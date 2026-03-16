package usecase

import (
	"context"
	"database/sql"

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

type TxManager interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
	WithinTransactionWithOptions(ctx context.Context, opts TxOptions, fn func(ctx context.Context) error) error
}

type TxOptions struct {
	Isolation sql.IsolationLevel
	ReadOnly  bool
}

type DraftService interface {
	GenerateDraft(ctx context.Context, rawIdea, language string) (model.Draft, model.DraftVersion, error)
	RefineDraft(ctx context.Context, draftID int64, sections []string, language string) (model.Draft, model.DraftVersion, error)
	GetDraft(ctx context.Context, draftID int64) (model.Draft, model.DraftVersion, error)
	ListDrafts(ctx context.Context, limit, offset int) ([]model.DraftListItem, error)
}
