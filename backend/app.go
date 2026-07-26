package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

func newRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"http://localhost:5173"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type"},
	}))

	r.Get("/api/engineers", getEngineers)
	r.Post("/api/engineers", createEngineer)
	r.Get("/api/notes", getNotes)
	r.Post("/api/notes", createNote)
	r.Put("/api/notes/{id}", updateNote)
	r.Delete("/api/notes/{id}", deleteNote)
	r.Get("/api/settings", getApplicationSettings)
	r.Put("/api/settings/{key}", updateApplicationSetting)
	r.Get("/api/integrations", getIntegrationCredentials)
	r.Put("/api/integrations/{provider}", saveIntegrationCredential)
	r.Delete("/api/integrations/{provider}", deleteIntegrationCredential)
	r.Get("/api/notes/{id}/attachments", getNoteAttachments)
	r.Post("/api/notes/{id}/attachments", uploadNoteAttachment)
	r.Get("/api/attachments/{id}/content", getAttachmentContent)
	r.Delete("/api/attachments/{id}", deleteAttachment)
	r.Post("/api/notes-with-attachment", createNoteWithAttachment)
	r.Get("/api/engineers/{engineerId}/goals", getGoals)
	r.Post("/api/engineers/{engineerId}/goals", createGoal)
	r.Get("/api/goals/{id}", getGoal)
	r.Put("/api/goals/{id}", updateGoal)
	r.Delete("/api/goals/{id}", deleteGoal)
	r.Get("/api/engineers/{engineerId}/one-on-ones", getOneOnOnes)
	r.Post("/api/engineers/{engineerId}/one-on-ones", createOneOnOne)
	r.Get("/api/one-on-ones/{id}", getOneOnOne)
	r.Put("/api/one-on-ones/{id}", updateOneOnOne)
	r.Delete("/api/one-on-ones/{id}", deleteOneOnOne)

	return r
}
