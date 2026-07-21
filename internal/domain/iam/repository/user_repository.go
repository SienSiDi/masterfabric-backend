package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/masterfabric/masterfabric_backend/internal/domain/iam/model"
)

type UserRepository interface {
	Create(ctx context.Context, email, passwordHash string) (*model.User, error)
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	UpdateEmail(ctx context.Context, id uuid.UUID, newEmail string) (*model.User, error)
	UpdatePassword(ctx context.Context, id uuid.UUID, newPasswordHash string) error
}
