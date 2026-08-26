// Package main is the entry point for the PatientTriage.ai server.
// This file is a placeholder that will be expanded in Ticket 6 (integration).
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/VirajMandavkar/PatientTriage/internal/db"
)

func main() {
	dbPath := "data/triage.db"
	if envPath := os.Getenv("TRIAGE_DB_PATH"); envPath != "" {
		dbPath = envPath
	}

	_, err := db.Init(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	fmt.Println("PatientTriage.ai — database initialized successfully")
	fmt.Println("Server implementation coming in Ticket 6...")
}
