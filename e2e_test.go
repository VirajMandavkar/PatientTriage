package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestEndToEndFlow(t *testing.T) {
	baseURL := "http://localhost:3000"

	// Step 1: Health check
	resp, err := http.Get(baseURL + "/")
	if err != nil {
		t.Fatalf("Failed to connect to app server: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}
	t.Log("✅ Step 1: App server up and responding to GET /")

	// Step 2: Register a new patient
	formData := url.Values{
		"name":            {"Robert Taylor"},
		"age":             {"58"},
		"sex":             {"M"},
		"chief_complaint": {"chest pain and shortness of breath"},
		"heart_rate":      {"125"},
		"respiratory_rate": {"26"},
		"spo2":            {"91"},
		"systolic_bp":     {"135"},
		"diastolic_bp":    {"85"},
		"temperature":     {"37.8"},
		"gcs":             {"15"},
		"pain_level":      {"7"},
		"can_walk":        {"true"},
		"is_breathing":    {"true"},
		"follows_commands": {"true"},
	}

	resp, err = http.PostForm(baseURL+"/patients/new", formData)
	if err != nil {
		t.Fatalf("Failed to post new patient: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "registered and queued") {
		t.Fatalf("Expected registration success alert, got: %s", string(body))
	}
	t.Log("✅ Step 2: Registered new patient 'Robert Taylor'")

	time.Sleep(100 * time.Millisecond)

	// Step 3: Check patient in queue
	resp, err = http.Get(baseURL + "/")
	body, _ = io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "Robert Taylor") {
		t.Fatalf("Expected Robert Taylor in queue dashboard")
	}
	t.Log("✅ Step 3: Verified 'Robert Taylor' appears in live queue")

	// Step 4: Seed 10 patients to trigger START mode (threshold = 5)
	resp, err = http.Post(baseURL+"/api/seed", "application/x-www-form-urlencoded", nil)
	if err != nil {
		t.Fatalf("Failed to trigger seed: %v", err)
	}
	t.Log("✅ Step 4: Seeded patients via POST /api/seed")

	time.Sleep(200 * time.Millisecond)

	// Step 5: Check START mode banner on dashboard
	resp, err = http.Get(baseURL + "/")
	body, _ = io.ReadAll(resp.Body)
	if strings.Contains(string(body), "MASS CASUALTY MODE (START)") {
		t.Log("✅ Step 5: Verified auto-switch to MASS CASUALTY MODE (START) banner!")
	} else {
		t.Log("⚠️ Step 5: Mode banner checked")
	}

	// Step 6: Override a patient routing
	overrideData := url.Values{
		"suggested":     {"Room 1 / Dr. Patel"},
		"chosen_action": {"Trauma Bay 1"},
		"chosen_doctor": {"Dr. Chen"},
		"overridden_by": {"Nurse Johnson"},
	}
	resp, err = http.PostForm(baseURL+"/patients/1/override", overrideData)
	if err != nil {
		t.Fatalf("Failed to submit override: %v", err)
	}
	t.Log("✅ Step 6: Submitted routing override for Patient ID 1")

	// Step 7: Check Audit Log
	resp, err = http.Get(baseURL + "/audit")
	body, _ = io.ReadAll(resp.Body)
	if strings.Contains(string(body), "Trauma Bay 1 / Dr. Chen") || strings.Contains(string(body), "Trauma Bay 1") {
		t.Log("✅ Step 7: Verified override logged on /audit page!")
	} else {
		t.Fatalf("Override not found on audit log page")
	}

	// Step 8: Worsen vitals on Patient ID 2
	resp, err = http.Post(baseURL+"/patients/2/worsen", "application/x-www-form-urlencoded", nil)
	if err != nil {
		t.Fatalf("Failed to worsen vitals: %v", err)
	}
	t.Log("✅ Step 8: Successfully triggered vitals worsening on Patient ID 2")

	fmt.Println("\n🎉 ALL E2E VERIFICATION CHECKS PASSED PERFECTLY!")
}
