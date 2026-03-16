package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"ai-product-assistant/internal/model"
)

type DraftRepository struct {
	db *sql.DB
}

func NewDraftRepository(db *sql.DB) *DraftRepository {
	return &DraftRepository{db: db}
}

func (r *DraftRepository) CreateDraft(ctx context.Context, input model.CreateDraftParams) (model.Draft, error) {
	const op = "create draft"
	const query = `
		INSERT INTO drafts (
			raw_idea,
			language
		)
		VALUES ($1, $2)
		RETURNING
			id,
			raw_idea,
			language,
			latest_version,
			created_at,
			updated_at
	`

	var draft model.Draft
	if err := r.db.QueryRowContext(ctx, query, input.RawIdea, input.Language).Scan(
		&draft.ID,
		&draft.RawIdea,
		&draft.Language,
		&draft.LatestVersion,
		&draft.CreatedAt,
		&draft.UpdatedAt,
	); err != nil {
		return model.Draft{}, wrapErr(op, err)
	}

	return draft, nil
}

func (r *DraftRepository) CreateDraftVersion(ctx context.Context, input model.CreateDraftVersionParams) (model.DraftVersion, error) {
	const op = "create draft version"
	const query = `
		WITH next_version AS (
			UPDATE drafts
			SET
				latest_version = latest_version + 1,
				updated_at = NOW()
			WHERE id = $1
			RETURNING
				id,
				latest_version
		)
		INSERT INTO draft_versions (
			draft_id,
			version,
			content,
			provider,
			model_name,
			prompt_tokens,
			completion_tokens,
			total_tokens
		)
		SELECT
			next_version.id,
			next_version.latest_version,
			$2::jsonb,
			$3,
			$4,
			$5,
			$6,
			$7
		FROM next_version
		RETURNING
			id,
			draft_id,
			version,
			content,
			provider,
			model_name,
			prompt_tokens,
			completion_tokens,
			total_tokens,
			created_at
	`

	contentJSON, err := marshalContent(input.Content)
	if err != nil {
		return model.DraftVersion{}, wrapErr(op, err)
	}

	var draftVersion model.DraftVersion
	var rawContent []byte
	if err := r.db.QueryRowContext(
		ctx,
		query,
		input.DraftID,
		string(contentJSON),
		input.Provider,
		input.ModelName,
		input.Usage.PromptTokens,
		input.Usage.CompletionTokens,
		input.Usage.TotalTokens,
	).Scan(
		&draftVersion.ID,
		&draftVersion.DraftID,
		&draftVersion.Version,
		&rawContent,
		&draftVersion.Provider,
		&draftVersion.ModelName,
		&draftVersion.Usage.PromptTokens,
		&draftVersion.Usage.CompletionTokens,
		&draftVersion.Usage.TotalTokens,
		&draftVersion.CreatedAt,
	); err != nil {
		return model.DraftVersion{}, wrapErr(op, err)
	}

	draftVersion.Content, err = unmarshalContent(rawContent)
	if err != nil {
		return model.DraftVersion{}, wrapErr(op, err)
	}

	return draftVersion, nil
}

func (r *DraftRepository) GetDraftByID(ctx context.Context, draftID int64) (model.Draft, error) {
	const op = "get draft by id"
	const query = `
		SELECT
			id,
			raw_idea,
			language,
			latest_version,
			created_at,
			updated_at
		FROM drafts
		WHERE id = $1
	`

	var draft model.Draft
	if err := r.db.QueryRowContext(ctx, query, draftID).Scan(
		&draft.ID,
		&draft.RawIdea,
		&draft.Language,
		&draft.LatestVersion,
		&draft.CreatedAt,
		&draft.UpdatedAt,
	); err != nil {
		return model.Draft{}, wrapErr(op, err)
	}

	return draft, nil
}

func (r *DraftRepository) GetLatestDraftVersion(ctx context.Context, draftID int64) (model.DraftVersion, error) {
	const op = "get latest draft version"
	const query = `
		SELECT
			id,
			draft_id,
			version,
			content,
			provider,
			model_name,
			prompt_tokens,
			completion_tokens,
			total_tokens,
			created_at
		FROM draft_versions
		WHERE draft_id = $1
		ORDER BY version DESC
		LIMIT 1
	`

	var draftVersion model.DraftVersion
	var rawContent []byte
	if err := r.db.QueryRowContext(ctx, query, draftID).Scan(
		&draftVersion.ID,
		&draftVersion.DraftID,
		&draftVersion.Version,
		&rawContent,
		&draftVersion.Provider,
		&draftVersion.ModelName,
		&draftVersion.Usage.PromptTokens,
		&draftVersion.Usage.CompletionTokens,
		&draftVersion.Usage.TotalTokens,
		&draftVersion.CreatedAt,
	); err != nil {
		return model.DraftVersion{}, wrapErr(op, err)
	}

	content, err := unmarshalContent(rawContent)
	if err != nil {
		return model.DraftVersion{}, wrapErr(op, err)
	}
	draftVersion.Content = content

	return draftVersion, nil
}

func (r *DraftRepository) ListDrafts(ctx context.Context, input model.ListDraftsParams) ([]model.DraftListItem, error) {
	const op = "list drafts"
	const query = `
		SELECT
			id,
			raw_idea,
			language,
			latest_version,
			created_at,
			updated_at
		FROM drafts
		ORDER BY created_at DESC, id DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.QueryContext(ctx, query, input.Limit, input.Offset)
	if err != nil {
		return nil, wrapErr(op, err)
	}
	defer rows.Close()

	items := make([]model.DraftListItem, 0)
	for rows.Next() {
		var item model.DraftListItem
		if err := rows.Scan(
			&item.ID,
			&item.RawIdea,
			&item.Language,
			&item.LatestVersion,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, wrapErr(op, err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, wrapErr(op, err)
	}

	return items, nil
}

func wrapErr(op string, err error) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("repository: %s: %w", op, err)
}

func marshalContent(content model.DraftContent) ([]byte, error) {
	data, err := json.Marshal(content)
	if err != nil {
		return nil, fmt.Errorf("marshal content: %w", err)
	}

	return data, nil
}

func unmarshalContent(data []byte) (model.DraftContent, error) {
	var content model.DraftContent
	if err := json.Unmarshal(data, &content); err != nil {
		return model.DraftContent{}, fmt.Errorf("unmarshal content: %w", err)
	}

	return content, nil
}
