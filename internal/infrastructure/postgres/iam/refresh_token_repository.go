package iam

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	iammodel "github.com/masterfabric/masterfabric_backend/internal/domain/iam/model"
	iamrepo "github.com/masterfabric/masterfabric_backend/internal/domain/iam/repository"
	domainerr "github.com/masterfabric/masterfabric_backend/internal/shared/errors"
)

var _ iamrepo.RefreshTokenRepository = (*RefreshTokenRepository)(nil)

type RefreshTokenRepository struct {
	pool *pgxpool.Pool
}

func NewRefreshTokenRepository(pool *pgxpool.Pool) *RefreshTokenRepository {
	return &RefreshTokenRepository{pool: pool}
}

func (r *RefreshTokenRepository) Create(ctx context.Context, token *iammodel.RefreshToken) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, created_at, last_seen_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, token.ID, token.UserID, token.TokenHash, token.ExpiresAt, token.CreatedAt, token.LastSeenAt)
	if err != nil {
		return domainerr.New(domainerr.CodeInternal, "create refresh token", fmt.Errorf("insert refresh_tokens: %w", err))
	}
	return nil
}

func (r *RefreshTokenRepository) FindByHash(ctx context.Context, tokenHash string) (*iammodel.RefreshToken, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, user_id, token_hash, expires_at, created_at, last_seen_at, revoked_at
		FROM refresh_tokens WHERE token_hash = $1
	`, tokenHash)
	t, err := scanRefreshToken(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerr.New(domainerr.CodeNotFound, "refresh token not found", err)
		}
		return nil, domainerr.New(domainerr.CodeInternal, "find refresh token by hash", fmt.Errorf("query refresh_tokens: %w", err))
	}
	return t, nil
}

func (r *RefreshTokenRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	cmd, err := r.pool.Exec(ctx, `
		UPDATE refresh_tokens SET revoked_at = now()
		WHERE id = $1 AND revoked_at IS NULL
	`, id)
	if err != nil {
		return domainerr.New(domainerr.CodeInternal, "revoke refresh token", fmt.Errorf("update refresh_tokens: %w", err))
	}
	if cmd.RowsAffected() == 0 {
		// either not found or already revoked — treat as not-found for caller simplicity
		return domainerr.New(domainerr.CodeNotFound, "refresh token not found or already revoked", nil)
	}
	return nil
}

func (r *RefreshTokenRepository) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE refresh_tokens SET revoked_at = now()
		WHERE user_id = $1 AND revoked_at IS NULL
	`, userID)
	if err != nil {
		return domainerr.New(domainerr.CodeInternal, "revoke all refresh tokens for user", fmt.Errorf("update refresh_tokens: %w", err))
	}
	return nil
}

func (r *RefreshTokenRepository) UpdateLastSeen(ctx context.Context, id uuid.UUID, at time.Time) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE refresh_tokens SET last_seen_at = $2 WHERE id = $1
	`, id, at)
	if err != nil {
		return domainerr.New(domainerr.CodeInternal, "update last_seen_at", fmt.Errorf("update refresh_tokens: %w", err))
	}
	return nil
}

func (r *RefreshTokenRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]iammodel.RefreshToken, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, token_hash, expires_at, created_at, last_seen_at, revoked_at
		FROM refresh_tokens
		WHERE user_id = $1
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, domainerr.New(domainerr.CodeInternal, "list refresh tokens by user", fmt.Errorf("query refresh_tokens: %w", err))
	}
	defer rows.Close()
	var out []iammodel.RefreshToken
	for rows.Next() {
		t, err := scanRefreshToken(rows)
		if err != nil {
			return nil, domainerr.New(domainerr.CodeInternal, "scan refresh token", err)
		}
		out = append(out, *t)
	}
	if err := rows.Err(); err != nil {
		return nil, domainerr.New(domainerr.CodeInternal, "iterate refresh tokens", err)
	}
	return out, nil
}

type refreshTokenScanner interface {
	Scan(dest ...any) error
}

func scanRefreshToken(row refreshTokenScanner) (*iammodel.RefreshToken, error) {
	var t iammodel.RefreshToken
	if err := row.Scan(&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &t.CreatedAt, &t.LastSeenAt, &t.RevokedAt); err != nil {
		return nil, err
	}
	return &t, nil
}
