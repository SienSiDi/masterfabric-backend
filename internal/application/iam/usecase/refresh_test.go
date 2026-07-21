package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	iammodel "github.com/masterfabric/masterfabric_backend/internal/domain/iam/model"
	"github.com/masterfabric/masterfabric_backend/internal/application/iam/dto"
	domainerr "github.com/masterfabric/masterfabric_backend/internal/shared/errors"
	"github.com/masterfabric/masterfabric_backend/internal/shared/events"
	"github.com/masterfabric/masterfabric_backend/internal/infrastructure/auth"
)

// seedLoginUser registers + logs in, returning the raw refresh token + user id.
func seedLoginUser(t *testing.T, userRepo *mockUserRepo, roleRepo *mockRoleRepo, refreshRepo *mockRefreshRepo, jwtSvc *auth.JWTService) (string, uuid.UUID) {
	t.Helper()
	bus := events.NewInProcessBus()
	regUC := NewRegisterUseCase(userRepo, roleRepo, bus)
	_, err := regUC.Execute(context.Background(), dto.RegisterRequest{
		Email: "refresher@example.com", Password: "a-strong-password-12",
	})
	if err != nil {
		t.Fatalf("seed register: %v", err)
	}
	loginUC := NewLoginUseCase(userRepo, roleRepo, refreshRepo, jwtSvc, bus, 168*time.Hour)
	resp, err := loginUC.Execute(context.Background(), dto.LoginRequest{
		Email: "refresher@example.com", Password: "a-strong-password-12",
	})
	if err != nil {
		t.Fatalf("seed login: %v", err)
	}
	return resp.RefreshToken, resp.User.ID
}

func TestRefreshUseCase_Execute_Success(t *testing.T) {
	userRepo := newMockUserRepo()
	roleRepo := newMockRoleRepo()
	refreshRepo := newMockRefreshRepo()
	blacklist := newMockBlacklist()
	jwtSvc := auth.NewJWTService("test-secret-at-least-32-chars-long", 15*time.Minute)

	rawRefresh, userID := seedLoginUser(t, userRepo, roleRepo, refreshRepo, jwtSvc)

	refreshUC := NewRefreshUseCase(refreshRepo, roleRepo, blacklist, jwtSvc, 168*time.Hour)
	resp, err := refreshUC.Execute(context.Background(), dto.RefreshRequest{RefreshToken: rawRefresh})
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
	if resp.ExpiresIn != 900 {
		t.Errorf("expected expiresIn=900, got %d", resp.ExpiresIn)
	}

	// Old refresh token must now be revoked
	hash := hashToken(rawRefresh)
	stored, _ := refreshRepo.FindByHash(context.Background(), hash)
	if stored == nil || !stored.IsRevoked() {
		t.Error("expected old refresh token to be revoked after rotation")
	}

	// New access token should still decode for the same user
	claims, err := jwtSvc.ParseAccess(resp.AccessToken)
	if err != nil {
		t.Fatalf("parse new access: %v", err)
	}
	if claims.UserID != userID {
		t.Errorf("expected user_id %s, got %s", userID, claims.UserID)
	}
}

func TestRefreshUseCase_Execute_RevokedRejected(t *testing.T) {
	userRepo := newMockUserRepo()
	roleRepo := newMockRoleRepo()
	refreshRepo := newMockRefreshRepo()
	blacklist := newMockBlacklist()
	jwtSvc := auth.NewJWTService("test-secret-at-least-32-chars-long", 15*time.Minute)

	rawRefresh, _ := seedLoginUser(t, userRepo, roleRepo, refreshRepo, jwtSvc)

	// Manually revoke via blacklist (simulates a previous logout)
	_ = blacklist.Revoke(context.Background(), hashToken(rawRefresh), time.Hour)

	refreshUC := NewRefreshUseCase(refreshRepo, roleRepo, blacklist, jwtSvc, 168*time.Hour)
	_, err := refreshUC.Execute(context.Background(), dto.RefreshRequest{RefreshToken: rawRefresh})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	de, ok := domainerr.As(err)
	if !ok {
		t.Fatalf("expected domain error, got %v", err)
	}
	if de.Code != domainerr.CodeUnauthorized {
		t.Errorf("expected UNAUTHORIZED, got %s", de.Code)
	}
}

func TestRefreshUseCase_Execute_ExpiredRejected(t *testing.T) {
	userRepo := newMockUserRepo()
	roleRepo := newMockRoleRepo()
	refreshRepo := newMockRefreshRepo()
	blacklist := newMockBlacklist()
	jwtSvc := auth.NewJWTService("test-secret-at-least-32-chars-long", 15*time.Minute)

	// Seed a registered user
	bus := events.NewInProcessBus()
	regUC := NewRegisterUseCase(userRepo, roleRepo, bus)
	_, _ = regUC.Execute(context.Background(), dto.RegisterRequest{
		Email: "expired@example.com", Password: "a-strong-password-12",
	})
	user, _ := userRepo.FindByEmail(context.Background(), "expired@example.com")

	// Insert an expired refresh token manually
	raw := "expired-token-raw-value-abc"
	expired := &iammodel.RefreshToken{
		ID: uuid.New(), UserID: user.ID, TokenHash: hashToken(raw),
		ExpiresAt:  time.Now().Add(-1 * time.Hour),
		CreatedAt:  time.Now().Add(-2 * time.Hour),
		LastSeenAt: time.Now().Add(-2 * time.Hour),
	}
	_ = refreshRepo.Create(context.Background(), expired)

	refreshUC := NewRefreshUseCase(refreshRepo, roleRepo, blacklist, jwtSvc, 168*time.Hour)
	_, err := refreshUC.Execute(context.Background(), dto.RefreshRequest{RefreshToken: raw})
	if err == nil {
		t.Fatal("expected error for expired refresh token")
	}
	de, _ := domainerr.As(err)
	if de.Code != domainerr.CodeUnauthorized {
		t.Errorf("expected UNAUTHORIZED, got %s", de.Code)
	}
}
