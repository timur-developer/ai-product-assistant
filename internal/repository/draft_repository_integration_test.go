package repository

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"sort"
	"sync"
	"testing"

	"ai-product-assistant/internal/model"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestDraftRepository_CreateAndReadDraft(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	resetTables(t, db)

	repo := NewDraftRepository(db)
	ctx := context.Background()

	created, err := repo.CreateDraft(ctx, model.CreateDraftParams{
		RawIdea:  "idea-1",
		Language: model.LanguageRU,
	})
	if err != nil {
		t.Fatalf("create draft: %v", err)
	}

	got, err := repo.GetDraftByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("get draft by id: %v", err)
	}

	if got.ID != created.ID {
		t.Fatalf("unexpected draft id: got=%d want=%d", got.ID, created.ID)
	}
	if got.RawIdea != "idea-1" {
		t.Fatalf("unexpected raw idea: got=%q", got.RawIdea)
	}
	if got.Language != model.LanguageRU {
		t.Fatalf("unexpected language: got=%q", got.Language)
	}
	if got.LatestVersion != 0 {
		t.Fatalf("unexpected latest version: got=%d", got.LatestVersion)
	}
}

func TestDraftRepository_CreateVersionAndGetLatest(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	resetTables(t, db)

	repo := NewDraftRepository(db)
	ctx := context.Background()

	draft, err := repo.CreateDraft(ctx, model.CreateDraftParams{
		RawIdea:  "idea-2",
		Language: model.LanguageRU,
	})
	if err != nil {
		t.Fatalf("create draft: %v", err)
	}

	v1, err := repo.CreateDraftVersion(ctx, model.CreateDraftVersionParams{
		DraftID: draft.ID,
		Content: model.DraftContent{
			Summary:   "summary-v1",
			Scenarios: []string{"s1"},
		},
		Provider:  "openai",
		ModelName: "gpt-test",
		Usage: model.LLMUsage{
			PromptTokens:     11,
			CompletionTokens: 22,
			TotalTokens:      33,
		},
	})
	if err != nil {
		t.Fatalf("create draft version v1: %v", err)
	}
	if v1.Version != 1 {
		t.Fatalf("unexpected version for v1: got=%d want=1", v1.Version)
	}

	v2, err := repo.CreateDraftVersion(ctx, model.CreateDraftVersionParams{
		DraftID: draft.ID,
		Content: model.DraftContent{
			Summary:   "summary-v2",
			Scenarios: []string{"s2", "s3"},
		},
		Provider:  "openai",
		ModelName: "gpt-test",
		Usage: model.LLMUsage{
			PromptTokens:     44,
			CompletionTokens: 55,
			TotalTokens:      99,
		},
	})
	if err != nil {
		t.Fatalf("create draft version v2: %v", err)
	}
	if v2.Version != 2 {
		t.Fatalf("unexpected version for v2: got=%d want=2", v2.Version)
	}

	latest, err := repo.GetLatestDraftVersion(ctx, draft.ID)
	if err != nil {
		t.Fatalf("get latest draft version: %v", err)
	}
	if latest.ID != v2.ID {
		t.Fatalf("unexpected latest version id: got=%d want=%d", latest.ID, v2.ID)
	}
	if latest.Content.Summary != "summary-v2" {
		t.Fatalf("unexpected latest summary: got=%q", latest.Content.Summary)
	}
	if latest.Usage.TotalTokens != 99 {
		t.Fatalf("unexpected latest usage total tokens: got=%d", latest.Usage.TotalTokens)
	}

	draftAfter, err := repo.GetDraftByID(ctx, draft.ID)
	if err != nil {
		t.Fatalf("get draft by id after versioning: %v", err)
	}
	if draftAfter.LatestVersion != 2 {
		t.Fatalf("unexpected latest version in drafts: got=%d want=2", draftAfter.LatestVersion)
	}
}

func TestDraftRepository_ListDraftsPagination(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	resetTables(t, db)

	repo := NewDraftRepository(db)
	ctx := context.Background()

	first, err := repo.CreateDraft(ctx, model.CreateDraftParams{
		RawIdea:  "first",
		Language: model.LanguageRU,
	})
	if err != nil {
		t.Fatalf("create first draft: %v", err)
	}

	second, err := repo.CreateDraft(ctx, model.CreateDraftParams{
		RawIdea:  "second",
		Language: model.LanguageRU,
	})
	if err != nil {
		t.Fatalf("create second draft: %v", err)
	}

	page1, err := repo.ListDrafts(ctx, model.ListDraftsParams{
		Limit:  1,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("list page1: %v", err)
	}
	if len(page1) != 1 {
		t.Fatalf("unexpected page1 size: got=%d want=1", len(page1))
	}
	if page1[0].ID != second.ID {
		t.Fatalf("unexpected page1 id: got=%d want=%d", page1[0].ID, second.ID)
	}

	page2, err := repo.ListDrafts(ctx, model.ListDraftsParams{
		Limit:  1,
		Offset: 1,
	})
	if err != nil {
		t.Fatalf("list page2: %v", err)
	}
	if len(page2) != 1 {
		t.Fatalf("unexpected page2 size: got=%d want=1", len(page2))
	}
	if page2[0].ID != first.ID {
		t.Fatalf("unexpected page2 id: got=%d want=%d", page2[0].ID, first.ID)
	}
}

func TestDraftRepository_CreateVersionNotFoundDraft(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	resetTables(t, db)

	repo := NewDraftRepository(db)
	ctx := context.Background()

	_, err := repo.CreateDraftVersion(ctx, model.CreateDraftVersionParams{
		DraftID: 999999,
		Content: model.DraftContent{
			Summary: "summary",
		},
		Provider:  "openai",
		ModelName: "gpt-test",
		Usage: model.LLMUsage{
			PromptTokens:     1,
			CompletionTokens: 1,
			TotalTokens:      2,
		},
	})
	if err == nil {
		t.Fatal("expected error for missing draft")
	}
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows, got: %v", err)
	}
}

func TestDraftRepository_CreateVersionConcurrentSequence(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	resetTables(t, db)

	repo := NewDraftRepository(db)
	ctx := context.Background()

	draft, err := repo.CreateDraft(ctx, model.CreateDraftParams{
		RawIdea:  "concurrent-idea",
		Language: model.LanguageRU,
	})
	if err != nil {
		t.Fatalf("create draft: %v", err)
	}

	const workers = 20
	versions := make(chan int, workers)
	errCh := make(chan error, workers)

	var wg sync.WaitGroup
	wg.Add(workers)

	for i := 0; i < workers; i++ {
		i := i
		go func() {
			defer wg.Done()

			version, err := repo.CreateDraftVersion(ctx, model.CreateDraftVersionParams{
				DraftID: draft.ID,
				Content: model.DraftContent{
					Summary: "summary",
				},
				Provider:  "openai",
				ModelName: "gpt-test",
				Usage: model.LLMUsage{
					PromptTokens:     i + 1,
					CompletionTokens: i + 2,
					TotalTokens:      i + 3,
				},
			})
			if err != nil {
				errCh <- err
				return
			}

			versions <- version.Version
		}()
	}

	wg.Wait()
	close(versions)
	close(errCh)

	for err := range errCh {
		if err != nil {
			t.Fatalf("create draft version in parallel: %v", err)
		}
	}

	gotVersions := make([]int, 0, workers)
	for v := range versions {
		gotVersions = append(gotVersions, v)
	}

	if len(gotVersions) != workers {
		t.Fatalf("unexpected versions count: got=%d want=%d", len(gotVersions), workers)
	}

	sort.Ints(gotVersions)
	for i, v := range gotVersions {
		want := i + 1
		if v != want {
			t.Fatalf("unexpected version sequence at idx=%d: got=%d want=%d", i, v, want)
		}
	}

	draftAfter, err := repo.GetDraftByID(ctx, draft.ID)
	if err != nil {
		t.Fatalf("get draft by id after concurrent versioning: %v", err)
	}
	if draftAfter.LatestVersion != workers {
		t.Fatalf("unexpected latest version in draft: got=%d want=%d", draftAfter.LatestVersion, workers)
	}
}

func TestDraftRepository_GetDraftByIDNotFound(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	resetTables(t, db)

	repo := NewDraftRepository(db)
	ctx := context.Background()

	_, err := repo.GetDraftByID(ctx, 999999)
	if err == nil {
		t.Fatal("expected error for missing draft")
	}
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows, got: %v", err)
	}
}

func TestDraftRepository_GetLatestDraftVersionNotFound(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	resetTables(t, db)

	repo := NewDraftRepository(db)
	ctx := context.Background()

	draft, err := repo.CreateDraft(ctx, model.CreateDraftParams{
		RawIdea:  "no-version",
		Language: model.LanguageRU,
	})
	if err != nil {
		t.Fatalf("create draft: %v", err)
	}

	_, err = repo.GetLatestDraftVersion(ctx, draft.ID)
	if err == nil {
		t.Fatal("expected error for missing draft version")
	}
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows, got: %v", err)
	}
}

func openTestDB(t *testing.T) *sql.DB {
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

func resetTables(t *testing.T, db *sql.DB) {
	t.Helper()

	if _, err := db.ExecContext(context.Background(), "TRUNCATE TABLE draft_versions, drafts RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("truncate tables: %v", err)
	}
}
