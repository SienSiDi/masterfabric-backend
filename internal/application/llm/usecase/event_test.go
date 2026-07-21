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

// mockEventRepo is an in-memory EventRepository for unit tests.
type mockEventRepo struct {
	bySession map[uuid.UUID][]*llmmodel.InferenceEvent
}

func newMockEventRepo() *mockEventRepo {
	return &mockEventRepo{bySession: make(map[uuid.UUID][]*llmmodel.InferenceEvent)}
}

func (m *mockEventRepo) Create(_ context.Context, e *llmmodel.InferenceEvent) (*llmmodel.InferenceEvent, error) {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now().UTC()
	}
	m.bySession[e.SessionID] = append(m.bySession[e.SessionID], e)
	return e, nil
}

func (m *mockEventRepo) ListBySession(_ context.Context, sessionID uuid.UUID, limit, offset int) ([]llmmodel.InferenceEvent, int, error) {
	all := m.bySession[sessionID]
	total := len(all)
	// return in reverse order (newest first) to mimic the SQL ORDER BY
	out := make([]llmmodel.InferenceEvent, 0, limit)
	for i := total - 1 - offset; i >= 0 && len(out) < limit; i-- {
		out = append(out, *all[i])
	}
	return out, total, nil
}

func TestRecordEventUseCase_Execute(t *testing.T) {
	sessionRepo := newMockSessionRepo()
	eventRepo := newMockEventRepo()
	bus := events.NewInProcessBus()

	userID := uuid.New()
	session := &llmmodel.Session{
		ID: uuid.New(), UserID: userID, ModelID: "gemma-2-2b-it-q4f32_1-MLC",
		CreatedAt: time.Now().UTC(),
	}
	sessionRepo.byID[session.ID] = session

	uc := NewRecordEventUseCase(eventRepo, sessionRepo, bus)
	resp, err := uc.Execute(context.Background(), userID, session.ID, dto.RecordEventRequest{
		Prompt: "Hello Gemma", Completion: "Hi there!",
		TokensIn: 5, TokensOut: 10, LatencyMs: 4200,
	})
	if err != nil {
		t.Fatalf("record event: %v", err)
	}
	if resp.ID == uuid.Nil {
		t.Error("expected non-zero event ID")
	}
	if resp.Prompt != "Hello Gemma" {
		t.Errorf("expected prompt 'Hello Gemma', got %s", resp.Prompt)
	}
	if resp.TokensIn != 5 || resp.TokensOut != 10 {
		t.Errorf("token counts wrong: in=%d out=%d", resp.TokensIn, resp.TokensOut)
	}

	// Verify it was saved
	events := eventRepo.bySession[session.ID]
	if len(events) != 1 {
		t.Errorf("expected 1 event saved, got %d", len(events))
	}
}

func TestRecordEventUseCase_NonOwnerForbidden(t *testing.T) {
	sessionRepo := newMockSessionRepo()
	eventRepo := newMockEventRepo()
	bus := events.NewInProcessBus()

	userID := uuid.New()
	otherUserID := uuid.New()
	session := &llmmodel.Session{
		ID: uuid.New(), UserID: userID, ModelID: "gemma-2-2b-it-q4f32_1-MLC",
		CreatedAt: time.Now().UTC(),
	}
	sessionRepo.byID[session.ID] = session

	uc := NewRecordEventUseCase(eventRepo, sessionRepo, bus)
	_, err := uc.Execute(context.Background(), otherUserID, session.ID, dto.RecordEventRequest{
		Prompt: "Hello", Completion: "Hi", TokensIn: 1, TokensOut: 1, LatencyMs: 100,
	})
	if err == nil {
		t.Fatal("expected error for non-owner")
	}
	de, ok := domainerr.As(err)
	if !ok || de.Code != domainerr.CodeForbidden {
		t.Errorf("expected FORBIDDEN, got %v", err)
	}
}

func TestRecordEventUseCase_UnknownSession(t *testing.T) {
	sessionRepo := newMockSessionRepo()
	eventRepo := newMockEventRepo()
	bus := events.NewInProcessBus()

	uc := NewRecordEventUseCase(eventRepo, sessionRepo, bus)
	_, err := uc.Execute(context.Background(), uuid.New(), uuid.New(), dto.RecordEventRequest{
		Prompt: "Hello", Completion: "Hi", TokensIn: 1, TokensOut: 1, LatencyMs: 100,
	})
	if err == nil {
		t.Fatal("expected error for unknown session")
	}
	de, _ := domainerr.As(err)
	if de.Code != domainerr.CodeNotFound {
		t.Errorf("expected NOT_FOUND, got %s", de.Code)
	}
}

func TestListEventsUseCase_Execute(t *testing.T) {
	sessionRepo := newMockSessionRepo()
	eventRepo := newMockEventRepo()
	bus := events.NewInProcessBus()

	userID := uuid.New()
	session := &llmmodel.Session{
		ID: uuid.New(), UserID: userID, ModelID: "gemma-2-2b-it-q4f32_1-MLC",
		CreatedAt: time.Now().UTC(),
	}
	sessionRepo.byID[session.ID] = session

	// Seed 3 events
	recordUC := NewRecordEventUseCase(eventRepo, sessionRepo, bus)
	for i := 0; i < 3; i++ {
		_, _ = recordUC.Execute(context.Background(), userID, session.ID, dto.RecordEventRequest{
			Prompt: "prompt", Completion: "completion", TokensIn: 1, TokensOut: 1, LatencyMs: 100 * i,
		})
	}

	listUC := NewListEventsUseCase(eventRepo, sessionRepo)
	resp, err := listUC.Execute(context.Background(), userID, session.ID, 1, 20)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if resp.Total != 3 {
		t.Errorf("expected total=3, got %d", resp.Total)
	}
	if len(resp.Events) != 3 {
		t.Errorf("expected 3 events, got %d", len(resp.Events))
	}
	// Newest first — the last event recorded should have the highest latency (200)
	if resp.Events[0].LatencyMs != 200 {
		t.Errorf("expected newest event latency=200, got %d", resp.Events[0].LatencyMs)
	}
}

func TestListEventsUseCase_NonOwnerForbidden(t *testing.T) {
	sessionRepo := newMockSessionRepo()
	eventRepo := newMockEventRepo()

	userID := uuid.New()
	otherUserID := uuid.New()
	session := &llmmodel.Session{
		ID: uuid.New(), UserID: userID, ModelID: "gemma-2-2b-it-q4f32_1-MLC",
		CreatedAt: time.Now().UTC(),
	}
	sessionRepo.byID[session.ID] = session

	listUC := NewListEventsUseCase(eventRepo, sessionRepo)
	_, err := listUC.Execute(context.Background(), otherUserID, session.ID, 1, 20)
	if err == nil {
		t.Fatal("expected error for non-owner")
	}
	de, _ := domainerr.As(err)
	if de.Code != domainerr.CodeForbidden {
		t.Errorf("expected FORBIDDEN, got %s", de.Code)
	}
}
