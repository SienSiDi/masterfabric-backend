package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"

	llmmodel "github.com/masterfabric/masterfabric_backend/internal/domain/llm/model"
	llmrepo "github.com/masterfabric/masterfabric_backend/internal/domain/llm/repository"
	domainerr "github.com/masterfabric/masterfabric_backend/internal/shared/errors"
	"github.com/masterfabric/masterfabric_backend/internal/shared/events"
	"github.com/masterfabric/masterfabric_backend/internal/shared/telemetry"
	"github.com/masterfabric/masterfabric_backend/internal/application/llm/dto"
)

type RecordEventUseCase struct {
	eventRepo   llmrepo.EventRepository
	sessionRepo llmrepo.SessionRepository
	eventBus    events.Bus
}

func NewRecordEventUseCase(eventRepo llmrepo.EventRepository, sessionRepo llmrepo.SessionRepository, eventBus events.Bus) *RecordEventUseCase {
	return &RecordEventUseCase{eventRepo: eventRepo, sessionRepo: sessionRepo, eventBus: eventBus}
}

// Execute records a single inference event from the browser-side WebLLM run.
// The session must exist and belong to the requesting user.
func (uc *RecordEventUseCase) Execute(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID, req dto.RecordEventRequest) (dto.EventDTO, error) {
	// Verify session exists + belongs to user
	session, err := uc.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return dto.EventDTO{}, err
	}
	if session.UserID != userID {
		return dto.EventDTO{}, domainerr.New(domainerr.CodeForbidden, "not owner of session", nil)
	}

	now := time.Now().UTC()
	event := &llmmodel.InferenceEvent{
		ID:         uuid.New(),
		SessionID:  sessionID,
		UserID:     userID,
		Prompt:     req.Prompt,
		Completion: req.Completion,
		TokensIn:   req.TokensIn,
		TokensOut:  req.TokensOut,
		LatencyMs:  req.LatencyMs,
		Error:      req.Error,
		CreatedAt:  now,
	}

	saved, err := uc.eventRepo.Create(ctx, event)
	if err != nil {
		return dto.EventDTO{}, err
	}

	// Increment Prometheus counters
	status := "ok"
	if req.Error != "" {
		status = "error"
	}
	telemetry.LLMEventsTotal.WithLabelValues(session.ModelID, status).Inc()
	telemetry.LLMTokensTotal.WithLabelValues(session.ModelID, "in").Add(float64(req.TokensIn))
	telemetry.LLMTokensTotal.WithLabelValues(session.ModelID, "out").Add(float64(req.TokensOut))

	uc.eventBus.Publish(ctx, events.NewEvent("llm.event.recorded", "masterfabric.llm", "llm.application.record_event", map[string]any{
		"event_id":    saved.ID.String(),
		"session_id":  sessionID.String(),
		"user_id":     userID.String(),
		"model_id":    session.ModelID,
		"latency_ms":  req.LatencyMs,
		"tokens_in":   req.TokensIn,
		"tokens_out":  req.TokensOut,
	}))

	return dto.EventDTO{
		ID:         saved.ID,
		Prompt:     saved.Prompt,
		Completion: saved.Completion,
		TokensIn:   saved.TokensIn,
		TokensOut:  saved.TokensOut,
		LatencyMs:  saved.LatencyMs,
		Error:      saved.Error,
		CreatedAt:  saved.CreatedAt,
	}, nil
}
