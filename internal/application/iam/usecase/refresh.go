package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"

	iammodel "github.com/masterfabric/masterfabric_backend/internal/domain/iam/model"
	iamrepo "github.com/masterfabric/masterfabric_backend/internal/domain/iam/repository"
	domainerr "github.com/masterfabric/masterfabric_backend/internal/shared/errors"

	"github.com/masterfabric/masterfabric_backend/internal/application/iam/dto"
	"github.com/masterfabric/masterfabric_backend/internal/infrastructure/auth"
)

type RefreshUseCase struct {
	refreshRepo iamrepo.RefreshTokenRepository
	roleRepo    iamrepo.RoleRepository
	blacklist   RefreshBlacklist
	jwt         *auth.JWTService
	refreshTTL  time.Duration
}

// RefreshBlacklist is the subset of the Redis token blacklist the refresh use case needs.
type RefreshBlacklist interface {
	IsRevoked(ctx context.Context, tokenHash string) (bool, error)
	Revoke(ctx context.Context, tokenHash string, ttl time.Duration) error
}

func NewRefreshUseCase(
	refreshRepo iamrepo.RefreshTokenRepository,
	roleRepo iamrepo.RoleRepository,
	blacklist RefreshBlacklist,
	jwt *auth.JWTService,
	refreshTTL time.Duration,
) *RefreshUseCase {
	return &RefreshUseCase{
		refreshRepo: refreshRepo,
		roleRepo:    roleRepo,
		blacklist:   blacklist,
		jwt:         jwt,
		refreshTTL:  refreshTTL,
	}
}

func (uc *RefreshUseCase) Execute(ctx context.Context, req dto.RefreshRequest) (dto.RefreshResponse, error) {
	hash := hashToken(req.RefreshToken)

	// 1. Redis blacklist first (fast path — no DB hit for revoked tokens)
	revoked, err := uc.blacklist.IsRevoked(ctx, hash)
	if err != nil {
		return dto.RefreshResponse{}, domainerr.New(domainerr.CodeInternal, "check blacklist", err)
	}
	if revoked {
		return dto.RefreshResponse{}, domainerr.New(domainerr.CodeUnauthorized, "refresh token revoked", nil)
	}

	// 2. Look up the token row
	stored, err := uc.refreshRepo.FindByHash(ctx, hash)
	if err != nil {
		if de, ok := domainerr.As(err); ok && de.Code == domainerr.CodeNotFound {
			return dto.RefreshResponse{}, domainerr.New(domainerr.CodeUnauthorized, "invalid refresh token", nil)
		}
		return dto.RefreshResponse{}, err
	}

	now := time.Now().UTC()
	if stored.IsExpired(now) {
		return dto.RefreshResponse{}, domainerr.New(domainerr.CodeUnauthorized, "refresh token expired", nil)
	}
	if stored.IsRevoked() {
		// race: revoked in DB but not yet in Redis — sync the blacklist and reject
		_ = uc.blacklist.Revoke(ctx, hash, stored.RemainingLifetime(now))
		return dto.RefreshResponse{}, domainerr.New(domainerr.CodeUnauthorized, "refresh token revoked", nil)
	}

	// 3. Load roles for the access token
	roles, err := uc.roleRepo.FindByUserID(ctx, stored.UserID)
	if err != nil {
		return dto.RefreshResponse{}, err
	}
	roleNames := make([]string, 0, len(roles))
	for _, r := range roles {
		roleNames = append(roleNames, r.Name)
	}

	// 4. Rotate: revoke the old token (DB + Redis) and issue a new one
	if err := uc.refreshRepo.Revoke(ctx, stored.ID); err != nil {
		return dto.RefreshResponse{}, err
	}
	_ = uc.blacklist.Revoke(ctx, hash, stored.RemainingLifetime(now))

	rawNew, err := generateOpaqueToken()
	if err != nil {
		return dto.RefreshResponse{}, domainerr.New(domainerr.CodeInternal, "generate refresh token", err)
	}
	newToken := &iammodel.RefreshToken{
		ID:         uuid.New(),
		UserID:     stored.UserID,
		TokenHash:  hashToken(rawNew),
		ExpiresAt:  now.Add(uc.refreshTTL),
		CreatedAt:  now,
		LastSeenAt: now,
	}
	if err := uc.refreshRepo.Create(ctx, newToken); err != nil {
		return dto.RefreshResponse{}, err
	}

	// 5. Issue a new access token
	accessToken, expiresIn, err := uc.jwt.GenerateAccess(stored.UserID, roleNames)
	if err != nil {
		return dto.RefreshResponse{}, domainerr.New(domainerr.CodeInternal, "generate access token", err)
	}

	return dto.RefreshResponse{
		AccessToken:  accessToken,
		RefreshToken: rawNew,
		ExpiresIn:    expiresIn,
	}, nil
}

// RefreshResponseWithToken is returned by Execute but also includes the new refresh token
// when the caller wants to do full rotation (used by some flows). For MVP we keep the
// response shape simple: only a new access token.
type RefreshResponseWithToken struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
}
