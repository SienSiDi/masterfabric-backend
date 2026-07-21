package usecase

import (
	"context"
	"strings"

	"github.com/google/uuid"

	iamrepo "github.com/masterfabric/masterfabric_backend/internal/domain/iam/repository"
	"github.com/masterfabric/masterfabric_backend/internal/application/iam/dto"
)

type UpdateMeUseCase struct {
	userRepo iamrepo.UserRepository
	roleRepo iamrepo.RoleRepository
}

func NewUpdateMeUseCase(userRepo iamrepo.UserRepository, roleRepo iamrepo.RoleRepository) *UpdateMeUseCase {
	return &UpdateMeUseCase{userRepo: userRepo, roleRepo: roleRepo}
}

func (uc *UpdateMeUseCase) Execute(ctx context.Context, userID uuid.UUID, req dto.UpdateMeRequest) (dto.MeResponse, error) {
	email := strings.TrimSpace(strings.ToLower(req.Email))
	user, err := uc.userRepo.UpdateEmail(ctx, userID, email)
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
