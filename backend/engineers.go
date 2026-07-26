package main

import (
	"encoding/json"
	"net/http"
)

func getEngineers(w http.ResponseWriter, _ *http.Request) {
	rows, err := db.Query(`
		SELECT id, name, role, level, team, COALESCE(career_goal, ''), review_cycle
		FROM engineers ORDER BY name`)
	if err != nil {
		http.Error(w, "Failed to retrieve engineers", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	engineers := make([]Engineer, 0)
	for rows.Next() {
		var engineer Engineer
		if err := rows.Scan(&engineer.ID, &engineer.Name, &engineer.Role,
			&engineer.Level, &engineer.Team, &engineer.CareerGoal,
			&engineer.ReviewCycle); err != nil {
			http.Error(w, "Failed to read engineer", http.StatusInternalServerError)
			return
		}
		engineers = append(engineers, engineer)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "Failed while reading engineers", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, engineers)
}

func createEngineer(w http.ResponseWriter, r *http.Request) {
	var engineer Engineer
	if err := json.NewDecoder(r.Body).Decode(&engineer); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if engineer.Name == "" || engineer.Role == "" || engineer.Level == "" ||
		engineer.Team == "" || engineer.ReviewCycle == "" {
		http.Error(w, "Name, role, level, team, and review cycle are required", http.StatusBadRequest)
		return
	}
	result, err := db.Exec(`
		INSERT INTO engineers (name, role, level, team, career_goal, review_cycle)
		VALUES (?, ?, ?, ?, ?, ?)`,
		engineer.Name, engineer.Role, engineer.Level, engineer.Team,
		engineer.CareerGoal, engineer.ReviewCycle)
	if err != nil {
		http.Error(w, "Failed to create engineer", http.StatusInternalServerError)
		return
	}
	id, err := result.LastInsertId()
	if err != nil {
		http.Error(w, "Engineer created but ID could not be retrieved", http.StatusInternalServerError)
		return
	}
	engineer.ID = int(id)
	writeJSON(w, http.StatusCreated, engineer)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
