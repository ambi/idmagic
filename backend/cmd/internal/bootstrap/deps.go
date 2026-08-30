package bootstrap

import (
	"context"
	"strings"

	"github.com/ambi/idmagic/backend/apitoken"
	"github.com/ambi/idmagic/backend/application"
	"github.com/ambi/idmagic/backend/audit"
	"github.com/ambi/idmagic/backend/authentication"
	webauthnusecases "github.com/ambi/idmagic/backend/authentication/webauthn/usecases"
	"github.com/ambi/idmagic/backend/authorization"
	"github.com/ambi/idmagic/backend/datakeys"
	"github.com/ambi/idmagic/backend/idgovernance"
	"github.com/ambi/idmagic/backend/idmanagement"
	"github.com/ambi/idmagic/backend/jobs"
	"github.com/ambi/idmagic/backend/oauth2"
	"github.com/ambi/idmagic/backend/provisioning"
	"github.com/ambi/idmagic/backend/saml"
	notification "github.com/ambi/idmagic/backend/shared/notification/ports"
	ratelimit "github.com/ambi/idmagic/backend/shared/ratelimit/ports"
	"github.com/ambi/idmagic/backend/sharedsignals"
	"github.com/ambi/idmagic/backend/signingkeys"
	"github.com/ambi/idmagic/backend/sourcing"
	"github.com/ambi/idmagic/backend/tenancy"
	"github.com/ambi/idmagic/backend/workloadidentity"
	"github.com/ambi/idmagic/backend/wsfederation"

	gowebauthn "github.com/go-webauthn/webauthn/webauthn"
)

// Dependencies は HTTP 層に渡す全境界をまとめた DI コンテナ。
// 永続層 (memory/postgres) や event sink の差分を本構造体で吸収する。
type Dependencies struct {
	Tenancy          tenancy.Module
	IdManagement     idmanagement.Module
	IdGovernance     idgovernance.Module
	Authentication   authentication.Module
	OAuth2           oauth2.Module
	SigningKeys      signingkeys.Module
	DataKeys         datakeys.Module
	Audit            audit.Module
	WsFederation     wsfederation.Module
	Saml             saml.Module
	Sourcing         sourcing.Module
	Application      application.Module
	ApiTokens        apitoken.Module
	Jobs             jobs.Module
	Provisioning     provisioning.Module
	WorkloadIdentity workloadidentity.Module
	SharedSignals    sharedsignals.Module
	Authorization    authorization.Module
	Notification     notification.Module
	RateLimit        ratelimit.Module
	// SecurityNotifications はアカウントのセキュリティ通知の起動時配線 (wi-90)。
	// AssembleNotification が locale と実行方法を埋め、issuer を知るプロセスが
	// IssuerResolver を差し込む。
	SecurityNotifications SecurityNotificationConfig
	Close                 func()
	DbPing                func(context.Context) error
}

// RuntimeConfig は /health などで露出するための実行時構成ラベルを集約する。
type RuntimeConfig struct {
	Persistence   string
	Observability string
	AuthZEN       string
	Features      FeatureRuntimeMetadata
}

func LoadRuntimeConfig(cfg SharedConfig) RuntimeConfig {
	return RuntimeConfig{
		Persistence:   cfg.Persistence,
		Observability: cfg.Observability,
		AuthZEN:       cfg.AuthZEN,
		Features:      cfg.Features.Metadata,
	}
}

// Assemble は cfg.Persistence に応じて memory/postgres いずれかの構成を組み立てる。
// 揮発性状態も PostgreSQL に統合済みで、選択肢は memory / postgres の 2 つ。cfg は
// 呼び出し側が LoadSharedConfig + ConfigLoader.Err() で既に検証済みであることを前提とし、
// ここでは I/O を伴う組み立てのみ行う (wi-103: 検証と組み立てのフェーズを分離し、
// 検証はネットワーク接続や listener 起動より前に集約して終える)。
func Assemble(ctx context.Context, cfg SharedConfig) (*Dependencies, error) {
	var deps *Dependencies
	var err error
	switch cfg.Persistence {
	case "postgres":
		deps, err = assemblePostgres(ctx, cfg)
	default:
		deps, err = assembleMemory(cfg)
	}
	if err != nil {
		return nil, err
	}
	// WebAuthn RP は永続層に依らず cfg から構築する (wi-26)。
	rp, err := loadWebAuthnRP(cfg)
	if err != nil {
		return nil, err
	}
	deps.Authentication.WebAuthnRP = rp
	// 通知はカタログ解決を含めてここで組み立て、API プロセスと worker プロセスが
	// 同じ経路 (テナント上書き・locale 解決を含む) で送るようにする (wi-288)。
	//nolint:contextcheck // 起動時の配線のみ。実際の I/O は送信ごとに呼び出し元の context で走る。
	if err := AssembleNotification(deps, cfg); err != nil {
		return nil, err
	}
	return deps, nil
}

// loadWebAuthnRP は cfg.WebAuthnRP* から RP を構築する。RP_ID 未設定なら WebAuthn は
// 無効 (nil) として扱う。RP_ID 設定時に origin が無い組み合わせは LoadSharedConfig が
// 既に fail-fast させているため、ここでは gowebauthn.New 自体が返す構築時エラーだけを
// 扱えばよい。
func loadWebAuthnRP(cfg SharedConfig) (*gowebauthn.WebAuthn, error) {
	if cfg.WebAuthnRPID == "" {
		return nil, nil //nolint:nilnil // RP_ID 未設定は WebAuthn 無効を表す正当な状態 (エラーではない)。
	}
	return webauthnusecases.NewWebAuthn(webauthnusecases.WebAuthnConfig{
		RPID:          cfg.WebAuthnRPID,
		RPDisplayName: cfg.WebAuthnRPDisplayName,
		RPOrigins:     cfg.WebAuthnRPOrigins,
	})
}

// splitAndTrim はカンマ区切り文字列を空要素を除いてトリムして分割する。
func splitAndTrim(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
