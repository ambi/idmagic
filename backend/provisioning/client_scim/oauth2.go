package client_scim

// 下流 SCIM への認証に使うトークンの供給。RFC7644-OUT-AUTHENTICATION は
// `Authorization: Bearer` の 1 方式だけを課すが、その値をどこから得るかは
// 接続の認証方式で決まる。`bearer_token` は保存した値をそのまま、
// `oauth2_client_credentials` はクライアント資格情報フロー (RFC 6749 §4.4) で
// 取得したアクセストークンを載せる。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/ambi/idmagic/backend/provisioning/domain"
)

const (
	// tokenRefreshSkew は失効時刻の手前でトークンを取り直す幅。時計ずれと
	// 往復時間の分だけ余裕を取る。ここを 0 にすると、期限ぎりぎりのトークンを
	// 載せた要求が下流に届く頃には失効している。
	tokenRefreshSkew = 60 * time.Second
	// maxTokenLifetime は下流が申告する有効期間の上限。誤った値 (あるいは
	// 悪意のある値) を受けて、失効しないトークンを抱え込まないための頭打ち。
	maxTokenLifetime = 24 * time.Hour
	// maxTokenResponseBytes はトークン応答の読み取り上限。
	maxTokenResponseBytes = 1 << 16
)

// tokenSource は 1 回の要求に載せるベアラートークンを返す。
// refresh が true なら保持している値を捨てて取り直す (下流が 401 を返した後)。
type tokenSource interface {
	token(ctx context.Context, refresh bool) (string, error)
}

// staticToken は `bearer_token` の接続が使う供給元。保存した値をそのまま返し、
// 取り直しても同じ値を返す —— 取り直す先が無いからである。
type staticToken string

func (t staticToken) token(context.Context, bool) (string, error) { return string(t), nil }

// oauth2ClientCredentials は接続に保存したクライアント資格情報フローの設定。
// ClientSecret 以外は秘密ではなく、接続の資格情報メタデータとして保存される。
type oauth2ClientCredentials struct {
	TokenURL     string
	ClientID     string
	ClientSecret string
	Scope        string
}

// tokenResponse は RFC 6749 §5.1 のトークン応答のうち、この用途で読む欄。
type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

// grant は取得したトークンと、それを再利用してよい期限。
type grant struct {
	AccessToken string
	ExpiresAt   time.Time
}

// usableAt は now の時点でこのトークンを再利用してよいかを返す。
// 失効そのものではなく、安全余裕の分だけ手前で使えないと判定する。
func (g grant) usableAt(now time.Time) bool {
	return g.AccessToken != "" && now.Add(tokenRefreshSkew).Before(g.ExpiresAt)
}

// tokenGrant はトークン応答と取得時刻から grant を決める純関数である。
// RFC 6749 §5.1 は `expires_in` を推奨に留めるので、欠落や 0 以下は
// 「1 度きり」として扱い、次の要求で取り直させる。極端に大きい値は頭打ちにする。
func tokenGrant(response tokenResponse, now time.Time) (grant, error) {
	if response.AccessToken == "" {
		// fail-closed: 空のトークンを Authorization に載せない。
		return grant{}, errors.New("provisioning/scim: token response has no access_token")
	}
	// 上限との比較は秒のまま行う。time.Duration へ先に変換すると、下流が申告した
	// 大きな値 (2^40 秒など) が int64 のナノ秒で溢れて負に回り込み、上限で切る
	// つもりの分岐を素通りする。
	seconds := max(response.ExpiresIn, 0)
	if maxSeconds := int64(maxTokenLifetime / time.Second); seconds > maxSeconds {
		seconds = maxSeconds
	}
	return grant{
		AccessToken: response.AccessToken,
		ExpiresAt:   now.Add(time.Duration(seconds) * time.Second),
	}, nil
}

// oauth2TokenSource はクライアント資格情報フローでアクセストークンを取得し、
// 失効まで再利用する。時刻は入力として受け取り、HTTP クライアントは注入する。
type oauth2TokenSource struct {
	credentials oauth2ClientCredentials
	httpClient  *http.Client
	now         func() time.Time

	mu   sync.Mutex
	held grant
}

func newOAuth2TokenSource(
	credentials oauth2ClientCredentials,
	httpClient *http.Client,
	now func() time.Time,
) (*oauth2TokenSource, error) {
	if credentials.TokenURL == "" || credentials.ClientID == "" || credentials.ClientSecret == "" {
		return nil, errors.New("provisioning/scim: oauth2_client_credentials needs a token URL, client id and client secret")
	}
	if now == nil {
		now = time.Now
	}
	return &oauth2TokenSource{credentials: credentials, httpClient: httpClient, now: now}, nil
}

func (s *oauth2TokenSource) token(ctx context.Context, refresh bool) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !refresh && s.held.usableAt(s.now()) {
		return s.held.AccessToken, nil
	}
	fetched, err := s.fetch(ctx)
	if err != nil {
		// 取れなかったときは保持していた値も捨てる。期限切れの値を後続の
		// 要求が使い回すと、失敗の原因が資格情報から遠ざかる。
		s.held = grant{}
		return "", err
	}
	s.held = fetched
	return fetched.AccessToken, nil
}

func (s *oauth2TokenSource) fetch(ctx context.Context) (grant, error) {
	form := url.Values{"grant_type": {"client_credentials"}}
	if s.credentials.Scope != "" {
		form.Set("scope", s.credentials.Scope)
	}
	request, err := http.NewRequestWithContext(
		ctx, http.MethodPost, s.credentials.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return grant{}, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Accept", "application/json")
	// client_secret_basic (RFC 6749 §2.3.1)。秘密を本文ではなくヘッダーへ置くので、
	// 本文を記録する中間装置に残りにくい。
	request.SetBasicAuth(
		url.QueryEscape(s.credentials.ClientID), url.QueryEscape(s.credentials.ClientSecret))

	response, err := s.httpClient.Do(request)
	if err != nil {
		return grant{}, fmt.Errorf("provisioning/scim: fetch access token: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	data, err := io.ReadAll(io.LimitReader(response.Body, maxTokenResponseBytes+1))
	if err != nil {
		return grant{}, err
	}
	if response.StatusCode != http.StatusOK {
		// 応答本文は載せない。トークンや秘密が含まれうる (RFC 6749 §5.2 の
		// エラー応答であっても、下流が何を書くかはこちらが決められない)。
		return grant{}, fmt.Errorf("provisioning/scim: token endpoint returned %d", response.StatusCode)
	}
	var decoded tokenResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		return grant{}, errors.New("provisioning/scim: token endpoint returned a body that is not a token response")
	}
	return tokenGrant(decoded, s.now())
}

// newTokenSource は接続の認証方式から供給元を組み立てる。認証方式による分岐は
// ここ 1 か所に閉じ、送出の 6 経路はどれも供給元の違いを知らない。
func newTokenSource(
	credential domain.ProvisioningConnectionCredentialMetadata,
	secret string,
	httpClient *http.Client,
	now func() time.Time,
) (tokenSource, error) {
	switch credential.AuthMethod {
	case domain.AuthOAuth2ClientCredentials:
		if err := domain.ValidateOutboundBaseURL(credential.OAuth2TokenURL); err != nil {
			return nil, fmt.Errorf("provisioning/scim: oauth2 token URL: %w", err)
		}
		return newOAuth2TokenSource(oauth2ClientCredentials{
			TokenURL:     credential.OAuth2TokenURL,
			ClientID:     credential.OAuth2ClientID,
			ClientSecret: secret,
			Scope:        credential.OAuth2Scope,
		}, httpClient, now)
	case domain.AuthBearerToken:
		return staticToken(secret), nil
	default:
		// fail-closed: 知らない認証方式で秘密をそのまま送らない。
		return nil, fmt.Errorf("provisioning/scim: unsupported auth method %q", credential.AuthMethod)
	}
}
