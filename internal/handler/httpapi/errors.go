package httpapi

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"ai-product-assistant/internal/usecase"
)

const (
	ErrCodeInvalidRequest = "invalid_request"
	ErrCodeNotFound       = "not_found"
	ErrCodeConflict       = "conflict"
	ErrCodeProviderFailed = "provider_failed"
	ErrCodeRateLimited    = "rate_limited"
	ErrCodeInternal       = "internal_error"
)

type ErrorEnvelope struct {
	Error ErrorBody `json:"error"`
}

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func WriteError(w http.ResponseWriter, statusCode int, code, message string) {
	err := WriteJSON(w, statusCode, ErrorEnvelope{
		Error: ErrorBody{
			Code:    code,
			Message: message,
		},
	})
	if err != nil {
		slog.Error("write error response failed", "error", err, "status_code", statusCode, "code", code)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func WriteUsecaseError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, usecase.ErrInvalidInput):
		WriteError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "invalid request")
	case errors.Is(err, usecase.ErrNotFound):
		WriteError(w, http.StatusNotFound, ErrCodeNotFound, "draft not found")
	case errors.Is(err, usecase.ErrConflict):
		WriteError(w, http.StatusConflict, ErrCodeConflict, "conflict")
	case errors.Is(err, usecase.ErrProviderFailed):
		WriteError(w, http.StatusBadGateway, ErrCodeProviderFailed, "provider failed")
	default:
		slog.Error("unexpected usecase error", "error", err)
		WriteError(w, http.StatusInternalServerError, ErrCodeInternal, "internal server error")
	}
}

func WriteInternalError(w http.ResponseWriter, err error, message string) {
	slog.Error(message, "error", err)
	WriteError(w, http.StatusInternalServerError, ErrCodeInternal, "internal server error")
}

func ParsePositiveInt64(raw string) (int64, error) {
	var v int64
	_, err := fmt.Sscan(raw, &v)
	if err != nil || v <= 0 {
		return 0, errors.New("must be a positive integer")
	}
	return v, nil
}
