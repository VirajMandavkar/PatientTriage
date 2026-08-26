package esi

import (
	"testing"

	"github.com/VirajMandavkar/PatientTriage/internal/models"
)

func TestScoreESI(t *testing.T) {
	tests := []struct {
		name           string
		chiefComplaint string
		hr             int
		rr             int
		spo2           float64
		sbp            int
		gcs            int
		pain           int
		expectedLevel  int
	}{
		{"1", "cardiac arrest", 0, 0, 0, 0, 3, 0, 1},
		{"2", "unresponsive", 40, 8, 75, 50, 3, 0, 1},
		{"3", "respiratory failure", 160, 45, 70, 70, 5, 0, 1},
		{"4", "major trauma", 110, 22, 95, 90, 12, 8, 1},
		{"5", "chest pain", 90, 18, 95, 130, 15, 7, 2},
		{"6", "severe headache", 80, 16, 98, 140, 10, 6, 2},
		{"7", "overdose", 70, 14, 97, 120, 14, 3, 2},
		{"8", "leg pain", 135, 20, 96, 130, 15, 9, 2},
		{"9", "abdominal pain", 85, 17, 97, 120, 15, 6, 3},
		{"10", "back pain with numbness", 75, 16, 98, 130, 15, 5, 3},
		{"11", "abdominal pain", 110, 22, 91, 120, 15, 6, 2},
		{"12", "ankle sprain", 75, 16, 99, 120, 15, 4, 4},
		{"13", "earache", 80, 18, 98, 125, 15, 3, 4},
		{"14", "medication refill", 70, 14, 99, 120, 15, 0, 5},
		{"15", "cold symptoms", 72, 15, 99, 125, 15, 1, 5},
		{"16", "mild headache", 68, 14, 99, 118, 15, 2, 5},
		{"17", "ankle sprain", 105, 22, 91, 120, 15, 4, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patient := models.Patient{
				ChiefComplaint: tt.chiefComplaint,
			}
			vitals := models.Vitals{
				HeartRate:       tt.hr,
				RespiratoryRate: tt.rr,
				SpO2:            tt.spo2,
				SystolicBP:      tt.sbp,
				GCS:             tt.gcs,
				PainLevel:       tt.pain,
			}
			level, rationale := ScoreESI(patient, vitals)
			if level != tt.expectedLevel {
				t.Errorf("ScoreESI() level = %v, want %v. Rationale: %s", level, tt.expectedLevel, rationale)
			}
			if rationale == "" {
				t.Errorf("ScoreESI() returned empty rationale")
			}
		})
	}
}
