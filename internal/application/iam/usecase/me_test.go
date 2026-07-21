package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/masterfabric/masterfabric_backend/internal/application/iam/dto"
	domainerr "github.com/masterfabric/masterfabric_backend/internal/shared/errors"
	"github.com/masterfabric/masterfabric_backend/internal/shared/events"
	"github.com/masterfabric/masterfabric_backend/internal/infrastructure/auth"
)

func TestMeUseCase_Execute(t *testing.T) {
	userRepo := newMockUserRepo()
	roleRepo := newMockRoleRepo()
	jwtSvc := auth.NewJWTService("test-secret-at-least-32-chars-long", 15*time.Minute)
	bus := events.NewInProcessBus()

	regUC := NewRegisterUseCase(userRepo, roleRepo, bus)
	regResp, err := regUC.Execute(context.Background(), dto.RegisterRequest{
		Email: "me@example.com", Password: "a-strong-password-12",
	})
	if err != nil {
		t.Fatalf("seed register: %v", err)
	}
	_ = jwtSvc // keep import referenced (used by other tests in this package)

	meUC := NewMeUseCase(userRepo, roleRepo)
	resp, err := meUC.Execute(context.Background(), regResp.UserID)
	if err != nil {
		t.Fatalf("me: %v", err)
	}
	if resp.ID != regResp.UserID {
		t.Errorf("expected id %s, got %s", regResp.UserID, resp.ID)
	}
	if resp.Email != "me@example.com" {
		t.Errorf("expected email me@example.com, got %s", resp.Email)
	}
	if len(resp.Roles) != 1 || resp.Roles[0] != "user" {
		t.Errorf("expected roles [user], got %v", resp.Roles)
	}
	if resp.CreatedAt.IsZero() {
		t.Error("expected non-zero createdAt")
	}
}

func TestUpdateMeUseCase_Execute(t *testing.T) {
	userRepo := newMockUserRepo()
	roleRepo := newMockRoleRepo()
	bus := events.NewInProcessBus()

	regUC := NewRegisterUseCase(userRepo, roleRepo, bus)
	regResp, _ := regUC.Execute(context.Background(), dto.RegisterRequest{
		Email: "old@example.com", Password: "a-strong-password-12",
	})

	updateUC := NewUpdateMeUseCase(userRepo, roleRepo)
	resp, err := updateUC.Execute(context.Background(), regResp.UserID, dto.UpdateMeRequest{
		Email: "new@example.com",
	})
	if err != nil {
		t.Fatalf("update me: %v", err)
	}
	if resp.Email != "new@example.com" {
		t.Errorf("expected new email, got %s", resp.Email)
	}

	// Verify the email actually changed in the repo
	stored, _ := userRepo.FindByID(context.Background(), regResp.UserID)
	if stored.Email != "new@example.com" {
		t.Errorf("repo still has old email: %s", stored.Email)
	}
}

func TestUpdateMeUseCase_DuplicateEmail(t *testing.T) {
	userRepo := newMockUserRepo()
	roleRepo := newMockRoleRepo()
	bus := events.NewInProcessBus()

	regUC := NewRegisterUseCase(userRepo, roleRepo, bus)
	regResp, _ := regUC.Execute(context.Background(), dto.RegisterRequest{
		Email: "alice@example.com", Password: "a-strong-password-12",
	})
	_, _ = regUC.Execute(context.Background(), dto.RegisterRequest{
		Email: "bob@example.com", Password: "a-strong-password-12",
	})

	updateUC := NewUpdateMeUseCase(userRepo, roleRepo)
	_, err := updateUC.Execute(context.Background(), regResp.UserID, dto.UpdateMeRequest{
		Email: "bob@example.com", // already taken by bob
	})
	if err == nil {
		t.Fatal("expected error for duplicate email")
	}
	de, ok := domainerr.As(err)
	if !ok || de.Code != domainerr.CodeConflict {
		t.Errorf("expected CONFLICT, got %v", err)
	}
}

func TestChangePasswordUseCase_Execute(t *testing.T) {
	userRepo := newMockUserRepo()
	roleRepo := newMockRoleRepo()
	refreshRepo := newMockRefreshRepo()
	jwtSvc := auth.NewJWTService("test-secret-at-least-32-chars-long", 15*time.Minute)
	bus := events.NewInProcessBus()

	regUC := NewRegisterUseCase(userRepo, roleRepo, bus)
	regResp, _ := regUC.Execute(context.Background(), dto.RegisterRequest{
		Email: "pwd@example.com", Password: "old-password-12345",
	})
	// Login to create a refresh token
	loginUC := NewLoginUseCase(userRepo, roleRepo, refreshRepo, jwtSvc, bus, 168*time.Hour)
	_, _ = loginUC.Execute(context.Background(), dto.LoginRequest{
		Email: "pwd@example.com", Password: "old-password-12345",
	})

	changeUC := NewChangePasswordUseCase(userRepo, refreshRepo)
	err := changeUC.Execute(context.Background(), regResp.UserID, dto.ChangePasswordRequest{
		CurrentPassword: "old-password-12345",
		NewPassword:     "new-password-12345",
	})
	if err != nil {
		t.Fatalf("change password: %v", err)
	}

	// Verify the hash changed
	stored, _ := userRepo.FindByID(context.Background(), regResp.UserID)
	if stored.PasswordHash == "old-password-12345" {
		t.Error("password hash was not updated")
	}

	// Verify all refresh tokens were revoked
	tokens := refreshRepo.saved
	for _, tok := range tokens {
		if tok.UserID == regResp.UserID && !tok.IsRevoked() {
			t.Error("expected all refresh tokens to be revoked after password change")
		}
	}
}

func TestChangePasswordUseCase_WrongCurrent(t *testing.T) {
	userRepo := newMockUserRepo()
	roleRepo := newMockRoleRepo()
	refreshRepo := newMockRefreshRepo()
	bus := events.NewInProcessBus()

	regUC := NewRegisterUseCase(userRepo, roleRepo, bus)
	regResp, _ := regUC.Execute(context.Background(), dto.RegisterRequest{
		Email: "wrong@example.com", Password: "correct-old-password-12",
	})

	changeUC := NewChangePasswordUseCase(userRepo, refreshRepo)
	err := changeUC.Execute(context.Background(), regResp.UserID, dto.ChangePasswordRequest{
		CurrentPassword: "wrong-current-password",
		NewPassword:     "new-password-12345",
	})
	if err == nil {
		t.Fatal("expected error for wrong current password")
	}
	de, _ := domainerr.As(err)
	if de.Code != domainerr.CodeUnauthorized {
		t.Errorf("expected UNAUTHORIZED, got %s", de.Code)
	}
}

func TestChangePasswordUseCase_SamePassword(t *testing.T) {
	userRepo := newMockUserRepo()
	roleRepo := newMockRoleRepo()
	refreshRepo := newMockRefreshRepo()
	bus := events.NewInProcessBus()

	regUC := NewRegisterUseCase(userRepo, roleRepo, bus)
	regResp, _ := regUC.Execute(context.Background(), dto.RegisterRequest{
		Email: "same@example.com", Password: "same-password-12345",
	})

	changeUC := NewChangePasswordUseCase(userRepo, refreshRepo)
	err := changeUC.Execute(context.Background(), regResp.UserID, dto.ChangePasswordRequest{
		CurrentPassword: "same-password-12345",
		NewPassword:     "same-password-12345", // same as current
	})
	if err == nil {
		t.Fatal("expected error for same password")
	}
	de, _ := domainerr.As(err)
	if de.Code != domainerr.CodeBadRequest {
		t.Errorf("expected BAD_REQUEST, got %s", de.Code)
	}
}
