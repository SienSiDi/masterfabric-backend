package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	domainerr "github.com/masterfabric/masterfabric_backend/internal/shared/errors"
	"github.com/masterfabric/masterfabric_backend/internal/shared/response"
)

func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic recovered", "panic", rec, "stack", string(debug.Stack()))
				response.Error(w, domainerr.New(domainerr.CodeInternal, "an internal error occurred", nil))
			}
		}()
		next.ServeHTTP(w, r)
	})
}
