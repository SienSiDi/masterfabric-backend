package middleware

import (
	"context"
	"net/http"
	"strings"

	domainerr "github.com/masterfabric/masterfabric_backend/internal/shared/errors"
	"github.com/masterfabric/masterfabric_backend/internal/shared/response"
)

// PermissionResolver maps role names (from the JWT) to permission strings.
// Implemented by internal/infrastructure/postgres/iam.PermissionResolver.
type PermissionResolver interface {
	PermissionsForRoles(ctx context.Context, roleNames []string) ([]string, error)
}

// RequirePermission returns middleware that checks whether the authenticated user
// has the wanted permission. Permission matching is wildcard-aware:
//   - "*" matches everything
//   - "org:*" matches "org:read", "org:write", etc.
//   - "*:read" matches "user:read", "org:read", etc.
//   - exact match otherwise
func RequirePermission(resolver PermissionResolver, want string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			roles, ok := RolesFromContext(r.Context())
			if !ok {
				response.Error(w, ErrNoUserID)
				return
			}
			perms, err := resolver.PermissionsForRoles(r.Context(), roles)
			if err != nil {
				response.Error(w, domainerr.New(domainerr.CodeForbidden, "permission check failed", err))
				return
			}
			if len(perms) == 0 {
				response.Error(w, domainerr.New(domainerr.CodeForbidden, "insufficient permissions", nil))
				return
			}
			if !matchPermission(perms, want) {
				response.Error(w, domainerr.New(domainerr.CodeForbidden, "insufficient permissions", nil))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// matchPermission checks whether any of the granted permissions matches the wanted one.
func matchPermission(granted []string, want string) bool {
	for _, g := range granted {
		if g == "*" || g == want {
			return true
		}
		if isPrefixWildcard(g, want) {
			return true
		}
	}
	return false
}

// isPrefixWildcard matches "org:*" against "org:read", or "*:read" against "user:read".
func isPrefixWildcard(pattern, want string) bool {
	n := len(pattern)
	if n < 2 || pattern[n-1] != '*' {
		return false
	}
	prefix := pattern[:n-1]
	if len(prefix) == 0 || prefix == ":" {
		return false
	}
	return len(want) >= len(prefix) && want[:len(prefix)] == prefix
}

// keep strings import referenced (used in future header parsing)
var _ = strings.TrimSpace
