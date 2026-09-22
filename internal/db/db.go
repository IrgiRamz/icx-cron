package db

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

func InitDB(dbPath string) (*sql.DB, error) {
	// Open SQLite database using pure Go driver
	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Connection pooling configuration for SQLite WAL
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	log.Println("[DB] SQLite database initialized successfully with WAL mode.")
	return db, nil
}

func createTables(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS jobs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL DEFAULT 'Unnamed Job',
		url TEXT NOT NULL,
		method TEXT NOT NULL DEFAULT 'GET',
		cron_expression TEXT NOT NULL,
		status INTEGER NOT NULL DEFAULT 1,
		timeout_seconds INTEGER NOT NULL DEFAULT 10,
		retry_count INTEGER NOT NULL DEFAULT 0,
		total_success INTEGER NOT NULL DEFAULT 0,
		total_fail INTEGER NOT NULL DEFAULT 0,
		consecutive_fail INTEGER NOT NULL DEFAULT 0,
		last_run_at DATETIME,
		next_run_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS job_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		job_id INTEGER NOT NULL,
		status_code INTEGER NOT NULL DEFAULT 0,
		execution_time_ms INTEGER NOT NULL DEFAULT 0,
		error_message TEXT DEFAULT '',
		response_body_snippet TEXT DEFAULT '',
		executed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_logs_job_id_executed ON job_logs(job_id, executed_at DESC);
	CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status);
	`

	_, err := db.Exec(schema)
	return err
}
