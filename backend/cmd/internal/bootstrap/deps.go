package bootstrap

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/ambi/idmagic/backend/apitoken"
	"github.com/ambi/idmagic/backend/application"
	"github.com/ambi/idmagic/backend/audit"
	"github.com/ambi/idmagic/backend/authentication"
	webauthnusecases "github.com/ambi/idmagic/backend/authentication/webauthn/usecases"
	"github.com/ambi/idmagic/backend/datakeys"
	"github.com/ambi/idmagic/backend/idgovernance"
	"github.com/ambi/idmagic/backend/idmanagement"
	"github.com/ambi/idmagic/backend/jobs"
	"github.com/ambi/idmagic/backend/oauth2"
	"github.com/ambi/idmagic/backend/provisioning"
	"github.com/ambi/idmagic/backend/saml"
	notification "github.com/ambi/idmagic/backend/shared/notification/ports"
	"github.com/ambi/idmagic/backend/signingkeys"
	"github.com/ambi/idmagic/backend/sourcing"
	"github.com/ambi/idmagic/backend/tenancy"
	"github.com/ambi/idmagic/backend/wsfederation"

	gowebauthn "github.com/go-webauthn/webauthn/webauthn"
)

// Dependencies は HTTP 層に渡す全境界をまとめた DI コンテナ。
// 永続層 (memory/postgres) や event sink の差分を本構造体で吸収する。
type Dependencies struct {
	Tenancy        tenancy.Module
	IdManagement   idmanagement.Module
	IdGovernance   idgovernance.Module
	Authentication authentication.Module
	OAuth2         oauth2.Module
	SigningKeys    signingkeys.Module
	DataKeys       datakeys.Module
	Audit          audit.Module
	WsFederation   wsfederation.Module
	Saml           saml.Module
	Sourcing       sourcing.Module
	Application    application.Module
	ApiTokens      apitoken.Module
	Jobs           jobs.Module
	Provisioning   provisioning.Module
	Notification   notification.Module
	Close          func()
	DbPing         func(context.Context) error
}

// RuntimeConfig は /health などで露出するための実行時構成ラベルを集約する。
type RuntimeConfig struct {
	Persistence   string
	Observability string
	AuthZEN       string
}

func LoadRuntimeConfig() RuntimeConfig {
	return RuntimeConfig{
		Persistence:   EnvDefault("PERSISTENCE", "memory"),
		Observability: EnvDefault("OBSERVABILITY", "noop"),
		AuthZEN:       EnvDefault("AUTHZEN", "local"),
	}
}

// assemble は PERSISTENCE 環境変数に応じて memory/postgres いずれかの構成を組み立てる。
// 揮発性状態も PostgreSQL に統合済みで、選択肢は memory / postgres の 2 つ (ADR-139)。
func Assemble(ctx context.Context) (*Dependencies, error) {
	var deps *Dependencies
	var err error
	switch EnvDefault("PERSISTENCE", "memory") {
	case "memory":
		deps, err = assembleMemory()
	case "postgres":
		deps, err = assemblePostgres(ctx)
	default:
		return nil, errors.New("PERSISTENCE must be memory or postgres")
	}
	if err != nil {
		return nil, err
	}
	// WebAuthn RP は永続層に依らず env config から構築する (wi-26 / ADR-087)。
	rp, err := loadWebAuthnRP()
	if err != nil {
		return nil, err
	}
	deps.Authentication.WebAuthnRP = rp
	// 通知はカタログ解決を含めてここで組み立て、API プロセスと worker プロセスが
	// 同じ経路 (テナント上書き・locale 解決を含む) で送るようにする (wi-288, ADR-142)。
	//nolint:contextcheck // 起動時の配線のみ。実際の I/O は送信ごとに呼び出し元の context で走る。
	if err := AssembleNotification(deps, os.Getenv); err != nil {
		return nil, err
	}
	return deps, nil
}

// loadWebAuthnRP は WEBAUTHN_RP_ID / WEBAUTHN_RP_ORIGINS / WEBAUTHN_RP_DISPLAY_NAME から RP を
// 構築する。RP_ID 未設定なら WebAuthn は無効 (nil) とし、RP_ID 設定時に origin が無ければ
// 起動を失敗させる (誤設定の silent 無効化を防ぐ起動時検証)。
func loadWebAuthnRP() (*gowebauthn.WebAuthn, error) {
	rpID := strings.TrimSpace(EnvDefault("WEBAUTHN_RP_ID", ""))
	if rpID == "" {
		return nil, nil //nolint:nilnil // RP_ID 未設定は WebAuthn 無効を表す正当な状態 (エラーではない)。
	}
	origins := splitAndTrim(EnvDefault("WEBAUTHN_RP_ORIGINS", ""))
	if len(origins) == 0 {
		return nil, errors.New("WEBAUTHN_RP_ORIGINS must be set when WEBAUTHN_RP_ID is set")
	}
	return webauthnusecases.NewWebAuthn(webauthnusecases.WebAuthnConfig{
		RPID:          rpID,
		RPDisplayName: EnvDefault("WEBAUTHN_RP_DISPLAY_NAME", "idmagic"),
		RPOrigins:     origins,
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
