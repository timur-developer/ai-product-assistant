package usecase

import "errors"

var (
	ErrInvalidInput   = errors.New("invalid input")
	ErrNotFound       = errors.New("not found")
	ErrConflict       = errors.New("conflict")
	ErrProviderFailed = errors.New("provider failed")
)
