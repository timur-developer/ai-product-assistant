package draft

import "ai-product-assistant/internal/model"

func toDraftResponse(d model.Draft, v model.DraftVersion) draftResponse {
	return draftResponse{
		ID:            d.ID,
		RawIdea:       d.RawIdea,
		Language:      d.Language,
		LatestVersion: d.LatestVersion,
		CreatedAt:     d.CreatedAt,
		UpdatedAt:     d.UpdatedAt,
		Version:       toDraftVersionResponse(v),
	}
}

func toDraftVersionResponse(v model.DraftVersion) draftVersionResponse {
	return draftVersionResponse{
		ID:        v.ID,
		DraftID:   v.DraftID,
		Version:   v.Version,
		Content:   toDraftContentResponse(v.Content),
		Provider:  v.Provider,
		ModelName: v.ModelName,
		Usage:     toUsageResponse(v.Usage),
		CreatedAt: v.CreatedAt,
	}
}

func toDraftContentResponse(c model.DraftContent) draftContentResponse {
	return draftContentResponse{
		Summary:     c.Summary,
		Audience:    c.Audience,
		Value:       c.Value,
		Scenarios:   c.Scenarios,
		Constraints: c.Constraints,
		Risks:       c.Risks,
		Questions:   c.Questions,
	}
}

func toUsageResponse(u model.LLMUsage) llmUsageResponse {
	return llmUsageResponse{
		PromptTokens:     u.PromptTokens,
		CompletionTokens: u.CompletionTokens,
		TotalTokens:      u.TotalTokens,
	}
}

func toListResponse(items []model.DraftListItem) listDraftsResponse {
	out := make([]draftListItemResponse, 0, len(items))
	for _, item := range items {
		out = append(out, draftListItemResponse{
			ID:            item.ID,
			RawIdea:       item.RawIdea,
			Language:      item.Language,
			LatestVersion: item.LatestVersion,
			CreatedAt:     item.CreatedAt,
			UpdatedAt:     item.UpdatedAt,
		})
	}

	return listDraftsResponse{Items: out}
}
