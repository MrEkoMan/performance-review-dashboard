package main

type Engineer struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Role        string `json:"role"`
	Level       string `json:"level"`
	Team        string `json:"team"`
	CareerGoal  string `json:"careerGoal"`
	ReviewCycle string `json:"reviewCycle"`
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
	Provider     string `json:"provider"`
	AccountLabel string `json:"accountLabel"`
	BaseURL      string `json:"baseUrl"`
	Secret       string `json:"secret"`
	Enabled      bool   `json:"enabled"`
}

type IntegrationCredentialResponse struct {
	Provider     string `json:"provider"`
	AccountLabel string `json:"accountLabel"`
	BaseURL      string `json:"baseUrl"`
	HasSecret    bool   `json:"hasSecret"`
	Enabled      bool   `json:"enabled"`
	UpdatedAt    string `json:"updatedAt"`
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
