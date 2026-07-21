package llm

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/masterfabric/masterfabric_backend/internal/application/llm/dto"
	llmusecase "github.com/masterfabric/masterfabric_backend/internal/application/llm/usecase"
	domainerr "github.com/masterfabric/masterfabric_backend/internal/shared/errors"
	"github.com/masterfabric/masterfabric_backend/internal/shared/middleware"
	"github.com/masterfabric/masterfabric_backend/internal/shared/pagination"
	"github.com/masterfabric/masterfabric_backend/internal/shared/response"
	"github.com/masterfabric/masterfabric_backend/internal/shared/validator"
)

type Handler struct {
	listModelsUC    *llmusecase.ListModelsUseCase
	createSessionUC *llmusecase.CreateSessionUseCase
	getSessionUC    *llmusecase.GetSessionUseCase
	recordEventUC   *llmusecase.RecordEventUseCase
	listEventsUC    *llmusecase.ListEventsUseCase
	recordScoreUC   *llmusecase.RecordScoreUseCase
	getMonitoringUC *llmusecase.GetMonitoringUseCase
}

func NewHandler(
	listModelsUC *llmusecase.ListModelsUseCase,
	createSessionUC *llmusecase.CreateSessionUseCase,
	getSessionUC *llmusecase.GetSessionUseCase,
	recordEventUC *llmusecase.RecordEventUseCase,
	listEventsUC *llmusecase.ListEventsUseCase,
	recordScoreUC *llmusecase.RecordScoreUseCase,
	getMonitoringUC *llmusecase.GetMonitoringUseCase,
) *Handler {
	return &Handler{
		listModelsUC:    listModelsUC,
		createSessionUC: createSessionUC,
		getSessionUC:    getSessionUC,
		recordEventUC:   recordEventUC,
		listEventsUC:    listEventsUC,
		recordScoreUC:   recordScoreUC,
		getMonitoringUC: getMonitoringUC,
	}
}

func (h *Handler) ListModels(w http.ResponseWriter, r *http.Request) {
	resp, err := h.listModelsUC.Execute(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (h *Handler) CreateSession(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromCtx(r)
	if !ok {
		response.Error(w, middleware.ErrNoUserID)
		return
	}
	var req dto.CreateSessionRequest
	if err := validator.DecodeAndValidate(r, &req); err != nil {
		response.Error(w, err)
		return
	}
	resp, err := h.createSessionUC.Execute(r.Context(), userID, req)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, resp)
}

func (h *Handler) GetSession(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromCtx(r)
	if !ok {
		response.Error(w, middleware.ErrNoUserID)
		return
	}
	sessionID, ok := sessionIDFromURL(r)
	if !ok {
		response.Error(w, domainerr.New(domainerr.CodeBadRequest, "invalid session id", nil))
		return
	}
	resp, err := h.getSessionUC.Execute(r.Context(), userID, sessionID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (h *Handler) RecordEvent(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromCtx(r)
	if !ok {
		response.Error(w, middleware.ErrNoUserID)
		return
	}
	sessionID, ok := sessionIDFromURL(r)
	if !ok {
		response.Error(w, domainerr.New(domainerr.CodeBadRequest, "invalid session id", nil))
		return
	}
	var req dto.RecordEventRequest
	if err := validator.DecodeAndValidate(r, &req); err != nil {
		response.Error(w, err)
		return
	}
	resp, err := h.recordEventUC.Execute(r.Context(), userID, sessionID, req)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, resp)
}

func (h *Handler) ListEvents(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromCtx(r)
	if !ok {
		response.Error(w, middleware.ErrNoUserID)
		return
	}
	sessionID, ok := sessionIDFromURL(r)
	if !ok {
		response.Error(w, domainerr.New(domainerr.CodeBadRequest, "invalid session id", nil))
		return
	}
	p := pagination.Parse(r, 20)
	resp, err := h.listEventsUC.Execute(r.Context(), userID, sessionID, p.Page, p.Limit)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (h *Handler) RecordScore(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromCtx(r)
	if !ok {
		response.Error(w, middleware.ErrNoUserID)
		return
	}
	sessionID, ok := sessionIDFromURL(r)
	if !ok {
		response.Error(w, domainerr.New(domainerr.CodeBadRequest, "invalid session id", nil))
		return
	}
	var req dto.RecordScoreRequest
	if err := validator.DecodeAndValidate(r, &req); err != nil {
		response.Error(w, err)
		return
	}
	if err := h.recordScoreUC.Execute(r.Context(), userID, sessionID, req); err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, map[string]string{"status": "recorded"})
}

func (h *Handler) GetMonitoring(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	now := time.Now().UTC()
	var from, to time.Time
	if v := q.Get("from"); v != "" {
		if parsed, err := time.Parse(time.RFC3339, v); err == nil {
			from = parsed
		}
	} else {
		from = now.Add(-24 * time.Hour)
	}
	if v := q.Get("to"); v != "" {
		if parsed, err := time.Parse(time.RFC3339, v); err == nil {
			to = parsed
		}
	} else {
		to = now
	}
	modelID := q.Get("modelId")
	report, err := h.getMonitoringUC.Execute(r.Context(), from, to, modelID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, llmusecase.ToDTO(report))
}

func userIDFromCtx(r *http.Request) (uuid.UUID, bool) {
	v, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		return uuid.Nil, false
	}
	uid, ok := v.(uuid.UUID)
	return uid, ok
}

func sessionIDFromURL(r *http.Request) (uuid.UUID, bool) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}
