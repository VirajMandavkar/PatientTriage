-- PatientTriage.ai SQLite Schema
-- All tables for the triage decision-support system.

CREATE TABLE IF NOT EXISTS patients (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    age INTEGER NOT NULL,
    sex TEXT NOT NULL CHECK(sex IN ('M','F','O')),
    chief_complaint TEXT NOT NULL,
    arrival_time DATETIME DEFAULT CURRENT_TIMESTAMP,
    status TEXT NOT NULL DEFAULT 'active' CHECK(status IN ('active','discharged','admitted')),
    can_walk BOOLEAN DEFAULT 1,
    is_breathing BOOLEAN DEFAULT 1,
    follows_commands BOOLEAN DEFAULT 1
);

CREATE TABLE IF NOT EXISTS vitals_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    patient_id INTEGER NOT NULL REFERENCES patients(id),
    heart_rate INTEGER,
    respiratory_rate INTEGER,
    systolic_bp INTEGER,
    diastolic_bp INTEGER,
    spo2 REAL,
    temperature REAL,
    gcs INTEGER DEFAULT 15,
    pain_level INTEGER DEFAULT 0,
    capillary_refill REAL DEFAULT 1.5,
    recorded_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS scores (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    patient_id INTEGER NOT NULL REFERENCES patients(id),
    esi_level INTEGER NOT NULL CHECK(esi_level BETWEEN 1 AND 5),
    rationale TEXT NOT NULL,
    start_tag TEXT CHECK(start_tag IN ('RED','YELLOW','GREEN','BLACK') OR start_tag IS NULL),
    scored_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS caution_flags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    patient_id INTEGER NOT NULL REFERENCES patients(id),
    flag BOOLEAN NOT NULL,
    reason TEXT,
    confidence REAL,
    llm_available BOOLEAN DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS overrides_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    patient_id INTEGER NOT NULL REFERENCES patients(id),
    suggested_action TEXT NOT NULL,
    chosen_action TEXT NOT NULL,
    overridden_by TEXT NOT NULL,
    override_reason TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS mode_state (
    id INTEGER PRIMARY KEY CHECK(id = 1),
    mode TEXT NOT NULL DEFAULT 'NORMAL' CHECK(mode IN ('NORMAL','START')),
    threshold INTEGER NOT NULL DEFAULT 5,
    switched_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    switched_by TEXT DEFAULT 'system'
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_vitals_patient ON vitals_history(patient_id, recorded_at DESC);
CREATE INDEX IF NOT EXISTS idx_scores_patient ON scores(patient_id, scored_at DESC);
CREATE INDEX IF NOT EXISTS idx_caution_patient ON caution_flags(patient_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_overrides_time ON overrides_log(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_patients_status ON patients(status);
