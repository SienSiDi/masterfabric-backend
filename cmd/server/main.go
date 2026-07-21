package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	configusecase "github.com/masterfabric/masterfabric_backend/internal/application/config/usecase"
	iamusecase "github.com/masterfabric/masterfabric_backend/internal/application/iam/usecase"
	llmusecase "github.com/masterfabric/masterfabric_backend/internal/application/llm/usecase"
	"github.com/masterfabric/masterfabric_backend/internal/infrastructure/auth"
	"github.com/masterfabric/masterfabric_backend/internal/infrastructure/http/router"
	iampg "github.com/masterfabric/masterfabric_backend/internal/infrastructure/postgres/iam"
	configpg "github.com/masterfabric/masterfabric_backend/internal/infrastructure/postgres/config"
	llmpg "github.com/masterfabric/masterfabric_backend/internal/infrastructure/postgres/llm"
	postgrespool "github.com/masterfabric/masterfabric_backend/internal/infrastructure/postgres"
	redisclient "github.com/masterfabric/masterfabric_backend/internal/infrastructure/redis"
	"github.com/masterfabric/masterfabric_backend/internal/shared/config"
	"github.com/masterfabric/masterfabric_backend/internal/shared/events"
	"github.com/masterfabric/masterfabric_backend/internal/shared/logger"
	"github.com/masterfabric/masterfabric_backend/internal/shared/middleware"
	"github.com/masterfabric/masterfabric_backend/internal/shared/telemetry"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}
	log := logger.New(cfg.Log.Level, cfg.Log.Format)
	slog.SetDefault(log)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := postgrespool.NewPool(ctx, cfg.DB.DSN, cfg.DB.MaxConns, cfg.DB.MinConns)
	if err != nil {
		log.Error("failed to connect postgres", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	rdb, err := redisclient.NewClient(ctx, cfg.Redis.URL)
	if err != nil {
		log.Warn("redis not available — rate limiting + token blacklist disabled", "error", err)
		rdb = nil
	}
	if rdb != nil {
		defer rdb.Close()
	}

	blacklist := redisclient.NewTokenBlacklist(rdb)
	rateLimiter := middleware.NewRateLimiter(rdb)
	eventBus := events.NewInProcessBus()

	// Config
	configRepo := configpg.New(pool)
	getConfigUC := configusecase.NewGetConfigUseCase(configRepo)
	updateConfigUC := configusecase.NewUpdateConfigUseCase(configRepo)

	// IAM
	userRepo := iampg.NewUserRepository(pool)
	roleRepo := iampg.NewRoleRepository(pool)
	refreshRepo := iampg.NewRefreshTokenRepository(pool)
	permResolver := iampg.NewPermissionResolver(pool)
	if err := permResolver.Refresh(ctx); err != nil {
		log.Error("failed to load permission cache", "error", err)
		os.Exit(1)
	}
	log.Info("permission cache loaded", "roles", len(permResolver.CacheSnapshot()))

	jwtSvc := auth.NewJWTService(cfg.JWT.Secret, cfg.JWT.AccessTTL)
	registerUC := iamusecase.NewRegisterUseCase(userRepo, roleRepo, eventBus)
	loginUC := iamusecase.NewLoginUseCase(userRepo, roleRepo, refreshRepo, jwtSvc, eventBus, cfg.JWT.RefreshTTL)
	refreshUC := iamusecase.NewRefreshUseCase(refreshRepo, roleRepo, blacklist, jwtSvc, cfg.JWT.RefreshTTL)
	logoutUC := iamusecase.NewLogoutUseCase(refreshRepo, blacklist)
	listSessUC := iamusecase.NewListSessionsUseCase(refreshRepo)
	meUC := iamusecase.NewMeUseCase(userRepo, roleRepo)
	updateMeUC := iamusecase.NewUpdateMeUseCase(userRepo, roleRepo)
	changePwdUC := iamusecase.NewChangePasswordUseCase(userRepo, refreshRepo)

	// LLM
	sessionRepo := llmpg.NewSessionRepository(pool)
	eventRepo := llmpg.NewEventRepository(pool)
	scoreRepo := llmpg.NewScoreRepository(pool)
	monitoringRepo := llmpg.NewMonitoringRepository(pool)
	listModelsUC := llmusecase.NewListModelsUseCase()
	createSessUC := llmusecase.NewCreateSessionUseCase(sessionRepo, eventBus)
	getSessUC := llmusecase.NewGetSessionUseCase(sessionRepo)
	recordEventUC := llmusecase.NewRecordEventUseCase(eventRepo, sessionRepo, eventBus)
	listEventsUC := llmusecase.NewListEventsUseCase(eventRepo, sessionRepo)
	recordScoreUC := llmusecase.NewRecordScoreUseCase(scoreRepo, eventRepo, sessionRepo, eventBus)
	getMonitoringUC := llmusecase.NewGetMonitoringUseCase(monitoringRepo)

	telemetry.Init()

	handler := router.New(router.Deps{
		Cfg:            cfg,
		PG:             pool,
		Redis:          rdb,
		Jwt:            jwtSvc,
		PermResolver:   permResolver,
		RateLimiter:    rateLimiter,
		GetConfigUC:    getConfigUC,
		UpdateConfigUC: updateConfigUC,
		RegisterUC:     registerUC,
		LoginUC:        loginUC,
		RefreshUC:      refreshUC,
		LogoutUC:       logoutUC,
		ListSessUC:     listSessUC,
		MeUC:           meUC,
		UpdateMeUC:     updateMeUC,
		ChangePwdUC:    changePwdUC,
		ListModelsUC:   listModelsUC,
		CreateSessUC:   createSessUC,
		GetSessUC:      getSessUC,
		RecordEventUC:  recordEventUC,
		ListEventsUC:   listEventsUC,
		RecordScoreUC:  recordScoreUC,
		GetMonitoringUC: getMonitoringUC,
	})

	server := &http.Server{
		Addr:         cfg.Server.Host + ":" + cfg.Server.Port,
		Handler:      handler,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	go func() {
		log.Info("server starting", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Info("shutdown signal received")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error("graceful shutdown failed", "error", err)
	}
	log.Info("server stopped")
}
