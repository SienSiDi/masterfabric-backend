package usecase

import (
	"context"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	iamrepo "github.com/masterfabric/masterfabric_backend/internal/domain/iam/repository"
	domainerr "github.com/masterfabric/masterfabric_backend/internal/shared/errors"

	"github.com/masterfabric/masterfabric_backend/internal/application/iam/dto"
)

type ChangePasswordUseCase struct {
	userRepo    iamrepo.UserRepository
	refreshRepo iamrepo.RefreshTokenRepository
}

func NewChangePasswordUseCase(userRepo iamrepo.UserRepository, refreshRepo iamrepo.RefreshTokenRepository) *ChangePasswordUseCase {
	return &ChangePasswordUseCase{userRepo: userRepo, refreshRepo: refreshRepo}
}

func (uc *ChangePasswordUseCase) Execute(ctx context.Context, userID uuid.UUID, req dto.ChangePasswordRequest) error {
	user, err := uc.userRepo.FindByID(ctx, userID)
	if err != nil {
		return err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		return domainerr.New(domainerr.CodeUnauthorized, "current password is incorrect", nil)
	}

	if req.CurrentPassword == req.NewPassword {
		return domainerr.New(domainerr.CodeBadRequest, "new password must be different from current", nil)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return domainerr.New(domainerr.CodeInternal, "hash password", err)
	}

	if err := uc.userRepo.UpdatePassword(ctx, userID, string(hash)); err != nil {
		return err
	}

	// Revoke all refresh tokens — force re-login on all devices.
	if err := uc.refreshRepo.RevokeAllForUser(ctx, userID); err != nil {
		return err
	}

	return nil
}
