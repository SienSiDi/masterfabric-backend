package config

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/masterfabric/masterfabric_backend/internal/domain/config/model"
	domainrepo "github.com/masterfabric/masterfabric_backend/internal/domain/config/repository"
	domainerr "github.com/masterfabric/masterfabric_backend/internal/shared/errors"
)

var _ domainrepo.ConfigRepository = (*Repository)(nil)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Get(ctx context.Context) (*model.AppConfig, error) {
	var raw []byte
	err := r.pool.QueryRow(ctx, `SELECT value FROM app_config WHERE key = 'app'`).Scan(&raw)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domainerr.New(domainerr.CodeNotFound, "app config not found", err)
		}
		return nil, domainerr.New(domainerr.CodeInternal, "get app config", fmt.Errorf("query app_config: %w", err))
	}
	var cfg model.AppConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, domainerr.New(domainerr.CodeInternal, "unmarshal app config", err)
	}
	return &cfg, nil
}

func (r *Repository) Update(ctx context.Context, cfg *model.AppConfig) error {
	raw, err := json.Marshal(cfg)
	if err != nil {
		return domainerr.New(domainerr.CodeInternal, "marshal app config", err)
	}
	if _, err := r.pool.Exec(ctx, `
		INSERT INTO app_config (key, value) VALUES ('app', $1)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now()
	`, raw); err != nil {
		return domainerr.New(domainerr.CodeInternal, "upsert app config", fmt.Errorf("exec upsert app_config: %w", err))
	}
	return nil
}
