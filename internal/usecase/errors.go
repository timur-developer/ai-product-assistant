package usecase

import "errors"

var (
	ErrInvalidInput   = errors.New("invalid input")
	ErrNotFound       = errors.New("not found")
	ErrProviderFailed = errors.New("provider failed")
)
