package usecase

import (
	"context"

	"github.com/google/uuid"

	llmrepo "github.com/masterfabric/masterfabric_backend/internal/domain/llm/repository"
	domainerr "github.com/masterfabric/masterfabric_backend/internal/shared/errors"
	"github.com/masterfabric/masterfabric_backend/internal/application/llm/dto"
)

type GetSessionUseCase struct {
	sessionRepo llmrepo.SessionRepository
}

func NewGetSessionUseCase(sessionRepo llmrepo.SessionRepository) *GetSessionUseCase {
	return &GetSessionUseCase{sessionRepo: sessionRepo}
}

// Execute returns the session if it exists AND belongs to the requesting user.
func (uc *GetSessionUseCase) Execute(ctx context.Context, requesterUserID, sessionID uuid.UUID) (dto.SessionDTO, error) {
	s, err := uc.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return dto.SessionDTO{}, err
	}
	if s.UserID != requesterUserID {
		return dto.SessionDTO{}, domainerr.New(domainerr.CodeForbidden, "not owner of session", nil)
	}
	return dto.SessionDTO{
		ID:        s.ID,
		ModelID:   s.ModelID,
		ModelHash: s.ModelHash,
		CreatedAt: s.CreatedAt,
		EndedAt:   s.EndedAt,
	}, nil
}
