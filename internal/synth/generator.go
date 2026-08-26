package synth

import (
	"database/sql"
	"math/rand"
	"time"

	"github.com/VirajMandavkar/PatientTriage/internal/models"
)

var firstNames = []string{"James", "Mary", "John", "Patricia", "Robert", "Jennifer", "Michael", "Linda", "William", "Elizabeth", "David", "Barbara", "Richard", "Susan", "Joseph", "Jessica", "Thomas", "Sarah", "Charles", "Karen"}
var lastNames = []string{"Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia", "Miller", "Davis", "Rodriguez", "Martinez", "Hernandez", "Lopez", "Gonzalez", "Wilson", "Anderson", "Thomas", "Taylor", "Moore", "Jackson", "Martin"}

type Generator struct {
	r *rand.Rand
}

func NewGenerator(seed int64) *Generator {
	return &Generator{
		r: rand.New(rand.NewSource(seed)),
	}
}

func (g *Generator) GeneratePatient(esiLevel int) models.Patient {
	p := models.Patient{}
	firstName := firstNames[g.r.Intn(len(firstNames))]
	lastName := lastNames[g.r.Intn(len(lastNames))]
	p.Name = firstName + " " + lastName
	
	if g.r.Intn(2) == 0 {
		p.Sex = "M"
	} else {
		p.Sex = "F"
	}

	switch esiLevel {
	case 1:
		p.Age = g.r.Intn(60) + 18 // 18-77
		complaints := []string{"cardiac arrest", "respiratory failure", "major trauma", "anaphylaxis", "status epilepticus"}
		p.ChiefComplaint = complaints[g.r.Intn(len(complaints))]
		p.CanWalk = false
		p.IsBreathing = g.r.Intn(2) == 1
		p.FollowsCommands = false
	case 2:
		p.Age = g.r.Intn(70) + 20
		complaints := []string{"chest pain", "stroke symptoms", "severe asthma", "high fever with rash", "acute abdomen"}
		p.ChiefComplaint = complaints[g.r.Intn(len(complaints))]
		p.CanWalk = g.r.Intn(2) == 1
		p.IsBreathing = true
		p.FollowsCommands = true
	case 3:
		p.Age = g.r.Intn(60) + 30
		complaints := []string{"abdominal pain", "moderate laceration", "back pain with numbness", "persistent vomiting", "urinary symptoms with fever"}
		p.ChiefComplaint = complaints[g.r.Intn(len(complaints))]
		p.CanWalk = true
		p.IsBreathing = true
		p.FollowsCommands = true
	case 4:
		p.Age = g.r.Intn(50) + 20
		complaints := []string{"simple laceration", "ankle sprain", "earache", "mild rash", "minor burn"}
		p.ChiefComplaint = complaints[g.r.Intn(len(complaints))]
		p.CanWalk = true
		p.IsBreathing = true
		p.FollowsCommands = true
	case 5:
		p.Age = g.r.Intn(60) + 20
		complaints := []string{"medication refill", "cold symptoms", "mild headache", "insect bite", "suture removal"}
		p.ChiefComplaint = complaints[g.r.Intn(len(complaints))]
		p.CanWalk = true
		p.IsBreathing = true
		p.FollowsCommands = true
	default:
		p.Age = 40
		p.ChiefComplaint = "unknown"
		p.CanWalk = true
		p.IsBreathing = true
		p.FollowsCommands = true
	}

	return p
}

func (g *Generator) randFloat64(min, max float64) float64 {
	return min + g.r.Float64()*(max-min)
}

func (g *Generator) randInt(min, max int) int {
	return min + g.r.Intn(max-min+1)
}

func (g *Generator) GenerateVitals(patientID int64, esiLevel int) models.Vitals {
	v := models.Vitals{
		PatientID:  patientID,
		RecordedAt: time.Now(),
	}

	switch esiLevel {
	case 1:
		v.HeartRate = g.randInt(120, 180)
		v.RespiratoryRate = g.randInt(30, 50)
		v.SpO2 = g.randFloat64(70, 85)
		v.SystolicBP = g.randInt(60, 80)
		v.DiastolicBP = g.randInt(60, 90)
		v.Temperature = g.randFloat64(39, 41)
		v.PainLevel = g.randInt(9, 10)
		v.GCS = g.randInt(3, 8)
		v.CapillaryRefill = g.randFloat64(2.5, 4.0)
	case 2:
		v.HeartRate = g.randInt(100, 140)
		v.RespiratoryRate = g.randInt(24, 35)
		v.SpO2 = g.randFloat64(85, 92)
		v.SystolicBP = g.randInt(80, 100)
		v.DiastolicBP = g.randInt(60, 90)
		v.Temperature = g.randFloat64(38.5, 40)
		v.PainLevel = g.randInt(7, 9)
		v.GCS = g.randInt(9, 13)
		v.CapillaryRefill = g.randFloat64(2.0, 3.0)
	case 3:
		v.HeartRate = g.randInt(80, 120)
		v.RespiratoryRate = g.randInt(18, 24)
		v.SpO2 = g.randFloat64(92, 96)
		v.SystolicBP = g.randInt(100, 140)
		v.DiastolicBP = g.randInt(60, 90)
		v.Temperature = g.randFloat64(37.5, 39)
		v.PainLevel = g.randInt(5, 7)
		v.GCS = g.randInt(14, 15)
		v.CapillaryRefill = g.randFloat64(1.0, 2.0)
	case 4:
		v.HeartRate = g.randInt(60, 100)
		v.RespiratoryRate = g.randInt(14, 20)
		v.SpO2 = g.randFloat64(96, 99)
		v.SystolicBP = g.randInt(110, 140)
		v.DiastolicBP = g.randInt(60, 90)
		v.Temperature = g.randFloat64(36.5, 38)
		v.PainLevel = g.randInt(3, 5)
		v.GCS = 15
		v.CapillaryRefill = g.randFloat64(1.0, 2.0)
	case 5:
		v.HeartRate = g.randInt(60, 90)
		v.RespiratoryRate = g.randInt(12, 18)
		v.SpO2 = g.randFloat64(97, 100)
		v.SystolicBP = g.randInt(110, 130)
		v.DiastolicBP = g.randInt(60, 90)
		v.Temperature = g.randFloat64(36, 37.5)
		v.PainLevel = g.randInt(0, 3)
		v.GCS = 15
		v.CapillaryRefill = g.randFloat64(1.0, 2.0)
	default:
		v.HeartRate = g.randInt(60, 90)
		v.RespiratoryRate = g.randInt(12, 18)
		v.SpO2 = g.randFloat64(97, 100)
		v.SystolicBP = g.randInt(110, 130)
		v.DiastolicBP = g.randInt(60, 90)
		v.Temperature = g.randFloat64(36, 37.5)
		v.PainLevel = 0
		v.GCS = 15
		v.CapillaryRefill = 1.0
	}

	return v
}

func (g *Generator) GenerateWorseningVitals(current models.Vitals, esiLevel int) models.Vitals {
	next := current
	next.RecordedAt = time.Now()

	if esiLevel > 1 {
		next.HeartRate += g.randInt(10, 20)
		next.SpO2 -= g.randFloat64(2, 4)
		next.RespiratoryRate += g.randInt(3, 6)
		next.SystolicBP -= g.randInt(10, 20)
		next.GCS -= g.randInt(0, 2)
		if next.GCS < 3 {
			next.GCS = 3
		}
	}
	return next
}

func SeedDatabase(db *sql.DB, gen *Generator, counts map[int]int) error {
	for esiLevel, count := range counts {
		for i := 0; i < count; i++ {
			p := gen.GeneratePatient(esiLevel)
			
			res, err := db.Exec(
				`INSERT INTO patients (name, age, sex, chief_complaint, can_walk, is_breathing, follows_commands) 
				VALUES (?, ?, ?, ?, ?, ?, ?)`,
				p.Name, p.Age, p.Sex, p.ChiefComplaint, p.CanWalk, p.IsBreathing, p.FollowsCommands,
			)
			if err != nil {
				return err
			}
			patientID, err := res.LastInsertId()
			if err != nil {
				return err
			}

			v := gen.GenerateVitals(patientID, esiLevel)
			_, err = db.Exec(
				`INSERT INTO vitals_history (patient_id, heart_rate, respiratory_rate, spo2, systolic_bp, diastolic_bp, temperature, pain_level, gcs, capillary_refill, recorded_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				v.PatientID, v.HeartRate, v.RespiratoryRate, v.SpO2, v.SystolicBP, v.DiastolicBP, v.Temperature, v.PainLevel, v.GCS, v.CapillaryRefill, v.RecordedAt,
			)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
