package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/masterfabric/masterfabric_backend/internal/infrastructure/auth"
	domainerr "github.com/masterfabric/masterfabric_backend/internal/shared/errors"
	"github.com/masterfabric/masterfabric_backend/internal/shared/response"
)

// ErrNoUserID is returned by handlers when an auth-required route ran without a user_id in ctx.
var ErrNoUserID = domainerr.New(domainerr.CodeUnauthorized, "no user in context", nil)

type ctxKeyUser string

const (
	UserIDKey ctxKeyUser = "user_id"
	RolesKey  ctxKeyUser = "roles"
)

// Auth parses the JWT from the Authorization: Bearer header, verifies it,
// and injects the user_id (uuid.UUID) and roles ([]string) into the request context.
// On failure it returns 401 with a generic message (no token detail leaks).
func Auth(jwt *auth.JWTService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				response.Error(w, domainerr.New(domainerr.CodeUnauthorized, "missing authorization header", nil))
				return
			}
			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") || parts[1] == "" {
				response.Error(w, domainerr.New(domainerr.CodeUnauthorized, "invalid authorization header", nil))
				return
			}
			claims, err := jwt.ParseAccess(parts[1])
			if err != nil {
				response.Error(w, domainerr.New(domainerr.CodeUnauthorized, "invalid or expired token", nil))
				return
			}
			ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, RolesKey, claims.Roles)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserIDFromContext returns the authenticated user's UUID, or false if not present.
func UserIDFromContext(ctx context.Context) (interface{}, bool) {
	v := ctx.Value(UserIDKey)
	if v == nil {
		return nil, false
	}
	return v, true
}

// RolesFromContext returns the authenticated user's roles, or false if not present.
func RolesFromContext(ctx context.Context) ([]string, bool) {
	v := ctx.Value(RolesKey)
	if v == nil {
		return nil, false
	}
	roles, ok := v.([]string)
	return roles, ok
}
