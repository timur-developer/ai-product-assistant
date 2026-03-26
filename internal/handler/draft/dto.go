package draft

import "time"

type generateDraftRequest struct {
	RawIdea  string `json:"raw_idea"`
	Language string `json:"language"`
}

type refineDraftRequest struct {
	Sections []string `json:"sections"`
	Language string   `json:"language"`
}

type draftContentResponse struct {
	Summary     string   `json:"summary"`
	Audience    string   `json:"audience"`
	Value       string   `json:"value"`
	Scenarios   []string `json:"scenarios"`
	Constraints []string `json:"constraints"`
	Risks       []string `json:"risks"`
	Questions   []string `json:"questions"`
}

type llmUsageResponse struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type draftVersionResponse struct {
	ID        int64                `json:"id"`
	DraftID   int64                `json:"draft_id"`
	Version   int                  `json:"version"`
	Content   draftContentResponse `json:"content"`
	Provider  string               `json:"provider"`
	ModelName string               `json:"model_name"`
	Usage     llmUsageResponse     `json:"usage"`
	CreatedAt time.Time            `json:"created_at"`
}

type draftResponse struct {
	ID            int64                `json:"id"`
	RawIdea       string               `json:"raw_idea"`
	Language      string               `json:"language"`
	LatestVersion int                  `json:"latest_version"`
	CreatedAt     time.Time            `json:"created_at"`
	UpdatedAt     time.Time            `json:"updated_at"`
	Version       draftVersionResponse `json:"version"`
}

type draftListItemResponse struct {
	ID            int64     `json:"id"`
	RawIdea       string    `json:"raw_idea"`
	Language      string    `json:"language"`
	LatestVersion int       `json:"latest_version"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type listDraftsResponse struct {
	Items []draftListItemResponse `json:"items"`
}
