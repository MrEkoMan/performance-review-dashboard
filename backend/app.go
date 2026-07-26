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
	r.Get("/api/engineers/{engineerId}/follow-ups", getFollowUps)
	r.Post("/api/engineers/{engineerId}/follow-ups", createFollowUp)
	r.Get("/api/follow-ups/{id}", getFollowUp)
	r.Put("/api/follow-ups/{id}", updateFollowUp)
	r.Delete("/api/follow-ups/{id}", deleteFollowUp)
	r.Get("/api/engineers/{engineerId}/recognitions", getRecognitions)
	r.Post("/api/engineers/{engineerId}/recognitions", createRecognition)
	r.Get("/api/recognitions/{id}", getRecognition)
	r.Put("/api/recognitions/{id}", updateRecognition)
	r.Delete("/api/recognitions/{id}", deleteRecognition)
	r.Get("/api/engineers/{engineerId}/timeline", getTimeline)
	r.Get("/api/dashboard/attention", getDashboardAttention)
	r.Get("/api/dashboard/upcoming-one-on-ones", getUpcomingOneOnOnes)
	r.Get("/api/dashboard/follow-ups", getDashboardFollowUps)
	r.Get("/api/dashboard/goals", getDashboardGoals)

	return r
}
