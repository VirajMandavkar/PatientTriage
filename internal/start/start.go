package start

import (
	"fmt"

	"github.com/VirajMandavkar/PatientTriage/internal/models"
)

// ScoreSTART evaluates a patient using the START mass-casualty triage algorithm.
// Returns a color tag (RED/YELLOW/GREEN/BLACK) and rationale.
func ScoreSTART(patient models.Patient, vitals models.Vitals) (tag string, rationale string) {
	if patient.CanWalk {
		return "GREEN", "Minor: patient is ambulatory"
	}
	if !patient.IsBreathing && vitals.RespiratoryRate == 0 {
		return "BLACK", "Deceased/Expectant: not breathing after airway repositioning"
	}
	if vitals.RespiratoryRate > 30 {
		return "RED", fmt.Sprintf("Immediate: respiratory rate %d > 30", vitals.RespiratoryRate)
	}
	if vitals.CapillaryRefill > 2.0 {
		return "RED", fmt.Sprintf("Immediate: capillary refill %.1fs > 2s, poor perfusion", vitals.CapillaryRefill)
	}
	if !patient.FollowsCommands {
		return "RED", "Immediate: altered mental status, cannot follow commands"
	}
	return "YELLOW", "Delayed: breathing <30/min, perfusion adequate, follows commands"
}
