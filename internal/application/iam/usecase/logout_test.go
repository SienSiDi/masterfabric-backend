package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/masterfabric/masterfabric_backend/internal/application/iam/dto"
	"github.com/masterfabric/masterfabric_backend/internal/shared/events"
	"github.com/masterfabric/masterfabric_backend/internal/infrastructure/auth"
)

func TestLogoutUseCase_Execute(t *testing.T) {
	userRepo := newMockUserRepo()
	roleRepo := newMockRoleRepo()
	refreshRepo := newMockRefreshRepo()
	blacklist := newMockBlacklist()
	jwtSvc := auth.NewJWTService("test-secret-at-least-32-chars-long", 15*time.Minute)
	bus := events.NewInProcessBus()

	regUC := NewRegisterUseCase(userRepo, roleRepo, bus)
	_, _ = regUC.Execute(context.Background(), dto.RegisterRequest{
		Email: "logout@example.com", Password: "a-strong-password-12",
	})
	loginUC := NewLoginUseCase(userRepo, roleRepo, refreshRepo, jwtSvc, bus, 168*time.Hour)
	resp, err := loginUC.Execute(context.Background(), dto.LoginRequest{
		Email: "logout@example.com", Password: "a-strong-password-12",
	})
	if err != nil {
		t.Fatalf("seed login: %v", err)
	}

	logoutUC := NewLogoutUseCase(refreshRepo, blacklist)
	if err := logoutUC.Execute(context.Background(), dto.LogoutRequest{RefreshToken: resp.RefreshToken}); err != nil {
		t.Fatalf("logout: %v", err)
	}

	// Token should now be revoked in DB
	stored, _ := refreshRepo.FindByHash(context.Background(), hashToken(resp.RefreshToken))
	if stored == nil {
		t.Fatal("expected refresh token to still exist in DB after logout")
	}
	if !stored.IsRevoked() {
		t.Error("expected refresh token to be revoked after logout")
	}

	// Token should be in the blacklist
	revoked, _ := blacklist.IsRevoked(context.Background(), hashToken(resp.RefreshToken))
	if !revoked {
		t.Error("expected refresh token to be in the Redis blacklist after logout")
	}

	// Subsequent refresh with the revoked token should fail
	refreshUC := NewRefreshUseCase(refreshRepo, roleRepo, blacklist, jwtSvc, 168*time.Hour)
	_, err = refreshUC.Execute(context.Background(), dto.RefreshRequest{RefreshToken: resp.RefreshToken})
	if err == nil {
		t.Fatal("expected refresh after logout to fail")
	}
}

func TestLogoutUseCase_Execute_Idempotent(t *testing.T) {
	refreshRepo := newMockRefreshRepo()
	blacklist := newMockBlacklist()

	logoutUC := NewLogoutUseCase(refreshRepo, blacklist)
	// Unknown token — should be idempotent (no error)
	if err := logoutUC.Execute(context.Background(), dto.LogoutRequest{RefreshToken: "never-existed"}); err != nil {
		t.Errorf("logout with unknown token should be idempotent, got: %v", err)
	}
}

func TestLogoutUseCase_Execute_TwiceSafe(t *testing.T) {
	userRepo := newMockUserRepo()
	roleRepo := newMockRoleRepo()
	refreshRepo := newMockRefreshRepo()
	blacklist := newMockBlacklist()
	jwtSvc := auth.NewJWTService("test-secret-at-least-32-chars-long", 15*time.Minute)
	bus := events.NewInProcessBus()

	regUC := NewRegisterUseCase(userRepo, roleRepo, bus)
	_, _ = regUC.Execute(context.Background(), dto.RegisterRequest{
		Email: "twice@example.com", Password: "a-strong-password-12",
	})
	loginUC := NewLoginUseCase(userRepo, roleRepo, refreshRepo, jwtSvc, bus, 168*time.Hour)
	resp, _ := loginUC.Execute(context.Background(), dto.LoginRequest{
		Email: "twice@example.com", Password: "a-strong-password-12",
	})

	logoutUC := NewLogoutUseCase(refreshRepo, blacklist)
	_ = logoutUC.Execute(context.Background(), dto.LogoutRequest{RefreshToken: resp.RefreshToken})
	// Second logout should not error (idempotent)
	if err := logoutUC.Execute(context.Background(), dto.LogoutRequest{RefreshToken: resp.RefreshToken}); err != nil {
		t.Errorf("second logout should be idempotent, got: %v", err)
	}
}

// keep time import referenced (used in refresh_test.go seed helpers)
var _ = time.Now
