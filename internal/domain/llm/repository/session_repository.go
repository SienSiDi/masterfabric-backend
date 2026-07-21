package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/masterfabric/masterfabric_backend/internal/domain/llm/model"
)

type SessionRepository interface {
	Create(ctx context.Context, s *model.Session) (*model.Session, error)
	FindByID(ctx context.Context, id uuid.UUID) (*model.Session, error)
}
