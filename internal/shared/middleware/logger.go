package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/masterfabric/masterfabric_backend/internal/shared/telemetry"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		dur := time.Since(start)
		slog.Info("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration_ms", dur.Milliseconds(),
			"request_id", r.Context().Value(RequestIDKey),
		)
		telemetry.Observe(r.Method, r.URL.Path, rec.status, dur)
	})
}
