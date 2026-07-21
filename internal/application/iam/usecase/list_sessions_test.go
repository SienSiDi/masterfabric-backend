package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	iammodel "github.com/masterfabric/masterfabric_backend/internal/domain/iam/model"
	"github.com/masterfabric/masterfabric_backend/internal/application/iam/dto"
	"github.com/masterfabric/masterfabric_backend/internal/shared/events"
	"github.com/masterfabric/masterfabric_backend/internal/infrastructure/auth"
)

func TestListSessionsUseCase_Execute(t *testing.T) {
	userRepo := newMockUserRepo()
	roleRepo := newMockRoleRepo()
	refreshRepo := newMockRefreshRepo()
	jwtSvc := auth.NewJWTService("test-secret-at-least-32-chars-long", 15*time.Minute)
	bus := events.NewInProcessBus()

	regUC := NewRegisterUseCase(userRepo, roleRepo, bus)
	_, _ = regUC.Execute(context.Background(), dto.RegisterRequest{
		Email: "sessions@example.com", Password: "a-strong-password-12",
	})
	loginUC := NewLoginUseCase(userRepo, roleRepo, refreshRepo, jwtSvc, bus, 168*time.Hour)
	resp1, _ := loginUC.Execute(context.Background(), dto.LoginRequest{
		Email: "sessions@example.com", Password: "a-strong-password-12",
	})
	resp2, _ := loginUC.Execute(context.Background(), dto.LoginRequest{
		Email: "sessions@example.com", Password: "a-strong-password-12",
	})

	user, _ := userRepo.FindByEmail(context.Background(), "sessions@example.com")

	uc := NewListSessionsUseCase(refreshRepo)
	out, err := uc.Execute(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(out.Sessions) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(out.Sessions))
	}

	// No token values should be present in the DTO (only id + timestamps)
	for _, s := range out.Sessions {
		if s.ID == uuid.Nil {
			t.Error("session ID is zero")
		}
		if s.CreatedAt.IsZero() {
			t.Error("createdAt is zero")
		}
	}

	// Mark the first one revoked and confirm it shows up
	_ = refreshRepo.Revoke(context.Background(), refreshRepo.saved[0].ID)
	out, _ = uc.Execute(context.Background(), user.ID)
	if out.Sessions[0].RevokedAt == nil && out.Sessions[1].RevokedAt == nil {
		t.Error("expected at least one session to show a revokedAt timestamp")
	}

	// keep resp vars used
	_ = resp1
	_ = resp2
}

func TestListSessionsUseCase_Empty(t *testing.T) {
	refreshRepo := newMockRefreshRepo()
	uc := NewListSessionsUseCase(refreshRepo)
	out, err := uc.Execute(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Sessions) != 0 {
		t.Errorf("expected 0 sessions for unknown user, got %d", len(out.Sessions))
	}
}

// ensure imports stay used
var _ = iammodel.RefreshToken{}
var _ = time.Now
