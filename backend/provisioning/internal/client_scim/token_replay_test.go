package client_scim

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/shared/testing_oauthtwin"
)

// 上流のトークンエンドポイントの実際の応答の形に対して、クライアント資格情報フローが得る結果を確かめる。
//
// 応答は RFC 6749 と RFC 6750 の例を転記した記録から返す。期待値は製品の定数からではなく、
// 記録が示す値と RFC の文面から書く。要求の方式、パス、Content-Type、Authorization、フォームの値が
// 記録と食い違うと、再生サーバーがテストを失敗させる。
//
//spec:covers RFC7644-OUT-AUTHENTICATION: クライアント資格情報フローの要求が RFC 6749 §4.4.2 の形であること、token_type が Bearer の応答のアクセストークンだけを使い、RFC 6749 §4.4.3 の例の種類 (example) のトークンを下流へ載せないこと
func TestTokenReplay_RFC6749Recordings(t *testing.T) {
	rec := testing_oauthtwin.RFC6749Recording(t)
	retrieved := time.Date(2026, 11, 26, 9, 0, 0, 0, time.UTC)

	fetch := func(t *testing.T, exchange string) (*oauth2TokenSource, string, error) {
		t.Helper()
		server := testing_oauthtwin.NewReplayServer(t, rec.Exchange(t, exchange))
		source, err := newOAuth2TokenSource(oauth2ClientCredentials{
			TokenURL:     server.URL + "/token",
			ClientID:     testing_oauthtwin.RecordedClientID,
			ClientSecret: testing_oauthtwin.RecordedClientSecret,
		}, server.Client(), func() time.Time { return retrieved })
		if err != nil {
			t.Fatal(err)
		}
		token, err := source.token(context.Background(), false)
		return source, token, err
	}

	t.Run("RFC 6750 §4 の Bearer の応答", func(t *testing.T) {
		source, token, err := fetch(t, "client-credentials-bearer")
		if err != nil {
			t.Fatalf("token() error = %v", err)
		}
		if token != "mF_9.B5f-4.1JqM" {
			t.Errorf("token() = %q, want the recorded access token", token)
		}
		// expires_in の 3600 秒から、失効時刻を取得時刻の 1 時間後と決める。
		if want := retrieved.Add(time.Hour); !source.held.ExpiresAt.Equal(want) {
			t.Errorf("ExpiresAt = %v, want %v", source.held.ExpiresAt, want)
		}
	})

	t.Run("RFC 6749 §4.4.3 の種類 example の応答", func(t *testing.T) {
		// RFC 6749 §7.1: 種類を理解できないアクセストークンを使ってはならない。
		_, token, err := fetch(t, "client-credentials-example-type")
		if err == nil {
			t.Fatalf("token() = %q, want an error for token_type example", token)
		}
	})
}
