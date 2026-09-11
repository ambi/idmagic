package domain

import (
	"testing"
	"time"

	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"

	"github.com/ambi/idmagic/backend/shared/spec"
)

func TestClientValidateRequiresGrantTypes(t *testing.T) {
	c := OAuth2Client{
		ClientID:                 "demo",
		ClientType:               spec.ClientConfidential,
		RedirectURIs:             []string{"https://app.example.com/cb"},
		GrantTypes:               nil,
		TokenEndpointAuthMethod:  AuthMethodClientSecretBasic,
		IDTokenSignedResponseAlg: signingdomain.SigAlgPS256,
		FapiProfile:              FapiNone,
		CreatedAt:                time.Now().UTC(),
	}
	if err := c.Validate(); err == nil {
		t.Fatal("expected error for empty grant_types")
	}
}

// fapi2TestClient は プロファイル以外がすべてそろった 1 件を返す。対照を作るときに
// 変えるのは引数の 2 つだけなので、落ちた理由がその差から来ていることが読める。
func fapi2TestClient(profile FapiProfile, method TokenEndpointAuthMethod) OAuth2Client {
	jwks := map[string]any{"keys": []any{map[string]any{"kty": "RSA", "kid": "k1"}}}
	subjectDN := "CN=client,O=example"
	return OAuth2Client{
		ClientID:                 "demo",
		ClientType:               spec.ClientConfidential,
		RedirectURIs:             []string{"https://app.example.com/cb"},
		GrantTypes:               []spec.GrantType{spec.GrantAuthorizationCode},
		TokenEndpointAuthMethod:  method,
		JWKS:                     jwks,
		TlsClientAuthSubjectDN:   &subjectDN,
		IDTokenSignedResponseAlg: signingdomain.SigAlgPS256,
		FapiProfile:              profile,
		CreatedAt:                time.Now().UTC(),
		UpdatedAt:                time.Now().UTC(),
	}
}

// FAPI2-CLIENT-AUTH: プロファイルを選択したクライアントのクライアント認証方式が
// `private_key_jwt` と `tls_client_auth` に限られることを固定する。
func TestOAuth2ClientRejectsSharedSecretAuthUnderFapi2(t *testing.T) {
	for _, method := range []TokenEndpointAuthMethod{
		AuthMethodClientSecretBasic, AuthMethodClientSecretPost, AuthMethodNone,
	} {
		t.Run(string(method), func(t *testing.T) {
			if err := fapi2TestClient(FapiSecurityProfileV2, method).Validate(); err == nil {
				t.Fatalf("%s の FAPI クライアントが検証を通った", method)
			}
		})
	}
	// 対照: 非対称な 2 方式は通る。ここが落ちるなら、方式の判定ではなくプロファイル
	// そのものを拒否している。
	for _, method := range []TokenEndpointAuthMethod{
		AuthMethodPrivateKeyJwt, AuthMethodTlsClientAuth,
	} {
		t.Run(string(method), func(t *testing.T) {
			if err := fapi2TestClient(FapiSecurityProfileV2, method).Validate(); err != nil {
				t.Fatalf("%s の FAPI クライアントが拒否された: %v", method, err)
			}
		})
	}
}

// FAPI2-SENDER-CONSTRAINT: プロファイルを選択したクライアントが、DPoP 証明も mTLS
// 証明書のサムプリントも持たない要求でトークンを得られないことを固定する。
func TestOAuth2ClientRequiresSenderConstraintEvidenceUnderFapi2(t *testing.T) {
	client := fapi2TestClient(FapiSecurityProfileV2, AuthMethodPrivateKeyJwt)
	if client.SenderConstraintSatisfied("", "") {
		t.Fatal("証拠の無い要求が送信者制約を満たすと判定された")
	}
	// 2 種類の証拠それぞれが単独で足りる。片方だけを見る実装はここで落ちる。
	if !client.SenderConstraintSatisfied("jkt-value", "") {
		t.Fatal("DPoP の鍵サムプリントが証拠として認められなかった")
	}
	if !client.SenderConstraintSatisfied("", "x5t-value") {
		t.Fatal("mTLS 証明書のサムプリントが証拠として認められなかった")
	}
}

// FAPI2-PROFILE-SELECTION: プロファイルを選択していないクライアントに、3 つの追加
// 制約が 1 つも掛からないことを固定する。制約が既定化した変更はここが落ちる。
func TestOAuth2ClientLeavesNonFapi2ClientsUnconstrained(t *testing.T) {
	client := fapi2TestClient(FapiNone, AuthMethodClientSecretBasic)
	if client.UsesFapi2SecurityProfile() {
		t.Fatal("プロファイルを選んでいないクライアントが選択済みと判定された")
	}
	if err := client.Validate(); err != nil {
		t.Fatalf("共有シークレットのクライアントが拒否された: %v", err)
	}
	if client.MustUsePushedAuthorizationRequests() {
		t.Fatal("PAR を要求していないクライアントに PAR が必須になった")
	}
	if !client.SenderConstraintSatisfied("", "") {
		t.Fatal("証拠の無い要求が送信者制約に引っかかった")
	}

	// 対照: クライアント個別の PAR 設定はプロファイルと独立に効き続ける。
	client.RequirePushedAuthorizationRequests = true
	if !client.MustUsePushedAuthorizationRequests() {
		t.Fatal("個別に PAR 必須としたクライアントで PAR が必須にならなかった")
	}
}

// FAPI2-PAR-PKCE: プロファイルの選択だけで PAR が必須になることを固定する。
func TestOAuth2ClientRequiresPushedAuthorizationRequestsUnderFapi2(t *testing.T) {
	client := fapi2TestClient(FapiSecurityProfileV2, AuthMethodPrivateKeyJwt)
	if !client.MustUsePushedAuthorizationRequests() {
		t.Fatal("プロファイルを選んだクライアントで PAR が必須にならなかった")
	}
}
