package usecase

import (
	"context"

	"ai-product-assistant/internal/model"
)

type DraftRepository interface {
	CreateDraft(ctx context.Context, input model.CreateDraftParams) (model.Draft, error)
	CreateDraftVersion(ctx context.Context, input model.CreateDraftVersionParams) (model.DraftVersion, error)
	GetDraftByID(ctx context.Context, draftID int64) (model.Draft, error)
	GetLatestDraftVersion(ctx context.Context, draftID int64) (model.DraftVersion, error)
	ListDrafts(ctx context.Context, input model.ListDraftsParams) ([]model.DraftListItem, error)
}

type LLMClient interface {
	GenerateDraft(ctx context.Context, input model.LLMGenerateParams) (model.LLMGenerateResult, error)
	RefineDraft(ctx context.Context, input model.LLMRefineParams) (model.LLMRefineResult, error)
}
