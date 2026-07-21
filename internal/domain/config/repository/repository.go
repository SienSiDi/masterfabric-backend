package repository

import (
	"context"

	"github.com/masterfabric/masterfabric_backend/internal/domain/config/model"
)

type ConfigRepository interface {
	Get(ctx context.Context) (*model.AppConfig, error)
	Update(ctx context.Context, cfg *model.AppConfig) error
}
