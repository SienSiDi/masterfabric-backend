package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/masterfabric/masterfabric_backend/internal/domain/llm/model"
)

type EventRepository interface {
	Create(ctx context.Context, e *model.InferenceEvent) (*model.InferenceEvent, error)
	ListBySession(ctx context.Context, sessionID uuid.UUID, limit, offset int) ([]model.InferenceEvent, int, error)
}
