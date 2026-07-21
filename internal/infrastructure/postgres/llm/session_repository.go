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

var _ llmrepo.SessionRepository = (*SessionRepository)(nil)

type SessionRepository struct {
	pool *pgxpool.Pool
}

func NewSessionRepository(pool *pgxpool.Pool) *SessionRepository {
	return &SessionRepository{pool: pool}
}

func (r *SessionRepository) Create(ctx context.Context, s *llmmodel.Session) (*llmmodel.Session, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO llm_sessions (id, user_id, model_id, model_hash, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, model_id, model_hash, created_at, ended_at
	`, s.ID, s.UserID, s.ModelID, s.ModelHash, s.CreatedAt)
	return scanSession(row)
}

func (r *SessionRepository) FindByID(ctx context.Context, id uuid.UUID) (*llmmodel.Session, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, user_id, model_id, model_hash, created_at, ended_at
		FROM llm_sessions WHERE id = $1
	`, id)
	s, err := scanSession(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerr.New(domainerr.CodeNotFound, "session not found", err)
		}
		return nil, domainerr.New(domainerr.CodeInternal, "find session by id", fmt.Errorf("query llm_sessions: %w", err))
	}
	return s, nil
}

type sessionScanner interface {
	Scan(dest ...any) error
}

func scanSession(row sessionScanner) (*llmmodel.Session, error) {
	var s llmmodel.Session
	if err := row.Scan(&s.ID, &s.UserID, &s.ModelID, &s.ModelHash, &s.CreatedAt, &s.EndedAt); err != nil {
		return nil, err
	}
	return &s, nil
}
