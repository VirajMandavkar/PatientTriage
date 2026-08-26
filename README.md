# PatientTriage.ai — Emergency Department Triage Decision Support

**PatientTriage.ai** is an emergency-department (ED) triage decision-support tool built to score patient acuity, flag clinical risks, and suggest initial routing actions while keeping **human clinicians in total control**.

> [!IMPORTANT]
> **Core Constraint: Suggest, Never Assign.**
> PatientTriage.ai never automatically assigns a patient to a room or physician. Every suggested action requires a **one-click human confirmation**. Any clinician override of a system suggestion is **permanently recorded in an immutable audit log** (who, when, what was suggested vs. chosen).

---

## Key Features

1. **Deterministic Primary Scoring (ESI Algorithm)**
   - Implements the 5-level **Emergency Severity Index (ESI)** decision tree (Immediate / Emergent / Urgent / Semi-urgent / Non-urgent).
   - **Zero ML in the critical path** — deterministic, fully explainable, with explicit rationale for every score decision branch taken.

2. **Secondary Local LLM Caution Flags (GBNF Constrained)**
   - Runs `llama-server` with `Qwen2.5-3B-Instruct-GGUF` locally on CPU.
   - Evaluates vitals trend windows (last N readings) + chief complaint to raise secondary caution flags.
   - Output is token-level constrained via a **GBNF grammar** to `{"flag": bool, "reason": string, "confidence": float}`.
   - **Architectural Isolation**: The caution-flag service has **zero write permissions or code access** to the ESI scores table. It can never change, hide, or override the ESI level.

3. **Mass-Casualty START Triage Mode**
   - Automatically switches from Normal mode into **START mode** (Simple Triage And Rapid Treatment) when active patient volume reaches/crosses a configurable threshold (default: 5 patients).
   - Tags patients into standard RPM categories (**RED** / **YELLOW** / **GREEN** / **BLACK**).
   - Mode state changes are logged and dynamically highlighted via a top-level banner.

4. **Live Re-scoring Queue & SSE Streaming**
   - Live queue powered by Go goroutines and channels (`queue.Supervisor`).
   - Re-evaluates patient priority continuously as new vitals arrive or wait times increment.
   - Streams updates via Server-Sent Events (SSE) using **htmx** (`hx-sse`) to swap updated table rows seamlessly without page reloads.

5. **Load-Bearing Audit Log Page**
   - Dedicated `/audit` page displaying every human clinician override.
   - Displays timestamp, patient, original system suggestion, nurse's chosen alternative, and override reason.

---

## Architecture Diagram

```mermaid
graph TD
    User["Clinician / Nurse (Browser Dashboard)"]
    Router["Go HTTP Router / Handler"]
    Queue["Queue Supervisor Loop (Goroutine)"]
    SSE["SSE Broadcast Hub"]
    ESI["ESI Scoring Engine (Pure Go)"]
    START["START Triage Engine (Pure Go)"]
    Caution["Caution Service (LLM Client)"]
    LLM["llama-server (Qwen2.5-3B + GBNF Grammar)"]
    DB[("SQLite (triage.db)")]

    User -->|"HTTP POST /confirm or /override"| Router
    User <--|"SSE Stream (htmx queue-update)"| SSE
    Router -->|"Record Override / Confirm"| DB
    Router -->|"Enqueue Vitals Event"| Queue
    Queue -->|"Read Vitals History"| DB
    Queue -->|"Score (Normal Mode)"| ESI
    Queue -->|"Score (START Mode)"| START
    Queue -->|"Store Score"| DB
    Queue -->|"Broadcast Row HTML"| SSE
    Router -->|"Async Trend Eval"| Caution
    Caution -->|"HTTP POST /completion (GBNF)"| LLM
    LLM -->|"Strict JSON Response"| Caution
    Caution -->|"Store Caution Flag"| DB
```

---

## Database Schema Overview

```sql
patients (id, name, age, sex, chief_complaint, arrival_time, status, can_walk, is_breathing, follows_commands)
vitals_history (id, patient_id, heart_rate, respiratory_rate, systolic_bp, diastolic_bp, spo2, temperature, gcs, pain_level, capillary_refill, recorded_at)
scores (id, patient_id, esi_level, rationale, start_tag, scored_at)
caution_flags (id, patient_id, flag, reason, confidence, llm_available, created_at)
overrides_log (id, patient_id, suggested_action, chosen_action, overridden_by, override_reason, created_at)
mode_state (id, mode, threshold, switched_at, switched_by)
```

---

## How to Run Locally

### Prerequisites
- **Go**: 1.22+
- **Build Tools**: `cmake`, `make`, `g++` (for compiling `llama.cpp` CPU binary)

### 1. Build llama.cpp and download model (Automated script available)
```bash
# Build llama.cpp and download Qwen2.5-3B-Instruct GGUF
cd /home/kernelghost/Desktop/GO
git clone --depth 1 https://github.com/ggerganov/llama.cpp.git
cd llama.cpp
cmake -B build -DGGML_CUDA=OFF -DGGML_METAL=OFF
cmake --build build --config Release -j$(nproc)

mkdir -p /home/kernelghost/Desktop/GO/models
wget -O /home/kernelghost/Desktop/GO/models/qwen2.5-3b-instruct-q4_k_m.gguf \
  https://huggingface.co/Qwen/Qwen2.5-3B-Instruct-GGUF/resolve/main/qwen2.5-3b-instruct-q4_k_m.gguf
```

### 2. Start the Local LLM Server
```bash
./scripts/start_llm.sh
```
*Serves `llama-server` on `http://localhost:8080` with GBNF grammar support.*

### 3. Start the PatientTriage.ai Application
```bash
go run .
```
*App starts on `http://localhost:3000`.*

### 4. Seed Demo Data
Navigate to `http://localhost:3000` in your browser and click **"Seed Demo Data"** to populate active patients across all ESI levels (1–5).

---

## Verification & Testing

Run full unit test suite (ESI 17-vector decision tree, START 6-vector triage, Caution isolation & adversarial prompt injection):

```bash
go test ./... -v
```

---

## License

Developed for Hackathon Project Demo. Designed for 48h decision-support proof of concept.
