package model

import "time"

const LanguageRU = "ru"

type Draft struct {
	ID            int64
	RawIdea       string
	Language      string
	LatestVersion int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type DraftListItem struct {
	ID            int64
	RawIdea       string
	Language      string
	LatestVersion int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type CreateDraftParams struct {
	RawIdea  string
	Language string
}

type ListDraftsParams struct {
	Limit  int
	Offset int
}
