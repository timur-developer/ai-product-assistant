package openai

import "errors"

var (
	ErrProvider        = errors.New("llm provider error")
	ErrTimeout         = errors.New("llm request timeout")
	ErrInvalidResponse = errors.New("llm invalid response")
)
