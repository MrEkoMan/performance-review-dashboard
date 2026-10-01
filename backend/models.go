package main

type Engineer struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Role         string `json:"role"`
	Level        string `json:"level"`
	Team         string `json:"team"`
	CareerGoal   string `json:"careerGoal"`
	ReviewCycle  string `json:"reviewCycle"`
	JiraUsername string `json:"jiraUsername,omitempty"`
	Archived     bool   `json:"archived"`
	DepartureDate   string `json:"departureDate,omitempty"`
	DepartureReason string `json:"departureReason,omitempty"`
	DepartureNotes  string `json:"departureNotes,omitempty"`
}

type PerformanceNote struct {
	ID             int    `json:"id"`
	EngineerID     int    `json:"engineerId"`
	EngineerName   string `json:"engineerName,omitempty"`
	NoteDate       string `json:"noteDate"`
	Category       string `json:"category"`
	Summary        string `json:"summary"`
	Details        string `json:"details"`
	Impact         string `json:"impact"`
	FollowUpNeeded bool   `json:"followUpNeeded"`
	ReviewCycle    string `json:"reviewCycle"`
}

type IntegrationCredentialInput struct {
	Provider       string `json:"provider"`
	AccountLabel   string `json:"accountLabel"`
	BaseURL        string `json:"baseUrl"`
	Secret         string `json:"secret"`
	Enabled        bool   `json:"enabled"`
	DeploymentType string `json:"deploymentType"`
}

type IntegrationCredentialResponse struct {
	Provider       string `json:"provider"`
	AccountLabel   string `json:"accountLabel"`
	BaseURL        string `json:"baseUrl"`
	HasSecret      bool   `json:"hasSecret"`
	Enabled        bool   `json:"enabled"`
	DeploymentType string `json:"deploymentType"`
	UpdatedAt      string `json:"updatedAt"`
}

type IntegrationConnectionResult struct {
	Provider   string `json:"provider"`
	Success    bool   `json:"success"`
	Category   string `json:"category"`
	Message    string `json:"message"`
	Identity   string `json:"identity"`
	StatusCode int    `json:"statusCode"`
	TestedAt   string `json:"testedAt"`
}

type AIProviderConfigurationInput struct {
	DisplayName string `json:"displayName"`
	BaseURL     string `json:"baseUrl"`
	Model       string `json:"model"`
	APIVersion  string `json:"apiVersion"`
	APIKey      string `json:"apiKey"`
	Enabled     bool   `json:"enabled"`
}

type AIProviderConfigurationResponse struct {
	Provider    string `json:"provider"`
	DisplayName string `json:"displayName"`
	BaseURL     string `json:"baseUrl"`
	Model       string `json:"model"`
	APIVersion  string `json:"apiVersion"`
	HasAPIKey   bool   `json:"hasApiKey"`
	Enabled     bool   `json:"enabled"`
	UpdatedAt   string `json:"updatedAt"`
}

// AIAnalysis is one persisted run of the AI examination for an engineer. The
// output is the durable artifact; the prompt context is not stored — it is
// recomputable from live data — so only a summary (per-source counts) and a
// SHA-256 hash of the prompt are kept for provenance.
type AIAnalysis struct {
	ID             int64  `json:"id"`
	EngineerID     int64  `json:"engineerId"`
	Provider       string `json:"provider"`
	Model          string `json:"model"`
	ReviewCycle    string `json:"reviewCycle"`
	ContextSummary string `json:"contextSummary"`
	ContextHash    string `json:"contextHash"`
	OutputMarkdown string `json:"outputMarkdown"`
	CreatedAt      string `json:"createdAt"`
}

type AIAnalysisInput struct {
	Provider    string `json:"provider"`
	ReviewCycle string `json:"reviewCycle"`
}

type ApplicationSetting struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Attachment struct {
	ID               int64  `json:"id"`
	OriginalFilename string `json:"originalFilename"`
	MimeType         string `json:"mimeType"`
	FileSize         int64  `json:"fileSize"`
	SHA256Hash       string `json:"sha256Hash"`
	SourceSystem     string `json:"sourceSystem"`
	SourceAuthor     string `json:"sourceAuthor"`
	SourceDate       string `json:"sourceDate"`
	Caption          string `json:"caption"`
	CreatedAt        string `json:"createdAt"`
	ContentURL       string `json:"contentUrl"`
}

type CreateNoteWithAttachmentInput struct {
	EngineerID     int    `json:"engineerId"`
	NoteDate       string `json:"noteDate"`
	Category       string `json:"category"`
	Summary        string `json:"summary"`
	Details        string `json:"details"`
	Impact         string `json:"impact"`
	FollowUpNeeded bool   `json:"followUpNeeded"`
	ReviewCycle    string `json:"reviewCycle"`
	SourceSystem   string `json:"sourceSystem"`
	SourceAuthor   string `json:"sourceAuthor"`
	SourceDate     string `json:"sourceDate"`
	Caption        string `json:"caption"`
}

type Goal struct {
	ID              int64  `json:"id"`
	EngineerID      int64  `json:"engineerId"`
	Title           string `json:"title"`
	Description     string `json:"description"`
	GoalType        string `json:"goalType"`
	Status          string `json:"status"`
	Priority        string `json:"priority"`
	StartDate       string `json:"startDate"`
	TargetDate      string `json:"targetDate"`
	CompletionDate  string `json:"completionDate"`
	ProgressPercent int    `json:"progressPercent"`
	SuccessCriteria string `json:"successCriteria"`
	ManagerNotes    string `json:"managerNotes"`
	EngineerNotes   string `json:"engineerNotes"`
	ReviewCycle     string `json:"reviewCycle"`
	CreatedAt       string `json:"createdAt"`
	UpdatedAt       string `json:"updatedAt"`
}

type OneOnOne struct {
	ID                  int64  `json:"id"`
	EngineerID          int64  `json:"engineerId"`
	MeetingDate         string `json:"meetingDate"`
	Wins                string `json:"wins"`
	Challenges          string `json:"challenges"`
	CareerDiscussion    string `json:"careerDiscussion"`
	Feedback            string `json:"feedback"`
	ManagerTopics       string `json:"managerTopics"`
	EngineerTopics      string `json:"engineerTopics"`
	PrivateManagerNotes string `json:"privateManagerNotes"`
	SharedNotes         string `json:"sharedNotes"`
	FollowUpDate        string `json:"followUpDate"`
	Status              string `json:"status"`
	CreatedAt           string `json:"createdAt"`
	UpdatedAt           string `json:"updatedAt"`
}

type OnboardingAnswers struct {
	CareerMotivation struct {
		EnjoyMost     string `json:"enjoyMost"`
		EnergyGivers  string `json:"energyGivers"`
		EnergyDrainers string `json:"energyDrainers"`
		SkillsThisYear string `json:"skillsThisYear"`
		CareerNext2to3 string `json:"careerNext2to3"`
	} `json:"careerMotivation"`
	TeamOrg struct {
		TeamDoesWell  string `json:"teamDoesWell"`
		Frustrations  string `json:"frustrations"`
		EMForADay     string `json:"emForADay"`
		SlowingUsDown string `json:"slowingUsDown"`
		TechDebtRisk  string `json:"techDebtRisk"`
	} `json:"teamOrg"`
	WorkingStyle struct {
		PreferredFeedback  string `json:"preferredFeedback"`
		CoachingVsAutonomy string `json:"coachingVsAutonomy"`
		GreatManager       string `json:"greatManager"`
		WorkedWellWithPrev string `json:"workedWellWithPrev"`
		HasntWorked        string `json:"hasntWorked"`
	} `json:"workingStyle"`
	CurrentWork struct {
		ProudOf         string `json:"proudOf"`
		WorkingOnNow    string `json:"workingOnNow"`
		RoadmapConcerns string `json:"roadmapConcerns"`
		Underutilized   string `json:"underutilized"`
	} `json:"currentWork"`
	OneThingToKnow string `json:"oneThingToKnow"`
}

type OnboardingProfile struct {
	ID          int64             `json:"id"`
	EngineerID  int64             `json:"engineerId"`
	Answers     OnboardingAnswers `json:"answers"`
	MeetingDate string            `json:"meetingDate"`
	CreatedAt   string            `json:"createdAt"`
	UpdatedAt   string            `json:"updatedAt"`
}

// DevelopmentPlanHeader captures the identifying metadata table at the top of a
// personal development plan (Developer / Current Role / Target Role / Plan
// Period / Manager).
type DevelopmentPlanHeader struct {
	Developer   string `json:"developer"`
	CurrentRole string `json:"currentRole"`
	TargetRole  string `json:"targetRole"`
	PlanPeriod  string `json:"planPeriod"`
	Manager     string `json:"manager"`
}

// DevelopmentPlanGoal is one SMART goal parsed from the plan, including the
// monthly progress rows that follow it. GoalID links the plan goal to a Goal
// record when the manager has created or connected one.
type DevelopmentPlanGoal struct {
	Title           string                  `json:"title"`
	Goal            string                  `json:"goal"`
	Why             string                  `json:"why"`
	SuccessLooksLike string               `json:"successLooksLike"`
	TargetDate      string                  `json:"targetDate"`
	MonthlyProgress []DevelopmentPlanMonth  `json:"monthlyProgress"`
}

// DevelopmentPlanMonth is one row of a goal's monthly progress table.
type DevelopmentPlanMonth struct {
	Month  string `json:"month"`
	Status string `json:"status"`
	Update string `json:"update"`
}

// DevelopmentPlanAccomplishment is one entry from the Additional
// Accomplishments section of the plan.
type DevelopmentPlanAccomplishment struct {
	Label         string `json:"label"`
	Accomplishment string `json:"accomplishment"`
	Problem       string `json:"problem"`
	ValueDelivered string `json:"valueDelivered"`
}

// DevelopmentPlanFields holds the structured data parsed from a development
// plan's markdown. The parser is tolerant: missing or paraphrased sections
// yield empty values rather than errors.
type DevelopmentPlanFields struct {
	Header          DevelopmentPlanHeader           `json:"header"`
	Strengths       []string                        `json:"strengths"`
	GrowthAreas     []string                        `json:"growthAreas"`
	NextRole        []DevelopmentPlanGap            `json:"nextRole"`
	FocusAreas      []string                        `json:"focusAreas"`
	Goals           []DevelopmentPlanGoal           `json:"goals"`
	Accomplishments []DevelopmentPlanAccomplishment `json:"accomplishments"`
}

// DevelopmentPlanGap is one row of the "What the next role looks like" table,
// pairing a next-role expectation with the gap from the engineer's current
// state.
type DevelopmentPlanGap struct {
	NextRoleLooksLike string `json:"nextRoleLooksLike"`
	Gap               string `json:"gap"`
}

type DevelopmentPlan struct {
	ID          int64                 `json:"id"`
	EngineerID   int64                 `json:"engineerId"`
	Title        string                `json:"title"`
	PlanDate     string                `json:"planDate"`
	ReviewCycle  string                `json:"reviewCycle"`
	RawMarkdown  string                `json:"rawMarkdown"`
	Fields       DevelopmentPlanFields `json:"fields"`
	LinkedGoalID *int64                `json:"linkedGoalId"`
	CreatedAt    string                `json:"createdAt"`
	UpdatedAt    string                `json:"updatedAt"`
}

type FollowUp struct {
	ID             int64  `json:"id"`
	EngineerID     int64  `json:"engineerId"`
	SourceType     string `json:"sourceType"`
	SourceID       *int64 `json:"sourceId"`
	Description    string `json:"description"`
	Owner          string `json:"owner"`
	DueDate        string `json:"dueDate"`
	Status         string `json:"status"`
	Priority       string `json:"priority"`
	CompletionDate string `json:"completionDate"`
	Notes          string `json:"notes"`
	CreatedAt      string `json:"createdAt"`
	UpdatedAt      string `json:"updatedAt"`
}

type Recognition struct {
	ID              int64  `json:"id"`
	EngineerID      int64  `json:"engineerId"`
	RecognitionDate string `json:"recognitionDate"`
	Source          string `json:"source"`
	SourceType      string `json:"sourceType"`
	Category        string `json:"category"`
	Summary         string `json:"summary"`
	Details         string `json:"details"`
	RelatedWork     string `json:"relatedWork"`
	ReviewCycle     string `json:"reviewCycle"`
	CreatedAt       string `json:"createdAt"`
	UpdatedAt       string `json:"updatedAt"`
}

type TimelineEvent struct {
	EventType   string `json:"eventType"`
	SourceID    int64  `json:"sourceId"`
	EventDate   string `json:"eventDate"`
	Title       string `json:"title"`
	Summary     string `json:"summary"`
	Status      string `json:"status"`
	ReviewCycle string `json:"reviewCycle"`
}

type AttentionItem struct {
	ItemType     string `json:"itemType"`
	Severity     string `json:"severity"`
	EngineerID   int64  `json:"engineerId"`
	EngineerName string `json:"engineerName"`
	Title        string `json:"title"`
	Reason       string `json:"reason"`
	DueDate      string `json:"dueDate"`
	SourceType   string `json:"sourceType"`
	SourceID     int64  `json:"sourceId"`
	TargetTab    string `json:"targetTab"`
}

type UpcomingOneOnOne struct {
	MeetingID              int64  `json:"meetingId"`
	EngineerID             int64  `json:"engineerId"`
	EngineerName           string `json:"engineerName"`
	MeetingDate            string `json:"meetingDate"`
	DaysUntil              int    `json:"daysUntil"`
	LastCompletedDate      string `json:"lastCompletedDate"`
	OpenFollowUps          int    `json:"openFollowUps"`
	BlockedGoals           int    `json:"blockedGoals"`
	OverdueGoals           int    `json:"overdueGoals"`
	RecentEvidenceCount    int    `json:"recentEvidenceCount"`
	RecentRecognitionCount int    `json:"recentRecognitionCount"`
}

type DashboardFollowUp struct {
	ID           int64  `json:"id"`
	EngineerID   int64  `json:"engineerId"`
	EngineerName string `json:"engineerName"`
	SourceType   string `json:"sourceType"`
	SourceID     *int64 `json:"sourceId"`
	Description  string `json:"description"`
	Owner        string `json:"owner"`
	DueDate      string `json:"dueDate"`
	DaysOverdue  int    `json:"daysOverdue"`
	Status       string `json:"status"`
	Priority     string `json:"priority"`
	Notes        string `json:"notes"`
}

type DashboardGoal struct {
	ID               int64  `json:"id"`
	EngineerID       int64  `json:"engineerId"`
	EngineerName     string `json:"engineerName"`
	Title            string `json:"title"`
	GoalType         string `json:"goalType"`
	Status           string `json:"status"`
	Priority         string `json:"priority"`
	StartDate        string `json:"startDate"`
	TargetDate       string `json:"targetDate"`
	ProgressPercent  int    `json:"progressPercent"`
	ExpectedProgress int    `json:"expectedProgress"`
	DaysToTarget     int    `json:"daysToTarget"`
	Health           string `json:"health"`
	ReviewCycle      string `json:"reviewCycle"`
}

type EvidenceRecency struct {
	EngineerID           int64  `json:"engineerId"`
	EngineerName         string `json:"engineerName"`
	Team                 string `json:"team"`
	ReviewCycle          string `json:"reviewCycle"`
	LastEvidenceDate     string `json:"lastEvidenceDate"`
	DaysSinceEvidence    int    `json:"daysSinceEvidence"`
	Recency              string `json:"recency"`
	TotalEvidence        int    `json:"totalEvidence"`
	EvidenceLast30Days   int    `json:"evidenceLast30Days"`
	CurrentCycleEvidence int    `json:"currentCycleEvidence"`
}

type ReviewPeriod struct {
	ID        int64  `json:"id"`
	Label     string `json:"label"`
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
	Phase     string `json:"phase"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

type ReviewReadiness struct {
	EngineerID            int64    `json:"engineerId"`
	EngineerName          string   `json:"engineerName"`
	Team                  string   `json:"team"`
	ReviewCycle           string   `json:"reviewCycle"`
	PeriodStart           string   `json:"periodStart"`
	PeriodEnd             string   `json:"periodEnd"`
	PeriodPhase           string   `json:"periodPhase"`
	DaysUntilEnd          int      `json:"daysUntilEnd"`
	Readiness             string   `json:"readiness"`
	EvidenceCount         int      `json:"evidenceCount"`
	EvidenceCategoryCount int      `json:"evidenceCategoryCount"`
	GoalCount             int      `json:"goalCount"`
	RecognitionCount      int      `json:"recognitionCount"`
	CompletedOneOnOnes    int      `json:"completedOneOnOnes"`
	OverdueFollowUps      int      `json:"overdueFollowUps"`
	MissingItems          []string `json:"missingItems"`
}
