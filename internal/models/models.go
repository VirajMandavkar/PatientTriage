// Package models defines shared data types used across the PatientTriage system.
// These structs map directly to the SQLite schema tables and are used by all
// subsystems (ESI scoring, caution flags, queue management, etc.).
package models

import "time"

// Patient represents a patient in the ED triage system.
type Patient struct {
	ID              int64     `json:"id"`
	Name            string    `json:"name"`
	Age             int       `json:"age"`
	Sex             string    `json:"sex"`
	ChiefComplaint  string    `json:"chief_complaint"`
	ArrivalTime     time.Time `json:"arrival_time"`
	Status          string    `json:"status"` // "active", "discharged", "admitted"
	CanWalk         bool      `json:"can_walk"`
	IsBreathing     bool      `json:"is_breathing"`
	FollowsCommands bool      `json:"follows_commands"`
}

// Vitals represents a single vitals reading for a patient.
type Vitals struct {
	ID              int64     `json:"id"`
	PatientID       int64     `json:"patient_id"`
	HeartRate       int       `json:"heart_rate"`
	RespiratoryRate int       `json:"respiratory_rate"`
	SystolicBP      int       `json:"systolic_bp"`
	DiastolicBP     int       `json:"diastolic_bp"`
	SpO2            float64   `json:"spo2"`
	Temperature     float64   `json:"temperature"`
	GCS             int       `json:"gcs"`
	PainLevel       int       `json:"pain_level"`
	CapillaryRefill float64   `json:"capillary_refill"`
	RecordedAt      time.Time `json:"recorded_at"`
}

// Score represents an ESI triage score for a patient.
type Score struct {
	ID       int64     `json:"id"`
	PatientID int64    `json:"patient_id"`
	ESILevel int       `json:"esi_level"` // 1-5
	Rationale string   `json:"rationale"`
	StartTag  string   `json:"start_tag,omitempty"` // RED, YELLOW, GREEN, BLACK (only in START mode)
	ScoredAt  time.Time `json:"scored_at"`
}

// CautionFlag represents an LLM-generated caution flag for a patient.
// This is a secondary advisory signal — it can NEVER modify the ESI score.
type CautionFlag struct {
	ID           int64     `json:"id"`
	PatientID    int64     `json:"patient_id"`
	Flag         bool      `json:"flag"`
	Reason       string    `json:"reason"`
	Confidence   float64   `json:"confidence"`
	LLMAvailable bool      `json:"llm_available"`
	CreatedAt    time.Time `json:"created_at"`
}

// Override represents a logged override of a system suggestion.
// Every time a human overrides the suggested routing, both the original
// suggestion and the chosen alternative are permanently recorded.
type Override struct {
	ID              int64     `json:"id"`
	PatientID       int64     `json:"patient_id"`
	SuggestedAction string    `json:"suggested_action"`
	ChosenAction    string    `json:"chosen_action"`
	OverriddenBy    string    `json:"overridden_by"`
	OverrideReason  string    `json:"override_reason"`
	CreatedAt       time.Time `json:"created_at"`
}

// ModeState represents the current operating mode of the triage system.
type ModeState struct {
	ID         int    `json:"id"`
	Mode       string `json:"mode"`      // "NORMAL" or "START"
	Threshold  int    `json:"threshold"`  // Number of active patients to trigger START mode
	SwitchedAt time.Time `json:"switched_at"`
	SwitchedBy string `json:"switched_by"`
}

// Masscasualties threshold default — triggers START mode when active patient count >= this.
const DefaultMassCasualtyThreshold = 5

// QueueEntry is the composite view of a patient in the live queue, combining
// patient data, latest vitals, current score, caution flag status, and suggested action.
type QueueEntry struct {
	Patient       Patient     `json:"patient"`
	LatestVitals  Vitals      `json:"latest_vitals"`
	CurrentScore  Score       `json:"current_score"`
	CautionFlag   CautionFlag `json:"caution_flag"`
	WaitMinutes   int         `json:"wait_minutes"`
	SuggestedRoom string      `json:"suggested_room"`
	SuggestedDoc  string      `json:"suggested_doctor"`
}

// Resource represents a room or doctor in the availability panel.
type Resource struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"` // "room" or "doctor"
	Available bool   `json:"available"`
}
