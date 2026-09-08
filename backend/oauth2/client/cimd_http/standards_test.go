package cimd_http

// docs/contexts/oauth2/standards.md の `OAuth Client ID Metadata Document`
// (draft-ietf-oauth-client-id-metadata-document-00) が宣言する 7 行を、
// 製品の解決経路そのもので観測する。
//
// 観測の入口は 2 つある。文書の形と取得を言う 6 行は
// ClientRepositoryWithCIMD.FindByID から観測する。ここが、登録簿を外した
// client_id が文書へ問い合わせに行く唯一の経路であり、cmd/internal/bootstrap
// の memory.go と postgres.go がこのデコレータを登録簿へ被せている。
// 認可リクエストの redirect_uri を言う 1 行だけは、解決したクライアントを
// 使う側の判断なので Authorize から観測する。
//
// 本番と差し替えているのは TLS の信頼点だけである。文書を配るのは
// httptest の自己署名サーバーなので、その証明書を信頼する http.Client を
// Fetcher へ渡す。URL 形状の検査、取得、上限、解析、キャッシュはすべて
// 製品の Fetcher と ClientRepositoryWithCIMD が行う。安全な取得側
// (SSRF 防護と非公開 IP の拒否) は NewFetcher を直接使う
// refusal_effects_test.go が持つ。

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	authorizationmemory "github.com/ambi/idmagic/backend/oauth2/authorization/db_memory"
	authorizationusecases "github.com/ambi/idmagic/backend/oauth2/authorization/usecases"
	clientmemory "github.com/ambi/idmagic/backend/oauth2/client/db_memory"
	clientdomain "github.com/ambi/idmagic/backend/oauth2/client/domain"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

const (
	documentPath       = "/oauth/client-metadata.json"
	documentClientName = "Example CIMD Client"
	documentRedirect   = "https://app.example/cb"
)

// metadataHost は Client ID Metadata Document を配るサーバーである。
// requests は受けた要求数で、「取得しなかった」ことを直接読むために持つ。
// lastPath は最後に要求された経路で、「その URL から取得した」ことを読むために持つ。
type metadataHost struct {
	server   *httptest.Server
	requests atomic.Int64
	lastPath atomic.Value
	document func(clientIDURL string) string
	headers  map[string]string
}

// newMetadataHost は document が返す本文を配る TLS サーバーを立てる。
// client_id は https でなければならないので、平文の httptest では代用できない。
func newMetadataHost(t *testing.T, document func(clientIDURL string) string) *metadataHost {
	t.Helper()
	return startMetadataHost(t, document, httptest.NewTLSServer)
}

// newPlaintextMetadataHost は同じ文書を http で配る。https の要求が形の検査で
// 止まっていることを、平文の待ち受けが要求を 1 件も受けないことで読む。
// TLS の待ち受けへ http で送っても接続そのものが成立しないので、平文の待ち受けを
// 用意しない限り「スキームの検査が効いている」と「TLS で失敗した」を区別できない。
func newPlaintextMetadataHost(t *testing.T, document func(clientIDURL string) string) *metadataHost {
	t.Helper()
	return startMetadataHost(t, document, httptest.NewServer)
}

func startMetadataHost(
	t *testing.T,
	document func(clientIDURL string) string,
	start func(http.Handler) *httptest.Server,
) *metadataHost {
	t.Helper()
	host := &metadataHost{document: document, headers: map[string]string{}}
	host.server = start(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host.requests.Add(1)
		host.lastPath.Store(r.URL.Path)
		for name, value := range host.headers {
			w.Header().Set(name, value)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, host.document(host.clientIDURL()))
	}))
	t.Cleanup(host.server.Close)
	return host
}

// clientIDURL は文書を配っている URL であり、それがそのまま client_id である。
func (h *metadataHost) clientIDURL() string { return h.server.URL + documentPath }

// resolution は、このサーバーの証明書を信頼する http.Client の上に
// 製品と同じ Fetcher と ClientRepositoryWithCIMD を組み立てる。
// 登録簿は空にする。CIMD の解決は登録簿を外したときにだけ走る。
func (h *metadataHost) resolution() (*ClientRepositoryWithCIMD, *[]spec.DomainEvent) {
	emitted := &[]spec.DomainEvent{}
	return &ClientRepositoryWithCIMD{
		OAuth2ClientRepository: clientmemory.NewClientRepository(),
		Fetcher:                newFetcherWithClient(h.server.Client()),
		Emit:                   func(event spec.DomainEvent) { *emitted = append(*emitted, event) },
	}, emitted
}

// validDocument は 7 行すべてを満たす文書である。個々のテストはここから
// 1 か所だけを崩す。崩していない事例が通ることを毎回対照に置くので、
// 拒否の理由が崩した 1 か所であることが読める。
func validDocument(clientIDURL string) string {
	return `{
		"client_id": "` + clientIDURL + `",
		"client_name": "` + documentClientName + `",
		"redirect_uris": ["` + documentRedirect + `", "https://app.example/cb2"]
	}`
}

// resolveDocument は与えた本文を配るサーバーを立て、その URL を client_id として
// 製品の解決経路へ通し、解決結果と記録された事象を返す。
func resolveDocument(t *testing.T, document func(clientIDURL string) string) (*clientdomain.OAuth2Client, []spec.DomainEvent) {
	t.Helper()
	host := newMetadataHost(t, document)
	repository, emitted := host.resolution()
	resolved, err := repository.FindByID(t.Context(), "default", host.clientIDURL())
	if err != nil {
		t.Fatalf("解決は「未知の client_id」へ畳まれるべきで、err ではない: %v", err)
	}
	return resolved, *emitted
}

// requireRefused は、文書が取り込まれず、拒否が監査に残ったことを読む。
// 応答は「未知の client_id」へ畳まれるので、拒否が起きたことは事象でしか読めない。
func requireRefused(t *testing.T, resolved *clientdomain.OAuth2Client, emitted []spec.DomainEvent) {
	t.Helper()
	if resolved != nil {
		t.Fatalf("拒否すべき文書からクライアントが組み立てられた: %#v", resolved)
	}
	rejected := 0
	for _, event := range emitted {
		switch event.(type) {
		case *clientdomain.ClientIdMetadataDocumentRejected:
			rejected++
		case *clientdomain.ClientIdMetadataDocumentResolved:
			t.Fatalf("拒否すべき文書が解決済みとして記録された: %#v", event)
		}
	}
	if rejected != 1 {
		t.Fatalf("拒否の記録が %d 件、期待は 1 件", rejected)
	}
}

// CIMD00-FETCH / CIMD00-URL-SHAPE:
// 登録簿を外した client_id が https かつ空でないパスを持つ URL の形をしていれば、
// その URL から文書を取得してクライアントを組み立てる。形が外れていれば、
// 同じ文書が同じホストで配られていても取得そのものを行わない。
//
// 取得数を読むのは、形の検査が通信の前に立っていることを固定するためである。
// 応答だけでは、取得してから捨てる実装と区別できない。
func TestClientIDMetadataDocumentIsFetchedOnlyForHTTPSURLClientIDs(t *testing.T) {
	host := newMetadataHost(t, validDocument)
	repository, emitted := host.resolution()

	// 対照: 正しい形の client_id は取得され、文書の内容がクライアントになる。
	resolved, err := repository.FindByID(t.Context(), "default", host.clientIDURL())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved == nil {
		t.Fatal("https かつパスを持つ client_id が解決されなかった")
	}
	if resolved.ClientID != host.clientIDURL() {
		t.Errorf("ClientID = %q, want %q", resolved.ClientID, host.clientIDURL())
	}
	if resolved.ClientName == nil || *resolved.ClientName != documentClientName {
		t.Errorf("ClientName = %v, want %q (取得した文書の値)", resolved.ClientName, documentClientName)
	}
	if len(resolved.RedirectURIs) != 2 || resolved.RedirectURIs[0] != documentRedirect {
		t.Errorf("RedirectURIs = %v, want 取得した文書の値", resolved.RedirectURIs)
	}
	if got := host.requests.Load(); got != 1 {
		t.Fatalf("サーバーが受けた要求は %d 件、期待は 1 件", got)
	}
	// 取得先は client_id が指す URL そのものである。別の場所から取ってきた文書を
	// この client_id のものとして扱う実装は、ここで落ちる。
	if got := host.lastPath.Load(); got != documentPath {
		t.Errorf("取得先の経路は %v、期待は %q", got, documentPath)
	}
	if len(*emitted) != 1 {
		t.Fatalf("記録された事象が %d 件、期待は 1 件", len(*emitted))
	}
	if _, ok := (*emitted)[0].(*clientdomain.ClientIdMetadataDocumentResolved); !ok {
		t.Errorf("記録は ClientIdMetadataDocumentResolved であるべき: %T", (*emitted)[0])
	}

	// 形が外れた client_id は、同じ文書が実際に配られていても取得しない。
	// http のものだけは平文の待ち受けから取り、残りは同じ TLS の待ち受けから取る。
	plaintext := newPlaintextMetadataHost(t, validDocument)
	for name, target := range map[string]struct {
		host     *metadataHost
		clientID string
	}{
		"http スキーム":    {plaintext, plaintext.clientIDURL()},
		"パスを持たない":      {host, host.server.URL},
		"userinfo を持つ": {host, strings.Replace(host.clientIDURL(), "https://", "https://user:pass@", 1)},
		"fragment を持つ": {host, host.clientIDURL() + "#frag"},
		"URL ではない":     {host, "opaque-client-id"},
	} {
		t.Run(name, func(t *testing.T) {
			before := target.host.requests.Load()
			resolved, err := repository.FindByID(t.Context(), "default", target.clientID)
			if err != nil {
				t.Fatalf("解決は「未知の client_id」へ畳まれるべきで、err ではない: %v", err)
			}
			if resolved != nil {
				t.Fatalf("CIMD の識別子として認められない形の client_id が解決された: %#v", resolved)
			}
			if got := target.host.requests.Load() - before; got != 0 {
				t.Fatalf("形の検査は取得の前に立つべきだが、要求が %d 件届いた", got)
			}
		})
	}
}

// CIMD00-CLIENT-ID-MATCH:
// 取得した文書の client_id フィールドが取得元 URL と厳密に一致しなければ、
// その文書をクライアントにしない。「厳密に」を読むため、別ホストだけでなく
// 末尾スラッシュ 1 文字違いとホスト名の大文字化も試す。正規化して受け入れる
// 実装へ緩めば、攻撃者は自分の配る文書で別の client_id を名乗れる。
func TestClientIDMetadataDocumentMustNameTheURLItWasFetchedFrom(t *testing.T) {
	// 対照: 完全に一致する文書は解決される。
	resolved, emitted := resolveDocument(t, validDocument)
	if resolved == nil {
		t.Fatal("取得元 URL と一致する client_id を持つ文書が解決されなかった")
	}
	if len(emitted) != 1 {
		t.Fatalf("記録された事象が %d 件、期待は 1 件", len(emitted))
	}

	for name, rewrite := range map[string]func(string) string{
		"別ホストの client_id": func(string) string { return "https://attacker.example/client.json" },
		"末尾スラッシュ 1 文字違い":  func(url string) string { return url + "/" },
		"空のクエリ 1 文字違い":    func(url string) string { return url + "?" },
		"パスの大文字化":         strings.ToUpper,
		"client_id が空文字列": func(string) string { return "" },
	} {
		t.Run(name, func(t *testing.T) {
			resolved, emitted := resolveDocument(t, func(clientIDURL string) string {
				return `{
					"client_id": "` + rewrite(clientIDURL) + `",
					"client_name": "` + documentClientName + `",
					"redirect_uris": ["` + documentRedirect + `"]
				}`
			})
			requireRefused(t, resolved, emitted)
		})
	}
}

// CIMD00-STRUCTURE:
// 文書が正しい JSON であり、client_id、client_name、空でない redirect_uris を
// 含むことを検証する。欠けていればフェイルクローズで拒否する。
//
// フェイルクローズであることは「拒否した」だけでは読めない。欠けた項目を
// 既定値で埋めて組み立てる実装も、その 1 点では同じに見える。ここでは
// クライアントが 1 つも組み立てられていないことと、拒否が監査に残ることを対で読む。
func TestMalformedClientIDMetadataDocumentIsRefusedFailClosed(t *testing.T) {
	for name, body := range map[string]string{
		"JSON ではない":   `not json at all`,
		"途中で切れた JSON": `{"client_id": "__SELF__",`,
		"3 項目の後に余分なデータがある": `{"client_id": "__SELF__", "client_name": "` + documentClientName +
			`", "redirect_uris": ["` + documentRedirect + `"]} <not json>`,
		// 必須の 3 項目はそろっていて、別の項目だけ型が違う。JSON としての
		// 妥当性を見ずに「読めた値」で組み立てる実装は、この事例でだけ区別できる。
		// ほかの不正な JSON は client_id が空のまま残るので、一致検査に先に捕まる。
		"必須の 3 項目は正しいが別の項目の型が違う": `{"client_id": "__SELF__", "client_name": "` + documentClientName +
			`", "redirect_uris": ["` + documentRedirect + `"], "grant_types": "authorization_code"}`,
		"オブジェクトではない":           `["https://app.example/c.json"]`,
		"client_name が無い":      `{"client_id": "__SELF__", "redirect_uris": ["` + documentRedirect + `"]}`,
		"client_name が空":       `{"client_id": "__SELF__", "client_name": "", "redirect_uris": ["` + documentRedirect + `"]}`,
		"redirect_uris が無い":    `{"client_id": "__SELF__", "client_name": "` + documentClientName + `"}`,
		"redirect_uris が空の配列":  `{"client_id": "__SELF__", "client_name": "` + documentClientName + `", "redirect_uris": []}`,
		"redirect_uris が配列でない": `{"client_id": "__SELF__", "client_name": "` + documentClientName + `", "redirect_uris": "` + documentRedirect + `"}`,
	} {
		t.Run(name, func(t *testing.T) {
			resolved, emitted := resolveDocument(t, func(clientIDURL string) string {
				return strings.ReplaceAll(body, "__SELF__", clientIDURL)
			})
			requireRefused(t, resolved, emitted)
		})
	}

	// 対照: 3 項目がそろった文書は解決される。上の拒否が、文書の取得や
	// 経路そのものの失敗ではないことを示す。
	if resolved, _ := resolveDocument(t, validDocument); resolved == nil {
		t.Fatal("前提が壊れている: 正しい文書が解決されない")
	}
}

// CIMD00-PRIVATE-KEY-JWT (excluded):
// private_key_jwt クライアント認証を、インラインの jwks または jwks_uri 経由で
// 提供してよい、という選択肢は採らない。
//
// この行の Statement は製品の制約ではなく標準側の機能を書いているので、観測は
// [[wi-500-back-api-tokens-standards-rows-with-tests]] の RFC6750-API-TOKEN-QUERY と
// 同じ向きになる。すなわち「その機能を使う要求が通らないこと」と、「通らなかった
// 結果として何が存在しないか」を対で読む。ここでは次の 2 つを読む。
//
//   - private_key_jwt を宣言した文書は、フェイルクローズで丸ごと拒否される。
//     認証方式だけ none へ落として残りを取り込む、という緩め方をしていない。
//   - 認証方式を宣言せず鍵材料だけ載せた文書は解決されるが、その鍵材料は
//     クライアントへ 1 つも入らない。文書に書いてあっても private_key_jwt の
//     資格情報にはならない。
func TestClientIDMetadataDocumentDoesNotOfferPrivateKeyJwtAuthentication(t *testing.T) {
	const inlineJWKS = `"jwks": {"keys": [{"kty": "RSA", "kid": "k1", "n": "AQAB", "e": "AQAB"}]}`

	for name, extra := range map[string]string{
		"インラインの jwks 付き": `"token_endpoint_auth_method": "private_key_jwt", ` + inlineJWKS,
		"jwks_uri 付き":    `"token_endpoint_auth_method": "private_key_jwt", "jwks_uri": "https://app.example/jwks.json"`,
		"鍵材料を伴わない宣言だけ":   `"token_endpoint_auth_method": "private_key_jwt"`,
	} {
		t.Run(name, func(t *testing.T) {
			resolved, emitted := resolveDocument(t, func(clientIDURL string) string {
				return `{
					"client_id": "` + clientIDURL + `",
					"client_name": "` + documentClientName + `",
					"redirect_uris": ["` + documentRedirect + `"],
					` + extra + `
				}`
			})
			requireRefused(t, resolved, emitted)
		})
	}

	// 拒否が防いだもの: 鍵材料は、文書に載っていてもクライアントへ入らない。
	// 認証方式を宣言しない文書は解決されるが、出来上がるのは none の公開
	// クライアントであり、private_key_jwt を成立させる資格情報を持たない。
	resolved, _ := resolveDocument(t, func(clientIDURL string) string {
		return `{
			"client_id": "` + clientIDURL + `",
			"client_name": "` + documentClientName + `",
			"redirect_uris": ["` + documentRedirect + `"],
			` + inlineJWKS + `,
			"jwks_uri": "https://app.example/jwks.json"
		}`
	})
	if resolved == nil {
		t.Fatal("認証方式を宣言しない文書は解決されるべき")
	}
	if resolved.TokenEndpointAuthMethod != clientdomain.AuthMethodNone {
		t.Errorf("TokenEndpointAuthMethod = %q, want none", resolved.TokenEndpointAuthMethod)
	}
	if resolved.JWKS != nil {
		t.Errorf("文書の jwks がクライアントへ取り込まれた: %#v", resolved.JWKS)
	}
	if resolved.JwksURI != nil {
		t.Errorf("文書の jwks_uri がクライアントへ取り込まれた: %q", *resolved.JwksURI)
	}
}

// CIMD00-CACHE (partial):
// 採っている範囲は「文書をキャッシュする」ことだけで、「HTTP キャッシュヘッダーに
// 従って」の部分は採っていない。Fetcher は解決に成功した文書を一律 5 分保持し、
// 応答の Cache-Control を読まない。partial はこの 2 面を対で読む。
//
// 採らなかった側を no-store で観測するのは、この行が partial である理由を
// 記録に残すためである。ここが required へ変わるなら、まずこのテストが落ちる。
func TestResolvedClientIDMetadataDocumentIsCachedRegardlessOfHTTPCacheHeaders(t *testing.T) {
	host := newMetadataHost(t, validDocument)
	// 採っていない側: 標準に従うなら、この応答は再利用してはならない。
	host.headers["Cache-Control"] = "no-store, no-cache, max-age=0"
	host.headers["Pragma"] = "no-cache"
	repository, _ := host.resolution()

	first, err := repository.FindByID(t.Context(), "default", host.clientIDURL())
	if err != nil || first == nil {
		t.Fatalf("1 回目の解決に失敗した: client=%#v err=%v", first, err)
	}
	second, err := repository.FindByID(t.Context(), "default", host.clientIDURL())
	if err != nil || second == nil {
		t.Fatalf("2 回目の解決に失敗した: client=%#v err=%v", second, err)
	}
	if got := host.requests.Load(); got != 1 {
		t.Fatalf("サーバーが受けた要求は %d 件、期待は 1 件 (2 回目はキャッシュから返る)", got)
	}
	if second != first {
		t.Errorf("2 回目が 1 回目と同じ文書から組み立てられていない")
	}

	// キャッシュの単位は client_id である。同じサーバーの別のパスは改めて取得する。
	// これが無いと、キャッシュではなく「2 回目を取得しない」実装と区別できない。
	if _, err := repository.FindByID(t.Context(), "default", host.server.URL+"/another.json"); err != nil {
		t.Fatalf("解決は「未知の client_id」へ畳まれるべきで、err ではない: %v", err)
	}
	if got := host.requests.Load(); got != 2 {
		t.Fatalf("サーバーが受けた要求は %d 件、期待は 2 件 (別の client_id は取得し直す)", got)
	}
}

// CIMD00-REDIRECT-VALIDATE:
// 認可リクエストの redirect_uri が、取得した文書の redirect_uris 一覧に
// 含まれることを検証する。
//
// この行だけは解決したクライアントを使う側の判断なので、Authorize から観測する。
// 拒否は「拒否した」だけでは足りない。認可リクエストが保存されていないことを
// 併せて読む。保存してから拒否する実装は、後続の経路へ材料を残す。
func TestAuthorizationRequestRedirectURIMustBeListedInTheFetchedDocument(t *testing.T) {
	host := newMetadataHost(t, validDocument)
	repository, _ := host.resolution()
	store := &countingRequestStore{AuthorizationRequestStore: authorizationmemory.NewAuthorizationRequestStore()}
	deps := authorizationusecases.AuthorizeDeps{ClientRepo: repository, RequestStore: store}

	request := func(redirectURI string) authorizationusecases.AuthorizeRequestInput {
		return authorizationusecases.AuthorizeRequestInput{
			ClientID: host.clientIDURL(), RedirectURI: redirectURI,
			ResponseType: "code", Scope: "openid",
			CodeChallenge: "challenge", CodeChallengeMethod: "S256",
		}
	}

	// 対照: 文書に載っている redirect_uri は通り、認可リクエストが保存される。
	out, err := authorizationusecases.Authorize(t.Context(), deps, request(documentRedirect))
	if err != nil {
		t.Fatalf("文書に載っている redirect_uri が拒否された: %v", err)
	}
	stored, err := store.Find(t.Context(), out.Request.ID)
	if err != nil || stored == nil {
		t.Fatalf("認可リクエストが保存されていない: %#v err=%v", stored, err)
	}
	if stored.RedirectURI != documentRedirect {
		t.Errorf("保存された RedirectURI = %q, want %q", stored.RedirectURI, documentRedirect)
	}

	for name, redirectURI := range map[string]string{
		"文書に無いホスト":  "https://attacker.example/cb",
		"文書に無いパス":   "https://app.example/other",
		"末尾スラッシュ違い": documentRedirect + "/",
		"クエリを足した":   documentRedirect + "?next=/",
	} {
		t.Run(name, func(t *testing.T) {
			out, err := authorizationusecases.Authorize(t.Context(), deps, request(redirectURI))
			if err == nil {
				t.Fatalf("文書の redirect_uris に無い %q が受理された: %#v", redirectURI, out)
			}
			var oauthErr *authorizationusecases.OAuthError
			if !errors.As(err, &oauthErr) || oauthErr.Code != "invalid_request" {
				t.Fatalf("拒否は invalid_request であるべき: %v", err)
			}
			// 拒否が防いだもの: 認可リクエストが 1 件も保存されていない。
			// 対照の 1 件から増えていなければ、後続の経路へ材料は残っていない。
			if got := store.saves.Load(); got != 1 {
				t.Fatalf("保存された認可リクエストが %d 件。拒否したリクエストは保存してはならない", got)
			}
		})
	}
}

// countingRequestStore は保存の回数を数える。拒否したリクエストが保存されて
// いないことを、保存そのものの回数で読む。
type countingRequestStore struct {
	*authorizationmemory.AuthorizationRequestStore
	saves atomic.Int64
}

func (s *countingRequestStore) Save(ctx context.Context, request *oauthdomain.AuthorizationRequest) error {
	s.saves.Add(1)
	return s.AuthorizationRequestStore.Save(ctx, request)
}
