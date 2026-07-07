package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

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
	EngineerName	string	`json:"engineerName"`
	NoteDate		string	`json:"noteData"`
	Category		string	`json:"category"`
	Summary			string	`json:"summary"`
	Details			string	`json:"details"`
	Impact			string	`json:"impact:`
	FollowUpNeeded	bool	`json:"followUpNeeded"`
	ReviewCycle		string	`json:"reviewCycle:`
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
		AllowedOrigins: []string{"https://localhost:5173"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:	[]string{"Accept", "Authorization", "Content-Type"},
	}))

	r.Get("/api/engineers", getEngineers)
	r.Post("/api/engineers", createEngineer)
	r.Get("/api/notes", getNotes)
	r.Post("/api/notes", createNote)

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

	var engineers []Engineer

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
			n.name,
			n.note_date,
			n.category,
			n.summary,
			n.details,
			n.impact,
			n.follow_up_needed,
			n.review_cycle
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var notes []PerformanceNote

	for rows.Next() {
		var n PerformanceNote
		rows.Scan(
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
		notes = append(notes, n)
	}

	json.NewEncoder(w).Encode(notes)
}

func createNote(w http.ResponseWriter, r *http.Request) {
	var n PerformanceNote
	json.NewDecoder(r.Body).Decode(&n)

	_, err := db.Exec(`
		INSERT INTO performance_notes
		(engineer_id, note_date, category, summary, details, impact, follow_up_needed, review_cycle)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, n.EngineerID, n.NoteDate, n.Category, n.Summary, n.Details, n.Impact, n.FollowUpNeeded, n.ReviewCycle)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}