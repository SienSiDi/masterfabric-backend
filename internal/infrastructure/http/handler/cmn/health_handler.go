package cmn

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/masterfabric/masterfabric_backend/internal/shared/response"
)

type Handler struct {
	pg  *pgxpool.Pool
	rdb *redis.Client
}

func New(pg *pgxpool.Pool, rdb *redis.Client) *Handler {
	return &Handler{pg: pg, rdb: rdb}
}

func (h *Handler) Live(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{"status": "alive"})
}

func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	services := map[string]string{"postgres": "healthy", "redis": "healthy"}

	if h.pg == nil {
		services["postgres"] = "unhealthy"
	} else if err := h.pg.Ping(ctx); err != nil {
		services["postgres"] = "unhealthy"
	}

	if h.rdb == nil {
		services["redis"] = "unhealthy"
	} else if err := h.rdb.Ping(ctx).Err(); err != nil {
		services["redis"] = "unhealthy"
	}

	status := "ready"
	for _, v := range services {
		if v != "healthy" {
			status = "unready"
			break
		}
	}
	response.JSON(w, http.StatusOK, map[string]any{"status": status, "services": services})
}
