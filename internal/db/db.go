// Package db handles SQLite database initialization, schema migration,
// and provides query helpers for the PatientTriage system.
package db

import (
	"database/sql"
	_ "embed"
	"fmt"
	"log"
	"sync"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

var (
	instance *sql.DB
	once     sync.Once
)

// Init opens (or creates) the SQLite database at the given path and runs
// the schema migration. It is safe to call multiple times — only the first
// call takes effect.
func Init(dbPath string) (*sql.DB, error) {
	var initErr error
	once.Do(func() {
		var err error
		// Use modernc.org/sqlite — pure Go, no CGO needed
		instance, err = sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on")
		if err != nil {
			initErr = fmt.Errorf("open database: %w", err)
			return
		}

		// Set connection pool settings suitable for SQLite
		instance.SetMaxOpenConns(1) // SQLite only supports one writer
		instance.SetMaxIdleConns(1)

		// Run schema migration
		if _, err := instance.Exec(schemaSQL); err != nil {
			initErr = fmt.Errorf("run schema migration: %w", err)
			return
		}

		// Seed the mode_state singleton row if it doesn't exist
		_, err = instance.Exec(`INSERT OR IGNORE INTO mode_state (id, mode, threshold) VALUES (1, 'NORMAL', 5)`)
		if err != nil {
			initErr = fmt.Errorf("seed mode_state: %w", err)
			return
		}

		log.Printf("[db] Database initialized at %s", dbPath)
	})
	return instance, initErr
}

// Get returns the initialized database instance. Panics if Init has not been called.
func Get() *sql.DB {
	if instance == nil {
		panic("db.Get() called before db.Init()")
	}
	return instance
}

// Close closes the database connection.
func Close() error {
	if instance != nil {
		return instance.Close()
	}
	return nil
}
