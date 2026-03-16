package openai

import (
	"fmt"
	"strings"

	"ai-product-assistant/internal/model"
)

func buildGenerateMessages(input model.LLMGenerateParams) []chatMessage {
	userPrompt := fmt.Sprintf(
		"Сырой ввод продуктовой идеи:\n%s\n\nВерни только JSON-объект по структуре: summary(string), audience(string), value(string), scenarios(string[]), constraints(string[]), risks(string[]), questions(string[]). Язык ответа: %s.",
		input.RawIdea,
		input.Language,
	)

	return []chatMessage{
		{
			Role:    "system",
			Content: "Ты продуктовый AI-ассистент. Верни только валидный JSON без markdown и пояснений.",
		},
		{
			Role:    "user",
			Content: userPrompt,
		},
	}
}

func buildRefineMessages(input model.LLMRefineParams) []chatMessage {
	sections := strings.Join(input.Sections, ", ")

	userPrompt := fmt.Sprintf(
		"Исходная сырая идея:\n%s\n\nТекущий структурированный черновик:\nsummary: %s\naudience: %s\nvalue: %s\nscenarios: %s\nconstraints: %s\nrisks: %s\nquestions: %s\n\nОбнови только секции: %s.\nВерни полный JSON-объект по структуре: summary(string), audience(string), value(string), scenarios(string[]), constraints(string[]), risks(string[]), questions(string[]). Язык ответа: %s.",
		input.RawIdea,
		input.CurrentContent.Summary,
		input.CurrentContent.Audience,
		input.CurrentContent.Value,
		strings.Join(input.CurrentContent.Scenarios, "; "),
		strings.Join(input.CurrentContent.Constraints, "; "),
		strings.Join(input.CurrentContent.Risks, "; "),
		strings.Join(input.CurrentContent.Questions, "; "),
		sections,
		input.Language,
	)

	return []chatMessage{
		{
			Role:    "system",
			Content: "Ты продуктовый AI-ассистент. Верни только валидный JSON без markdown и пояснений.",
		},
		{
			Role:    "user",
			Content: userPrompt,
		},
	}
}
