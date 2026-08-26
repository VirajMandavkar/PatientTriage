package caution

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/VirajMandavkar/PatientTriage/internal/models"
)

type LLMResponse struct {
	Flag       bool    `json:"flag"`
	Reason     string  `json:"reason"`
	Confidence float64 `json:"confidence"`
}

type Service struct {
	llmEndpoint string
	grammarStr  string
	httpClient  *http.Client
}

func NewService(llmEndpoint string, grammarPath string) (*Service, error) {
	b, err := os.ReadFile(grammarPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read grammar file: %w", err)
	}

	return &Service{
		llmEndpoint: llmEndpoint,
		grammarStr:  string(b),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

func (s *Service) EvaluateTrend(patient models.Patient, vitalsHistory []models.Vitals) (*LLMResponse, error) {
	prompt := fmt.Sprintf("Patient Chief Complaint: %s\n", patient.ChiefComplaint)
	for i, v := range vitalsHistory {
		prompt += fmt.Sprintf("Reading %d: HR:%d RR:%d BP:%d/%d SpO2:%.1f Temp:%.1f\n", i+1, v.HeartRate, v.RespiratoryRate, v.SystolicBP, v.DiastolicBP, v.SpO2, v.Temperature)
	}

	reqBody, _ := json.Marshal(map[string]interface{}{
		"prompt":      prompt,
		"grammar":     s.grammarStr,
		"n_predict":   200,
		"temperature": 0.3,
	})

	resp, err := s.httpClient.Post(s.llmEndpoint+"/completion", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var llamaResp struct {
		Content string `json:"content"`
	}
	if err := json.Unmarshal(body, &llamaResp); err == nil && llamaResp.Content != "" {
		return ValidateLLMResponse([]byte(llamaResp.Content))
	}

	return ValidateLLMResponse(body)
}

func (s *Service) EvaluateAndStore(db *sql.DB, patient models.Patient, vitalsHistory []models.Vitals) (*models.CautionFlag, error) {
	resp, err := s.EvaluateTrend(patient, vitalsHistory)
	
	cf := &models.CautionFlag{
		PatientID:    patient.ID,
		CreatedAt:    time.Now(),
	}

	if err != nil || resp == nil {
		cf.Flag = false
		cf.Reason = ""
		cf.Confidence = 0.0
		cf.LLMAvailable = false
	} else {
		cf.Flag = resp.Flag
		cf.Reason = resp.Reason
		cf.Confidence = resp.Confidence
		cf.LLMAvailable = true
	}

	// Only insert into caution_flags, no reference to other tables
	res, err := db.Exec("INSERT INTO caution_flags (patient_id, flag, reason, confidence, llm_available) VALUES (?, ?, ?, ?, ?)",
		cf.PatientID, cf.Flag, cf.Reason, cf.Confidence, cf.LLMAvailable)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	cf.ID = id

	return cf, nil
}

func ValidateLLMResponse(raw []byte) (*LLMResponse, error) {
	var resp LLMResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}
	if resp.Flag && resp.Reason == "" {
		return nil, errors.New("reason cannot be empty if flag is true")
	}
	if resp.Confidence < 0.0 || resp.Confidence > 1.0 {
		return nil, errors.New("confidence must be between 0.0 and 1.0")
	}
	return &resp, nil
}

func (s *Service) IsAvailable() bool {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(s.llmEndpoint + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
