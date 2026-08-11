// Package support: HTTP アダプタの共有基盤。
package support_http

import (
	"context"
	"sync/atomic"
	"time"

	apitokenports "github.com/ambi/idmagic/backend/apitoken/ports"
	appports "github.com/ambi/idmagic/backend/application/ports"
	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	sessionusecases "github.com/ambi/idmagic/backend/authentication/session/usecases"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
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

// Authenticator は認証・認可の共通ロジックに必要な依存を保持する。
type Authenticator struct {
	UserRepo              userports.UserRepository
	GroupRepo             groupports.GroupRepository
	SessionManager        *sessionusecases.SessionManager
	TokenIntrospector     oauthports.TokenIntrospector
	ApiTokenAuthenticator apitokenports.Authenticator
	DpopReplayStore       oauthports.DpopReplayStore
	AuthnResolver         authdomain.AuthenticationContextResolver
}

// ApplicationGate はフェデレーション開始時のアプリ割当ゲートに必要な依存を保持する。
type ApplicationGate struct {
	ApplicationRepo             appports.ApplicationRepository
	ApplicationAssignmentRepo   appports.AssignmentRepository
	GroupRepo                   groupports.GroupRepository
	ApplicationSignInPolicyRepo appports.SignInPolicyRepository
	DefaultSignInPolicyRepo     appports.DefaultSignInPolicyRepository
	GateTrustedForwardedHops    int
}

// HealthInfo は bootstrap が決定した実行時構成のラベル。
// /health がそのまま JSON で返すだけの読み取り専用情報を保持する。
type HealthInfo struct {
	Persistence   string
	EventSink     string
	Observability string
	AuthZEN       string
}
