package client_scim

// RFC7644-OUT-AUTHENTICATION: 接続が `oauth2_client_credentials` を選んだとき、
// 下流へ提示するのはクライアント資格情報フロー (RFC 6749 §4.4) で取得した
// アクセストークンでなければならない。保存した client_secret をそのまま
// ベアラートークンとして送ると、下流が正しく実装されていれば必ず 401 になり、
// 正しく実装されていなければ秘密がアクセストークンとして通ってしまう。

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/provisioning/domain"
)

// oauth2Downstream は、指定されたアクセストークンだけを受理する下流を組み立てる。
// トークンエンドポイントと SCIM 接点を 1 つのサーバーに同居させ、それぞれの
// 呼ばれ方を数える。
type oauth2Downstream struct {
	server        *httptest.Server
	tokenRequests *atomic.Int32
	scimRequests  *atomic.Int32
	lastTokenForm *atomic.Pointer[string]
	lastScimAuth  *atomic.Pointer[string]
	issued        *atomic.Int32
	expiresIn     int
}

func newOAuth2Downstream(t *testing.T, expiresIn int) *oauth2Downstream {
	t.Helper()
	down := &oauth2Downstream{
		tokenRequests: &atomic.Int32{}, scimRequests: &atomic.Int32{},
		lastTokenForm: &atomic.Pointer[string]{}, lastScimAuth: &atomic.Pointer[string]{},
		issued: &atomic.Int32{}, expiresIn: expiresIn,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth2/token", func(w http.ResponseWriter, r *http.Request) {
		down.tokenRequests.Add(1)
		if err := r.ParseForm(); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		form := r.Form.Encode()
		down.lastTokenForm.Store(&form)
		// クライアント認証は client_secret_post か Basic のどちらかで来る。
		id, secret, hasBasic := r.BasicAuth()
		if !hasBasic {
			id, secret = r.Form.Get("client_id"), r.Form.Get("client_secret")
		}
		if id != "downstream-client" || secret != "downstream-secret" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		nth := down.issued.Add(1)
		body := map[string]any{"access_token": accessTokenFor(nth), "token_type": "Bearer"}
		if down.expiresIn > 0 {
			body["expires_in"] = down.expiresIn
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(body)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		down.scimRequests.Add(1)
		auth := r.Header.Get("Authorization")
		down.lastScimAuth.Store(&auth)
		// 現在有効なトークンだけを受理する。client_secret をそのまま送れば 401。
		if auth != "Bearer "+accessTokenFor(down.issued.Load()) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/scim+json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "remote-1"})
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	down.server = server
	return down
}

func accessTokenFor(nth int32) string {
	return "downstream-access-token-" + strconv.Itoa(int(nth))
}

func (d *oauth2Downstream) client(t *testing.T, now func() time.Time) *Client {
	t.Helper()
	source, err := newOAuth2TokenSource(oauth2ClientCredentials{
		TokenURL:     d.server.URL + "/oauth2/token",
		ClientID:     "downstream-client",
		ClientSecret: "downstream-secret",
		Scope:        "scim",
	}, d.server.Client(), now)
	if err != nil {
		t.Fatal(err)
	}
	return &Client{HTTPClient: d.server.Client(), BaseURL: d.server.URL, tokenSource: source}
}

func createOnce(t *testing.T, client *Client) error {
	t.Helper()
	_, _, err := client.CreateUser(context.Background(),
		[]domain.AttributeMappingRule{simpleRule("userName", "username")},
		map[string]any{"username": "alice"})
	return err
}

func TestClient_OAuth2ClientCredentials_FetchesAndPresentsAnAccessToken(t *testing.T) {
	down := newOAuth2Downstream(t, 3600)
	client := down.client(t, time.Now)

	if err := createOnce(t, client); err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if got := down.tokenRequests.Load(); got != 1 {
		t.Fatalf("トークン取得の回数 = %d, want 1", got)
	}
	// 提示されたのは取得したアクセストークンであって、保存した client_secret ではない。
	auth := *down.lastScimAuth.Load()
	if auth != "Bearer "+accessTokenFor(1) {
		t.Fatalf("Authorization = %q, want the fetched access token", auth)
	}
	if strings.Contains(auth, "downstream-secret") {
		t.Fatalf("client_secret を下流へ提示している: %q", auth)
	}
	// 取得要求は RFC 6749 §4.4 の grant_type と scope を伴う。
	form := *down.lastTokenForm.Load()
	if !strings.Contains(form, "grant_type=client_credentials") {
		t.Fatalf("token form = %q, want grant_type=client_credentials", form)
	}
	if !strings.Contains(form, "scope=scim") {
		t.Fatalf("token form = %q, want scope=scim", form)
	}
}

func TestClient_OAuth2ClientCredentials_ReusesTokenWithinExpiry(t *testing.T) {
	down := newOAuth2Downstream(t, 3600)
	client := down.client(t, time.Now)

	for range 3 {
		if err := createOnce(t, client); err != nil {
			t.Fatalf("CreateUser() error = %v", err)
		}
	}
	if got := down.tokenRequests.Load(); got != 1 {
		t.Fatalf("トークン取得の回数 = %d, want 1 (期限内は再利用する)", got)
	}
	if got := down.scimRequests.Load(); got != 3 {
		t.Fatalf("SCIM 要求の回数 = %d, want 3", got)
	}
}

func TestClient_OAuth2ClientCredentials_RefetchesAfterExpiry(t *testing.T) {
	down := newOAuth2Downstream(t, 120)
	current := time.Now()
	client := down.client(t, func() time.Time { return current })

	if err := createOnce(t, client); err != nil {
		t.Fatalf("1 回目 CreateUser() error = %v", err)
	}
	// 安全余裕 (60 秒) の内側へ入れる。期限そのものより手前で取り直す。
	current = current.Add(70 * time.Second)
	if err := createOnce(t, client); err != nil {
		t.Fatalf("2 回目 CreateUser() error = %v", err)
	}
	if got := down.tokenRequests.Load(); got != 2 {
		t.Fatalf("トークン取得の回数 = %d, want 2 (期限が近づいたら取り直す)", got)
	}
}

func TestClient_OAuth2ClientCredentials_RefetchesOnceOn401(t *testing.T) {
	// 下流がトークンを先に失効させた場合。保持しているトークンを捨てて
	// 1 度だけ取り直し、同じ要求を再送する。
	down := newOAuth2Downstream(t, 3600)
	client := down.client(t, time.Now)

	if err := createOnce(t, client); err != nil {
		t.Fatalf("1 回目 CreateUser() error = %v", err)
	}
	// 下流が新しいトークンを発行し、古いトークンは受理しなくなる。
	down.issued.Add(1)

	if err := createOnce(t, client); err != nil {
		t.Fatalf("401 のあと取り直して成功するはず: %v", err)
	}
	if got := down.tokenRequests.Load(); got != 2 {
		t.Fatalf("トークン取得の回数 = %d, want 2", got)
	}
}

func TestClient_OAuth2ClientCredentials_DoesNotLoopWhenCredentialsAreWrong(t *testing.T) {
	// 資格情報そのものが誤っている接続が、下流のトークンエンドポイントを
	// 叩き続けないこと。取り直しは 1 度だけである。
	down := newOAuth2Downstream(t, 3600)
	source, err := newOAuth2TokenSource(oauth2ClientCredentials{
		TokenURL:     down.server.URL + "/oauth2/token",
		ClientID:     "downstream-client",
		ClientSecret: "wrong-secret",
	}, down.server.Client(), time.Now)
	if err != nil {
		t.Fatal(err)
	}
	client := &Client{HTTPClient: down.server.Client(), BaseURL: down.server.URL, tokenSource: source}

	if err := createOnce(t, client); err == nil {
		t.Fatal("誤った資格情報で成功してはならない")
	}
	if got := down.tokenRequests.Load(); got > 2 {
		t.Fatalf("トークン取得の回数 = %d, want <= 2 (繰り返さない)", got)
	}
	if got := down.scimRequests.Load(); got != 0 {
		t.Fatalf("トークンを取得できていないのに SCIM 要求を %d 件送っている", got)
	}
}

func TestClient_OAuth2ClientCredentials_ErrorOmitsTheDownstreamBody(t *testing.T) {
	// トークン取得の失敗を報告するとき、下流の応答本文を載せない。
	// 本文にはトークンや秘密が含まれうる。
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_client","access_token":"leaked-secret-value"}`))
	}))
	t.Cleanup(server.Close)

	source, err := newOAuth2TokenSource(oauth2ClientCredentials{
		TokenURL: server.URL, ClientID: "c", ClientSecret: "s",
	}, server.Client(), time.Now)
	if err != nil {
		t.Fatal(err)
	}
	_, tokenErr := source.token(context.Background(), false)
	if tokenErr == nil {
		t.Fatal("400 は失敗として扱うはず")
	}
	if strings.Contains(tokenErr.Error(), "leaked-secret-value") {
		t.Fatalf("エラーが下流の応答本文を含んでいる: %v", tokenErr)
	}
	if !strings.Contains(tokenErr.Error(), "400") {
		t.Fatalf("エラーが状態コードを含んでいない: %v", tokenErr)
	}
}

// tokenGrant はトークン応答と現在時刻から、載せるトークンと失効時刻を決める
// 純関数である。RFC 6749 §5.1 は `expires_in` を推奨に留めるので、欠落や
// おかしな値をここで正規化する。
func TestTokenGrant_NormalizesExpiry(t *testing.T) {
	base := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)

	t.Run("expires_in があれば失効時刻を計算する", func(t *testing.T) {
		grant, err := tokenGrant(tokenResponse{AccessToken: "t", ExpiresIn: 3600}, base)
		if err != nil {
			t.Fatal(err)
		}
		if !grant.ExpiresAt.Equal(base.Add(3600 * time.Second)) {
			t.Fatalf("ExpiresAt = %v, want %v", grant.ExpiresAt, base.Add(3600*time.Second))
		}
		if !grant.usableAt(base) {
			t.Fatal("取得直後のトークンは使えるはず")
		}
	})

	t.Run("安全余裕の内側では使えないと判定する", func(t *testing.T) {
		grant, err := tokenGrant(tokenResponse{AccessToken: "t", ExpiresIn: 120}, base)
		if err != nil {
			t.Fatal(err)
		}
		if !grant.usableAt(base.Add(50 * time.Second)) {
			t.Fatal("余裕の外側 (残り 70 秒) はまだ使えるはず")
		}
		if grant.usableAt(base.Add(70 * time.Second)) {
			t.Fatal("余裕の内側 (残り 50 秒) は使えないと判定するはず")
		}
	})

	t.Run("expires_in が無ければ 1 度きりのトークンとして扱う", func(t *testing.T) {
		grant, err := tokenGrant(tokenResponse{AccessToken: "t"}, base)
		if err != nil {
			t.Fatal(err)
		}
		if grant.usableAt(base) {
			t.Fatal("expires_in が無いトークンは次の要求で取り直すはず")
		}
	})

	t.Run("expires_in が 0 以下なら 1 度きりのトークンとして扱う", func(t *testing.T) {
		for _, seconds := range []int64{0, -1, -3600} {
			grant, err := tokenGrant(tokenResponse{AccessToken: "t", ExpiresIn: seconds}, base)
			if err != nil {
				t.Fatalf("expires_in=%d: %v", seconds, err)
			}
			if grant.usableAt(base) {
				t.Fatalf("expires_in=%d を再利用可能と判定している", seconds)
			}
		}
	})

	t.Run("極端に大きい expires_in は上限で切る", func(t *testing.T) {
		// 下流が誤った値を返しても、失効しないトークンを抱え込まない。
		// 上限そのものと、time.Duration のナノ秒で溢れる大きさの両方を見る。
		// 溢れる値だけを見る検査は、上限で切る分岐が無くても
		// 「負に回り込んだので上限より前」で通ってしまい、何も識別しない。
		for _, seconds := range []int64{
			365 * 24 * 60 * 60, // 1 年。溢れないが上限より大きい。
			1 << 40,            // time.Duration のナノ秒では溢れる大きさ。
			1<<63 - 1,          // int64 の最大値。
		} {
			grant, err := tokenGrant(tokenResponse{AccessToken: "t", ExpiresIn: seconds}, base)
			if err != nil {
				t.Fatalf("expires_in=%d: %v", seconds, err)
			}
			want := base.Add(maxTokenLifetime)
			if !grant.ExpiresAt.Equal(want) {
				t.Fatalf("expires_in=%d: ExpiresAt = %v, want exactly %v", seconds, grant.ExpiresAt, want)
			}
		}
	})

	t.Run("access_token が空の応答は拒否する", func(t *testing.T) {
		// fail-closed: 空のトークンを Authorization に載せない。
		if _, err := tokenGrant(tokenResponse{ExpiresIn: 3600}, base); err == nil {
			t.Fatal("access_token が空の応答を受理してはならない")
		}
	})
}

// newTokenSource は認証方式から供給元を選ぶ。ここが分岐していなければ、
// oauth2 の接続が保存した client_secret をそのままベアラートークンとして
// 提示する元の欠陥に戻る。個々の供給元の振る舞いを見る検査は
// newOAuth2TokenSource を直接組み立てるので、この分岐は素通りしてしまう。
func TestNewTokenSource_SelectsBySAuthMethod(t *testing.T) {
	const secret = "stored-credential"

	t.Run("bearer_token は保存した値をそのまま載せる", func(t *testing.T) {
		source, err := newTokenSource(domain.ProvisioningConnectionCredentialMetadata{
			AuthMethod: domain.AuthBearerToken,
		}, secret, http.DefaultClient, time.Now)
		if err != nil {
			t.Fatal(err)
		}
		token, err := source.token(context.Background(), false)
		if err != nil || token != secret {
			t.Fatalf("token = (%q, %v), want the stored credential", token, err)
		}
	})

	t.Run("oauth2_client_credentials は秘密をそのまま載せない", func(t *testing.T) {
		source, err := newTokenSource(domain.ProvisioningConnectionCredentialMetadata{
			AuthMethod:     domain.AuthOAuth2ClientCredentials,
			OAuth2TokenURL: "https://downstream.example.com/oauth2/token",
			OAuth2ClientID: "idmagic-provisioner",
		}, secret, http.DefaultClient, time.Now)
		if err != nil {
			t.Fatal(err)
		}
		// 型で分ける。staticToken が返ってきたら、それは秘密をそのまま
		// 載せる供給元であり、この認証方式では誤りである。
		if _, isStatic := source.(staticToken); isStatic {
			t.Fatal("oauth2_client_credentials に固定トークンの供給元が選ばれている (秘密をそのまま提示する)")
		}
		if _, isOAuth2 := source.(*oauth2TokenSource); !isOAuth2 {
			t.Fatalf("供給元の型 = %T, want *oauth2TokenSource", source)
		}
	})

	t.Run("トークンエンドポイントが非公開アドレスなら組み立てを拒否する", func(t *testing.T) {
		// SSRF: 運用者が入れた URL なので、SCIM の接点と同じ検査を通す。
		_, err := newTokenSource(domain.ProvisioningConnectionCredentialMetadata{
			AuthMethod:     domain.AuthOAuth2ClientCredentials,
			OAuth2TokenURL: "http://127.0.0.1/oauth2/token",
			OAuth2ClientID: "c",
		}, secret, http.DefaultClient, time.Now)
		if err == nil {
			t.Fatal("ループバックのトークンエンドポイントを受理してはならない")
		}
	})

	t.Run("知らない認証方式は拒否する", func(t *testing.T) {
		// fail-closed: 知らない方式で秘密をそのまま送らない。
		if _, err := newTokenSource(domain.ProvisioningConnectionCredentialMetadata{
			AuthMethod: domain.ProvisioningAuthMethod("mutual_tls"),
		}, secret, http.DefaultClient, time.Now); err == nil {
			t.Fatal("知らない認証方式を受理してはならない")
		}
	})
}

// 下流が何を提示しても 401 を返す場合。取り直しは 1 度だけで、要求は
// 2 件で止まる。無条件に繰り返す実装だと、資格情報が誤っている接続が
// 下流を叩き続ける (最悪は終わらない)。
func TestClient_OAuth2ClientCredentials_RetriesOnceWhenTheDownstreamAlwaysRejects(t *testing.T) {
	tokenRequests := &atomic.Int32{}
	scimRequests := &atomic.Int32{}
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth2/token", func(w http.ResponseWriter, _ *http.Request) {
		tokenRequests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "issued", "token_type": "Bearer", "expires_in": 3600,
		})
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		// トークンは発行されるが、SCIM の接点は常に拒否する。
		scimRequests.Add(1)
		w.WriteHeader(http.StatusUnauthorized)
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	source, err := newOAuth2TokenSource(oauth2ClientCredentials{
		TokenURL: server.URL + "/oauth2/token", ClientID: "c", ClientSecret: "s",
	}, server.Client(), time.Now)
	if err != nil {
		t.Fatal(err)
	}
	client := &Client{HTTPClient: server.Client(), BaseURL: server.URL, tokenSource: source}

	if err := createOnce(t, client); err == nil {
		t.Fatal("常に 401 を返す下流に対して成功してはならない")
	}
	if got := scimRequests.Load(); got != 2 {
		t.Fatalf("SCIM 要求の回数 = %d, want 2 (最初の 1 件と、取り直し後の 1 件だけ)", got)
	}
	if got := tokenRequests.Load(); got != 2 {
		t.Fatalf("トークン取得の回数 = %d, want 2", got)
	}
}
