package usecase

import (
	"context"
	"time"

	iamrepo "github.com/masterfabric/masterfabric_backend/internal/domain/iam/repository"
	domainerr "github.com/masterfabric/masterfabric_backend/internal/shared/errors"

	"github.com/masterfabric/masterfabric_backend/internal/application/iam/dto"
)

type LogoutUseCase struct {
	refreshRepo iamrepo.RefreshTokenRepository
	blacklist   RefreshBlacklist
}

func NewLogoutUseCase(refreshRepo iamrepo.RefreshTokenRepository, blacklist RefreshBlacklist) *LogoutUseCase {
	return &LogoutUseCase{refreshRepo: refreshRepo, blacklist: blacklist}
}

func (uc *LogoutUseCase) Execute(ctx context.Context, req dto.LogoutRequest) error {
	hash := hashToken(req.RefreshToken)

	stored, err := uc.refreshRepo.FindByHash(ctx, hash)
	if err != nil {
		// Idempotent logout: if the token is unknown, just succeed.
		if de, ok := domainerr.As(err); ok && de.Code == domainerr.CodeNotFound {
			return nil
		}
		return err
	}

	// Revoke in DB (sets revoked_at) — no-op if already revoked
	_ = uc.refreshRepo.Revoke(ctx, stored.ID)

	// Add to Redis blacklist with the remaining TTL so /auth/refresh can reject fast
	now := time.Now().UTC()
	_ = uc.blacklist.Revoke(ctx, hash, stored.RemainingLifetime(now))

	return nil
}
