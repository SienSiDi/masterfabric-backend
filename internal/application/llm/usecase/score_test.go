package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	llmmodel "github.com/masterfabric/masterfabric_backend/internal/domain/llm/model"
	domainerr "github.com/masterfabric/masterfabric_backend/internal/shared/errors"
	"github.com/masterfabric/masterfabric_backend/internal/shared/events"
	"github.com/masterfabric/masterfabric_backend/internal/application/llm/dto"
)

// mockScoreRepo is an in-memory ScoreRepository.
type mockScoreRepo struct {
	saved        []*llmmodel.DecisionScore
	byEventID    map[uuid.UUID]*llmmodel.DecisionScore
	createErr    error
}

func newMockScoreRepo() *mockScoreRepo {
	return &mockScoreRepo{byEventID: make(map[uuid.UUID]*llmmodel.DecisionScore)}
}

func (m *mockScoreRepo) Create(_ context.Context, s *llmmodel.DecisionScore) error {
	if m.createErr != nil {
		return m.createErr
	}
	if _, exists := m.byEventID[s.EventID]; exists {
		return domainerr.New(domainerr.CodeConflict, "score already exists for this event", nil)
	}
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	if s.CreatedAt.IsZero() {
		s.CreatedAt = time.Now().UTC()
	}
	m.saved = append(m.saved, s)
	m.byEventID[s.EventID] = s
	return nil
}

func (m *mockScoreRepo) FindByEventID(_ context.Context, eventID uuid.UUID) (*llmmodel.DecisionScore, error) {
	if s, ok := m.byEventID[eventID]; ok {
		return s, nil
	}
	return nil, domainerr.New(domainerr.CodeNotFound, "score not found", nil)
}

func TestRecordScoreUseCase_Execute(t *testing.T) {
	sessionRepo := newMockSessionRepo()
	eventRepo := newMockEventRepo()
	scoreRepo := newMockScoreRepo()
	bus := events.NewInProcessBus()

	userID := uuid.New()
	session := &llmmodel.Session{
		ID: uuid.New(), UserID: userID, ModelID: "gemma-2b-q4f32_1-MLC",
		CreatedAt: time.Now().UTC(),
	}
	sessionRepo.byID[session.ID] = session

	// Seed an event
	recordEventUC := NewRecordEventUseCase(eventRepo, sessionRepo, bus)
	eventResp, _ := recordEventUC.Execute(context.Background(), userID, session.ID, dto.RecordEventRequest{
		Prompt: "hi", Completion: "hello", TokensIn: 2, TokensOut: 3, LatencyMs: 500,
	})

	uc := NewRecordScoreUseCase(scoreRepo, eventRepo, sessionRepo, bus)
	err := uc.Execute(context.Background(), userID, session.ID, dto.RecordScoreRequest{
		EventID:       eventResp.ID,
		Correctness:   0.8,
		LatencyScore:  1.0,
		SafetyFlag:    false,
		CostScore:     0.9,
		UserSignal:    "accept",
		Composite:     0.85,
	})
	if err != nil {
		t.Fatalf("record score: %v", err)
	}

	saved := scoreRepo.byEventID[eventResp.ID]
	if saved == nil {
		t.Fatal("score not saved")
	}
	if saved.Composite != 0.85 {
		t.Errorf("expected composite 0.85, got %f", saved.Composite)
	}
}

func TestRecordScoreUseCase_SafetyFlagZeroesComposite(t *testing.T) {
	sessionRepo := newMockSessionRepo()
	eventRepo := newMockEventRepo()
	scoreRepo := newMockScoreRepo()
	bus := events.NewInProcessBus()

	userID := uuid.New()
	session := &llmmodel.Session{ID: uuid.New(), UserID: userID, ModelID: "gemma", CreatedAt: time.Now().UTC()}
	sessionRepo.byID[session.ID] = session

	recordEventUC := NewRecordEventUseCase(eventRepo, sessionRepo, bus)
	eventResp, _ := recordEventUC.Execute(context.Background(), userID, session.ID, dto.RecordEventRequest{
		Prompt: "x", Completion: "y", TokensIn: 1, TokensOut: 1, LatencyMs: 100,
	})

	uc := NewRecordScoreUseCase(scoreRepo, eventRepo, sessionRepo, bus)
	// Client sends composite=0.99 but safetyFlag=true → server must override to 0.0
	err := uc.Execute(context.Background(), userID, session.ID, dto.RecordScoreRequest{
		EventID:      eventResp.ID,
		Correctness:  0.9,
		LatencyScore: 1.0,
		SafetyFlag:   true,
		CostScore:    0.9,
		UserSignal:   "accept",
		Composite:    0.99, // should be overridden
	})
	if err != nil {
		t.Fatalf("record score: %v", err)
	}

	saved := scoreRepo.byEventID[eventResp.ID]
	if saved.Composite != 0.0 {
		t.Errorf("safetyFlag=true must force composite=0.0, got %f", saved.Composite)
	}
	if !saved.SafetyFlag {
		t.Error("safetyFlag should be stored as true")
	}
}

func TestRecordScoreUseCase_NonOwnerForbidden(t *testing.T) {
	sessionRepo := newMockSessionRepo()
	eventRepo := newMockEventRepo()
	scoreRepo := newMockScoreRepo()
	bus := events.NewInProcessBus()

	userID := uuid.New()
	otherUserID := uuid.New()
	session := &llmmodel.Session{ID: uuid.New(), UserID: userID, ModelID: "gemma", CreatedAt: time.Now().UTC()}
	sessionRepo.byID[session.ID] = session

	uc := NewRecordScoreUseCase(scoreRepo, eventRepo, sessionRepo, bus)
	err := uc.Execute(context.Background(), otherUserID, session.ID, dto.RecordScoreRequest{
		EventID: uuid.New(), Composite: 0.5,
	})
	if err == nil {
		t.Fatal("expected error for non-owner")
	}
	de, _ := domainerr.As(err)
	if de.Code != domainerr.CodeForbidden {
		t.Errorf("expected FORBIDDEN, got %s", de.Code)
	}
}

func TestRecordScoreUseCase_EventNotInSession(t *testing.T) {
	sessionRepo := newMockSessionRepo()
	eventRepo := newMockEventRepo()
	scoreRepo := newMockScoreRepo()
	bus := events.NewInProcessBus()

	userID := uuid.New()
	session := &llmmodel.Session{ID: uuid.New(), UserID: userID, ModelID: "gemma", CreatedAt: time.Now().UTC()}
	sessionRepo.byID[session.ID] = session

	uc := NewRecordScoreUseCase(scoreRepo, eventRepo, sessionRepo, bus)
	err := uc.Execute(context.Background(), userID, session.ID, dto.RecordScoreRequest{
		EventID:   uuid.New(), // not in this session
		Composite: 0.5,
	})
	if err == nil {
		t.Fatal("expected error for unknown event")
	}
	de, _ := domainerr.As(err)
	if de.Code != domainerr.CodeNotFound {
		t.Errorf("expected NOT_FOUND, got %s", de.Code)
	}
}

func TestRecordScoreUseCase_DoubleScoreConflict(t *testing.T) {
	sessionRepo := newMockSessionRepo()
	eventRepo := newMockEventRepo()
	scoreRepo := newMockScoreRepo()
	bus := events.NewInProcessBus()

	userID := uuid.New()
	session := &llmmodel.Session{ID: uuid.New(), UserID: userID, ModelID: "gemma", CreatedAt: time.Now().UTC()}
	sessionRepo.byID[session.ID] = session

	recordEventUC := NewRecordEventUseCase(eventRepo, sessionRepo, bus)
	eventResp, _ := recordEventUC.Execute(context.Background(), userID, session.ID, dto.RecordEventRequest{
		Prompt: "x", Completion: "y", TokensIn: 1, TokensOut: 1, LatencyMs: 100,
	})

	uc := NewRecordScoreUseCase(scoreRepo, eventRepo, sessionRepo, bus)
	_ = uc.Execute(context.Background(), userID, session.ID, dto.RecordScoreRequest{
		EventID: eventResp.ID, Composite: 0.5,
	})
	// Second score for same event → conflict
	err := uc.Execute(context.Background(), userID, session.ID, dto.RecordScoreRequest{
		EventID: eventResp.ID, Composite: 0.6,
	})
	if err == nil {
		t.Fatal("expected error for double score")
	}
	de, _ := domainerr.As(err)
	if de.Code != domainerr.CodeConflict {
		t.Errorf("expected CONFLICT, got %s", de.Code)
	}
}
