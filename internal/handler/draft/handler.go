package draft

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"ai-product-assistant/internal/handler/httpapi"
	"ai-product-assistant/internal/usecase"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	service usecase.DraftService
}

func NewHandler(service usecase.DraftService) (*Handler, error) {
	if service == nil {
		return nil, errors.New("draft handler: service is required")
	}

	return &Handler{service: service}, nil
}

func (h *Handler) GenerateDraft(w http.ResponseWriter, r *http.Request) {
	var req generateDraftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, httpapi.ErrCodeInvalidRequest, "invalid json")
		return
	}

	req.RawIdea = strings.TrimSpace(req.RawIdea)
	req.Language = strings.ToLower(strings.TrimSpace(req.Language))
	if err := validateGenerateRequest(req); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, httpapi.ErrCodeInvalidRequest, err.Error())
		return
	}

	draftModel, versionModel, err := h.service.GenerateDraft(r.Context(), req.RawIdea, req.Language)
	if err != nil {
		httpapi.WriteUsecaseError(w, err)
		return
	}

	if err := httpapi.WriteJSON(w, http.StatusCreated, toDraftResponse(draftModel, versionModel)); err != nil {
		httpapi.WriteInternalError(w, err, "write generate response failed")
	}
}

func (h *Handler) RefineDraft(w http.ResponseWriter, r *http.Request) {
	draftID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || draftID <= 0 {
		httpapi.WriteError(w, http.StatusBadRequest, httpapi.ErrCodeInvalidRequest, "invalid draft id")
		return
	}

	var req refineDraftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, httpapi.ErrCodeInvalidRequest, "invalid json")
		return
	}

	req.Language = strings.ToLower(strings.TrimSpace(req.Language))
	if err := validateRefineRequest(req); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, httpapi.ErrCodeInvalidRequest, err.Error())
		return
	}

	draftModel, versionModel, err := h.service.RefineDraft(r.Context(), draftID, req.Sections, req.Language)
	if err != nil {
		httpapi.WriteUsecaseError(w, err)
		return
	}

	if err := httpapi.WriteJSON(w, http.StatusOK, toDraftResponse(draftModel, versionModel)); err != nil {
		httpapi.WriteInternalError(w, err, "write refine response failed")
	}
}

func (h *Handler) GetDraftByID(w http.ResponseWriter, r *http.Request) {
	draftID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || draftID <= 0 {
		httpapi.WriteError(w, http.StatusBadRequest, httpapi.ErrCodeInvalidRequest, "invalid draft id")
		return
	}

	draftModel, versionModel, err := h.service.GetDraft(r.Context(), draftID)
	if err != nil {
		httpapi.WriteUsecaseError(w, err)
		return
	}

	if err := httpapi.WriteJSON(w, http.StatusOK, toDraftResponse(draftModel, versionModel)); err != nil {
		httpapi.WriteInternalError(w, err, "write get by id response failed")
	}
}

func (h *Handler) ListDrafts(w http.ResponseWriter, r *http.Request) {
	limit, offset, err := parseListParams(r.URL.Query().Get("limit"), r.URL.Query().Get("offset"))
	if err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, httpapi.ErrCodeInvalidRequest, err.Error())
		return
	}

	items, err := h.service.ListDrafts(r.Context(), limit, offset)
	if err != nil {
		httpapi.WriteUsecaseError(w, err)
		return
	}

	if err := httpapi.WriteJSON(w, http.StatusOK, toListResponse(items)); err != nil {
		httpapi.WriteInternalError(w, err, "write list response failed")
	}
}
