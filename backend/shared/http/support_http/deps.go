// Package support: HTTP アダプタの共有基盤。
package support_http

import (
	"context"
	"sync/atomic"
	"time"

	rlports "github.com/ambi/idmagic/backend/shared/ratelimit/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// Deps は全 HTTP ハンドラが共有する依存集約のうち、HTTP 横断設定とライフサイクルに関連するもの。
type Deps struct {
	Issuer   string
	Contract *spec.RuntimeContract
	// TenantBaseDomain は subdomain style のテナントが載る親ドメイン。
	// 空なら host ベースのテナント解決そのものが無効になり、path prefix 経路だけが
	// 残る。ワイルドカード DNS / 証明書を用意できない配備を一級市民として保つための
	// 既定であり、この値が空の間はどのテナントも subdomain style を選べない。
	TenantBaseDomain     string
	TrustedForwardedHops int
	// RateLimiter is the shared endpoint rate limiter, distinct from the
	// per-account/per-IP login throttle. nil-safe: callers must check before use, matching
	// LoginAttemptThrottle's optional-by-construction convention in tests that don't wire it.
	RateLimiter               rlports.RateLimiter
	OperationTimeout          time.Duration
	DetachedCompletionTimeout time.Duration
	AbortMetrics              HTTPAbortMetrics
	Metrics                   Metrics
	Emit                      func(spec.DomainEvent)
	HealthInfo                HealthInfo
	DbPing                    func(context.Context) error
	ShuttingDown              *atomic.Bool
	StartupComplete           *atomic.Bool
	TenantRepo                tenantports.TenantRepository
	// PaginationCodec signs/verifies keyset pagination cursors.
	// nil-safe: handlers that don't paginate never touch it.
	PaginationCodec *CursorCodec
}

// HealthInfo は bootstrap が決定した実行時構成のラベル。
// /health がそのまま JSON で返すだけの読み取り専用情報を保持する。
type HealthInfo struct {
	Persistence   string
	EventSink     string
	Observability string
	AuthZEN       string
	Features      FeatureRuntimeMetadata
}

type RuntimeFeatureMetadata struct {
	ID           string `json:"id"`
	Version      string `json:"version"`
	Maturity     string `json:"maturity"`
	UpdatePolicy string `json:"update_policy"`
}

type FeatureRuntimeMetadata struct {
	SchemaVersion string                   `json:"schema_version"`
	Enabled       []RuntimeFeatureMetadata `json:"enabled"`
}
