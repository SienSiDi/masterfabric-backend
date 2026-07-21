package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/masterfabric/masterfabric_backend/internal/domain/config/model"
)

func main() {
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "set DATABASE_DSN")
		os.Exit(1)
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		fmt.Fprintln(os.Stderr, "connect:", err)
		os.Exit(1)
	}
	defer pool.Close()

	roles := []struct {
		name  string
		perms string
	}{
		{"admin", `["*"]`},
		{"user", `["own:read","own:write"]`},
	}
	for _, r := range roles {
		if _, err := pool.Exec(ctx, `
			INSERT INTO roles (name, permissions) VALUES ($1, $2::jsonb)
			ON CONFLICT (name) DO NOTHING
		`, r.name, r.perms); err != nil {
			fmt.Fprintln(os.Stderr, "seed role:", err)
			os.Exit(1)
		}
	}

	cfg := model.Default()
	raw, _ := json.Marshal(cfg)
	if _, err := pool.Exec(ctx, `
		INSERT INTO app_config (key, value) VALUES ('app', $1::jsonb)
		ON CONFLICT (key) DO NOTHING
	`, raw); err != nil {
		fmt.Fprintln(os.Stderr, "seed config:", err)
		os.Exit(1)
	}
	fmt.Println("seed done")
}
