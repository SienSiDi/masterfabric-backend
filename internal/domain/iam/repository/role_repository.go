package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/masterfabric/masterfabric_backend/internal/domain/iam/model"
)

type RoleRepository interface {
	FindByName(ctx context.Context, name string) (*model.Role, error)
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]model.Role, error)
	AssignToUser(ctx context.Context, userID, roleID uuid.UUID) error
}
