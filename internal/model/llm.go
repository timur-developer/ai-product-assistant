package model

type LLMGenerateParams struct {
	RawIdea  string
	Language string
}

type LLMGenerateResult struct {
	Content DraftContent
	Usage   LLMUsage
}

type LLMRefineParams struct {
	RawIdea        string
	CurrentContent DraftContent
	Sections       []string
	Language       string
}

type LLMRefineResult struct {
	Content DraftContent
	Usage   LLMUsage
}
