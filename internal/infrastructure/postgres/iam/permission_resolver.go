package iam

import (
	"context"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PermissionResolver maps role names to permission strings, cached in memory.
// It is safe for concurrent use. Call Refresh() at startup and periodically.
type PermissionResolver struct {
	pool   *pgxpool.Pool
	mu     sync.RWMutex
	cache  map[string][]string // role name -> permission strings
}

func NewPermissionResolver(pool *pgxpool.Pool) *PermissionResolver {
	return &PermissionResolver{pool: pool, cache: make(map[string][]string)}
}

// Refresh loads all roles + permissions from Postgres into the in-memory cache.
func (r *PermissionResolver) Refresh(ctx context.Context) error {
	rows, err := r.pool.Query(ctx, `SELECT name, permissions FROM roles`)
	if err != nil {
		return err
	}
	defer rows.Close()
	tmp := make(map[string][]string)
	for rows.Next() {
		var name string
		var permsRaw []byte
		if err := rows.Scan(&name, &permsRaw); err != nil {
			return err
		}
		perms, err := parsePermissionsJSON(permsRaw)
		if err != nil {
			return err
		}
		tmp[name] = perms
	}
	if err := rows.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	r.cache = tmp
	r.mu.Unlock()
	return nil
}

// PermissionsForRoles returns the deduplicated set of permission strings for the
// given role names. Unknown roles are silently skipped.
func (r *PermissionResolver) PermissionsForRoles(_ context.Context, roleNames []string) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	seen := make(map[string]bool)
	out := make([]string, 0)
	for _, rn := range roleNames {
		for _, p := range r.cache[rn] {
			if !seen[p] {
				seen[p] = true
				out = append(out, p)
			}
		}
	}
	return out, nil
}

func parsePermissionsJSON(raw []byte) ([]string, error) {
	var perms []string
	if len(raw) == 0 {
		return perms, nil
	}
	if err := jsonUnmarshal(raw, &perms); err != nil {
		return nil, err
	}
	return perms, nil
}

// CacheSnapshot returns a copy of the current role->permissions cache (for logging / introspection).
func (r *PermissionResolver) CacheSnapshot() map[string][]string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string][]string, len(r.cache))
	for k, v := range r.cache {
		out[k] = append([]string(nil), v...)
	}
	return out
}
