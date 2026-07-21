package llm

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	llmmodel "github.com/masterfabric/masterfabric_backend/internal/domain/llm/model"
	llmrepo "github.com/masterfabric/masterfabric_backend/internal/domain/llm/repository"
	domainerr "github.com/masterfabric/masterfabric_backend/internal/shared/errors"
)

var _ llmrepo.EventRepository = (*EventRepository)(nil)

type EventRepository struct {
	pool *pgxpool.Pool
}

func NewEventRepository(pool *pgxpool.Pool) *EventRepository {
	return &EventRepository{pool: pool}
}

func (r *EventRepository) Create(ctx context.Context, e *llmmodel.InferenceEvent) (*llmmodel.InferenceEvent, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO inference_events (id, session_id, user_id, prompt, completion, tokens_in, tokens_out, latency_ms, error, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, session_id, user_id, prompt, completion, tokens_in, tokens_out, latency_ms, error, created_at
	`, e.ID, e.SessionID, e.UserID, e.Prompt, e.Completion, e.TokensIn, e.TokensOut, e.LatencyMs, e.Error, e.CreatedAt)
	return scanEvent(row)
}

func (r *EventRepository) ListBySession(ctx context.Context, sessionID uuid.UUID, limit, offset int) ([]llmmodel.InferenceEvent, int, error) {
	// Total count first
	var total int
	err := r.pool.QueryRow(ctx, `SELECT count(*) FROM inference_events WHERE session_id = $1`, sessionID).Scan(&total)
	if err != nil {
		return nil, 0, domainerr.New(domainerr.CodeInternal, "count events", fmt.Errorf("count inference_events: %w", err))
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, session_id, user_id, prompt, completion, tokens_in, tokens_out, latency_ms, error, created_at
		FROM inference_events
		WHERE session_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, sessionID, limit, offset)
	if err != nil {
		return nil, 0, domainerr.New(domainerr.CodeInternal, "list events", fmt.Errorf("query inference_events: %w", err))
	}
	defer rows.Close()

	var out []llmmodel.InferenceEvent
	for rows.Next() {
		e, err := scanEvent(rows)
		if err != nil {
			return nil, 0, domainerr.New(domainerr.CodeInternal, "scan event", err)
		}
		out = append(out, *e)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, domainerr.New(domainerr.CodeInternal, "iterate events", err)
	}
	return out, total, nil
}

type eventScanner interface {
	Scan(dest ...any) error
}

func scanEvent(row eventScanner) (*llmmodel.InferenceEvent, error) {
	var e llmmodel.InferenceEvent
	if err := row.Scan(&e.ID, &e.SessionID, &e.UserID, &e.Prompt, &e.Completion, &e.TokensIn, &e.TokensOut, &e.LatencyMs, &e.Error, &e.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerr.New(domainerr.CodeNotFound, "event not found", err)
		}
		return nil, err
	}
	return &e, nil
}
