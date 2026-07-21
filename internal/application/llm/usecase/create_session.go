package usecase

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	llmmodel "github.com/masterfabric/masterfabric_backend/internal/domain/llm/model"
	llmrepo "github.com/masterfabric/masterfabric_backend/internal/domain/llm/repository"
	"github.com/masterfabric/masterfabric_backend/internal/shared/events"
	"github.com/masterfabric/masterfabric_backend/internal/application/llm/dto"
)

type CreateSessionUseCase struct {
	sessionRepo llmrepo.SessionRepository
	eventBus    events.Bus
}

func NewCreateSessionUseCase(sessionRepo llmrepo.SessionRepository, eventBus events.Bus) *CreateSessionUseCase {
	return &CreateSessionUseCase{sessionRepo: sessionRepo, eventBus: eventBus}
}

func (uc *CreateSessionUseCase) Execute(ctx context.Context, userID uuid.UUID, req dto.CreateSessionRequest) (dto.SessionDTO, error) {
	modelID := strings.TrimSpace(req.ModelID)
	if modelID == "" {
		// validator should catch this, but double-check at the use case boundary
		modelID = "gemma-2-2b-it-q4f32_1-MLC"
	}
	now := time.Now().UTC()
	session := &llmmodel.Session{
		ID:        uuid.New(),
		UserID:    userID,
		ModelID:   modelID,
		ModelHash: strings.TrimSpace(req.ModelHash),
		CreatedAt: now,
	}
	saved, err := uc.sessionRepo.Create(ctx, session)
	if err != nil {
		return dto.SessionDTO{}, err
	}
	uc.eventBus.Publish(ctx, events.NewEvent("llm.session.created", "masterfabric.llm", "llm.application.create_session", map[string]any{
		"session_id": saved.ID.String(),
		"user_id":    saved.UserID.String(),
		"model_id":   saved.ModelID,
	}))
	return dto.SessionDTO{
		ID:        saved.ID,
		ModelID:   saved.ModelID,
		ModelHash: saved.ModelHash,
		CreatedAt: saved.CreatedAt,
		EndedAt:   saved.EndedAt,
	}, nil
}
