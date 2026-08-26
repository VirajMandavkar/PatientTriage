package start

import (
	"testing"

	"github.com/VirajMandavkar/PatientTriage/internal/models"
)

func TestScoreSTART(t *testing.T) {
	tests := []struct {
		name            string
		canWalk         bool
		isBreathing     bool
		followsCommands bool
		rr              int
		capRefill       float64
		expected        string
	}{
		{"1", true, true, true, 20, 1.5, "GREEN"},
		{"2", false, false, false, 0, 0, "BLACK"},
		{"3", false, true, true, 35, 1.5, "RED"},
		{"4", false, true, true, 25, 3.0, "RED"},
		{"5", false, true, false, 20, 1.5, "RED"},
		{"6", false, true, true, 25, 1.5, "YELLOW"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patient := models.Patient{
				CanWalk:         tt.canWalk,
				IsBreathing:     tt.isBreathing,
				FollowsCommands: tt.followsCommands,
			}
			vitals := models.Vitals{
				RespiratoryRate: tt.rr,
				CapillaryRefill: tt.capRefill,
			}
			tag, rationale := ScoreSTART(patient, vitals)
			if tag != tt.expected {
				t.Errorf("ScoreSTART() tag = %v, want %v. Rationale: %s", tag, tt.expected, rationale)
			}
			if rationale == "" {
				t.Errorf("ScoreSTART() returned empty rationale")
			}
		})
	}
}
