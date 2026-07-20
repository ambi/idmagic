// /health: 起動構成をそのまま返す簡易ヘルスエンドポイント。
// /livez, /readyz, /startupz: Kubernetes 準拠のヘルスプローブエンドポイント。
package server_http

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v5"
)

type DependencyStatus struct {
	Status  string `json:"status"` // healthy / unavailable
	Message string `json:"message,omitempty"`
}

type ProbeResponse struct {
	Status       string                      `json:"status"`
	Dependencies map[string]DependencyStatus `json:"dependencies,omitempty"`
}

func (d Deps) handleHealth(c *echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status":        "ok",
		"persistence":   d.HealthInfo.Persistence,
		"event_sink":    d.HealthInfo.EventSink,
		"observability": d.HealthInfo.Observability,
		"authzen":       d.HealthInfo.AuthZEN,
	})
}

// handleMetrics serves GET /metrics (system.yaml MetricsExposition). The
// route is always registered, matching /health and the probe endpoints, but
// returns 503 when no MetricsHandler was wired at composition root so an
// unconfigured deployment fails the scrape loudly rather than 404ing.
func (d Deps) handleMetrics(c *echo.Context) error {
	if d.MetricsHandler == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
	}
	d.MetricsHandler.ServeHTTP(c.Response(), c.Request())
	return nil
}

func (d Deps) handleLivez(c *echo.Context) error {
	// Liveness: プロセス自体が動作していれば常に healthy。
	// デッドロック検知などを組み込む余地を残すが、一時的な依存障害では fail させない。
	return c.JSON(http.StatusOK, map[string]string{"status": "healthy"})
}

func (d Deps) handleStartupz(c *echo.Context) error {
	// Startup: 初期化が完了しているか確認。
	if d.StartupComplete == nil || !d.StartupComplete.Load() {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"status": "starting"})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "healthy"})
}

func (d Deps) handleReadyz(c *echo.Context) error {
	// Readiness: シャットダウン中なら即座に unready
	if d.ShuttingDown != nil && d.ShuttingDown.Load() {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
	}

	// 起動完了前も unready
	if d.StartupComplete == nil || !d.StartupComplete.Load() {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"status": "starting"})
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), 1*time.Second)
	defer cancel()

	var (
		pgErr error
		vkErr error
		wg    sync.WaitGroup
	)

	persistence := d.HealthInfo.Persistence
	if persistence == "postgres_valkey" {
		wg.Add(2)
		go func() {
			defer wg.Done()
			if d.DbPing != nil {
				pgErr = d.DbPing(ctx)
			}
		}()
		go func() {
			defer wg.Done()
			if d.ValkeyPing != nil {
				vkErr = d.ValkeyPing(ctx)
			}
		}()
		wg.Wait()
	}

	isHealthy := pgErr == nil && vkErr == nil
	status := "healthy"
	if !isHealthy {
		status = "unavailable"
	}

	verbose := c.QueryParam("verbose") != ""
	if verbose {
		details := make(map[string]DependencyStatus)
		if persistence == "postgres_valkey" {
			if pgErr != nil {
				details["postgres"] = DependencyStatus{Status: "unavailable", Message: pgErr.Error()}
			} else {
				details["postgres"] = DependencyStatus{Status: "healthy"}
			}

			if vkErr != nil {
				details["valkey"] = DependencyStatus{Status: "unavailable", Message: vkErr.Error()}
			} else {
				details["valkey"] = DependencyStatus{Status: "healthy"}
			}
		}

		resp := ProbeResponse{
			Status: status,
		}
		if len(details) > 0 {
			resp.Dependencies = details
		}

		if !isHealthy {
			return c.JSON(http.StatusServiceUnavailable, resp)
		}
		return c.JSON(http.StatusOK, resp)
	}

	if !isHealthy {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"status": status})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": status})
}
