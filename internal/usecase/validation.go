package usecase

import (
	"fmt"
	"strings"

	"ai-product-assistant/internal/model"
)

const (
	defaultListLimit = 20
	maxListLimit     = 100
)

var allowedRefineSections = map[string]struct{}{
	"summary":     {},
	"audience":    {},
	"value":       {},
	"scenarios":   {},
	"constraints": {},
	"risks":       {},
	"questions":   {},
}

func validateRawIdea(rawIdea string) (string, error) {
	normalized := strings.TrimSpace(rawIdea)
	if normalized == "" {
		return "", fmt.Errorf("%w: raw idea is required", ErrInvalidInput)
	}

	return normalized, nil
}

func validateLanguage(language string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(language))
	if normalized == "" {
		return "", fmt.Errorf("%w: language is required", ErrInvalidInput)
	}
	if normalized != model.LanguageRU {
		return "", fmt.Errorf("%w: unsupported language", ErrInvalidInput)
	}

	return normalized, nil
}

func normalizeRefineSections(sections []string) ([]string, error) {
	if len(sections) == 0 {
		return nil, fmt.Errorf("%w: sections are required", ErrInvalidInput)
	}

	normalized := make([]string, 0, len(sections))
	seen := make(map[string]struct{}, len(sections))
	for _, section := range sections {
		value := strings.ToLower(strings.TrimSpace(section))
		if value == "" {
			continue
		}

		if _, ok := allowedRefineSections[value]; !ok {
			return nil, fmt.Errorf("%w: unsupported section %q", ErrInvalidInput, section)
		}
		if _, ok := seen[value]; ok {
			continue
		}

		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}

	if len(normalized) == 0 {
		return nil, fmt.Errorf("%w: sections are required", ErrInvalidInput)
	}

	return normalized, nil
}

func normalizeListParams(limit, offset int) (model.ListDraftsParams, error) {
	if offset < 0 {
		return model.ListDraftsParams{}, fmt.Errorf("%w: offset must be non-negative", ErrInvalidInput)
	}

	if limit <= 0 {
		limit = defaultListLimit
	}
	if limit > maxListLimit {
		limit = maxListLimit
	}

	return model.ListDraftsParams{
		Limit:  limit,
		Offset: offset,
	}, nil
}
