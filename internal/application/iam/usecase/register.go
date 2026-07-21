package usecase

import (
	"context"
	"strings"

	"golang.org/x/crypto/bcrypt"

	iamrepo "github.com/masterfabric/masterfabric_backend/internal/domain/iam/repository"
	domainerr "github.com/masterfabric/masterfabric_backend/internal/shared/errors"
	"github.com/masterfabric/masterfabric_backend/internal/shared/events"

	"github.com/masterfabric/masterfabric_backend/internal/application/iam/dto"
)

type RegisterUseCase struct {
	userRepo iamrepo.UserRepository
	roleRepo iamrepo.RoleRepository
	eventBus events.Bus
}

func NewRegisterUseCase(userRepo iamrepo.UserRepository, roleRepo iamrepo.RoleRepository, eventBus events.Bus) *RegisterUseCase {
	return &RegisterUseCase{userRepo: userRepo, roleRepo: roleRepo, eventBus: eventBus}
}

func (uc *RegisterUseCase) Execute(ctx context.Context, req dto.RegisterRequest) (dto.RegisterResponse, error) {
	email := strings.TrimSpace(strings.ToLower(req.Email))

	existing, err := uc.userRepo.FindByEmail(ctx, email)
	if err != nil {
		// NotFound is expected for a new email; anything else is internal.
		de, ok := domainerr.As(err)
		if !ok || de.Code != domainerr.CodeNotFound {
			return dto.RegisterResponse{}, err
		}
	}
	if existing != nil {
		return dto.RegisterResponse{}, domainerr.New(domainerr.CodeConflict, "email already registered", nil)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return dto.RegisterResponse{}, domainerr.New(domainerr.CodeInternal, "hash password", err)
	}

	user, err := uc.userRepo.Create(ctx, email, string(hash))
	if err != nil {
		return dto.RegisterResponse{}, err
	}

	// Assign default "user" role. Non-fatal — login returns empty roles if this fails.
	if role, err := uc.roleRepo.FindByName(ctx, "user"); err == nil {
		_ = uc.roleRepo.AssignToUser(ctx, user.ID, role.ID)
	}

	uc.eventBus.Publish(ctx, events.NewEvent("user.registered", "masterfabric.iam", "iam.application.register", map[string]any{
		"user_id": user.ID.String(),
		"email":   user.Email,
	}))

	return dto.RegisterResponse{UserID: user.ID, Email: user.Email}, nil
}
