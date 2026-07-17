package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	_ "modernc.org/sqlite"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

type Engineer struct {
	ID			int 	`json:"id"`
	Name 		string 	`json:"name"`
	Role		string	`json:"role"`
	Level		string	`json:"level"`
	Team		string	`json:"team"`
	CareerGoal	string 	`json:"careerGoal"`
	ReviewCycle	string	`json:reviewCycle"`
}

type PerformanceNote struct {
	ID				int		`json:"id"`
	EngineerID		int		`json:"engineerId"`
	EngineerName	string	`json:"engineerName,omitempty"`
	NoteDate		string	`json:"noteDate"`
	Category		string	`json:"category"`
	Summary			string	`json:"summary"`
	Details			string	`json:"details"`
	Impact			string	`json:"impact"`
	FollowUpNeeded	bool	`json:"followUpNeeded"`
	ReviewCycle		string	`json:"reviewCycle`
}

var db *sql.DB

func main() {
	var err error
	db, err = sql.Open("sqlite", "./data/performance.db")
	if err != nil {
		log.Fatal(err)
	}

	initDB()

	r := chi.NewRouter()

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"http://localhost:5173"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:	[]string{"Accept", "Authorization", "Content-Type"},
	}))

	r.Get("/api/engineers", getEngineers)
	r.Post("/api/engineers", createEngineer)
	r.Get("/api/notes", getNotes)
	r.Post("/api/notes", createNote)
	r.Put("/api/notes/{id}", updateNote)
	r.Delete("/api/notes/{id}", deleteNote)

	log.Println("API running on https://localhost:8080")
	http.ListenAndServe(":8080", r)
}

func initDB() {
	schema := `
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
	`

	if _, err := db.Exec(schema); err != nil {
		log.Fatal(err)
	}
}

func getEngineers(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
		SELECT id, name, role, level, team, career_goal, review_cycle
		FROM engineers
		ORDER BY name
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	engineers := make([]Engineer, 0)

	for rows.Next() {
		var e Engineer
		rows.Scan(&e.ID, &e.Name, &e.Role, &e.Level, &e.Team, &e.CareerGoal, &e.ReviewCycle)
		engineers = append(engineers, e)
	}

	json.NewEncoder(w).Encode(engineers)
}

func createEngineer(w http.ResponseWriter, r *http.Request) {
	var e Engineer
	json.NewDecoder(r.Body).Decode(&e)

	_, err := db.Exec(`
		INSERT INTO engineers
		(name, role, level, team, career_goal, review_cycle)
		VALUES (?, ?, ?, ?, ?, ?)
	`, e.Name, e.Role, e.Level, e.Team, e.CareerGoal, e.ReviewCycle)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func getNotes(w http.ResponseWriter, r *http.Request) {
	engineerID := r.URL.Query().Get("engineerId")

	query := `
		SELECT
			n.id,
			n.engineer_id,
			e.name,
			n.note_date,
			n.category,
			n.summary,
			COALESCE(n.details, ''),
			COALESCE(n.impact, ''),
			COALESCE(n.follow_up_needed, 0),
			COALESCE(n.review_cycle, '')
		FROM performance_notes n
		JOIN engineers e on e.id = n.engineer_id
	`

	args := []any{}

	if engineerID != "" {
		query += ` WHERE n.engineer_id = ?`
		args = append(args, engineerID)
	}

	query += ` ORDER BY n.note_date DESC`

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Printf("getNotes query failed: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	notes := make([]PerformanceNote, 0)

	for rows.Next() {
		var n PerformanceNote
		err := rows.Scan(
			&n.ID,
			&n.EngineerID,
			&n.EngineerName,
			&n.NoteDate,
			&n.Category,
			&n.Summary,
			&n.Details,
			&n.Impact,
			&n.FollowUpNeeded,
			&n.ReviewCycle,
		)

		if err != nil {
			log.Printf("getNotes scan failed: %v", err)
			http.Error(w, "Failed to read note data", http.StatusInternalServerError)
			return
		}

		notes = append(notes, n)
	}

	if err := rows.Err(); err != nil {
		log.Printf("getNotes row iteration failed: %v", err)
		http.Error(w, "Failed while reading notes", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(notes); err != nil {
		log.Printf("getNotes encoding failed: %v", err)
	}
}

func createNote(w http.ResponseWriter, r *http.Request) {
	var note PerformanceNote
	
	if err := json.NewDecoder(r.Body).Decode(&note); err != nil {
		log.Printf("createNote decode failed: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("Creating note with date: %v", note.NoteDate)

	result, err := db.Exec(`
		INSERT INTO performance_notes (
			engineer_id, 
			note_date, 
			category, 
			summary, 
			details, 
			impact, 
			follow_up_needed, 
			review_cycle
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		note.EngineerID,
		note.NoteDate,
		note.Category, 
		note.Summary,
		note.Details, 
		note.Impact,
		note.FollowUpNeeded, 
		note.ReviewCycle,
	)

	if err != nil {
		log.Printf("createNote insert failed: %v", err)
		http.Error(w, "Failed to create note", http.StatusInternalServerError)
		return
	}

	id, err := result.LastInsertId()
	if err != nil {
		log.Printf("createdNote ID lookup failed: %v", err)
		http.Error(w, "Note created but ID could not be retrieved", http.StatusInternalServerError)
		return
	}

	note.ID = int(id)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(note); err != nil {
		log.Printf("createNote response encoding failed: %v", err)
	}
}

func updateNote(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if id == "" {
		http.Error(w, "Missing note ID", http.StatusBadRequest)
		return
	}

	var note PerformanceNote

	if err := json.NewDecoder(r.Body).Decode(&note); err != nil {
		log.Printf("updateNote decode failed: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	result, err :=db.Exec(`
		UPDATE performance_notes
		SET
			engineer_id = ?,
			note_date = ?,
			category = ?,
			summary = ?,
			details = ?,
			impact = ?,
			follow_up_needed = ?,
			review_cycle = ?
		WHERE id = ?
	`,
		note.EngineerID,
		note.NoteDate,
		note.Category,
		note.Summary,
		note.Details,
		note.Impact,
		note.FollowUpNeeded,
		note.ReviewCycle,
		id,
	)

	if err != nil {
		log.Printf("updateNote failed: %v", err)
		http.Error(w, "Failed to update note", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, "Failed to confirm update", http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		http.Error(w, "Note not found", http.StatusNotFound)
		return
	}

	noteID, err := strconv.Atoi(id)
	if err != nil {
		http.Error(w, "Invalid note ID", http.StatusBadRequest)
		return
	}

	note.ID = noteID

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(note); err != nil {
		log.Printf("updateNote encode failed: %v", err)
	}
}

func deleteNote(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	result, err := db.Exec(`
		DELETE FROM performance_notes
		WHERE id = ?
	`, id)

	if err != nil {
		log.Printf("deleteNote failed: %v", err)
		http.Error(w, "Failed to delete note", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, "Failed to confirm deletion", http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		http.Error(w, "Failed to confirm deletion", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}