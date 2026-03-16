package usecase

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"ai-product-assistant/internal/model"
)

type contextKey string

const txMarkerKey contextKey = "tx_marker"

type mockDraftRepository struct {
	createDraftFn          func(ctx context.Context, input model.CreateDraftParams) (model.Draft, error)
	createDraftVersionFn   func(ctx context.Context, input model.CreateDraftVersionParams) (model.DraftVersion, error)
	getDraftByIDFn         func(ctx context.Context, draftID int64) (model.Draft, error)
	getLatestVersionByIDFn func(ctx context.Context, draftID int64) (model.DraftVersion, error)
	listDraftsFn           func(ctx context.Context, input model.ListDraftsParams) ([]model.DraftListItem, error)
}

func (m *mockDraftRepository) CreateDraft(ctx context.Context, input model.CreateDraftParams) (model.Draft, error) {
	return m.createDraftFn(ctx, input)
}

func (m *mockDraftRepository) CreateDraftVersion(ctx context.Context, input model.CreateDraftVersionParams) (model.DraftVersion, error) {
	return m.createDraftVersionFn(ctx, input)
}

func (m *mockDraftRepository) GetDraftByID(ctx context.Context, draftID int64) (model.Draft, error) {
	return m.getDraftByIDFn(ctx, draftID)
}

func (m *mockDraftRepository) GetLatestDraftVersion(ctx context.Context, draftID int64) (model.DraftVersion, error) {
	return m.getLatestVersionByIDFn(ctx, draftID)
}

func (m *mockDraftRepository) ListDrafts(ctx context.Context, input model.ListDraftsParams) ([]model.DraftListItem, error) {
	return m.listDraftsFn(ctx, input)
}

type mockLLMClient struct {
	generateDraftFn func(ctx context.Context, input model.LLMGenerateParams) (model.LLMGenerateResult, error)
	refineDraftFn   func(ctx context.Context, input model.LLMRefineParams) (model.LLMRefineResult, error)
}

func (m *mockLLMClient) GenerateDraft(ctx context.Context, input model.LLMGenerateParams) (model.LLMGenerateResult, error) {
	return m.generateDraftFn(ctx, input)
}

func (m *mockLLMClient) RefineDraft(ctx context.Context, input model.LLMRefineParams) (model.LLMRefineResult, error) {
	return m.refineDraftFn(ctx, input)
}

type mockTxManager struct {
	withinFn            func(ctx context.Context, fn func(ctx context.Context) error) error
	withinWithOptionsFn func(ctx context.Context, opts TxOptions, fn func(ctx context.Context) error) error
}

func (m *mockTxManager) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return m.withinFn(ctx, fn)
}

func (m *mockTxManager) WithinTransactionWithOptions(ctx context.Context, opts TxOptions, fn func(ctx context.Context) error) error {
	if m.withinWithOptionsFn != nil {
		return m.withinWithOptionsFn(ctx, opts, fn)
	}

	return m.WithinTransaction(ctx, fn)
}

func TestServiceGenerateDraftSuccess(t *testing.T) {
	repo := &mockDraftRepository{
		createDraftFn: func(ctx context.Context, input model.CreateDraftParams) (model.Draft, error) {
			if ctx.Value(txMarkerKey) != true {
				t.Fatalf("expected tx context marker")
			}
			if input.RawIdea != "idea" {
				t.Fatalf("unexpected raw idea: %s", input.RawIdea)
			}
			if input.Language != model.LanguageRU {
				t.Fatalf("unexpected language: %s", input.Language)
			}

			return model.Draft{ID: 10, RawIdea: input.RawIdea, Language: input.Language, LatestVersion: 0}, nil
		},
		createDraftVersionFn: func(ctx context.Context, input model.CreateDraftVersionParams) (model.DraftVersion, error) {
			if ctx.Value(txMarkerKey) != true {
				t.Fatalf("expected tx context marker")
			}
			if input.DraftID != 10 {
				t.Fatalf("unexpected draft id: %d", input.DraftID)
			}
			if input.Provider != "openai" {
				t.Fatalf("unexpected provider: %s", input.Provider)
			}
			if input.ModelName != "gpt-4o-mini" {
				t.Fatalf("unexpected model name: %s", input.ModelName)
			}

			return model.DraftVersion{
				ID:      100,
				DraftID: input.DraftID,
				Version: 1,
				Content: input.Content,
				Usage:   input.Usage,
			}, nil
		},
	}

	llm := &mockLLMClient{
		generateDraftFn: func(_ context.Context, input model.LLMGenerateParams) (model.LLMGenerateResult, error) {
			if input.RawIdea != "idea" {
				t.Fatalf("unexpected raw idea for llm: %s", input.RawIdea)
			}
			if input.Language != model.LanguageRU {
				t.Fatalf("unexpected llm language: %s", input.Language)
			}

			return model.LLMGenerateResult{
				Content: model.DraftContent{
					Summary: "summary",
				},
				Usage: model.LLMUsage{TotalTokens: 50},
			}, nil
		},
	}

	txManager := &mockTxManager{
		withinFn: func(ctx context.Context, fn func(ctx context.Context) error) error {
			return fn(context.WithValue(ctx, txMarkerKey, true))
		},
	}

	svc, err := NewService(repo, llm, txManager, "openai", "gpt-4o-mini")
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	draft, version, err := svc.GenerateDraft(context.Background(), "idea", "ru")
	if err != nil {
		t.Fatalf("generate draft: %v", err)
	}

	if draft.ID != 10 {
		t.Fatalf("unexpected draft id: %d", draft.ID)
	}
	if version.Version != 1 {
		t.Fatalf("unexpected version number: %d", version.Version)
	}
	if version.Usage.TotalTokens != 50 {
		t.Fatalf("unexpected total tokens: %d", version.Usage.TotalTokens)
	}
}

func TestServiceGenerateDraftInvalidInput(t *testing.T) {
	svc, err := NewService(
		&mockDraftRepository{},
		&mockLLMClient{},
		&mockTxManager{withinFn: func(_ context.Context, fn func(ctx context.Context) error) error { return fn(context.Background()) }},
		"openai",
		"gpt-4o-mini",
	)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	_, _, err = svc.GenerateDraft(context.Background(), " ", "ru")
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got: %v", err)
	}
}

func TestServiceGenerateDraftProviderError(t *testing.T) {
	svc, err := NewService(
		&mockDraftRepository{},
		&mockLLMClient{
			generateDraftFn: func(_ context.Context, _ model.LLMGenerateParams) (model.LLMGenerateResult, error) {
				return model.LLMGenerateResult{}, errors.New("provider failed")
			},
		},
		&mockTxManager{withinFn: func(_ context.Context, fn func(ctx context.Context) error) error { return fn(context.Background()) }},
		"openai",
		"gpt-4o-mini",
	)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	_, _, err = svc.GenerateDraft(context.Background(), "idea", "ru")
	if !errors.Is(err, ErrProviderFailed) {
		t.Fatalf("expected ErrProviderFailed, got: %v", err)
	}
}

func TestServiceGenerateDraftInvalidLanguage(t *testing.T) {
	svc, err := NewService(
		&mockDraftRepository{},
		&mockLLMClient{},
		&mockTxManager{withinFn: func(_ context.Context, fn func(ctx context.Context) error) error { return fn(context.Background()) }},
		"openai",
		"gpt-4o-mini",
	)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	_, _, err = svc.GenerateDraft(context.Background(), "idea", "en")
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got: %v", err)
	}
}

func TestServiceRefineDraftSuccessWithDeduplicateSections(t *testing.T) {
	var gotSections []string

	repo := &mockDraftRepository{
		getDraftByIDFn: func(_ context.Context, draftID int64) (model.Draft, error) {
			return model.Draft{
				ID:            draftID,
				RawIdea:       "idea",
				Language:      model.LanguageRU,
				LatestVersion: 1,
			}, nil
		},
		getLatestVersionByIDFn: func(_ context.Context, draftID int64) (model.DraftVersion, error) {
			return model.DraftVersion{
				ID:      11,
				DraftID: draftID,
				Version: 1,
				Content: model.DraftContent{
					Summary: "old",
				},
			}, nil
		},
		createDraftVersionFn: func(_ context.Context, input model.CreateDraftVersionParams) (model.DraftVersion, error) {
			return model.DraftVersion{
				ID:      12,
				DraftID: input.DraftID,
				Version: 2,
				Content: input.Content,
			}, nil
		},
	}

	llm := &mockLLMClient{
		refineDraftFn: func(_ context.Context, input model.LLMRefineParams) (model.LLMRefineResult, error) {
			gotSections = input.Sections
			return model.LLMRefineResult{
				Content: model.DraftContent{
					Summary: "new",
				},
			}, nil
		},
	}

	txManager := &mockTxManager{
		withinFn: func(ctx context.Context, fn func(ctx context.Context) error) error {
			return fn(ctx)
		},
	}

	svc, err := NewService(repo, llm, txManager, "openai", "gpt-4o-mini")
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	_, version, err := svc.RefineDraft(context.Background(), 1, []string{"summary", "Summary", "audience", "summary"}, "ru")
	if err != nil {
		t.Fatalf("refine draft: %v", err)
	}

	if len(gotSections) != 2 {
		t.Fatalf("unexpected sections length: %d", len(gotSections))
	}
	if gotSections[0] != "summary" || gotSections[1] != "audience" {
		t.Fatalf("unexpected normalized sections: %#v", gotSections)
	}
	if version.Version != 2 {
		t.Fatalf("unexpected version: %d", version.Version)
	}
}

func TestServiceRefineDraftNotFound(t *testing.T) {
	svc, err := NewService(
		&mockDraftRepository{
			getDraftByIDFn: func(_ context.Context, _ int64) (model.Draft, error) {
				return model.Draft{}, sql.ErrNoRows
			},
		},
		&mockLLMClient{},
		&mockTxManager{withinFn: func(ctx context.Context, fn func(ctx context.Context) error) error { return fn(ctx) }},
		"openai",
		"gpt-4o-mini",
	)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	_, _, err = svc.RefineDraft(context.Background(), 1, []string{"summary"}, "ru")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}
}

func TestServiceRefineDraftInvalidSections(t *testing.T) {
	svc, err := NewService(
		&mockDraftRepository{},
		&mockLLMClient{},
		&mockTxManager{withinFn: func(ctx context.Context, fn func(ctx context.Context) error) error { return fn(ctx) }},
		"openai",
		"gpt-4o-mini",
	)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	_, _, err = svc.RefineDraft(context.Background(), 1, []string{"unknown"}, "ru")
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got: %v", err)
	}
}

func TestServiceRefineDraftInvalidLanguage(t *testing.T) {
	svc, err := NewService(
		&mockDraftRepository{},
		&mockLLMClient{},
		&mockTxManager{withinFn: func(ctx context.Context, fn func(ctx context.Context) error) error { return fn(ctx) }},
		"openai",
		"gpt-4o-mini",
	)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	_, _, err = svc.RefineDraft(context.Background(), 1, []string{"summary"}, "en")
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got: %v", err)
	}
}

func TestServiceGenerateDraftRollbackOnVersionError(t *testing.T) {
	rollbackCalled := false

	repo := &mockDraftRepository{
		createDraftFn: func(_ context.Context, _ model.CreateDraftParams) (model.Draft, error) {
			return model.Draft{ID: 1}, nil
		},
		createDraftVersionFn: func(_ context.Context, _ model.CreateDraftVersionParams) (model.DraftVersion, error) {
			return model.DraftVersion{}, errors.New("version insert failed")
		},
	}

	llm := &mockLLMClient{
		generateDraftFn: func(_ context.Context, _ model.LLMGenerateParams) (model.LLMGenerateResult, error) {
			return model.LLMGenerateResult{
				Content: model.DraftContent{Summary: "summary"},
			}, nil
		},
	}

	txManager := &mockTxManager{
		withinFn: func(ctx context.Context, fn func(ctx context.Context) error) error {
			err := fn(ctx)
			if err != nil {
				rollbackCalled = true
			}
			return err
		},
	}

	svc, err := NewService(repo, llm, txManager, "openai", "gpt-4o-mini")
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	_, _, err = svc.GenerateDraft(context.Background(), "idea", "ru")
	if err == nil {
		t.Fatal("expected error")
	}
	if !rollbackCalled {
		t.Fatal("expected rollback to be called")
	}
}

func TestServiceGetDraftSuccess(t *testing.T) {
	svc, err := NewService(
		&mockDraftRepository{
			getDraftByIDFn: func(_ context.Context, draftID int64) (model.Draft, error) {
				return model.Draft{ID: draftID, RawIdea: "idea"}, nil
			},
			getLatestVersionByIDFn: func(_ context.Context, draftID int64) (model.DraftVersion, error) {
				return model.DraftVersion{DraftID: draftID, Version: 1}, nil
			},
		},
		&mockLLMClient{},
		&mockTxManager{withinFn: func(ctx context.Context, fn func(ctx context.Context) error) error { return fn(ctx) }},
		"openai",
		"gpt-4o-mini",
	)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	draft, version, err := svc.GetDraft(context.Background(), 7)
	if err != nil {
		t.Fatalf("get draft: %v", err)
	}
	if draft.ID != 7 || version.Version != 1 {
		t.Fatalf("unexpected output: draft=%+v version=%+v", draft, version)
	}
}

func TestServiceListDraftsAppliesDefaultsAndLimits(t *testing.T) {
	var gotInput model.ListDraftsParams

	svc, err := NewService(
		&mockDraftRepository{
			listDraftsFn: func(_ context.Context, input model.ListDraftsParams) ([]model.DraftListItem, error) {
				gotInput = input
				return []model.DraftListItem{{ID: 1}}, nil
			},
		},
		&mockLLMClient{},
		&mockTxManager{withinFn: func(ctx context.Context, fn func(ctx context.Context) error) error { return fn(ctx) }},
		"openai",
		"gpt-4o-mini",
	)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	items, err := svc.ListDrafts(context.Background(), 999, 0)
	if err != nil {
		t.Fatalf("list drafts: %v", err)
	}

	if gotInput.Limit != maxListLimit {
		t.Fatalf("unexpected limit: %d", gotInput.Limit)
	}
	if len(items) != 1 {
		t.Fatalf("unexpected items count: %d", len(items))
	}
}

func TestServiceRefineDraftConflictOnConcurrentUpdate(t *testing.T) {
	createCalled := false
	latestReadCount := 0

	repo := &mockDraftRepository{
		getDraftByIDFn: func(_ context.Context, draftID int64) (model.Draft, error) {
			return model.Draft{
				ID:            draftID,
				RawIdea:       "idea",
				Language:      model.LanguageRU,
				LatestVersion: 1,
			}, nil
		},
		getLatestVersionByIDFn: func(ctx context.Context, draftID int64) (model.DraftVersion, error) {
			latestReadCount++
			if latestReadCount == 1 {
				return model.DraftVersion{
					DraftID: draftID,
					Version: 1,
					Content: model.DraftContent{Summary: "v1"},
				}, nil
			}

			if ctx.Value(txMarkerKey) != true {
				t.Fatalf("expected transaction context marker")
			}

			return model.DraftVersion{
				DraftID: draftID,
				Version: 2,
				Content: model.DraftContent{Summary: "v2"},
			}, nil
		},
		createDraftVersionFn: func(_ context.Context, _ model.CreateDraftVersionParams) (model.DraftVersion, error) {
			createCalled = true
			return model.DraftVersion{}, nil
		},
	}

	llm := &mockLLMClient{
		refineDraftFn: func(_ context.Context, _ model.LLMRefineParams) (model.LLMRefineResult, error) {
			return model.LLMRefineResult{
				Content: model.DraftContent{Summary: "refined"},
			}, nil
		},
	}

	txManager := &mockTxManager{
		withinFn: func(ctx context.Context, fn func(ctx context.Context) error) error {
			return fn(context.WithValue(ctx, txMarkerKey, true))
		},
	}

	svc, err := NewService(repo, llm, txManager, "openai", "gpt-4o-mini")
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	_, _, err = svc.RefineDraft(context.Background(), 1, []string{"summary"}, "ru")
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected ErrConflict on conflict, got: %v", err)
	}
	if createCalled {
		t.Fatal("create version must not be called on conflict")
	}
}

func TestServiceGetDraftReadsInSingleTransaction(t *testing.T) {
	repo := &mockDraftRepository{
		getDraftByIDFn: func(ctx context.Context, draftID int64) (model.Draft, error) {
			if ctx.Value(txMarkerKey) != true {
				t.Fatal("expected transaction context marker in get draft")
			}
			return model.Draft{ID: draftID}, nil
		},
		getLatestVersionByIDFn: func(ctx context.Context, draftID int64) (model.DraftVersion, error) {
			if ctx.Value(txMarkerKey) != true {
				t.Fatal("expected transaction context marker in get latest version")
			}
			return model.DraftVersion{DraftID: draftID, Version: 3}, nil
		},
	}

	txManager := &mockTxManager{
		withinFn: func(ctx context.Context, fn func(ctx context.Context) error) error {
			return fn(context.WithValue(ctx, txMarkerKey, true))
		},
		withinWithOptionsFn: func(ctx context.Context, _ TxOptions, fn func(ctx context.Context) error) error {
			return fn(context.WithValue(ctx, txMarkerKey, true))
		},
	}

	svc, err := NewService(repo, &mockLLMClient{}, txManager, "openai", "gpt-4o-mini")
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	_, version, err := svc.GetDraft(context.Background(), 12)
	if err != nil {
		t.Fatalf("get draft: %v", err)
	}
	if version.Version != 3 {
		t.Fatalf("unexpected version: %d", version.Version)
	}
}

func TestServiceGetDraftUsesRepeatableReadOptions(t *testing.T) {
	repo := &mockDraftRepository{
		getDraftByIDFn: func(_ context.Context, draftID int64) (model.Draft, error) {
			return model.Draft{ID: draftID}, nil
		},
		getLatestVersionByIDFn: func(_ context.Context, draftID int64) (model.DraftVersion, error) {
			return model.DraftVersion{DraftID: draftID, Version: 1}, nil
		},
	}

	txManager := &mockTxManager{
		withinFn: func(ctx context.Context, fn func(ctx context.Context) error) error {
			return fn(ctx)
		},
		withinWithOptionsFn: func(ctx context.Context, opts TxOptions, fn func(ctx context.Context) error) error {
			if opts.Isolation != sql.LevelRepeatableRead {
				t.Fatalf("unexpected isolation level: %v", opts.Isolation)
			}
			if !opts.ReadOnly {
				t.Fatal("expected readonly transaction")
			}
			return fn(ctx)
		},
	}

	svc, err := NewService(repo, &mockLLMClient{}, txManager, "openai", "gpt-4o-mini")
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	_, _, err = svc.GetDraft(context.Background(), 15)
	if err != nil {
		t.Fatalf("get draft: %v", err)
	}
}

func TestServiceGetDraftNotFound(t *testing.T) {
	svc, err := NewService(
		&mockDraftRepository{
			getDraftByIDFn: func(_ context.Context, _ int64) (model.Draft, error) {
				return model.Draft{}, sql.ErrNoRows
			},
			getLatestVersionByIDFn: func(_ context.Context, _ int64) (model.DraftVersion, error) {
				return model.DraftVersion{}, nil
			},
		},
		&mockLLMClient{},
		&mockTxManager{withinFn: func(ctx context.Context, fn func(ctx context.Context) error) error { return fn(ctx) }},
		"openai",
		"gpt-4o-mini",
	)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	_, _, err = svc.GetDraft(context.Background(), 1)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}
}

func TestServiceListDraftsInvalidOffset(t *testing.T) {
	svc, err := NewService(
		&mockDraftRepository{
			listDraftsFn: func(_ context.Context, _ model.ListDraftsParams) ([]model.DraftListItem, error) {
				return nil, nil
			},
		},
		&mockLLMClient{},
		&mockTxManager{withinFn: func(ctx context.Context, fn func(ctx context.Context) error) error { return fn(ctx) }},
		"openai",
		"gpt-4o-mini",
	)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	_, err = svc.ListDrafts(context.Background(), 10, -1)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got: %v", err)
	}
}

func TestNewServiceValidation(t *testing.T) {
	testCases := []struct {
		name      string
		repo      DraftRepository
		llm       LLMClient
		tx        TxManager
		provider  string
		modelName string
	}{
		{
			name:      "nil repo",
			repo:      nil,
			llm:       &mockLLMClient{},
			tx:        &mockTxManager{withinFn: func(ctx context.Context, fn func(ctx context.Context) error) error { return fn(ctx) }},
			provider:  "openai",
			modelName: "gpt-4o-mini",
		},
		{
			name:      "nil llm",
			repo:      &mockDraftRepository{},
			llm:       nil,
			tx:        &mockTxManager{withinFn: func(ctx context.Context, fn func(ctx context.Context) error) error { return fn(ctx) }},
			provider:  "openai",
			modelName: "gpt-4o-mini",
		},
		{
			name:      "nil tx",
			repo:      &mockDraftRepository{},
			llm:       &mockLLMClient{},
			tx:        nil,
			provider:  "openai",
			modelName: "gpt-4o-mini",
		},
		{
			name:      "empty provider",
			repo:      &mockDraftRepository{},
			llm:       &mockLLMClient{},
			tx:        &mockTxManager{withinFn: func(ctx context.Context, fn func(ctx context.Context) error) error { return fn(ctx) }},
			provider:  "",
			modelName: "gpt-4o-mini",
		},
		{
			name:      "empty model name",
			repo:      &mockDraftRepository{},
			llm:       &mockLLMClient{},
			tx:        &mockTxManager{withinFn: func(ctx context.Context, fn func(ctx context.Context) error) error { return fn(ctx) }},
			provider:  "openai",
			modelName: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewService(tc.repo, tc.llm, tc.tx, tc.provider, tc.modelName)
			if err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}
