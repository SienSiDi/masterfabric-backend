package usecase

import (
	"context"

	"github.com/google/uuid"

	iamrepo "github.com/masterfabric/masterfabric_backend/internal/domain/iam/repository"
	domainerr "github.com/masterfabric/masterfabric_backend/internal/shared/errors"

	"github.com/masterfabric/masterfabric_backend/internal/application/iam/dto"
)

type MeUseCase struct {
	userRepo iamrepo.UserRepository
	roleRepo iamrepo.RoleRepository
}

func NewMeUseCase(userRepo iamrepo.UserRepository, roleRepo iamrepo.RoleRepository) *MeUseCase {
	return &MeUseCase{userRepo: userRepo, roleRepo: roleRepo}
}

func (uc *MeUseCase) Execute(ctx context.Context, userID uuid.UUID) (dto.MeResponse, error) {
	user, err := uc.userRepo.FindByID(ctx, userID)
	if err != nil {
		return dto.MeResponse{}, err
	}
	roles, err := uc.roleRepo.FindByUserID(ctx, userID)
	if err != nil {
		return dto.MeResponse{}, err
	}
	roleNames := make([]string, 0, len(roles))
	for _, r := range roles {
		roleNames = append(roleNames, r.Name)
	}
	return dto.MeResponse{
		ID:        user.ID,
		Email:     user.Email,
		Roles:     roleNames,
		CreatedAt: user.CreatedAt,
	}, nil
}

// keep domainerr referenced for future use (e.g. NotFound mapping)
var _ = domainerr.ErrNotFound
