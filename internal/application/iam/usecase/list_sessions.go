package usecase

import (
	"context"

	"github.com/google/uuid"

	iamrepo "github.com/masterfabric/masterfabric_backend/internal/domain/iam/repository"

	"github.com/masterfabric/masterfabric_backend/internal/application/iam/dto"
)

type ListSessionsUseCase struct {
	refreshRepo iamrepo.RefreshTokenRepository
}

func NewListSessionsUseCase(refreshRepo iamrepo.RefreshTokenRepository) *ListSessionsUseCase {
	return &ListSessionsUseCase{refreshRepo: refreshRepo}
}

func (uc *ListSessionsUseCase) Execute(ctx context.Context, userID uuid.UUID) (dto.ListSessionsResponse, error) {
	tokens, err := uc.refreshRepo.ListByUser(ctx, userID)
	if err != nil {
		return dto.ListSessionsResponse{}, err
	}
	sessions := make([]dto.SessionDTO, 0, len(tokens))
	for _, t := range tokens {
		sessions = append(sessions, dto.SessionDTO{
			ID:         t.ID,
			CreatedAt:  t.CreatedAt,
			LastSeenAt: t.LastSeenAt,
			RevokedAt:  t.RevokedAt,
		})
	}
	return dto.ListSessionsResponse{Sessions: sessions}, nil
}
