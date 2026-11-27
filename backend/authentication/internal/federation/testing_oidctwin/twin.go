package testing_oidctwin

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"maps"
	"math/big"
	"mime"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/shared/security/tokens_jose"
	"github.com/ambi/idmagic/backend/shared/testing_httprecord"
)

// 模倣が受け付けるクライアント認証の方式である（OpenID Connect Core 1.0 §9）。
const (
	ClientSecretBasic = "client_secret_basic"
	ClientSecretPost  = "client_secret_post"
)

// Twin は一つのクライアントを登録した OpenID Connect の OP の模倣である。
//
// Discovery（`/.well-known/openid-configuration`）、認可エンドポイント（`/authorize`）、
// トークンエンドポイント（`/token`）、JWK Set（`/jwks.json`）を issuer の下に持つ。
// 記録にない組み合わせ（PKCE の照合、使用済みの認可コード、鍵の回転、`kid` の欠落、回転に遅れた JWK Set）と、
// 経路ごとの 5xx や不正な本文を作るために使う。応答の形は記録と一致することをこのパッケージのテストが確かめる。
type Twin struct {
	mu           sync.Mutex
	issuer       string
	clientID     string
	clientSecret string
	codes        map[string]AuthorizationCode
	keys         []signingKey // 公開する鍵。最後の鍵で署名する
	staleKeys    []signingKey // 直前の回転の前に公開していた鍵
	issued       int
	faults       map[string][]Fault

	// AuthMethods はトークンエンドポイントが受け付けるクライアント認証の方式であり、Discovery でも示す。
	// デフォルトは client_secret_basic と client_secret_post である。
	AuthMethods []string
	// SignedInSubject と SignedInClaims は、認可エンドポイントでログインした利用者の sub と追加のクレームである。
	SignedInSubject string
	SignedInClaims  map[string]any
	// OmitKid が真なら、ID トークンの JOSE ヘッダーと JWK Set の鍵に kid を載せない（Core §10.1 は一つの鍵なら省略を許す）。
	OmitKid bool
	// StaleJWKS が真なら、JWK Set は直前の回転の前の鍵を配信する。回転した鍵の公開が署名に遅れる相手を表す。
	StaleJWKS bool
	// Now は ID トークンの iat と exp の基準である。nil なら現在時刻である。
	Now func() time.Time
}

// AuthorizationCode は一度だけ交換できる認可コードと、それに結び付いた認可の要求である。
type AuthorizationCode struct {
	Code string
	// RedirectURI が空でなければ、交換の要求に同じ値を求める。
	RedirectURI string
	// CodeChallenge が空でなければ、交換の要求の code_verifier の S256 が一致することを求める（RFC 7636 §4.6）。
	CodeChallenge string
	Subject       string
	Nonce         string
	Claims        map[string]any
}

// Fault は、その経路の次の要求に返す障害である。要求の内容に依らず返し、状態を変えない。
type Fault struct {
	// Status は返す状態コードである。0 ならその要求には障害を返さず、通常どおり処理する。
	Status int
	// ContentType は応答の Content-Type である。空なら載せない。
	ContentType string
	// Body は応答の本文である。
	Body string
}

// MalformedJSON は、JSON として解釈できない本文の 200 である。
var MalformedJSON = Fault{Status: http.StatusOK, ContentType: "application/json", Body: `{"keys":`}

type signingKey struct {
	kid string
	key *rsa.PrivateKey
}

// NewTwin は、issuer の下に一つのクライアントと一つの署名鍵を持つ模倣を作る。
func NewTwin(tb testing.TB, issuer, clientID, clientSecret string) *Twin {
	tb.Helper()
	tw := &Twin{
		issuer: strings.TrimRight(issuer, "/"), clientID: clientID, clientSecret: clientSecret,
		codes: map[string]AuthorizationCode{}, faults: map[string][]Fault{},
		AuthMethods:     []string{ClientSecretBasic, ClientSecretPost},
		SignedInSubject: "twin-subject",
	}
	tw.keys = []signingKey{tw.newKey(tb)}
	return tw
}

// Client は模倣を httptest.Server として起動し、どのホストへの要求も模倣へ送る HTTP クライアントを返す。
// 製品のクライアントへは issuer の URL をそのまま渡せる。リダイレクトは追わない。
func (tw *Twin) Client(tb testing.TB) *http.Client {
	tb.Helper()
	server := httptest.NewServer(tw)
	tb.Cleanup(server.Close)
	return testing_httprecord.RoutedClient(server)
}

// AddAuthorizationCode は一度だけ交換できる認可コードを置く。
func (tw *Twin) AddAuthorizationCode(code AuthorizationCode) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	tw.codes[code.Code] = code
}

// RotateSigningKey は新しい鍵で署名を始める。それまでの鍵は JWK Set に残す。
func (tw *Twin) RotateSigningKey(tb testing.TB) {
	tb.Helper()
	key := tw.newKey(tb)
	tw.mu.Lock()
	defer tw.mu.Unlock()
	tw.staleKeys = slices.Clone(tw.keys)
	tw.keys = append(tw.keys, key)
}

// InjectFaults は、path（`/token` など）の次の要求から順に返す障害を積む。
func (tw *Twin) InjectFaults(path string, faults ...Fault) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	tw.faults[path] = append(tw.faults[path], faults...)
}

func (tw *Twin) newKey(tb testing.TB) signingKey {
	tb.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		tb.Fatalf("generate twin signing key: %v", err)
	}
	tw.mu.Lock()
	defer tw.mu.Unlock()
	tw.issued++
	return signingKey{kid: "twin-key-" + strconv.Itoa(tw.issued), key: key}
}

func (tw *Twin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if queued := tw.faults[r.URL.Path]; len(queued) > 0 {
		tw.faults[r.URL.Path] = queued[1:]
		if queued[0].Status != 0 {
			writeFault(w, queued[0])
			return
		}
	}
	switch r.URL.Path {
	case "/.well-known/openid-configuration":
		tw.discovery(w, r)
	case "/authorize":
		tw.authorize(w, r)
	case "/token":
		tw.token(w, r)
	case "/jwks.json":
		tw.jwks(w, r)
	default:
		http.NotFound(w, r)
	}
}

// discovery は Discovery §3 の必須のメタデータと、クライアント認証の方式を返す。
func (tw *Twin) discovery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"issuer":                                tw.issuer,
		"authorization_endpoint":                tw.issuer + "/authorize",
		"token_endpoint":                        tw.issuer + "/token",
		"jwks_uri":                              tw.issuer + "/jwks.json",
		"token_endpoint_auth_methods_supported": tw.AuthMethods,
		"response_types_supported":              []string{"code"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
	}, false)
}

// authorize は利用者が同意したものとして認可コードを発行し、Core §3.1.2.5 の形で redirect_uri へ戻す。
// 要求の誤りは、リダイレクト先を信頼できないので 400 で返す。
func (tw *Twin) authorize(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	redirectURI := query.Get("redirect_uri")
	scopes := strings.Fields(query.Get("scope"))
	challengeMethod := query.Get("code_challenge_method")
	if query.Get("response_type") != "code" || query.Get("client_id") != tw.clientID || redirectURI == "" ||
		!slices.Contains(scopes, "openid") || (query.Get("code_challenge") != "" && challengeMethod != "S256") {
		writeError(w, http.StatusBadRequest, "invalid_request", "")
		return
	}
	tw.issued++
	code := AuthorizationCode{
		Code: "twin-code-" + strconv.Itoa(tw.issued), RedirectURI: redirectURI,
		CodeChallenge: query.Get("code_challenge"), Subject: tw.SignedInSubject,
		Nonce: query.Get("nonce"), Claims: maps.Clone(tw.SignedInClaims),
	}
	tw.codes[code.Code] = code
	values := url.Values{"code": {code.Code}}
	if state := query.Get("state"); state != "" {
		values.Set("state", state)
	}
	w.Header().Set("Location", redirectURI+"?"+values.Encode())
	w.WriteHeader(http.StatusFound)
}

func (tw *Twin) token(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err != nil || mediaType != "application/x-www-form-urlencoded" {
		writeError(w, http.StatusBadRequest, "invalid_request", "")
		return
	}
	if err := r.ParseForm(); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "")
		return
	}
	form := r.PostForm
	if status, code, challenge := tw.authenticate(r, form); code != "" {
		writeError(w, status, code, challenge)
		return
	}
	switch form.Get("grant_type") {
	case "":
		writeError(w, http.StatusBadRequest, "invalid_request", "")
	case "authorization_code":
		tw.exchangeCode(w, form)
	default:
		writeError(w, http.StatusBadRequest, "unsupported_grant_type", "")
	}
}

// authenticate は、AuthMethods の方式でクライアントを認証し、拒否なら状態コード、エラー、WWW-Authenticate を返す。
// Basic の資格情報は form-urlencoded を戻してから比べる（RFC 6749 §2.3.1）。二つの方式を同時に使う要求は拒否する。
func (tw *Twin) authenticate(r *http.Request, form url.Values) (int, string, string) {
	user, password, basic := r.BasicAuth()
	_, inBody := form["client_secret"]
	switch {
	case basic && inBody:
		return http.StatusBadRequest, "invalid_request", ""
	case basic && slices.Contains(tw.AuthMethods, ClientSecretBasic):
		id, errID := url.QueryUnescape(user)
		secret, errSecret := url.QueryUnescape(password)
		if errID != nil || errSecret != nil || id != tw.clientID || secret != tw.clientSecret {
			return http.StatusUnauthorized, "invalid_client", `Basic realm="token"`
		}
	case inBody && slices.Contains(tw.AuthMethods, ClientSecretPost):
		if form.Get("client_id") != tw.clientID || form.Get("client_secret") != tw.clientSecret {
			return http.StatusUnauthorized, "invalid_client", ""
		}
	default:
		// 受け付けない方式か、認証のない要求である。RFC 6749 §5.2 の invalid_client にする。
		return http.StatusUnauthorized, "invalid_client", `Basic realm="token"`
	}
	return 0, "", ""
}

func (tw *Twin) exchangeCode(w http.ResponseWriter, form url.Values) {
	value := form.Get("code")
	if value == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "")
		return
	}
	code, known := tw.codes[value]
	if !known || (code.RedirectURI != "" && form.Get("redirect_uri") != code.RedirectURI) || !pkceMatches(code.CodeChallenge, form.Get("code_verifier")) {
		writeError(w, http.StatusBadRequest, "invalid_grant", "")
		return
	}
	delete(tw.codes, value)
	idToken, err := tw.signIDToken(code)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tw.issued++
	writeJSON(w, http.StatusOK, map[string]any{
		"access_token":  "twin-access-" + strconv.Itoa(tw.issued),
		"token_type":    "Bearer",
		"refresh_token": "twin-refresh-" + strconv.Itoa(tw.issued),
		"expires_in":    3600,
		"id_token":      idToken,
	}, true)
}

func pkceMatches(challenge, verifier string) bool {
	if challenge == "" {
		return true
	}
	sum := sha256.Sum256([]byte(verifier))
	return subtle.ConstantTimeCompare([]byte(base64.RawURLEncoding.EncodeToString(sum[:])), []byte(challenge)) == 1
}

// signIDToken は Core §2 の必須のクレームを持つ ID トークンを、最後の鍵の RS256 で署名する。
func (tw *Twin) signIDToken(code AuthorizationCode) (string, error) {
	now := time.Now().UTC()
	if tw.Now != nil {
		now = tw.Now().UTC()
	}
	claims := maps.Clone(code.Claims)
	if claims == nil {
		claims = map[string]any{}
	}
	claims["iss"] = tw.issuer
	claims["sub"] = code.Subject
	claims["aud"] = tw.clientID
	claims["iat"] = now.Unix()
	claims["exp"] = now.Add(5 * time.Minute).Unix()
	if code.Nonce != "" {
		claims["nonce"] = code.Nonce
	}
	key := tw.keys[len(tw.keys)-1]
	header := map[string]any{"typ": "JWT"}
	if !tw.OmitKid {
		header["kid"] = key.kid
	}
	return tokens_jose.SignJWT(key.key, "RS256", header, claims)
}

// jwks は公開する鍵を RFC 7517 §5 の JWK Set で返す。
func (tw *Twin) jwks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	published := tw.keys
	if tw.StaleJWKS && tw.staleKeys != nil {
		published = tw.staleKeys
	}
	keys := make([]map[string]any, 0, len(published))
	for _, key := range published {
		jwk := map[string]any{
			"kty": "RSA", "use": "sig", "alg": "RS256",
			"n": base64.RawURLEncoding.EncodeToString(key.key.N.Bytes()),
			"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.key.E)).Bytes()),
		}
		if !tw.OmitKid {
			jwk["kid"] = key.kid
		}
		keys = append(keys, jwk)
	}
	writeJSON(w, http.StatusOK, map[string]any{"keys": keys}, false)
}

func writeError(w http.ResponseWriter, status int, code, challenge string) {
	if challenge != "" {
		w.Header().Set("WWW-Authenticate", challenge)
	}
	writeJSON(w, status, map[string]any{"error": code}, true)
}

// writeJSON は本文を application/json で書く。noStore が真なら、Core §3.1.3.3 と §3.1.3.4 のとおり Cache-Control: no-store を付ける。
func writeJSON(w http.ResponseWriter, status int, body map[string]any, noStore bool) {
	data, err := json.Marshal(body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if noStore {
		w.Header().Set("Cache-Control", "no-store")
	}
	w.WriteHeader(status)
	_, _ = w.Write(data)
}

func writeFault(w http.ResponseWriter, fault Fault) {
	if fault.ContentType != "" {
		w.Header().Set("Content-Type", fault.ContentType)
	}
	w.WriteHeader(fault.Status)
	_, _ = w.Write([]byte(fault.Body))
}
