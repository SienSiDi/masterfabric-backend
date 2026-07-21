package usecase

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	iammodel "github.com/masterfabric/masterfabric_backend/internal/domain/iam/model"
	iamrepo "github.com/masterfabric/masterfabric_backend/internal/domain/iam/repository"
	domainerr "github.com/masterfabric/masterfabric_backend/internal/shared/errors"
	"github.com/masterfabric/masterfabric_backend/internal/shared/events"

	"github.com/masterfabric/masterfabric_backend/internal/application/iam/dto"
	"github.com/masterfabric/masterfabric_backend/internal/infrastructure/auth"
)

type LoginUseCase struct {
	userRepo    iamrepo.UserRepository
	roleRepo    iamrepo.RoleRepository
	refreshRepo iamrepo.RefreshTokenRepository
	jwt         *auth.JWTService
	eventBus    events.Bus
	refreshTTL  time.Duration
}

func NewLoginUseCase(
	userRepo iamrepo.UserRepository,
	roleRepo iamrepo.RoleRepository,
	refreshRepo iamrepo.RefreshTokenRepository,
	jwt *auth.JWTService,
	eventBus events.Bus,
	refreshTTL time.Duration,
) *LoginUseCase {
	return &LoginUseCase{
		userRepo:    userRepo,
		roleRepo:    roleRepo,
		refreshRepo: refreshRepo,
		jwt:         jwt,
		eventBus:    eventBus,
		refreshTTL:  refreshTTL,
	}
}

func (uc *LoginUseCase) Execute(ctx context.Context, req dto.LoginRequest) (dto.LoginResponse, error) {
	email := strings.TrimSpace(strings.ToLower(req.Email))

	user, err := uc.userRepo.FindByEmail(ctx, email)
	if err != nil {
		if de, ok := domainerr.As(err); ok && de.Code == domainerr.CodeNotFound {
			return dto.LoginResponse{}, domainerr.New(domainerr.CodeUnauthorized, "invalid credentials", nil)
		}
		return dto.LoginResponse{}, err
	}
	if user == nil {
		return dto.LoginResponse{}, domainerr.New(domainerr.CodeUnauthorized, "invalid credentials", nil)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return dto.LoginResponse{}, domainerr.New(domainerr.CodeUnauthorized, "invalid credentials", nil)
	}

	roles, err := uc.roleRepo.FindByUserID(ctx, user.ID)
	if err != nil {
		return dto.LoginResponse{}, err
	}
	roleNames := make([]string, 0, len(roles))
	for _, r := range roles {
		roleNames = append(roleNames, r.Name)
	}

	accessToken, expiresIn, err := uc.jwt.GenerateAccess(user.ID, roleNames)
	if err != nil {
		return dto.LoginResponse{}, domainerr.New(domainerr.CodeInternal, "generate access token", err)
	}

	rawRefresh, err := generateOpaqueToken()
	if err != nil {
		return dto.LoginResponse{}, domainerr.New(domainerr.CodeInternal, "generate refresh token", err)
	}
	hash := hashToken(rawRefresh)
	now := time.Now().UTC()
	refresh := &iammodel.RefreshToken{
		ID:         uuid.New(),
		UserID:     user.ID,
		TokenHash:  hash,
		ExpiresAt:  now.Add(uc.refreshTTL),
		CreatedAt:  now,
		LastSeenAt: now,
	}
	if err := uc.refreshRepo.Create(ctx, refresh); err != nil {
		return dto.LoginResponse{}, err
	}

	uc.eventBus.Publish(ctx, events.NewEvent("user.logged_in", "masterfabric.iam", "iam.application.login", map[string]any{
		"user_id": user.ID.String(),
	}))

	return dto.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
		ExpiresIn:    expiresIn,
		User: dto.UserDTO{
			ID:    user.ID,
			Email: user.Email,
			Roles: roleNames,
		},
	}, nil
}
