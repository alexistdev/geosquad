package handler

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/alexistdev/geosquad/api/internal/cache"
	"github.com/alexistdev/geosquad/api/internal/response"
)

type HealthHandler struct {
	db      *sql.DB
	redis   *cache.Client
	version string
}

func NewHealthHandler(db *sql.DB, redis *cache.Client, version string) *HealthHandler {
	return &HealthHandler{db: db, redis: redis, version: version}
}

// Health memeriksa dependensi, bukan sekadar membalas "ok". Endpoint yang
// selalu hijau tidak berguna buat load balancer.
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	checks := map[string]string{"mysql": "ok", "redis": "ok"}
	healthy := true

	if err := h.db.PingContext(ctx); err != nil {
		checks["mysql"] = "gagal: " + err.Error()
		healthy = false
	}
	if err := h.redis.Ping(ctx).Err(); err != nil {
		checks["redis"] = "gagal: " + err.Error()
		healthy = false
	}

	payload := map[string]any{"version": h.version, "checks": checks}
	if !healthy {
		// Status 503 supaya load balancer menarik instance ini dari rotasi.
		// Detail per dependensi tetap ikut, karena endpoint ini dibaca operator.
		response.Unhealthy(w, "Sebagian dependensi tidak sehat.", payload)
		return
	}
	response.OK(w, "Service sehat.", payload)
}
