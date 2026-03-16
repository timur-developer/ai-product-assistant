package model

import "time"

type DraftVersion struct {
	ID        int64
	DraftID   int64
	Version   int
	Content   DraftContent
	Provider  string
	ModelName string
	Usage     LLMUsage
	CreatedAt time.Time
}

type CreateDraftVersionParams struct {
	DraftID   int64
	Content   DraftContent
	Provider  string
	ModelName string
	Usage     LLMUsage
}
