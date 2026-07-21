package iam

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	iammodel "github.com/masterfabric/masterfabric_backend/internal/domain/iam/model"
	iamrepo "github.com/masterfabric/masterfabric_backend/internal/domain/iam/repository"
	domainerr "github.com/masterfabric/masterfabric_backend/internal/shared/errors"
)

var _ iamrepo.UserRepository = (*UserRepository)(nil)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) Create(ctx context.Context, email, passwordHash string) (*iammodel.User, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash) VALUES ($1, $2)
		RETURNING id, email, password_hash, created_at
	`, email, passwordHash)
	u, err := scanUser(row)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			return nil, domainerr.New(domainerr.CodeConflict, "email already registered", err)
		}
		return nil, domainerr.New(domainerr.CodeInternal, "create user", fmt.Errorf("insert user: %w", err))
	}
	return u, nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*iammodel.User, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, created_at FROM users WHERE email = $1
	`, email)
	u, err := scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerr.New(domainerr.CodeNotFound, "user not found", err)
		}
		return nil, domainerr.New(domainerr.CodeInternal, "find user by email", fmt.Errorf("query user: %w", err))
	}
	return u, nil
}

func (r *UserRepository) FindByID(ctx context.Context, id uuid.UUID) (*iammodel.User, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, created_at FROM users WHERE id = $1
	`, id)
	u, err := scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerr.New(domainerr.CodeNotFound, "user not found", err)
		}
		return nil, domainerr.New(domainerr.CodeInternal, "find user by id", fmt.Errorf("query user: %w", err))
	}
	return u, nil
}

func (r *UserRepository) UpdateEmail(ctx context.Context, id uuid.UUID, newEmail string) (*iammodel.User, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE users SET email = $2 WHERE id = $1
		RETURNING id, email, password_hash, created_at
	`, id, newEmail)
	u, err := scanUser(row)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, domainerr.New(domainerr.CodeConflict, "email already registered", err)
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerr.New(domainerr.CodeNotFound, "user not found", err)
		}
		return nil, domainerr.New(domainerr.CodeInternal, "update user email", fmt.Errorf("update users: %w", err))
	}
	return u, nil
}

func (r *UserRepository) UpdatePassword(ctx context.Context, id uuid.UUID, newPasswordHash string) error {
	cmd, err := r.pool.Exec(ctx, `
		UPDATE users SET password_hash = $2 WHERE id = $1
	`, id, newPasswordHash)
	if err != nil {
		return domainerr.New(domainerr.CodeInternal, "update user password", fmt.Errorf("update users: %w", err))
	}
	if cmd.RowsAffected() == 0 {
		return domainerr.New(domainerr.CodeNotFound, "user not found", nil)
	}
	return nil
}

type scannable interface {
	Scan(dest ...any) error
}

func scanUser(row scannable) (*iammodel.User, error) {
	var u iammodel.User
	if err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt); err != nil {
		return nil, err
	}
	return &u, nil
}
