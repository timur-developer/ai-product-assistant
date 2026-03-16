package model

type DraftContent struct {
	Summary     string
	Audience    string
	Value       string
	Scenarios   []string
	Constraints []string
	Risks       []string
	Questions   []string
}

type LLMUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}
