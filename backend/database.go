package main

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

var db *sql.DB

const schema = `
CREATE TABLE IF NOT EXISTS engineers (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL,
	role TEXT NOT NULL,
	level TEXT NOT NULL,
	team TEXT NOT NULL,
	career_goal TEXT,
	review_cycle TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS performance_notes (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	engineer_id INTEGER NOT NULL,
	note_date TEXT NOT NULL,
	category TEXT NOT NULL,
	summary TEXT NOT NULL,
	details TEXT,
	impact TEXT,
	follow_up_needed BOOLEAN DEFAULT FALSE,
	review_cycle TEXT,
	FOREIGN KEY(engineer_id) REFERENCES engineers(id)
);
CREATE TABLE IF NOT EXISTS integration_credentials (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	provider TEXT NOT NULL UNIQUE,
	account_label TEXT,
	base_url TEXT,
	encrypted_secret TEXT NOT NULL,
	enabled BOOLEAN NOT NULL DEFAULT TRUE,
	updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS application_settings (
	setting_key TEXT PRIMARY KEY,
	setting_value TEXT NOT NULL,
	updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
INSERT OR IGNORE INTO application_settings (setting_key, setting_value)
VALUES ('theme', 'light');
CREATE TABLE IF NOT EXISTS attachments (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	original_filename TEXT NOT NULL,
	stored_filename TEXT NOT NULL UNIQUE,
	mime_type TEXT NOT NULL,
	file_size INTEGER NOT NULL,
	sha256_hash TEXT NOT NULL,
	source_system TEXT,
	source_author TEXT,
	source_date TEXT,
	caption TEXT,
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS performance_note_attachments (
	note_id INTEGER NOT NULL,
	attachment_id INTEGER NOT NULL,
	PRIMARY KEY (note_id, attachment_id),
	FOREIGN KEY (note_id) REFERENCES performance_notes(id) ON DELETE CASCADE,
	FOREIGN KEY (attachment_id) REFERENCES attachments(id) ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS goals (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	engineer_id INTEGER NOT NULL,
	title TEXT NOT NULL,
	description TEXT,
	goal_type TEXT NOT NULL,
	status TEXT NOT NULL,
	priority TEXT NOT NULL,
	start_date TEXT,
	target_date TEXT,
	completion_date TEXT,
	progress_percentage INTEGER NOT NULL DEFAULT 0
		CHECK (progress_percentage BETWEEN 0 AND 100),
	success_criteria TEXT,
	manager_notes TEXT,
	engineer_notes TEXT,
	review_cycle TEXT,
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (engineer_id) REFERENCES engineers(id) ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS one_on_ones (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	engineer_id INTEGER NOT NULL,
	meeting_date TEXT NOT NULL,
	wins TEXT,
	challenges TEXT,
	career_discussion TEXT,
	feedback TEXT,
	manager_topics TEXT,
	engineer_topics TEXT,
	private_manager_notes TEXT,
	shared_notes TEXT,
	follow_up_date TEXT,
	status TEXT NOT NULL,
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (engineer_id) REFERENCES engineers(id) ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS follow_ups (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	engineer_id INTEGER NOT NULL,
	source_type TEXT NOT NULL DEFAULT 'manual',
	source_id INTEGER,
	description TEXT NOT NULL,
	owner TEXT NOT NULL,
	due_date TEXT,
	status TEXT NOT NULL,
	priority TEXT NOT NULL,
	completion_date TEXT,
	notes TEXT,
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (engineer_id) REFERENCES engineers(id) ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS recognitions (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	engineer_id INTEGER NOT NULL,
	recognition_date TEXT NOT NULL,
	source TEXT NOT NULL,
	source_type TEXT NOT NULL,
	category TEXT NOT NULL,
	summary TEXT NOT NULL,
	details TEXT,
	related_work TEXT,
	review_cycle TEXT,
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (engineer_id) REFERENCES engineers(id) ON DELETE CASCADE
);`

func openDatabase(path string) (*sql.DB, error) {
	database, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	database.SetMaxOpenConns(1)
	if err := initializeDatabase(database); err != nil {
		database.Close()
		return nil, err
	}
	return database, nil
}

func initializeDatabase(database *sql.DB) error {
	if _, err := database.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return fmt.Errorf("enable foreign keys: %w", err)
	}
	if _, err := database.Exec(schema); err != nil {
		return fmt.Errorf("initialize schema: %w", err)
	}
	return nil
}
