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

func TestLoginUseCase_Execute(t *testing.T) {
	// Seed a registered user via the RegisterUseCase so the bcrypt hash matches.
	userRepo := newMockUserRepo()
	roleRepo := newMockRoleRepo()
	refreshRepo := newMockRefreshRepo()
	bus := events.NewInProcessBus()
	jwtSvc := auth.NewJWTService("test-secret-at-least-32-chars-long", 15*time.Minute)

	regUC := NewRegisterUseCase(userRepo, roleRepo, bus)
	_, err := regUC.Execute(context.Background(), dto.RegisterRequest{
		Email:    "alice@example.com",
		Password: "a-strong-password-12",
	})
	if err != nil {
		t.Fatalf("seed register: %v", err)
	}

	loginUC := NewLoginUseCase(userRepo, roleRepo, refreshRepo, jwtSvc, bus, 168*time.Hour)

	tests := []struct {
		name     string
		req      dto.LoginRequest
		wantErr  bool
		wantCode domainerr.Code
	}{
		{
			name:    "success: correct credentials",
			req:     dto.LoginRequest{Email: "alice@example.com", Password: "a-strong-password-12"},
			wantErr: false,
		},
		{
			name:     "fail: wrong password",
			req:      dto.LoginRequest{Email: "alice@example.com", Password: "wrong-password-xxxxx"},
			wantErr:  true,
			wantCode: domainerr.CodeUnauthorized,
		},
		{
			name:     "fail: unknown email",
			req:      dto.LoginRequest{Email: "nobody@example.com", Password: "a-strong-password-12"},
			wantErr:  true,
			wantCode: domainerr.CodeUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := loginUC.Execute(context.Background(), tt.req)
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
			if resp.AccessToken == "" {
				t.Error("expected non-empty access token")
			}
			if resp.RefreshToken == "" {
				t.Error("expected non-empty refresh token")
			}
			if resp.ExpiresIn != 900 {
				t.Errorf("expected expiresIn=900 (15min), got %d", resp.ExpiresIn)
			}
			if resp.User.Email != tt.req.Email {
				t.Errorf("expected user email %s, got %s", tt.req.Email, resp.User.Email)
			}
			if len(resp.User.Roles) == 0 {
				t.Error("expected at least one role (user)")
			}
			if len(refreshRepo.saved) == 0 {
				t.Error("expected a refresh token to be persisted")
			}
		})
	}
}

func TestLoginUseCase_AccessTokenDecodes(t *testing.T) {
	userRepo := newMockUserRepo()
	roleRepo := newMockRoleRepo()
	refreshRepo := newMockRefreshRepo()
	bus := events.NewInProcessBus()
	jwtSvc := auth.NewJWTService("test-secret-at-least-32-chars-long", 15*time.Minute)

	regUC := NewRegisterUseCase(userRepo, roleRepo, bus)
	regResp, err := regUC.Execute(context.Background(), dto.RegisterRequest{
		Email:    "bob@example.com",
		Password: "a-strong-password-12",
	})
	if err != nil {
		t.Fatalf("seed register: %v", err)
	}

	loginUC := NewLoginUseCase(userRepo, roleRepo, refreshRepo, jwtSvc, bus, 168*time.Hour)
	resp, err := loginUC.Execute(context.Background(), dto.LoginRequest{
		Email:    "bob@example.com",
		Password: "a-strong-password-12",
	})
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	claims, err := jwtSvc.ParseAccess(resp.AccessToken)
	if err != nil {
		t.Fatalf("parse access token: %v", err)
	}
	if claims.UserID != regResp.UserID {
		t.Errorf("expected user_id %s, got %s", regResp.UserID, claims.UserID)
	}
	if len(claims.Roles) == 0 || claims.Roles[0] != "user" {
		t.Errorf("expected roles [user], got %v", claims.Roles)
	}
}
