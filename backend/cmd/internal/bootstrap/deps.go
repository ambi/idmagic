package bootstrap

import (
	"context"
	"errors"
	"strings"

	"github.com/ambi/idmagic/backend/application"
	"github.com/ambi/idmagic/backend/audit"
	"github.com/ambi/idmagic/backend/authentication"
	webauthnusecases "github.com/ambi/idmagic/backend/authentication/webauthn/usecases"
	"github.com/ambi/idmagic/backend/idgovernance"
	"github.com/ambi/idmagic/backend/idmanagement"
	"github.com/ambi/idmagic/backend/jobs"
	"github.com/ambi/idmagic/backend/oauth2"
	"github.com/ambi/idmagic/backend/provisioning"
	"github.com/ambi/idmagic/backend/saml"
	"github.com/ambi/idmagic/backend/scim"
	"github.com/ambi/idmagic/backend/signingkeys"
	"github.com/ambi/idmagic/backend/tenancy"
	"github.com/ambi/idmagic/backend/wsfederation"

	gowebauthn "github.com/go-webauthn/webauthn/webauthn"
)

// Dependencies は HTTP 層に渡す全境界をまとめた DI コンテナ。
// 永続層 (memory/postgres_valkey) や event sink の差分を本構造体で吸収する。
type Dependencies struct {
	Tenancy        tenancy.Module
	IdManagement   idmanagement.Module
	IdGovernance   idgovernance.Module
	Authentication authentication.Module
	OAuth2         oauth2.Module
	SigningKeys    signingkeys.Module
	Audit          audit.Module
	WsFederation   wsfederation.Module
	Saml           saml.Module
	Scim           scim.Module
	Application    application.Module
	Jobs           jobs.Module
	Provisioning   provisioning.Module
	Close          func()
	DbPing         func(context.Context) error
	ValkeyPing     func(context.Context) error
}

// RuntimeConfig は /health などで露出するための実行時構成ラベルを集約する。
type RuntimeConfig struct {
	Persistence   string
	EventSink     string
	Observability string
	AuthZEN       string
}

func LoadRuntimeConfig() RuntimeConfig {
	return RuntimeConfig{
		Persistence:   EnvDefault("PERSISTENCE", "memory"),
		EventSink:     EnvDefault("EVENT_SINK", "console"),
		Observability: EnvDefault("OBSERVABILITY", "noop"),
		AuthZEN:       EnvDefault("AUTHZEN", "local"),
	}
}

// assemble は PERSISTENCE 環境変数に応じて memory/postgres_valkey いずれかの構成を組み立てる。
func Assemble(ctx context.Context) (*Dependencies, error) {
	var deps *Dependencies
	var err error
	switch EnvDefault("PERSISTENCE", "memory") {
	case "memory":
		deps, err = assembleMemory()
	case "postgres_valkey":
		deps, err = assemblePostgresValkey(ctx)
	default:
		return nil, errors.New("PERSISTENCE must be memory or postgres_valkey")
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
