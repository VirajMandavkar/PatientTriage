package queue

import (
	"bytes"
	"database/sql"
	"html/template"
	"time"

	"github.com/VirajMandavkar/PatientTriage/internal/esi"
	"github.com/VirajMandavkar/PatientTriage/internal/models"
	"github.com/VirajMandavkar/PatientTriage/internal/sse"
	"github.com/VirajMandavkar/PatientTriage/internal/start"
)

type Supervisor struct {
	db        *sql.DB
	hub       *sse.Hub
	events    chan Event
	mode      string
	threshold int
	templates *template.Template
}

type Event struct {
	Type      string
	PatientID int64
}

func NewSupervisor(db *sql.DB, hub *sse.Hub, threshold int) *Supervisor {
	tmpl := template.Must(template.ParseFiles(
		"templates/partials/queue_row.html",
		"templates/partials/caution_badge.html",
		"templates/partials/mode_banner.html",
	))
	return &Supervisor{
		db:        db,
		hub:       hub,
		events:    make(chan Event, 100),
		mode:      "NORMAL",
		threshold: threshold,
		templates: tmpl,
	}
}

func (s *Supervisor) Enqueue(event Event) {
	s.events <- event
}

func (s *Supervisor) GetMode() string {
	return s.mode
}

func (s *Supervisor) Run() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case event := <-s.events:
			s.handleEvent(event)
		case <-ticker.C:
			s.handleEvent(Event{Type: "tick"})
		}
	}
}

func (s *Supervisor) handleEvent(event Event) {
	switch event.Type {
	case "vitals_update":
		s.rescorePatient(event.PatientID)
		s.broadcastPatientRow(event.PatientID)
	case "patient_added":
		s.rescorePatient(event.PatientID)
		s.CheckModeSwitch()
		s.broadcastPatientRow(event.PatientID)
	case "patient_discharged":
		s.CheckModeSwitch()
	case "tick":
		s.broadcastAllRows()
	}
}

func (s *Supervisor) rescorePatient(patientID int64) {
	var p models.Patient
	err := s.db.QueryRow("SELECT id, name, age, sex, chief_complaint, arrival_time, status, can_walk, is_breathing, follows_commands FROM patients WHERE id = ?", patientID).
		Scan(&p.ID, &p.Name, &p.Age, &p.Sex, &p.ChiefComplaint, &p.ArrivalTime, &p.Status, &p.CanWalk, &p.IsBreathing, &p.FollowsCommands)
	if err != nil {
		return
	}

	var v models.Vitals
	err = s.db.QueryRow("SELECT id, patient_id, heart_rate, respiratory_rate, systolic_bp, diastolic_bp, spo2, temperature, gcs, pain_level, capillary_refill, recorded_at FROM vitals_history WHERE patient_id = ? ORDER BY recorded_at DESC LIMIT 1", patientID).
		Scan(&v.ID, &v.PatientID, &v.HeartRate, &v.RespiratoryRate, &v.SystolicBP, &v.DiastolicBP, &v.SpO2, &v.Temperature, &v.GCS, &v.PainLevel, &v.CapillaryRefill, &v.RecordedAt)

	var esiLevel int = 5
	var rationale string = "No vitals"
	var startTag string = ""

	if err != sql.ErrNoRows {
		if s.mode == "START" {
			startTag, rationale = start.ScoreSTART(p, v)
			switch startTag {
			case "RED":
				esiLevel = 1
			case "YELLOW":
				esiLevel = 2
			case "GREEN":
				esiLevel = 3
			case "BLACK":
				esiLevel = 5
			}
		} else {
			esiLevel, rationale = esi.ScoreESI(p, v)
		}
	}

	s.db.Exec("INSERT INTO scores (patient_id, esi_level, rationale, start_tag) VALUES (?, ?, ?, ?)",
		patientID, esiLevel, rationale, sql.NullString{String: startTag, Valid: startTag != ""})
}

func (s *Supervisor) CheckModeSwitch() {
	var count int
	s.db.QueryRow("SELECT COUNT(*) FROM patients WHERE status = 'active'").Scan(&count)

	newMode := "NORMAL"
	if count >= s.threshold {
		newMode = "START"
	}

	if s.mode != newMode {
		s.mode = newMode
		s.db.Exec("UPDATE mode_state SET mode = ?, switched_at = CURRENT_TIMESTAMP WHERE id = 1", s.mode)
		s.broadcastMode()
		s.rescoreAllActivePatients()
	}
}

func (s *Supervisor) rescoreAllActivePatients() {
	rows, err := s.db.Query("SELECT id FROM patients WHERE status = 'active'")
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		rows.Scan(&id)
		s.rescorePatient(id)
		s.broadcastPatientRow(id)
	}
}

func (s *Supervisor) GetQueue() ([]models.QueueEntry, error) {
	query := `
	SELECT p.id, p.name, p.age, p.sex, p.chief_complaint, p.arrival_time, p.status, p.can_walk, p.is_breathing, p.follows_commands,
	       COALESCE(s.esi_level, 5), COALESCE(s.rationale, ''), COALESCE(s.start_tag, ''), COALESCE(s.scored_at, p.arrival_time),
	       COALESCE(c.flag, 0), COALESCE(c.reason, ''), COALESCE(c.llm_available, 1)
	FROM patients p
	LEFT JOIN (
	    SELECT patient_id, esi_level, rationale, start_tag, scored_at
	    FROM scores s1
	    WHERE scored_at = (SELECT MAX(scored_at) FROM scores s2 WHERE s2.patient_id = s1.patient_id)
	) s ON p.id = s.patient_id
	LEFT JOIN (
	    SELECT patient_id, flag, reason, llm_available
	    FROM caution_flags c1
	    WHERE created_at = (SELECT MAX(created_at) FROM caution_flags c2 WHERE c2.patient_id = c1.patient_id)
	) c ON p.id = c.patient_id
	WHERE p.status = 'active'
	ORDER BY COALESCE(s.esi_level, 5) ASC, p.arrival_time ASC
	`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var queue []models.QueueEntry
	for rows.Next() {
		var q models.QueueEntry
		var startTag sql.NullString
		err := rows.Scan(
			&q.Patient.ID, &q.Patient.Name, &q.Patient.Age, &q.Patient.Sex, &q.Patient.ChiefComplaint, &q.Patient.ArrivalTime, &q.Patient.Status, &q.Patient.CanWalk, &q.Patient.IsBreathing, &q.Patient.FollowsCommands,
			&q.CurrentScore.ESILevel, &q.CurrentScore.Rationale, &startTag, &q.CurrentScore.ScoredAt,
			&q.CautionFlag.Flag, &q.CautionFlag.Reason, &q.CautionFlag.LLMAvailable,
		)
		if err == nil {
			q.CurrentScore.StartTag = startTag.String
			q.WaitMinutes = int(time.Since(q.Patient.ArrivalTime).Minutes())
			queue = append(queue, q)
		}
	}

	return queue, nil
}

func (s *Supervisor) getQueueEntry(patientID int64) (models.QueueEntry, error) {
	queue, err := s.GetQueue()
	if err != nil {
		return models.QueueEntry{}, err
	}
	for _, q := range queue {
		if q.Patient.ID == patientID {
			return q, nil
		}
	}
	return models.QueueEntry{}, sql.ErrNoRows
}

func (s *Supervisor) broadcastPatientRow(patientID int64) {
	q, err := s.getQueueEntry(patientID)
	if err != nil {
		return
	}
	var buf bytes.Buffer
	err = s.templates.ExecuteTemplate(&buf, "queue_row.html", q)
	if err == nil {
		s.hub.Broadcast("queue-update", buf.String())
	}
}

func (s *Supervisor) broadcastAllRows() {
	queue, err := s.GetQueue()
	if err != nil {
		return
	}
	for _, q := range queue {
		var buf bytes.Buffer
		err = s.templates.ExecuteTemplate(&buf, "queue_row.html", q)
		if err == nil {
			s.hub.Broadcast("queue-update", buf.String())
		}
	}
}

func (s *Supervisor) broadcastMode() {
	var buf bytes.Buffer
	var count int
	s.db.QueryRow("SELECT COUNT(*) FROM patients WHERE status = 'active'").Scan(&count)
	data := struct {
		Mode        string
		ActiveCount int
	}{
		Mode:        s.mode,
		ActiveCount: count,
	}
	s.templates.ExecuteTemplate(&buf, "mode_banner.html", data)
	s.hub.Broadcast("mode-update", buf.String())
}
