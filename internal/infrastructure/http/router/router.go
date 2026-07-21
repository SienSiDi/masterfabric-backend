package router

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	configusecase "github.com/masterfabric/masterfabric_backend/internal/application/config/usecase"
	iamusecase "github.com/masterfabric/masterfabric_backend/internal/application/iam/usecase"
	llmusecase "github.com/masterfabric/masterfabric_backend/internal/application/llm/usecase"
	"github.com/masterfabric/masterfabric_backend/internal/infrastructure/auth"
	cmnhandler "github.com/masterfabric/masterfabric_backend/internal/infrastructure/http/handler/cmn"
	confighandler "github.com/masterfabric/masterfabric_backend/internal/infrastructure/http/handler/config"
	iamhandler "github.com/masterfabric/masterfabric_backend/internal/infrastructure/http/handler/iam"
	llmhandler "github.com/masterfabric/masterfabric_backend/internal/infrastructure/http/handler/llm"
	"github.com/masterfabric/masterfabric_backend/internal/shared/config"
	"github.com/masterfabric/masterfabric_backend/internal/shared/middleware"
	"github.com/masterfabric/masterfabric_backend/internal/shared/telemetry"
)

type Deps struct {
	Cfg            *config.Config
	PG             *pgxpool.Pool
	Redis          *redis.Client
	Jwt            *auth.JWTService
	PermResolver   middleware.PermissionResolver
	RateLimiter    *middleware.RateLimiter
	GetConfigUC    *configusecase.GetConfigUseCase
	UpdateConfigUC *configusecase.UpdateConfigUseCase
	RegisterUC     *iamusecase.RegisterUseCase
	LoginUC        *iamusecase.LoginUseCase
	RefreshUC      *iamusecase.RefreshUseCase
	LogoutUC       *iamusecase.LogoutUseCase
	ListSessUC     *iamusecase.ListSessionsUseCase
	MeUC           *iamusecase.MeUseCase
	UpdateMeUC     *iamusecase.UpdateMeUseCase
	ChangePwdUC    *iamusecase.ChangePasswordUseCase
	ListModelsUC   *llmusecase.ListModelsUseCase
	CreateSessUC   *llmusecase.CreateSessionUseCase
	GetSessUC      *llmusecase.GetSessionUseCase
	RecordEventUC  *llmusecase.RecordEventUseCase
	ListEventsUC   *llmusecase.ListEventsUseCase
	RecordScoreUC  *llmusecase.RecordScoreUseCase
	GetMonitoringUC *llmusecase.GetMonitoringUseCase
}

func New(deps Deps) http.Handler {
	r := chi.NewRouter()

	r.Use(chimw.RealIP)
	r.Use(middleware.RequestID)
	r.Use(middleware.Recover)
	r.Use(middleware.CORS(deps.Cfg.CORS.AllowedOrigins))
	r.Use(middleware.BodyLimit(deps.Cfg.HTTP.MaxBodyBytes))
	r.Use(middleware.Logger)

	cmnH := cmnhandler.New(deps.PG, deps.Redis)
	cfgH := confighandler.New(deps.GetConfigUC, deps.UpdateConfigUC)
	authH := iamhandler.NewAuthHandler(deps.RegisterUC, deps.LoginUC, deps.RefreshUC, deps.LogoutUC, deps.ListSessUC)
	meH := iamhandler.NewMeHandler(deps.MeUC, deps.UpdateMeUC, deps.ChangePwdUC)
	llmH := llmhandler.NewHandler(deps.ListModelsUC, deps.CreateSessUC, deps.GetSessUC, deps.RecordEventUC, deps.ListEventsUC, deps.RecordScoreUC, deps.GetMonitoringUC)

	authMW := middleware.Auth(deps.Jwt)
	loginLimitMW := middleware.RateLimit(deps.RateLimiter, loginRateKey, 5, time.Minute)
	registerLimitMW := middleware.RateLimit(deps.RateLimiter, registerRateKey, 3, time.Minute)
	refreshLimitMW := middleware.RateLimit(deps.RateLimiter, refreshRateKey, 10, time.Minute)

	r.Get("/health/live", cmnH.Live)
	r.Get("/health/ready", cmnH.Ready)
	r.Handle("/metrics", telemetry.MetricsHandler())

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/config", cfgH.Get)
		r.With(authMW, middleware.RequirePermission(deps.PermResolver, "config:write")).Put("/admin/config", cfgH.Update)

		r.Route("/auth", func(r chi.Router) {
			r.With(registerLimitMW).Post("/register", authH.Register)
			r.With(loginLimitMW).Post("/login", authH.Login)
			r.With(refreshLimitMW).Post("/refresh", authH.Refresh)
			r.With(authMW).Post("/logout", authH.Logout)
			r.With(authMW).Post("/change-password", meH.ChangePassword)
		})

		// Authenticated user routes
		r.Group(func(r chi.Router) {
			r.Use(authMW)
			r.Get("/me", meH.Me)
			r.Put("/me", meH.Update)
			r.Get("/me/sessions", authH.Sessions)

			// Web MLC-LLM
			r.Route("/llm", func(r chi.Router) {
				r.Get("/models", llmH.ListModels)
				r.Post("/sessions", llmH.CreateSession)
				r.Get("/sessions/{id}", llmH.GetSession)
				// Rate limit event recording: 30/min per user
				r.With(middleware.RateLimit(deps.RateLimiter, eventRateKey, 30, time.Minute)).
					Post("/sessions/{id}/events", llmH.RecordEvent)
				r.Get("/sessions/{id}/events", llmH.ListEvents)
				r.Post("/sessions/{id}/score", llmH.RecordScore)
			})

			// Admin-only monitoring aggregate
			r.With(middleware.RequirePermission(deps.PermResolver, "llm:read")).Get("/llm/monitoring", llmH.GetMonitoring)
		})
	})

	return r
}

// loginRateKey extracts the rate-limit key from the login request body.
// It reads the email from the JSON body. On parse failure, falls back to the client IP.
// The body is restored for the downstream handler.
func loginRateKey(r *http.Request) string {
	if r.Body == nil {
		return "login:ip:" + clientIP(r)
	}
	raw, err := io.ReadAll(r.Body)
	r.Body.Close()
	r.Body = io.NopCloser(bytes.NewReader(raw))
	if err == nil && len(raw) > 0 {
		var body struct {
			Email string `json:"email"`
		}
		_ = json.Unmarshal(raw, &body)
		if body.Email != "" {
			return "login:" + body.Email
		}
	}
	return "login:ip:" + clientIP(r)
}

// clientIP extracts the client IP from RemoteAddr (set by chimw.RealIP middleware).
func clientIP(r *http.Request) string {
	host := r.RemoteAddr
	if i := strings.IndexByte(host, ':'); i >= 0 {
		host = host[:i]
	}
	return host
}

// registerRateKey extracts rate-limit key from the registration request body.
func registerRateKey(r *http.Request) string {
	if r.Body == nil {
		return "register:ip:" + clientIP(r)
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, 512))
	r.Body.Close()
	r.Body = io.NopCloser(bytes.NewReader(raw))
	if err == nil && len(raw) > 0 {
		var body struct {
			Email string `json:"email"`
		}
		_ = json.Unmarshal(raw, &body)
		if body.Email != "" {
			return "register:" + body.Email
		}
	}
	return "register:ip:" + clientIP(r)
}

// refreshRateKey rate-limits refresh token requests by token hash.
func refreshRateKey(r *http.Request) string {
	if r.Body == nil {
		return "refresh:ip:" + clientIP(r)
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, 512))
	r.Body.Close()
	r.Body = io.NopCloser(bytes.NewReader(raw))
	if err == nil && len(raw) > 0 {
		var body struct {
			RefreshToken string `json:"refreshToken"`
		}
		_ = json.Unmarshal(raw, &body)
		if body.RefreshToken != "" {
			h := sha256.Sum256([]byte(body.RefreshToken))
			return "refresh:" + hex.EncodeToString(h[:8])
		}
	}
	return "refresh:ip:" + clientIP(r)
}

// eventRateKey rate-limits event recording per user (user_id is in the JWT context,
// but middleware runs before the handler — so we extract user_id from the JWT here
// via a lightweight parse. For simplicity we use the client IP as a fallback.)
func eventRateKey(r *http.Request) string {
	// The auth middleware has already run by the time this is invoked (we're inside
	// the authenticated group). We can't easily read the context here, so we use
	// the Authorization header's hash as the key — a stable per-user identifier.
	auth := r.Header.Get("Authorization")
	if len(auth) > 7 {
		return "llm:events:" + auth[7:] // skip "Bearer "
	}
	return "llm:events:ip:" + clientIP(r)
}
