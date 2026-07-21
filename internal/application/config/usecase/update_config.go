package usecase

import (
	"context"

	"github.com/masterfabric/masterfabric_backend/internal/domain/config/model"
	"github.com/masterfabric/masterfabric_backend/internal/domain/config/repository"
	domainerr "github.com/masterfabric/masterfabric_backend/internal/shared/errors"
)

type UpdateConfigUseCase struct {
	repo repository.ConfigRepository
}

func NewUpdateConfigUseCase(repo repository.ConfigRepository) *UpdateConfigUseCase {
	return &UpdateConfigUseCase{repo: repo}
}

func (uc *UpdateConfigUseCase) Execute(ctx context.Context, req *model.AppConfig) (*model.AppConfig, error) {
	if req == nil {
		return nil, domainerr.New(domainerr.CodeBadRequest, "config body is required", nil)
	}
	if err := uc.repo.Update(ctx, req); err != nil {
		return nil, err
	}
	return uc.repo.Get(ctx)
}
