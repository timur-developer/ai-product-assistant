package usecase

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"ai-product-assistant/internal/model"
)

type Service struct {
	repo      DraftRepository
	llmClient LLMClient
	txManager TxManager
	provider  string
	modelName string
}

func NewService(repo DraftRepository, llmClient LLMClient, txManager TxManager, provider, modelName string) (*Service, error) {
	if repo == nil {
		return nil, errors.New("usecase: draft repository is required")
	}
	if llmClient == nil {
		return nil, errors.New("usecase: llm client is required")
	}
	if txManager == nil {
		return nil, errors.New("usecase: transaction manager is required")
	}
	if strings.TrimSpace(provider) == "" {
		return nil, errors.New("usecase: provider is required")
	}
	if strings.TrimSpace(modelName) == "" {
		return nil, errors.New("usecase: model name is required")
	}

	return &Service{
		repo:      repo,
		llmClient: llmClient,
		txManager: txManager,
		provider:  provider,
		modelName: modelName,
	}, nil
}

func (s *Service) GenerateDraft(ctx context.Context, rawIdea, language string) (model.Draft, model.DraftVersion, error) {
	rawIdea, err := validateRawIdea(rawIdea)
	if err != nil {
		return model.Draft{}, model.DraftVersion{}, err
	}

	language, err = validateLanguage(language)
	if err != nil {
		return model.Draft{}, model.DraftVersion{}, err
	}

	llmResult, err := s.llmClient.GenerateDraft(ctx, model.LLMGenerateParams{
		RawIdea:  rawIdea,
		Language: language,
	})
	if err != nil {
		return model.Draft{}, model.DraftVersion{}, fmt.Errorf("generate draft: %w", errors.Join(ErrProviderFailed, err))
	}

	var draft model.Draft
	var version model.DraftVersion
	if err := s.txManager.WithinTransaction(ctx, func(txCtx context.Context) error {
		createdDraft, err := s.repo.CreateDraft(txCtx, model.CreateDraftParams{
			RawIdea:  rawIdea,
			Language: language,
		})
		if err != nil {
			return fmt.Errorf("create draft: %w", err)
		}

		createdVersion, err := s.repo.CreateDraftVersion(txCtx, model.CreateDraftVersionParams{
			DraftID:   createdDraft.ID,
			Content:   llmResult.Content,
			Provider:  s.provider,
			ModelName: s.modelName,
			Usage:     llmResult.Usage,
		})
		if err != nil {
			return fmt.Errorf("create draft version: %w", err)
		}

		draft = createdDraft
		version = createdVersion

		return nil
	}); err != nil {
		return model.Draft{}, model.DraftVersion{}, fmt.Errorf("generate draft transaction: %w", err)
	}

	return draft, version, nil
}

func (s *Service) RefineDraft(ctx context.Context, draftID int64, sections []string, language string) (model.Draft, model.DraftVersion, error) {
	if draftID <= 0 {
		return model.Draft{}, model.DraftVersion{}, fmt.Errorf("%w: draft id must be positive", ErrInvalidInput)
	}

	language, err := validateLanguage(language)
	if err != nil {
		return model.Draft{}, model.DraftVersion{}, err
	}

	sections, err = normalizeRefineSections(sections)
	if err != nil {
		return model.Draft{}, model.DraftVersion{}, err
	}

	draft, err := s.repo.GetDraftByID(ctx, draftID)
	if err != nil {
		return model.Draft{}, model.DraftVersion{}, mapRepositoryErr("get draft", err)
	}

	baseVersion, err := s.repo.GetLatestDraftVersion(ctx, draftID)
	if err != nil {
		return model.Draft{}, model.DraftVersion{}, mapRepositoryErr("get latest draft version", err)
	}

	llmResult, err := s.llmClient.RefineDraft(ctx, model.LLMRefineParams{
		RawIdea:        draft.RawIdea,
		CurrentContent: baseVersion.Content,
		Sections:       sections,
		Language:       language,
	})
	if err != nil {
		return model.Draft{}, model.DraftVersion{}, fmt.Errorf("refine draft: %w", errors.Join(ErrProviderFailed, err))
	}

	var version model.DraftVersion
	if err := s.txManager.WithinTransaction(ctx, func(txCtx context.Context) error {
		currentVersion, err := s.repo.GetLatestDraftVersion(txCtx, draft.ID)
		if err != nil {
			return mapRepositoryErr("get latest draft version in transaction", err)
		}
		if currentVersion.Version != baseVersion.Version {
			return fmt.Errorf("%w: draft was updated, retry refine", ErrConflict)
		}

		createdVersion, err := s.repo.CreateDraftVersion(txCtx, model.CreateDraftVersionParams{
			DraftID:   draft.ID,
			Content:   llmResult.Content,
			Provider:  s.provider,
			ModelName: s.modelName,
			Usage:     llmResult.Usage,
		})
		if err != nil {
			return fmt.Errorf("create draft version: %w", err)
		}

		draft.LatestVersion = createdVersion.Version
		version = createdVersion

		return nil
	}); err != nil {
		return model.Draft{}, model.DraftVersion{}, fmt.Errorf("refine draft transaction: %w", err)
	}

	return draft, version, nil
}

func (s *Service) GetDraft(ctx context.Context, draftID int64) (model.Draft, model.DraftVersion, error) {
	if draftID <= 0 {
		return model.Draft{}, model.DraftVersion{}, fmt.Errorf("%w: draft id must be positive", ErrInvalidInput)
	}

	var draft model.Draft
	var version model.DraftVersion
	if err := s.txManager.WithinTransactionWithOptions(ctx, TxOptions{
		Isolation: sql.LevelRepeatableRead,
		ReadOnly:  true,
	}, func(txCtx context.Context) error {
		loadedDraft, err := s.repo.GetDraftByID(txCtx, draftID)
		if err != nil {
			return mapRepositoryErr("get draft", err)
		}

		loadedVersion, err := s.repo.GetLatestDraftVersion(txCtx, draftID)
		if err != nil {
			return mapRepositoryErr("get latest draft version", err)
		}

		draft = loadedDraft
		version = loadedVersion
		return nil
	}); err != nil {
		return model.Draft{}, model.DraftVersion{}, err
	}

	return draft, version, nil
}

func (s *Service) ListDrafts(ctx context.Context, limit, offset int) ([]model.DraftListItem, error) {
	params, err := normalizeListParams(limit, offset)
	if err != nil {
		return nil, err
	}

	items, err := s.repo.ListDrafts(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("list drafts: %w", err)
	}

	return items, nil
}

func mapRepositoryErr(op string, err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%s: %w", op, ErrNotFound)
	}

	return fmt.Errorf("%s: %w", op, err)
}
