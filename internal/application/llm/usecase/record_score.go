package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"

	llmmodel "github.com/masterfabric/masterfabric_backend/internal/domain/llm/model"
	llmrepo "github.com/masterfabric/masterfabric_backend/internal/domain/llm/repository"
	domainerr "github.com/masterfabric/masterfabric_backend/internal/shared/errors"
	"github.com/masterfabric/masterfabric_backend/internal/shared/events"
	"github.com/masterfabric/masterfabric_backend/internal/shared/telemetry"
	"github.com/masterfabric/masterfabric_backend/internal/application/llm/dto"
)

type RecordScoreUseCase struct {
	scoreRepo   llmrepo.ScoreRepository
	eventRepo   llmrepo.EventRepository
	sessionRepo llmrepo.SessionRepository
	eventBus    events.Bus
}

func NewRecordScoreUseCase(
	scoreRepo llmrepo.ScoreRepository,
	eventRepo llmrepo.EventRepository,
	sessionRepo llmrepo.SessionRepository,
	eventBus events.Bus,
) *RecordScoreUseCase {
	return &RecordScoreUseCase{scoreRepo: scoreRepo, eventRepo: eventRepo, sessionRepo: sessionRepo, eventBus: eventBus}
}

// Execute records a decision score for an inference event.
// Server-side rule: if safetyFlag is true, composite is forced to 0.0
// regardless of what the client sent (prevents FE drift / tampering).
func (uc *RecordScoreUseCase) Execute(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID, req dto.RecordScoreRequest) error {
	// 1. Verify session ownership
	session, err := uc.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return err
	}
	if session.UserID != userID {
		return domainerr.New(domainerr.CodeForbidden, "not owner of session", nil)
	}

	// 2. The event must belong to this session (look it up by id)
	// We don't have a FindByID on EventRepository yet — use ListBySession and find it.
	// For efficiency we could add FindByID, but for MVP this is fine (events per session are small).
	evs, _, err := uc.eventRepo.ListBySession(ctx, sessionID, 1000, 0)
	if err != nil {
		return err
	}
	var found bool
	for _, e := range evs {
		if e.ID == req.EventID {
			found = true
			break
		}
	}
	if !found {
		return domainerr.New(domainerr.CodeNotFound, "event not found in this session", nil)
	}

	// 3. Server-side composite enforcement
	composite := req.Composite
	if req.SafetyFlag {
		composite = 0.0
	}

	// 4. Insert
	score := &llmmodel.DecisionScore{
		ID:            uuid.New(),
		EventID:       req.EventID,
		Correctness:   req.Correctness,
		LatencyScore:  req.LatencyScore,
		SafetyFlag:    req.SafetyFlag,
		CostScore:     req.CostScore,
		UserSignal:    req.UserSignal,
		Composite:     composite,
		CreatedAt:     time.Now().UTC(),
	}
	if err := uc.scoreRepo.Create(ctx, score); err != nil {
		return err
	}

	// 5. Prometheus: track composite distribution
	telemetry.LLMDecisionScoreSum.WithLabelValues(session.ModelID).Add(composite)

	uc.eventBus.Publish(ctx, events.NewEvent("llm.score.recorded", "masterfabric.llm", "llm.application.record_score", map[string]any{
		"score_id":     score.ID.String(),
		"event_id":     req.EventID.String(),
		"session_id":   sessionID.String(),
		"composite":    composite,
		"safety_flag":  req.SafetyFlag,
		"user_signal":  req.UserSignal,
	}))

	return nil
}
