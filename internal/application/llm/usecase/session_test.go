package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	llmmodel "github.com/masterfabric/masterfabric_backend/internal/domain/llm/model"
	llmrepo "github.com/masterfabric/masterfabric_backend/internal/domain/llm/repository"
	domainerr "github.com/masterfabric/masterfabric_backend/internal/shared/errors"
	"github.com/masterfabric/masterfabric_backend/internal/shared/events"
	"github.com/masterfabric/masterfabric_backend/internal/application/llm/dto"
)

// mockSessionRepo is an in-memory SessionRepository for unit tests.
type mockSessionRepo struct {
	byID map[uuid.UUID]*llmmodel.Session
}

func newMockSessionRepo() *mockSessionRepo {
	return &mockSessionRepo{byID: make(map[uuid.UUID]*llmmodel.Session)}
}

func (m *mockSessionRepo) Create(_ context.Context, s *llmmodel.Session) (*llmmodel.Session, error) {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	if s.CreatedAt.IsZero() {
		s.CreatedAt = time.Now().UTC()
	}
	m.byID[s.ID] = s
	return s, nil
}

func (m *mockSessionRepo) FindByID(_ context.Context, id uuid.UUID) (*llmmodel.Session, error) {
	if s, ok := m.byID[id]; ok {
		return s, nil
	}
	return nil, domainerr.New(domainerr.CodeNotFound, "session not found", nil)
}

// keep llmrepo referenced
var _ llmrepo.SessionRepository = (*mockSessionRepo)(nil)

func TestCreateSessionUseCase_Execute(t *testing.T) {
	sessionRepo := newMockSessionRepo()
	bus := events.NewInProcessBus()
	uc := NewCreateSessionUseCase(sessionRepo, bus)

	userID := uuid.New()
	resp, err := uc.Execute(context.Background(), userID, dto.CreateSessionRequest{
		ModelID:   "gemma-2-2b-it-q4f32_1-MLC",
		ModelHash: "sha256:abc123",
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if resp.ID == uuid.Nil {
		t.Error("expected non-zero session ID")
	}
	if resp.ModelID != "gemma-2-2b-it-q4f32_1-MLC" {
		t.Errorf("expected modelId gemma-2-2b-it-q4f32_1-MLC, got %s", resp.ModelID)
	}
	if resp.ModelHash != "sha256:abc123" {
		t.Errorf("expected modelHash sha256:abc123, got %s", resp.ModelHash)
	}

	// Verify it was actually saved
	stored, _ := sessionRepo.FindByID(context.Background(), resp.ID)
	if stored == nil {
		t.Fatal("session not found in repo after create")
	}
	if stored.UserID != userID {
		t.Errorf("expected userID %s, got %s", userID, stored.UserID)
	}
}

func TestCreateSessionUseCase_DefaultModel(t *testing.T) {
	sessionRepo := newMockSessionRepo()
	bus := events.NewInProcessBus()
	uc := NewCreateSessionUseCase(sessionRepo, bus)

	resp, err := uc.Execute(context.Background(), uuid.New(), dto.CreateSessionRequest{
		ModelID: "", // empty — should default
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if resp.ModelID != "gemma-2-2b-it-q4f32_1-MLC" {
		t.Errorf("expected default model, got %s", resp.ModelID)
	}
}

func TestGetSessionUseCase_Execute(t *testing.T) {
	sessionRepo := newMockSessionRepo()
	uc := NewGetSessionUseCase(sessionRepo)

	userID := uuid.New()
	otherUserID := uuid.New()

	// Seed a session
	sess := &llmmodel.Session{
		ID: uuid.New(), UserID: userID, ModelID: "gemma-2-2b-it-q4f32_1-MLC",
		CreatedAt: time.Now().UTC(),
	}
	sessionRepo.byID[sess.ID] = sess

	// Owner can fetch
	resp, err := uc.Execute(context.Background(), userID, sess.ID)
	if err != nil {
		t.Fatalf("get session as owner: %v", err)
	}
	if resp.ID != sess.ID {
		t.Errorf("expected id %s, got %s", sess.ID, resp.ID)
	}

	// Non-owner gets 403
	_, err = uc.Execute(context.Background(), otherUserID, sess.ID)
	if err == nil {
		t.Fatal("expected error for non-owner")
	}
	de, ok := domainerr.As(err)
	if !ok || de.Code != domainerr.CodeForbidden {
		t.Errorf("expected FORBIDDEN, got %v", err)
	}
}

func TestGetSessionUseCase_NotFound(t *testing.T) {
	sessionRepo := newMockSessionRepo()
	uc := NewGetSessionUseCase(sessionRepo)

	_, err := uc.Execute(context.Background(), uuid.New(), uuid.New())
	if err == nil {
		t.Fatal("expected error for unknown session")
	}
	de, _ := domainerr.As(err)
	if de.Code != domainerr.CodeNotFound {
		t.Errorf("expected NOT_FOUND, got %s", de.Code)
	}
}
