// Package main is the entry point for PatientTriage.ai — an emergency department
// triage decision-support system. It wires up the HTTP server, SSE hub, queue
// supervisor, and all route handlers.
//
// The system scores, flags, and suggests — but never assigns. A human confirms
// every routing action, and every override is permanently logged.
package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/VirajMandavkar/PatientTriage/internal/caution"
	"github.com/VirajMandavkar/PatientTriage/internal/db"
	"github.com/VirajMandavkar/PatientTriage/internal/models"
	"github.com/VirajMandavkar/PatientTriage/internal/queue"
	"github.com/VirajMandavkar/PatientTriage/internal/sse"
	"github.com/VirajMandavkar/PatientTriage/internal/synth"
)

// Mock resources for the demo
var rooms = []models.Resource{
	{ID: "r1", Name: "Room 1", Type: "room", Available: true},
	{ID: "r2", Name: "Room 2", Type: "room", Available: true},
	{ID: "r3", Name: "Room 3", Type: "room", Available: true},
	{ID: "r4", Name: "Room 4", Type: "room", Available: false},
	{ID: "r5", Name: "Room 5", Type: "room", Available: true},
	{ID: "r6", Name: "Room 6", Type: "room", Available: false},
	{ID: "r7", Name: "Trauma Bay 1", Type: "room", Available: true},
	{ID: "r8", Name: "Trauma Bay 2", Type: "room", Available: true},
}

var doctors = []models.Resource{
	{ID: "d1", Name: "Dr. Patel", Type: "doctor", Available: true},
	{ID: "d2", Name: "Dr. Chen", Type: "doctor", Available: true},
	{ID: "d3", Name: "Dr. Williams", Type: "doctor", Available: false},
	{ID: "d4", Name: "Dr. Garcia", Type: "doctor", Available: true},
}

type App struct {
	db         *sql.DB
	hub        *sse.Hub
	supervisor *queue.Supervisor
	caution    *caution.Service
	templates  *template.Template
}

func main() {
	// Config
	dbPath := "data/triage.db"
	if envPath := os.Getenv("TRIAGE_DB_PATH"); envPath != "" {
		dbPath = envPath
	}
	port := "3000"
	if envPort := os.Getenv("TRIAGE_PORT"); envPort != "" {
		port = envPort
	}
	llmEndpoint := "http://localhost:8080"
	if envLLM := os.Getenv("LLM_ENDPOINT"); envLLM != "" {
		llmEndpoint = envLLM
	}

	// Initialize database
	database, err := db.Init(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize SSE hub
	hub := sse.NewHub()
	go hub.Run()

	// Initialize queue supervisor
	supervisor := queue.NewSupervisor(database, hub, models.DefaultMassCasualtyThreshold)
	go supervisor.Run()

	// Initialize caution flag service (graceful if LLM unavailable)
	cautionSvc, err := caution.NewService(llmEndpoint, "grammar/caution_flag.gbnf")
	if err != nil {
		log.Printf("[caution] Warning: Could not initialize caution service: %v", err)
		log.Printf("[caution] ESI scoring will continue without LLM caution flags")
	}

	// Parse templates
	tmpl := template.Must(template.ParseGlob("templates/*.html"))
	template.Must(tmpl.ParseGlob("templates/partials/*.html"))

	app := &App{
		db:         database,
		hub:        hub,
		supervisor: supervisor,
		caution:    cautionSvc,
		templates:  tmpl,
	}

	// Routes
	mux := http.NewServeMux()

	// Pages
	mux.HandleFunc("/", app.handleDashboard)
	mux.HandleFunc("/patients/new", app.handlePatientForm)
	mux.HandleFunc("/audit", app.handleAudit)

	// API
	mux.HandleFunc("/patients/", app.handlePatientActions)
	mux.HandleFunc("/api/seed", app.handleSeed)

	// SSE
	mux.Handle("/sse/queue", hub)

	// Static files
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	log.Printf("🏥 PatientTriage.ai starting on http://localhost:%s", port)
	log.Printf("   LLM endpoint: %s (available: %v)", llmEndpoint, cautionSvc != nil && cautionSvc.IsAvailable())
	log.Printf("   Mass-casualty threshold: %d patients", models.DefaultMassCasualtyThreshold)

	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// ============================================================================
// Handlers
// ============================================================================

func (app *App) handleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	queueEntries, err := app.supervisor.GetQueue()
	if err != nil {
		log.Printf("[dashboard] Error getting queue: %v", err)
		queueEntries = []models.QueueEntry{}
	}

	// Assign suggested rooms and doctors
	roomIdx := 0
	docIdx := 0
	for i := range queueEntries {
		for roomIdx < len(rooms) && !rooms[roomIdx].Available {
			roomIdx++
		}
		for docIdx < len(doctors) && !doctors[docIdx].Available {
			docIdx++
		}
		if roomIdx < len(rooms) {
			queueEntries[i].SuggestedRoom = rooms[roomIdx].Name
		}
		if docIdx < len(doctors) {
			queueEntries[i].SuggestedDoc = doctors[docIdx].Name
		}
		// Cycle through available resources
		roomIdx++
		if roomIdx >= len(rooms) {
			roomIdx = 0
		}
		docIdx++
		if docIdx >= len(doctors) {
			docIdx = 0
		}
	}

	var activeCount, criticalCount int
	var totalWait int
	for _, q := range queueEntries {
		activeCount++
		if q.CurrentScore.ESILevel <= 2 {
			criticalCount++
		}
		totalWait += q.WaitMinutes
	}
	avgWait := 0
	if activeCount > 0 {
		avgWait = totalWait / activeCount
	}

	data := map[string]interface{}{
		"Queue":         queueEntries,
		"Mode":          app.supervisor.GetMode(),
		"ActiveCount":   activeCount,
		"CriticalCount": criticalCount,
		"AvgWait":       avgWait,
		"Threshold":     models.DefaultMassCasualtyThreshold,
		"Rooms":         rooms,
		"Doctors":       doctors,
	}

	app.renderPage(w, "dashboard.html", data)
}

func (app *App) handlePatientForm(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		data := map[string]interface{}{
			"Mode": app.supervisor.GetMode(),
		}
		app.renderPage(w, "patient_form.html", data)
		return
	}

	// POST: Add new patient
	if err := r.ParseForm(); err != nil {
		app.renderAlert(w, "error", "Invalid form data")
		return
	}

	age, _ := strconv.Atoi(r.FormValue("age"))
	hr, _ := strconv.Atoi(r.FormValue("heart_rate"))
	rr, _ := strconv.Atoi(r.FormValue("respiratory_rate"))
	spo2, _ := strconv.ParseFloat(r.FormValue("spo2"), 64)
	sbp, _ := strconv.Atoi(r.FormValue("systolic_bp"))
	dbp, _ := strconv.Atoi(r.FormValue("diastolic_bp"))
	temp, _ := strconv.ParseFloat(r.FormValue("temperature"), 64)
	gcs, _ := strconv.Atoi(r.FormValue("gcs"))
	pain, _ := strconv.Atoi(r.FormValue("pain_level"))

	canWalk := r.FormValue("can_walk") == "true"
	isBreathing := r.FormValue("is_breathing") == "true"
	followsCommands := r.FormValue("follows_commands") == "true"

	// Insert patient
	result, err := app.db.Exec(
		`INSERT INTO patients (name, age, sex, chief_complaint, can_walk, is_breathing, follows_commands) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		r.FormValue("name"), age, r.FormValue("sex"), r.FormValue("chief_complaint"),
		canWalk, isBreathing, followsCommands,
	)
	if err != nil {
		app.renderAlert(w, "error", "Failed to register patient: "+err.Error())
		return
	}

	patientID, _ := result.LastInsertId()

	// Insert vitals
	_, err = app.db.Exec(
		`INSERT INTO vitals_history (patient_id, heart_rate, respiratory_rate, systolic_bp, diastolic_bp, spo2, temperature, gcs, pain_level) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		patientID, hr, rr, sbp, dbp, spo2, temp, gcs, pain,
	)
	if err != nil {
		app.renderAlert(w, "error", "Failed to record vitals: "+err.Error())
		return
	}

	// Trigger scoring via supervisor
	app.supervisor.Enqueue(queue.Event{Type: "patient_added", PatientID: patientID})

	// Trigger caution flag evaluation (async, non-blocking)
	if app.caution != nil {
		go func() {
			p := models.Patient{ID: patientID, ChiefComplaint: r.FormValue("chief_complaint")}
			v := []models.Vitals{{
				PatientID:       patientID,
				HeartRate:       hr,
				RespiratoryRate: rr,
				SpO2:            spo2,
				SystolicBP:      sbp,
				DiastolicBP:     dbp,
				Temperature:     temp,
				GCS:             gcs,
				PainLevel:       pain,
			}}
			app.caution.EvaluateAndStore(app.db, p, v)
		}()
	}

	app.renderAlert(w, "success", fmt.Sprintf("Patient %s registered and queued for triage (ID: %d)", r.FormValue("name"), patientID))
}

func (app *App) handleAudit(w http.ResponseWriter, r *http.Request) {
	type AuditEntry struct {
		models.Override
		PatientName string
	}

	rows, err := app.db.Query(`
		SELECT o.id, o.patient_id, o.suggested_action, o.chosen_action, o.overridden_by, 
		       COALESCE(o.override_reason, ''), o.created_at, COALESCE(p.name, 'Unknown')
		FROM overrides_log o
		LEFT JOIN patients p ON o.patient_id = p.id
		ORDER BY o.created_at DESC
	`)
	if err != nil {
		log.Printf("[audit] Error querying overrides: %v", err)
	}

	var overrides []AuditEntry
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var a AuditEntry
			err := rows.Scan(
				&a.ID, &a.PatientID, &a.SuggestedAction, &a.ChosenAction,
				&a.OverriddenBy, &a.OverrideReason, &a.CreatedAt, &a.PatientName,
			)
			if err == nil {
				overrides = append(overrides, a)
			}
		}
	}

	data := map[string]interface{}{
		"Mode":      app.supervisor.GetMode(),
		"Overrides": overrides,
	}
	app.renderPage(w, "audit.html", data)
}

func (app *App) handlePatientActions(w http.ResponseWriter, r *http.Request) {
	// Parse /patients/{id}/{action}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/patients/"), "/")
	if len(parts) < 2 {
		http.NotFound(w, r)
		return
	}

	patientID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	action := parts[1]

	switch action {
	case "confirm":
		app.handleConfirm(w, r, patientID)
	case "override":
		app.handleOverride(w, r, patientID)
	case "worsen":
		app.handleWorsen(w, r, patientID)
	case "vitals":
		app.handleVitalsUpdate(w, r, patientID)
	default:
		http.NotFound(w, r)
	}
}

func (app *App) handleConfirm(w http.ResponseWriter, r *http.Request, patientID int64) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	r.ParseForm()
	suggested := r.FormValue("suggested")

	// Log the confirmation (as a non-override)
	app.db.Exec(
		`INSERT INTO overrides_log (patient_id, suggested_action, chosen_action, overridden_by) VALUES (?, ?, ?, ?)`,
		patientID, suggested, suggested, "Triage Nurse",
	)

	// Mark patient as admitted
	app.db.Exec(`UPDATE patients SET status = 'admitted' WHERE id = ?`, patientID)
	app.supervisor.Enqueue(queue.Event{Type: "patient_discharged", PatientID: patientID})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *App) handleOverride(w http.ResponseWriter, r *http.Request, patientID int64) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	r.ParseForm()

	suggested := r.FormValue("suggested")
	chosenRoom := r.FormValue("chosen_action")
	chosenDoc := r.FormValue("chosen_doctor")
	chosen := chosenRoom
	if chosenDoc != "" {
		chosen = chosenRoom + " / " + chosenDoc
	}
	overriddenBy := r.FormValue("overridden_by")
	if overriddenBy == "" {
		overriddenBy = "Triage Nurse"
	}

	// Log the override — this is the load-bearing audit trail
	_, err := app.db.Exec(
		`INSERT INTO overrides_log (patient_id, suggested_action, chosen_action, overridden_by, override_reason) VALUES (?, ?, ?, ?, ?)`,
		patientID, suggested, chosen, overriddenBy, "Manual override",
	)
	if err != nil {
		log.Printf("[override] Error logging override: %v", err)
	}

	// Mark patient as admitted
	app.db.Exec(`UPDATE patients SET status = 'admitted' WHERE id = ?`, patientID)
	app.supervisor.Enqueue(queue.Event{Type: "patient_discharged", PatientID: patientID})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *App) handleWorsen(w http.ResponseWriter, r *http.Request, patientID int64) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get current vitals
	var v models.Vitals
	err := app.db.QueryRow(
		`SELECT heart_rate, respiratory_rate, systolic_bp, diastolic_bp, spo2, temperature, gcs, pain_level, capillary_refill FROM vitals_history WHERE patient_id = ? ORDER BY recorded_at DESC LIMIT 1`,
		patientID,
	).Scan(&v.HeartRate, &v.RespiratoryRate, &v.SystolicBP, &v.DiastolicBP, &v.SpO2, &v.Temperature, &v.GCS, &v.PainLevel, &v.CapillaryRefill)
	if err != nil {
		log.Printf("[worsen] Error getting vitals for patient %d: %v", patientID, err)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Get patient's current ESI level
	var esiLevel int
	app.db.QueryRow(`SELECT COALESCE(esi_level, 3) FROM scores WHERE patient_id = ? ORDER BY scored_at DESC LIMIT 1`, patientID).Scan(&esiLevel)

	// Generate worsened vitals
	gen := synth.NewGenerator(time.Now().UnixNano())
	v.PatientID = patientID
	worsened := gen.GenerateWorseningVitals(v, esiLevel)

	// Insert new vitals
	app.db.Exec(
		`INSERT INTO vitals_history (patient_id, heart_rate, respiratory_rate, systolic_bp, diastolic_bp, spo2, temperature, gcs, pain_level, capillary_refill) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		patientID, worsened.HeartRate, worsened.RespiratoryRate, worsened.SystolicBP, worsened.DiastolicBP,
		worsened.SpO2, worsened.Temperature, worsened.GCS, worsened.PainLevel, worsened.CapillaryRefill,
	)

	// Trigger rescore
	app.supervisor.Enqueue(queue.Event{Type: "vitals_update", PatientID: patientID})

	// Trigger caution flag re-evaluation (async)
	if app.caution != nil {
		go func() {
			var p models.Patient
			app.db.QueryRow(`SELECT id, chief_complaint FROM patients WHERE id = ?`, patientID).Scan(&p.ID, &p.ChiefComplaint)

			// Get vitals history for trend
			rows, err := app.db.Query(
				`SELECT heart_rate, respiratory_rate, spo2, systolic_bp, temperature, gcs, pain_level FROM vitals_history WHERE patient_id = ? ORDER BY recorded_at DESC LIMIT 5`,
				patientID,
			)
			if err != nil {
				return
			}
			defer rows.Close()

			var history []models.Vitals
			for rows.Next() {
				var vt models.Vitals
				rows.Scan(&vt.HeartRate, &vt.RespiratoryRate, &vt.SpO2, &vt.SystolicBP, &vt.Temperature, &vt.GCS, &vt.PainLevel)
				history = append(history, vt)
			}
			app.caution.EvaluateAndStore(app.db, p, history)
		}()
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *App) handleVitalsUpdate(w http.ResponseWriter, r *http.Request, patientID int64) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	r.ParseForm()

	hr, _ := strconv.Atoi(r.FormValue("heart_rate"))
	rr, _ := strconv.Atoi(r.FormValue("respiratory_rate"))
	spo2, _ := strconv.ParseFloat(r.FormValue("spo2"), 64)
	sbp, _ := strconv.Atoi(r.FormValue("systolic_bp"))
	dbp, _ := strconv.Atoi(r.FormValue("diastolic_bp"))
	temp, _ := strconv.ParseFloat(r.FormValue("temperature"), 64)
	gcs, _ := strconv.Atoi(r.FormValue("gcs"))
	pain, _ := strconv.Atoi(r.FormValue("pain_level"))

	app.db.Exec(
		`INSERT INTO vitals_history (patient_id, heart_rate, respiratory_rate, systolic_bp, diastolic_bp, spo2, temperature, gcs, pain_level) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		patientID, hr, rr, sbp, dbp, spo2, temp, gcs, pain,
	)

	app.supervisor.Enqueue(queue.Event{Type: "vitals_update", PatientID: patientID})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *App) handleSeed(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	gen := synth.NewGenerator(42) // Fixed seed for reproducibility
	counts := map[int]int{
		1: 1, // 1 ESI-1 patient
		2: 2, // 2 ESI-2 patients
		3: 3, // 3 ESI-3 patients
		4: 2, // 2 ESI-4 patients
		5: 2, // 2 ESI-5 patients
	}

	err := synth.SeedDatabase(app.db, gen, counts)
	if err != nil {
		log.Printf("[seed] Error seeding database: %v", err)
		http.Error(w, "Failed to seed database", http.StatusInternalServerError)
		return
	}

	// Trigger scoring for all new patients
	rows, err := app.db.Query(`SELECT id FROM patients WHERE status = 'active'`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var id int64
			rows.Scan(&id)
			app.supervisor.Enqueue(queue.Event{Type: "patient_added", PatientID: id})
		}
	}

	log.Printf("[seed] Database seeded with %d patients", 10)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// ============================================================================
// Template Helpers
// ============================================================================

func (app *App) renderPage(w http.ResponseWriter, pageFile string, data interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl, err := template.ParseFiles(
		"templates/layout.html",
		"templates/"+pageFile,
		"templates/partials/queue_row.html",
		"templates/partials/caution_badge.html",
		"templates/partials/mode_banner.html",
	)
	if err != nil {
		log.Printf("[render] Error parsing templates for %s: %v", pageFile, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	err = tmpl.ExecuteTemplate(w, "layout.html", data)
	if err != nil {
		log.Printf("[render] Error rendering %s: %v", pageFile, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (app *App) renderAlert(w http.ResponseWriter, alertType string, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<div class="alert alert-%s">%s</div>`, alertType, message)
}
