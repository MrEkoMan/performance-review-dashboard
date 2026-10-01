package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

// newRouter assembles the full route table. Every non-auth route runs behind
// the authenticate middleware, which resolves the session cookie into the
// request context; when no accounts exist yet the tool stays usable as before
// (open mode, callers treated as the manager). Engineer-accessible read
// routes are wrapped with engineerSelf so a logged-in engineer can only reach
// their own engineerId; everything else is manager-only.
func newRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	}))
	r.Use(authenticate)

	// Authentication endpoints are public: login mints the session, logout
	// ends it (and is safe to call without one), me reflects the caller.
	// registration-status tells the login page whether the manager account
	// can still be claimed; register is the claiming/recovery path.
	r.Post("/api/auth/login", loginHandler)
	r.Post("/api/auth/logout", logoutHandler)
	r.Get("/api/auth/me", meHandler)
	r.Get("/api/auth/registration-status", registrationStatusHandler)
	r.Post("/api/auth/register", registerHandler)

	// Engineer-scoped reads: manager for anyone, engineer only for self.
	r.Get("/api/engineers/{engineerId}", engineerSelf(getEngineer))
	r.Get("/api/engineers/{engineerId}/goals", engineerSelf(getGoals))
	r.Get("/api/engineers/{engineerId}/one-on-ones", engineerSelf(getOneOnOnes))
	r.Get("/api/engineers/{engineerId}/follow-ups", engineerSelf(getFollowUps))
	r.Get("/api/engineers/{engineerId}/recognitions", engineerSelf(getRecognitions))
	r.Get("/api/engineers/{engineerId}/timeline", engineerSelf(getTimeline))
	r.Get("/api/engineers/{engineerId}/development-plans", engineerSelf(getDevelopmentPlans))
	r.Get("/api/engineers/{engineerId}/onboarding-profile", engineerSelf(getOnboardingProfile))
	r.Get("/api/engineers/{engineerId}/jira-evidence", managerOnly(getJiraEvidence))
	r.Post("/api/engineers/{engineerId}/jira-evidence/accept", managerOnly(acceptJiraEvidence))

	// The engineer list is reachable by all callers; getEngineers narrows the
	// result to the caller's own record when the caller is an engineer.
	r.Get("/api/engineers", getEngineers)

	// Manager-only engineer management and portal access.
	r.Post("/api/engineers", managerOnly(createEngineer))
	r.Put("/api/engineers/{engineerId}", managerOnly(updateEngineer))
	r.Post("/api/engineers/{engineerId}/archive", managerOnly(archiveEngineer))
	r.Post("/api/engineers/{engineerId}/restore", managerOnly(restoreEngineer))
	r.Post("/api/engineers/{engineerId}/portal-access", managerOnly(createPortalAccess))
	r.Put("/api/engineers/{engineerId}/portal-access", managerOnly(updatePortalAccess))
	r.Delete("/api/engineers/{engineerId}/portal-access", managerOnly(deletePortalAccess))
	r.Get("/api/engineers/{engineerId}/portal-access", managerOnly(getPortalAccess))

	// Notes are the engineer-writable surface: engineers may append context on
	// their own record and edit only their own engineer-authored entries.
	r.Get("/api/notes", engineerSelf(getNotes))
	r.Post("/api/notes", createNote)
	r.Put("/api/notes/{id}", updateNote)
	r.Delete("/api/notes/{id}", deleteNote)

	r.Get("/api/notes/{id}/attachments", engineerOwnNote(getNoteAttachments))
	r.Post("/api/notes/{id}/attachments", managerOnly(uploadNoteAttachment))
	r.Get("/api/attachments/{id}/content", engineerOwnAttachment(getAttachmentContent))
	r.Delete("/api/attachments/{id}", managerOnly(deleteAttachment))
	r.Post("/api/notes-with-attachment", managerOnly(createNoteWithAttachment))

	// Manager-only mutations on child records.
	r.Post("/api/engineers/{engineerId}/goals", managerOnly(createGoal))
	r.Get("/api/goals/{id}", managerOnly(getGoal))
	r.Put("/api/goals/{id}", managerOnly(updateGoal))
	r.Delete("/api/goals/{id}", managerOnly(deleteGoal))
	r.Post("/api/engineers/{engineerId}/one-on-ones", managerOnly(createOneOnOne))
	r.Get("/api/one-on-ones/{id}", engineerSelf(forEngineerOneOnOne(getOneOnOne)))
	r.Put("/api/one-on-ones/{id}", managerOnly(updateOneOnOne))
	r.Delete("/api/one-on-ones/{id}", managerOnly(deleteOneOnOne))
	r.Put("/api/engineers/{engineerId}/onboarding-profile", managerOnly(upsertOnboardingProfile))
	r.Post("/api/onboarding-profile/parse", managerOnly(parseOnboardingFile))
	r.Post("/api/engineers/{engineerId}/development-plans", managerOnly(createDevelopmentPlan))
	r.Get("/api/development-plans/{id}", managerOnly(getDevelopmentPlan))
	r.Put("/api/development-plans/{id}", managerOnly(updateDevelopmentPlan))
	r.Delete("/api/development-plans/{id}", managerOnly(deleteDevelopmentPlan))
	r.Post("/api/development-plans/parse", managerOnly(parseDevelopmentPlanFile))
	r.Post("/api/engineers/{engineerId}/follow-ups", managerOnly(createFollowUp))
	r.Get("/api/follow-ups/{id}", managerOnly(getFollowUp))
	r.Put("/api/follow-ups/{id}", managerOnly(updateFollowUp))
	r.Delete("/api/follow-ups/{id}", managerOnly(deleteFollowUp))
	r.Post("/api/engineers/{engineerId}/recognitions", managerOnly(createRecognition))
	r.Get("/api/recognitions/{id}", managerOnly(getRecognition))
	r.Put("/api/recognitions/{id}", managerOnly(updateRecognition))
	r.Delete("/api/recognitions/{id}", managerOnly(deleteRecognition))
	r.Get("/api/recognitions/{id}/attachments", managerOnly(getRecognitionAttachments))
	r.Post("/api/recognitions/{id}/attachments", managerOnly(uploadRecognitionAttachment))
	r.Post("/api/engineers/{engineerId}/recognitions-with-attachment", managerOnly(createRecognitionWithAttachment))

	// AI analyses embed private manager notes in their output, so engineers
	// cannot access them at all until the prompt source is filtered.
	r.Post("/api/engineers/{engineerId}/ai-analyses", managerOnly(createAIAnalysis))
	r.Get("/api/engineers/{engineerId}/ai-analyses", managerOnly(getAIAnalyses))
	r.Delete("/api/ai-analyses/{id}", managerOnly(deleteAIAnalysis))

	// Dashboards, settings, integrations, and review-period mutations stay
	// manager-only; review periods stay readable so date pickers keep working.
	r.Get("/api/dashboard/attention", managerOnly(getDashboardAttention))
	r.Get("/api/dashboard/upcoming-one-on-ones", managerOnly(getUpcomingOneOnOnes))
	r.Get("/api/dashboard/follow-ups", managerOnly(getDashboardFollowUps))
	r.Get("/api/dashboard/goals", managerOnly(getDashboardGoals))
	r.Get("/api/dashboard/evidence-recency", managerOnly(getEvidenceRecency))
	r.Get("/api/review-periods", getReviewPeriods)
	r.Post("/api/review-periods", managerOnly(createReviewPeriod))
	r.Put("/api/review-periods/{id}", managerOnly(updateReviewPeriod))
	r.Delete("/api/review-periods/{id}", managerOnly(deleteReviewPeriod))
	r.Get("/api/dashboard/review-readiness", managerOnly(getReviewReadiness))
	r.Get("/api/settings", managerOnly(getApplicationSettings))
	r.Put("/api/settings/{key}", managerOnly(updateApplicationSetting))
	r.Get("/api/integrations", managerOnly(getIntegrationCredentials))
	r.Put("/api/integrations/{provider}", managerOnly(saveIntegrationCredential))
	r.Delete("/api/integrations/{provider}", managerOnly(deleteIntegrationCredential))
	r.Post("/api/integrations/{provider}/test", managerOnly(testIntegrationConnection))
	r.Get("/api/ai-providers", managerOnly(getAIProviderConfigurations))
	r.Put("/api/ai-providers/{provider}", managerOnly(saveAIProviderConfiguration))
	r.Delete("/api/ai-providers/{provider}", managerOnly(deleteAIProviderConfiguration))

	return r
}

// forEngineerOneOnOne wraps the single-1:1 GET handler: the URL does not
// carry the engineerId, so the ownership check looks the record up itself.
// Redaction of private manager notes happens in the one-on-ones handler via
// redactPrivateManagerNotesForEngineer.
func forEngineerOneOnOne(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := currentUser(r)
		if !ok {
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}
		if user.Role == roleManager {
			next(w, r)
			return
		}
		id, err := positiveID(chi.URLParam(r, "id"))
		if err != nil {
			http.Error(w, "Invalid 1:1 ID", http.StatusBadRequest)
			return
		}
		var engineerID int64
		if err := db.QueryRow(
			`SELECT engineer_id FROM one_on_ones WHERE id = ?`, id).Scan(&engineerID); err != nil {
			// Let the handler produce its canonical 404/400.
			next(w, r)
			return
		}
		if engineerID != user.EngineerID {
			http.Error(w, "You can only access your own records", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}
