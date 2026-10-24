package authentication

import (
	"github.com/ambi/idmagic/backend/authentication/internal/negotiate/config_file"
	"github.com/ambi/idmagic/backend/authentication/internal/negotiate/protocol_kerberos"
	"github.com/ambi/idmagic/backend/authentication/internal/negotiate/usecases"
	signinhttp "github.com/ambi/idmagic/backend/authentication/internal/signin/handlers_http"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/labstack/echo/v5"
)

// ConfigureNegotiate は起動時だけに呼び、管理者が配備した信頼設定を読み込む。
// 返す fingerprint は配備の監査用であり、鍵と principal の内容をログへ出さない。
func (m *Module) ConfigureNegotiate(path string, users userports.UserRepository) (fingerprint string, err error) {
	if path == "" {
		return "", nil
	}
	config, secret, fingerprint, err := config_file.Load(path)
	if err != nil {
		return "", err
	}
	service, err := usecases.New(config.Tenants, protocol_kerberos.Verifier{}, m.FederationReplayStore, users)
	if err != nil {
		return "", err
	}
	m.negotiateService, m.negotiateAttemptSecret, m.automaticNegotiate = service, secret, config.Automatic
	return fingerprint, nil
}

// NegotiateSignIn は、検証済みのパッシブ要求から認証を試す公開の操作である。
// nil は配備で無効なことを表す。WS-Fed に鍵と principal の管理を渡さない。
func (m Module) NegotiateSignIn(d RouteDeps) func(*echo.Context, string, bool) (bool, error) {
	if m.negotiateService == nil {
		return nil
	}
	deps := signinhttp.Deps{Deps: m.httpDeps(d)}
	return deps.NegotiatePassive
}
