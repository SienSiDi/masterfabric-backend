package usecase

import (
	"context"

	"github.com/google/uuid"

	llmrepo "github.com/masterfabric/masterfabric_backend/internal/domain/llm/repository"
	domainerr "github.com/masterfabric/masterfabric_backend/internal/shared/errors"
	"github.com/masterfabric/masterfabric_backend/internal/application/llm/dto"
)

type ListEventsUseCase struct {
	eventRepo   llmrepo.EventRepository
	sessionRepo llmrepo.SessionRepository
}

func NewListEventsUseCase(eventRepo llmrepo.EventRepository, sessionRepo llmrepo.SessionRepository) *ListEventsUseCase {
	return &ListEventsUseCase{eventRepo: eventRepo, sessionRepo: sessionRepo}
}

// Execute returns a paginated list of events for a session (newest first).
// The session must exist and belong to the requesting user.
func (uc *ListEventsUseCase) Execute(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID, page, limit int) (dto.ListEventsResponse, error) {
	// Verify session ownership
	session, err := uc.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return dto.ListEventsResponse{}, err
	}
	if session.UserID != userID {
		return dto.ListEventsResponse{}, domainerr.New(domainerr.CodeForbidden, "not owner of session", nil)
	}

	offset := (page - 1) * limit
	events, total, err := uc.eventRepo.ListBySession(ctx, sessionID, limit, offset)
	if err != nil {
		return dto.ListEventsResponse{}, err
	}

	out := make([]dto.EventDTO, 0, len(events))
	for _, e := range events {
		out = append(out, dto.EventDTO{
			ID:         e.ID,
			Prompt:     e.Prompt,
			Completion: e.Completion,
			TokensIn:   e.TokensIn,
			TokensOut:  e.TokensOut,
			LatencyMs:  e.LatencyMs,
			Error:      e.Error,
			CreatedAt:  e.CreatedAt,
		})
	}
	return dto.ListEventsResponse{
		Events: out,
		Page:   page,
		Limit:  limit,
		Total:  total,
	}, nil
}
