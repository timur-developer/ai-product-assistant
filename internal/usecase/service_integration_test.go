package usecase_test

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"ai-product-assistant/internal/model"
	"ai-product-assistant/internal/repository"
	"ai-product-assistant/internal/storage/transaction"
	"ai-product-assistant/internal/usecase"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type repoWithSync struct {
	inner     usecase.DraftRepository
	afterRead chan<- struct{}
	waitWrite <-chan struct{}
}

func (r *repoWithSync) CreateDraft(ctx context.Context, input model.CreateDraftParams) (model.Draft, error) {
	return r.inner.CreateDraft(ctx, input)
}

func (r *repoWithSync) CreateDraftVersion(ctx context.Context, input model.CreateDraftVersionParams) (model.DraftVersion, error) {
	return r.inner.CreateDraftVersion(ctx, input)
}

func (r *repoWithSync) GetDraftByID(ctx context.Context, draftID int64) (model.Draft, error) {
	draft, err := r.inner.GetDraftByID(ctx, draftID)
	if err != nil {
		return model.Draft{}, err
	}

	select {
	case r.afterRead <- struct{}{}:
	default:
	}

	<-r.waitWrite
	return draft, nil
}

func (r *repoWithSync) GetLatestDraftVersion(ctx context.Context, draftID int64) (model.DraftVersion, error) {
	return r.inner.GetLatestDraftVersion(ctx, draftID)
}

func (r *repoWithSync) ListDrafts(ctx context.Context, input model.ListDraftsParams) ([]model.DraftListItem, error) {
	return r.inner.ListDrafts(ctx, input)
}

type noopLLMClient struct{}

func (c *noopLLMClient) GenerateDraft(_ context.Context, _ model.LLMGenerateParams) (model.LLMGenerateResult, error) {
	return model.LLMGenerateResult{}, nil
}

func (c *noopLLMClient) RefineDraft(_ context.Context, _ model.LLMRefineParams) (model.LLMRefineResult, error) {
	return model.LLMRefineResult{}, nil
}

func TestServiceGetDraftSnapshotConsistency(t *testing.T) {
	db := openUsecaseTestDB(t)
	defer db.Close()
	resetUsecaseTables(t, db)

	repo := repository.NewDraftRepository(db)
	ctx := context.Background()

	draft, err := repo.CreateDraft(ctx, model.CreateDraftParams{
		RawIdea:  "idea",
		Language: model.LanguageRU,
	})
	if err != nil {
		t.Fatalf("create draft: %v", err)
	}

	_, err = repo.CreateDraftVersion(ctx, model.CreateDraftVersionParams{
		DraftID:   draft.ID,
		Content:   model.DraftContent{Summary: "v1"},
		Provider:  "openai",
		ModelName: "gpt-4o-mini",
		Usage:     model.LLMUsage{TotalTokens: 1},
	})
	if err != nil {
		t.Fatalf("create v1: %v", err)
	}

	txManager, err := transaction.NewManager(db)
	if err != nil {
		t.Fatalf("new tx manager: %v", err)
	}

	afterRead := make(chan struct{}, 1)
	writeDone := make(chan struct{})
	errCh := make(chan error, 1)

	svcRepo := &repoWithSync{
		inner:     repo,
		afterRead: afterRead,
		waitWrite: writeDone,
	}

	svc, err := usecase.NewService(svcRepo, &noopLLMClient{}, txManager, "openai", "gpt-4o-mini")
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	go func() {
		<-afterRead
		_, writeErr := repo.CreateDraftVersion(context.Background(), model.CreateDraftVersionParams{
			DraftID:   draft.ID,
			Content:   model.DraftContent{Summary: "v2"},
			Provider:  "openai",
			ModelName: "gpt-4o-mini",
			Usage:     model.LLMUsage{TotalTokens: 2},
		})
		errCh <- writeErr
		close(writeDone)
	}()

	gotDraft, gotVersion, err := svc.GetDraft(ctx, draft.ID)
	if err != nil {
		t.Fatalf("get draft: %v", err)
	}

	select {
	case writeErr := <-errCh:
		if writeErr != nil {
			t.Fatalf("concurrent write failed: %v", writeErr)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting concurrent writer")
	}

	if gotDraft.LatestVersion != 1 {
		t.Fatalf("expected snapshot latest_version=1, got=%d", gotDraft.LatestVersion)
	}
	if gotVersion.Version != 1 {
		t.Fatalf("expected snapshot draft_version=1, got=%d", gotVersion.Version)
	}

	latestDraft, err := repo.GetDraftByID(ctx, draft.ID)
	if err != nil {
		t.Fatalf("get draft after write: %v", err)
	}
	if latestDraft.LatestVersion != 2 {
		t.Fatalf("expected persisted latest_version=2, got=%d", latestDraft.LatestVersion)
	}
}

func openUsecaseTestDB(t *testing.T) *sql.DB {
	t.Helper()

	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("TEST_POSTGRES_DSN is not set")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	if err := db.PingContext(context.Background()); err != nil {
		db.Close()
		t.Fatalf("ping db: %v", err)
	}

	return db
}

func resetUsecaseTables(t *testing.T, db *sql.DB) {
	t.Helper()

	if _, err := db.ExecContext(context.Background(), "TRUNCATE TABLE draft_versions, drafts RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("truncate tables: %v", err)
	}
}
