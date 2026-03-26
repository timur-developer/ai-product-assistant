package draft

import (
	"errors"
	"strconv"
	"strings"

	"ai-product-assistant/internal/model"
)

const maxRawIdeaLength = 4000

func validateGenerateRequest(req generateDraftRequest) error {
	rawIdea := strings.TrimSpace(req.RawIdea)
	if rawIdea == "" {
		return errors.New("raw_idea is required")
	}
	if len([]rune(rawIdea)) > maxRawIdeaLength {
		return errors.New("raw_idea is too long")
	}

	language := strings.ToLower(strings.TrimSpace(req.Language))
	if language == "" {
		return errors.New("language is required")
	}
	if language != model.LanguageRU {
		return errors.New("unsupported language")
	}

	return nil
}

func validateRefineRequest(req refineDraftRequest) error {
	if len(req.Sections) == 0 {
		return errors.New("sections are required")
	}

	language := strings.ToLower(strings.TrimSpace(req.Language))
	if language == "" {
		return errors.New("language is required")
	}
	if language != model.LanguageRU {
		return errors.New("unsupported language")
	}

	return nil
}

func parseListParams(rawLimit, rawOffset string) (int, int, error) {
	limit := 0
	offset := 0

	if rawLimit != "" {
		v, err := strconv.Atoi(rawLimit)
		if err != nil {
			return 0, 0, errors.New("limit must be integer")
		}
		if v < 0 {
			return 0, 0, errors.New("limit must be non-negative")
		}
		limit = v
	}

	if rawOffset != "" {
		v, err := strconv.Atoi(rawOffset)
		if err != nil {
			return 0, 0, errors.New("offset must be integer")
		}
		if v < 0 {
			return 0, 0, errors.New("offset must be non-negative")
		}
		offset = v
	}

	return limit, offset, nil
}
