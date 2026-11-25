package upstream_http

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/shared/testing_oauthtwin"
	tokenvaultdomain "github.com/ambi/idmagic/backend/tokenvault/internal/domain"
	tokenvaultports "github.com/ambi/idmagic/backend/tokenvault/internal/ports"
)

// newTwinClient は模倣を起動し、そのトークンエンドポイントを指すクライアントを返す。
func newTwinClient(t *testing.T, tw *testing_oauthtwin.Twin) (*Client, tokenvaultports.OAuthClient) {
	t.Helper()
	server := tw.Start(t)
	return &Client{HTTP: withoutRedirects(server.Client())}, tokenvaultports.OAuthClient{
		TokenEndpoint: server.URL + "/token", ClientID: "client", ClientSecret: "secret",
	}
}

// 上流が申告する有効期間を、期限なし、そのまま、上限のいずれかにする。
//
//spec:covers REQ-TOKENVAULT-012: expires_in が欠落するか 0 以下なら期限を持たず、365 日を超えるなら 365 日で頭打ちにすること
func TestTokenTwin_BoundsUpstreamLifetime(t *testing.T) {
	year := tokenvaultdomain.MaxTokenLifetime
	tests := []struct {
		name      string
		setup     func(tw *testing_oauthtwin.Twin)
		want      *time.Duration
		wantLabel string
	}{
		{"欠落", func(tw *testing_oauthtwin.Twin) { tw.OmitExpiresIn = true }, nil, "nil"},
		{"0", func(tw *testing_oauthtwin.Twin) { tw.ExpiresIn = 0 }, nil, "nil"},
		{"負", func(tw *testing_oauthtwin.Twin) { tw.ExpiresIn = -1 }, nil, "nil"},
		{"1 秒", func(tw *testing_oauthtwin.Twin) { tw.ExpiresIn = 1 }, new(time.Second), "1s"},
		{"365 日ちょうど", func(tw *testing_oauthtwin.Twin) { tw.ExpiresIn = int64(year / time.Second) }, &year, "365d"},
		{"365 日と 1 秒", func(tw *testing_oauthtwin.Twin) { tw.ExpiresIn = int64(year/time.Second) + 1 }, &year, "365d"},
		// time.Duration のナノ秒で溢れる大きさ。秒のまま比べないと負に回り込み、上限の分岐を素通りする。
		{"2^40 秒", func(tw *testing_oauthtwin.Twin) { tw.ExpiresIn = 1 << 40 }, &year, "365d"},
		{"int64 の最大値", func(tw *testing_oauthtwin.Twin) { tw.ExpiresIn = 1<<63 - 1 }, &year, "365d"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tw := testing_oauthtwin.NewTwin("client", "secret")
			tw.AddRefreshToken("old")
			tc.setup(tw)
			client, oauth := newTwinClient(t, tw)
			response, err := client.Refresh(context.Background(), oauth, "old")
			if err != nil {
				t.Fatal(err)
			}
			switch {
			case tc.want == nil && response.ExpiresIn != nil:
				t.Errorf("ExpiresIn = %v, want %s", *response.ExpiresIn, tc.wantLabel)
			case tc.want != nil && (response.ExpiresIn == nil || *response.ExpiresIn != *tc.want):
				t.Errorf("ExpiresIn = %v, want %s", response.ExpiresIn, tc.wantLabel)
			}
		})
	}
}

// 更新トークンの回転：回転する上流では新しい値を返し、旧い値は invalid_grant で拒否される。
// 回転しない上流では応答が更新トークンを持たず、保存した値を維持する判断はユースケースが行う。
//
//spec:covers REQ-TOKENVAULT-012: 応答が新しい更新トークンを持つときはその値を、持たないときは空を返すこと
func TestTokenTwin_RefreshTokenRotation(t *testing.T) {
	ctx := context.Background()
	t.Run("回転する", func(t *testing.T) {
		tw := testing_oauthtwin.NewTwin("client", "secret")
		tw.AddRefreshToken("old")
		client, oauth := newTwinClient(t, tw)
		response, err := client.Refresh(ctx, oauth, "old")
		if err != nil || response.RefreshToken == "" || response.RefreshToken == "old" {
			t.Fatalf("Refresh() = %+v, %v, want a rotated refresh token", response, err)
		}
		_, err = client.Refresh(ctx, oauth, "old")
		if refused, ok := errors.AsType[*tokenvaultports.TokenEndpointError](err); !ok || refused.OAuthError != "invalid_grant" {
			t.Errorf("Refresh(old) error = %v, want invalid_grant", err)
		}
		if _, err := client.Refresh(ctx, oauth, response.RefreshToken); err != nil {
			t.Errorf("Refresh(rotated) error = %v", err)
		}
	})
	t.Run("回転しない", func(t *testing.T) {
		tw := testing_oauthtwin.NewTwin("client", "secret")
		tw.RotateRefreshTokens = false
		tw.AddRefreshToken("old")
		client, oauth := newTwinClient(t, tw)
		for range 2 {
			response, err := client.Refresh(ctx, oauth, "old")
			if err != nil || response.RefreshToken != "" {
				t.Fatalf("Refresh() = %+v, %v, want no refresh token", response, err)
			}
		}
	})
}

// 上流の拒否と障害を、再認可で直るもの、直らないもの、不正な応答に分けて返す。
//
//spec:covers REQ-TOKENVAULT-013: invalid_grant、invalid_client、429、5xx、不正な本文を区別して返すこと
func TestTokenTwin_ClassifiesRefusalsAndFaults(t *testing.T) {
	tests := []struct {
		name           string
		setup          func(tw *testing_oauthtwin.Twin, oauth *tokenvaultports.OAuthClient)
		wantStatus     int
		wantOAuthError string
		wantOutcome    tokenvaultdomain.RefreshOutcome
	}{
		{"未知の更新トークン", func(*testing_oauthtwin.Twin, *tokenvaultports.OAuthClient) {}, 400, "invalid_grant", tokenvaultdomain.RefreshRequiresReauthorization},
		{"クライアントの資格情報の誤り", func(_ *testing_oauthtwin.Twin, oauth *tokenvaultports.OAuthClient) { oauth.ClientSecret = "wrong" }, 401, "invalid_client", tokenvaultdomain.RefreshTransient},
		{"429", func(tw *testing_oauthtwin.Twin, _ *tokenvaultports.OAuthClient) {
			tw.InjectFaults(testing_oauthtwin.Fault{Status: 429, RetryAfter: "7"})
		}, 429, "", tokenvaultdomain.RefreshTransient},
		{"503 の HTML", func(tw *testing_oauthtwin.Twin, _ *tokenvaultports.OAuthClient) {
			tw.InjectFaults(testing_oauthtwin.Fault{Status: 503, ContentType: "text/html", Body: "<html>"})
		}, 503, "", tokenvaultdomain.RefreshTransient},
		{"不正な本文", func(tw *testing_oauthtwin.Twin, _ *tokenvaultports.OAuthClient) {
			tw.InjectFaults(testing_oauthtwin.MalformedBody)
		}, 200, "", tokenvaultdomain.RefreshTransient},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tw := testing_oauthtwin.NewTwin("client", "secret")
			client, oauth := newTwinClient(t, tw)
			tc.setup(tw, &oauth)
			_, err := client.Refresh(context.Background(), oauth, "unknown")
			refused, ok := errors.AsType[*tokenvaultports.TokenEndpointError](err)
			if !ok || refused.Status != tc.wantStatus || refused.OAuthError != tc.wantOAuthError {
				t.Fatalf("error = %#v, want TokenEndpointError{%d, %q}", err, tc.wantStatus, tc.wantOAuthError)
			}
			if got := tokenvaultdomain.ClassifyRefreshFailure(refused.OAuthError); got != tc.wantOutcome {
				t.Errorf("ClassifyRefreshFailure(%q) = %v, want %v", refused.OAuthError, got, tc.wantOutcome)
			}
		})
	}
}
