package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/masterfabric/masterfabric_backend/internal/domain/iam/model"
)

type RefreshTokenRepository interface {
	Create(ctx context.Context, token *model.RefreshToken) error
	FindByHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error)
	Revoke(ctx context.Context, id uuid.UUID) error
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
	UpdateLastSeen(ctx context.Context, id uuid.UUID, at time.Time) error
	ListByUser(ctx context.Context, userID uuid.UUID) ([]model.RefreshToken, error)
}
