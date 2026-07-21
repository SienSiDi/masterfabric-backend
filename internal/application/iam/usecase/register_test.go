package usecase

import (
	"context"
	"testing"

	"github.com/masterfabric/masterfabric_backend/internal/application/iam/dto"
	domainerr "github.com/masterfabric/masterfabric_backend/internal/shared/errors"
	"github.com/masterfabric/masterfabric_backend/internal/shared/events"
)

func TestRegisterUseCase_Execute(t *testing.T) {
	tests := []struct {
		name    string
		seed    map[string]string // pre-existing users: email -> passwordHash
		req     dto.RegisterRequest
		wantErr bool
		wantCode domainerr.Code
	}{
		{
			name:    "success: new email + strong password",
			req:     dto.RegisterRequest{Email: "alice@example.com", Password: "supersecretpass"},
			wantErr: false,
		},
		{
			name:    "fail: duplicate email",
			seed:    map[string]string{"bob@example.com": "$2a$10$somehash"},
			req:     dto.RegisterRequest{Email: "bob@example.com", Password: "supersecretpass"},
			wantErr: true,
			wantCode: domainerr.CodeConflict,
		},
		{
			name:    "fail: short password (caught by validator, not use case)",
			req:     dto.RegisterRequest{Email: "carol@example.com", Password: "short"},
			// Note: validation happens in the handler via validator.DecodeAndValidate.
			// The use case itself doesn't re-validate length — that's the handler's job.
			// So the use case will succeed here. This test documents that boundary.
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo := newMockUserRepo()
			for email, hash := range tt.seed {
				_, _ = userRepo.Create(context.Background(), email, hash)
			}
			roleRepo := newMockRoleRepo()
			bus := events.NewInProcessBus()

			uc := NewRegisterUseCase(userRepo, roleRepo, bus)
			resp, err := uc.Execute(context.Background(), tt.req)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				de, ok := domainerr.As(err)
				if !ok {
					t.Fatalf("expected domain error, got %v", err)
				}
				if tt.wantCode != "" && de.Code != tt.wantCode {
					t.Errorf("expected code %s, got %s", tt.wantCode, de.Code)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.Email != tt.req.Email {
				t.Errorf("expected email %s, got %s", tt.req.Email, resp.Email)
			}
			if roleRepo.assignCalls == 0 {
				t.Error("expected AssignToUser to be called at least once")
			}
		})
	}
}

func TestRegisterUseCase_BcryptHashIsStored(t *testing.T) {
	userRepo := newMockUserRepo()
	roleRepo := newMockRoleRepo()
	bus := events.NewInProcessBus()

	uc := NewRegisterUseCase(userRepo, roleRepo, bus)
	_, err := uc.Execute(context.Background(), dto.RegisterRequest{
		Email:    "dave@example.com",
		Password: "a-strong-password-12",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stored, err := userRepo.FindByEmail(context.Background(), "dave@example.com")
	if err != nil {
		t.Fatalf("find user: %v", err)
	}
	if stored.PasswordHash == "a-strong-password-12" {
		t.Error("password was stored in plaintext — bcrypt hash expected")
	}
	if len(stored.PasswordHash) < 50 {
		t.Errorf("bcrypt hash too short (%d chars) — expected ~60", len(stored.PasswordHash))
	}
}
