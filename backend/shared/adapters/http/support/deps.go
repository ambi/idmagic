// Package support: HTTP アダプタの共有基盤。
package support

import (
	"context"
	"sync/atomic"
	"time"

	appports "github.com/ambi/idmagic/backend/application/ports"
	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
	idmports "github.com/ambi/idmagic/backend/identitymanagement/ports"
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
	UserRepo          idmports.UserRepository
	GroupRepo         idmports.GroupRepository
	SessionManager    *authusecases.SessionManager
	TokenIntrospector oauthports.TokenIntrospector
	AuthnResolver     authdomain.AuthenticationContextResolver
}

// ApplicationGate はフェデレーション開始時のアプリ割当ゲートに必要な依存を保持する。
type ApplicationGate struct {
	ApplicationRepo             appports.ApplicationRepository
	ApplicationAssignmentRepo   appports.AssignmentRepository
	GroupRepo                   idmports.GroupRepository
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
