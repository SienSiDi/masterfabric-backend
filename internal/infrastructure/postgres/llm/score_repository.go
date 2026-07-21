package llm

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	llmmodel "github.com/masterfabric/masterfabric_backend/internal/domain/llm/model"
	llmrepo "github.com/masterfabric/masterfabric_backend/internal/domain/llm/repository"
	domainerr "github.com/masterfabric/masterfabric_backend/internal/shared/errors"
)

var _ llmrepo.ScoreRepository = (*ScoreRepository)(nil)

type ScoreRepository struct {
	pool *pgxpool.Pool
}

func NewScoreRepository(pool *pgxpool.Pool) *ScoreRepository {
	return &ScoreRepository{pool: pool}
}

func (r *ScoreRepository) Create(ctx context.Context, s *llmmodel.DecisionScore) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO decision_scores (id, event_id, correctness, latency_score, safety_flag, cost_score, user_signal, composite, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, s.ID, s.EventID, s.Correctness, s.LatencyScore, s.SafetyFlag, s.CostScore, s.UserSignal, s.Composite, s.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation on event_id
			return domainerr.New(domainerr.CodeConflict, "score already exists for this event", err)
		}
		return domainerr.New(domainerr.CodeInternal, "create decision score", fmt.Errorf("insert decision_scores: %w", err))
	}
	return nil
}

// FindByEventID returns the score for an event, or ErrNotFound if none.
func (r *ScoreRepository) FindByEventID(ctx context.Context, eventID uuid.UUID) (*llmmodel.DecisionScore, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, event_id, correctness, latency_score, safety_flag, cost_score, user_signal, composite, created_at
		FROM decision_scores WHERE event_id = $1
	`, eventID)
	var s llmmodel.DecisionScore
	if err := row.Scan(&s.ID, &s.EventID, &s.Correctness, &s.LatencyScore, &s.SafetyFlag, &s.CostScore, &s.UserSignal, &s.Composite, &s.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerr.New(domainerr.CodeNotFound, "score not found", err)
		}
		return nil, domainerr.New(domainerr.CodeInternal, "find score by event", fmt.Errorf("query decision_scores: %w", err))
	}
	return &s, nil
}
