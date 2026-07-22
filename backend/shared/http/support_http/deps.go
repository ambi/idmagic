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
	"github.com/ambi/idmagic/backend/shared/spec"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// Deps は全 HTTP ハンドラが共有する依存集約のうち、HTTP 横断設定とライフサイクルに関連するもの。
type Deps struct {
	Issuer                    string
	SCL                       *spec.SCL
	LegacyBareIssuer          bool
	TrustedForwardedHops      int
	OperationTimeout          time.Duration
	DetachedCompletionTimeout time.Duration
	AbortMetrics              HTTPAbortMetrics
	Metrics                   Metrics
	Emit                      func(spec.DomainEvent)
	HealthInfo                HealthInfo
	DbPing                    func(context.Context) error
	ValkeyPing                func(context.Context) error
	ShuttingDown              *atomic.Bool
	StartupComplete           *atomic.Bool
	TenantRepo                tenantports.TenantRepository
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
