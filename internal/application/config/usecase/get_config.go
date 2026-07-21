package usecase

import (
	"context"

	"github.com/masterfabric/masterfabric_backend/internal/domain/config/model"
	"github.com/masterfabric/masterfabric_backend/internal/domain/config/repository"
)

type GetConfigUseCase struct {
	repo repository.ConfigRepository
}

func NewGetConfigUseCase(repo repository.ConfigRepository) *GetConfigUseCase {
	return &GetConfigUseCase{repo: repo}
}

func (uc *GetConfigUseCase) Execute(ctx context.Context) (*model.AppConfig, error) {
	cfg, err := uc.repo.Get(ctx)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		d := model.Default()
		return &d, nil
	}
	return cfg, nil
}
