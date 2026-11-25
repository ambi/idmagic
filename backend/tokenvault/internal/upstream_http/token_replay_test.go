package upstream_http

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/shared/testing_oauthtwin"
	tokenvaultports "github.com/ambi/idmagic/backend/tokenvault/internal/ports"
)

// 上流のトークンエンドポイントの実際の応答の形に対して、コードの交換と更新が取り出す結果を確かめる。
//
// 応答は RFC 6749 と RFC 6750 の例を転記した記録から返す。期待値は製品の定数からではなく、
// 記録が示す値と RFC の文面から書く。要求の方式、パス、Content-Type、Authorization、フォームの値が
// 記録と食い違うと、再生サーバーがテストを失敗させる。PKCE の code_verifier は記録にない欄として許す。
//
//spec:covers REQ-TOKENVAULT-004, REQ-TOKENVAULT-013: コードの交換の要求が RFC 6749 §4.1.3 の形であること、token_type が Bearer でない応答 (§4.1.4 の example) を交換の失敗にすること、§5.2 のエラー応答の error を拒否の理由として区別すること
func TestTokenReplay_RFC6749Recordings(t *testing.T) {
	rec := testing_oauthtwin.RFC6749Recording(t)
	ctx := context.Background()

	start := func(t *testing.T, exchange string) (*Client, tokenvaultports.OAuthClient) {
		t.Helper()
		server := testing_oauthtwin.NewReplayServer(t, rec.Exchange(t, exchange))
		return &Client{HTTP: withoutRedirects(server.Client())}, tokenvaultports.OAuthClient{
			TokenEndpoint: server.URL + "/token",
			ClientID:      testing_oauthtwin.RecordedClientID,
			ClientSecret:  testing_oauthtwin.RecordedClientSecret,
		}
	}
	wantRecordedBearer := func(t *testing.T, response tokenvaultports.TokenResponse, err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if response.AccessToken != "mF_9.B5f-4.1JqM" || response.RefreshToken != "tGzv3JOkF0XG5Qx2TlKWIA" {
			t.Errorf("tokens = (%q, %q), want the recorded tokens", response.AccessToken, response.RefreshToken)
		}
		if response.ExpiresIn == nil || *response.ExpiresIn != time.Hour {
			t.Errorf("ExpiresIn = %v, want 1h from expires_in 3600", response.ExpiresIn)
		}
		// 応答が scope を持たないので、付与スコープは要求したものになる（REQ-TOKENVAULT-004）。
		if response.Scopes != nil {
			t.Errorf("Scopes = %v, want nil when the response has no scope", response.Scopes)
		}
	}

	t.Run("コードの交換と RFC 6750 §4 の Bearer の応答", func(t *testing.T) {
		client, oauth := start(t, "authorization-code-bearer")
		response, err := client.ExchangeCode(ctx, oauth, testing_oauthtwin.RecordedCode, "verifier", testing_oauthtwin.RecordedRedirectURI)
		wantRecordedBearer(t, response, err)
	})
	t.Run("コードの交換と RFC 6749 §4.1.4 の種類 example の応答", func(t *testing.T) {
		client, oauth := start(t, "authorization-code-example-type")
		_, err := client.ExchangeCode(ctx, oauth, testing_oauthtwin.RecordedCode, "verifier", testing_oauthtwin.RecordedRedirectURI)
		if refused, ok := errors.AsType[*tokenvaultports.TokenEndpointError](err); !ok || refused.OAuthError != "" {
			t.Errorf("error = %v, want a malformed-response TokenEndpointError", err)
		}
	})
	t.Run("更新と RFC 6750 §4 の Bearer の応答", func(t *testing.T) {
		client, oauth := start(t, "refresh-bearer")
		response, err := client.Refresh(ctx, oauth, testing_oauthtwin.RecordedRefreshToken)
		wantRecordedBearer(t, response, err)
	})
	t.Run("RFC 6749 §5.2 のエラー応答", func(t *testing.T) {
		client, oauth := start(t, "refresh-invalid-request")
		_, err := client.Refresh(ctx, oauth, "")
		refused, ok := errors.AsType[*tokenvaultports.TokenEndpointError](err)
		if !ok || refused.Status != 400 || refused.OAuthError != "invalid_request" {
			t.Errorf("error = %v, want TokenEndpointError{400, invalid_request}", err)
		}
	})
}
