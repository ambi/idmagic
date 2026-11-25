package testing_oauthtwin

import (
	"encoding/json"
	"maps"
	"mime"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"sync"
	"testing"
)

// Twin は一つのクライアントを登録した OAuth 2.0 のトークンエンドポイントの模倣である。
//
// 記録にない操作の組み合わせ（更新トークンの回転の後の旧トークン、使用済みの認可コード）、
// 拒否、429 と 5xx、不正な本文、`expires_in` の欠落と任意の値を作るために使う。
// 応答の形は記録と一致することをこのパッケージのテストが確かめる。PKCE の検証はしない。
type Twin struct {
	mu            sync.Mutex
	clientID      string
	clientSecret  string
	codes         map[string]string // 認可コード → 登録したリダイレクト URI
	refreshTokens map[string]bool   // 有効な更新トークン
	issued        int
	faults        []Fault

	// TokenType は発行する応答の token_type である。既定は Bearer である。
	TokenType string
	// ExpiresIn は発行する応答の expires_in である。既定は 3600 である。
	ExpiresIn int64
	// OmitExpiresIn が真なら、発行する応答に expires_in を含めない（RFC 6749 §5.1 は推奨に留める）。
	OmitExpiresIn bool
	// RotateRefreshTokens が真なら、更新のたびに新しい更新トークンを返し、使った更新トークンを無効にする。
	// 偽なら更新の応答に refresh_token を含めず、使った更新トークンを有効なまま残す。既定は真である。
	RotateRefreshTokens bool
	// Extra は発行する応答へ加える拡張の欄である。
	Extra map[string]any
}

// Fault は次の要求に返す障害である。要求の内容に依らず返し、状態を変えない。
type Fault struct {
	// Status は返す状態コードである。0 ならその要求には障害を返さず、通常どおり処理する。
	Status int
	// ContentType は応答の Content-Type である。空なら載せない。
	ContentType string
	// Body は応答の本文である。
	Body string
	// RetryAfter は Retry-After ヘッダーの値である。空なら載せない。
	RetryAfter string
}

// MalformedBody は、JSON として解釈できない本文の 200 である。
var MalformedBody = Fault{Status: http.StatusOK, ContentType: "application/json", Body: `{"access_token":`}

// NewTwin は一つのクライアントを登録した模倣を作る。
func NewTwin(clientID, clientSecret string) *Twin {
	return &Twin{
		clientID: clientID, clientSecret: clientSecret,
		codes: map[string]string{}, refreshTokens: map[string]bool{},
		TokenType: "Bearer", ExpiresIn: 3600, RotateRefreshTokens: true,
	}
}

// Start は模倣を httptest.Server として起動する。トークンエンドポイントは `<URL>/token` である。
func (tw *Twin) Start(tb testing.TB) *httptest.Server {
	tb.Helper()
	server := httptest.NewServer(tw)
	tb.Cleanup(server.Close)
	return server
}

// AddAuthorizationCode は一度だけ交換できる認可コードを置く。redirectURI が空でなければ、交換の要求に同じ値を求める。
func (tw *Twin) AddAuthorizationCode(code, redirectURI string) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	tw.codes[code] = redirectURI
}

// AddRefreshToken は有効な更新トークンを置く。
func (tw *Twin) AddRefreshToken(token string) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	tw.refreshTokens[token] = true
}

// InjectFaults は次の要求から順に返す障害を積む。
func (tw *Twin) InjectFaults(faults ...Fault) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	tw.faults = append(tw.faults, faults...)
}

func (tw *Twin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if len(tw.faults) > 0 {
		fault := tw.faults[0]
		tw.faults = tw.faults[1:]
		if fault.Status != 0 {
			writeFault(w, fault)
			return
		}
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err != nil || mediaType != "application/x-www-form-urlencoded" {
		writeError(w, http.StatusBadRequest, "invalid_request")
		return
	}
	if err := r.ParseForm(); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request")
		return
	}
	form := r.PostForm
	if status, code := tw.authenticate(r, form); code != "" {
		if status == http.StatusUnauthorized {
			// RFC 6749 §5.2: Authorization で認証を試みたクライアントには 401 と WWW-Authenticate を返す。
			w.Header().Set("WWW-Authenticate", `Basic realm="token"`)
		}
		writeError(w, status, code)
		return
	}
	switch form.Get("grant_type") {
	case "":
		writeError(w, http.StatusBadRequest, "invalid_request")
	case "authorization_code":
		tw.exchangeCode(w, form)
	case "refresh_token":
		tw.refresh(w, form)
	case "client_credentials":
		tw.issue(w, false)
	default:
		writeError(w, http.StatusBadRequest, "unsupported_grant_type")
	}
}

// authenticate は RFC 6749 §2.3.1 のクライアント認証を確かめ、拒否なら状態コードとエラーを返す。
// Basic の資格情報は form-urlencoded を戻してから比べる。二つの方式を同時に使う要求は拒否する。
func (tw *Twin) authenticate(r *http.Request, form url.Values) (int, string) {
	user, password, basic := r.BasicAuth()
	_, inBody := form["client_secret"]
	switch {
	case basic && inBody:
		return http.StatusBadRequest, "invalid_request"
	case basic:
		id, errID := url.QueryUnescape(user)
		secret, errSecret := url.QueryUnescape(password)
		if errID != nil || errSecret != nil || id != tw.clientID || secret != tw.clientSecret {
			return http.StatusUnauthorized, "invalid_client"
		}
	case inBody:
		if form.Get("client_id") != tw.clientID || form.Get("client_secret") != tw.clientSecret {
			return http.StatusUnauthorized, "invalid_client"
		}
	default:
		return http.StatusUnauthorized, "invalid_client"
	}
	return 0, ""
}

func (tw *Twin) exchangeCode(w http.ResponseWriter, form url.Values) {
	code := form.Get("code")
	if code == "" {
		writeError(w, http.StatusBadRequest, "invalid_request")
		return
	}
	redirectURI, known := tw.codes[code]
	if !known || (redirectURI != "" && form.Get("redirect_uri") != redirectURI) {
		writeError(w, http.StatusBadRequest, "invalid_grant")
		return
	}
	delete(tw.codes, code)
	tw.issue(w, true)
}

func (tw *Twin) refresh(w http.ResponseWriter, form url.Values) {
	token := form.Get("refresh_token")
	if token == "" {
		writeError(w, http.StatusBadRequest, "invalid_request")
		return
	}
	if !tw.refreshTokens[token] {
		writeError(w, http.StatusBadRequest, "invalid_grant")
		return
	}
	if tw.RotateRefreshTokens {
		delete(tw.refreshTokens, token)
	}
	tw.issue(w, tw.RotateRefreshTokens)
}

// issue はアクセストークンを、withRefresh が真なら新しい更新トークンも発行する。
func (tw *Twin) issue(w http.ResponseWriter, withRefresh bool) {
	tw.issued++
	body := map[string]any{}
	maps.Copy(body, tw.Extra)
	body["access_token"] = "twin-access-" + strconv.Itoa(tw.issued)
	body["token_type"] = tw.TokenType
	if !tw.OmitExpiresIn {
		body["expires_in"] = tw.ExpiresIn
	}
	if withRefresh {
		refresh := "twin-refresh-" + strconv.Itoa(tw.issued)
		tw.refreshTokens[refresh] = true
		body["refresh_token"] = refresh
	}
	writeJSON(w, http.StatusOK, body)
}

func writeError(w http.ResponseWriter, status int, code string) {
	writeJSON(w, status, map[string]any{"error": code})
}

// writeJSON は RFC 6749 §5.1 と §5.2 の例と同じヘッダーで本文を書く。
func writeJSON(w http.ResponseWriter, status int, body map[string]any) {
	data, err := json.Marshal(body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.WriteHeader(status)
	_, _ = w.Write(data)
}

func writeFault(w http.ResponseWriter, fault Fault) {
	if fault.ContentType != "" {
		w.Header().Set("Content-Type", fault.ContentType)
	}
	if fault.RetryAfter != "" {
		w.Header().Set("Retry-After", fault.RetryAfter)
	}
	w.WriteHeader(fault.Status)
	_, _ = w.Write([]byte(fault.Body))
}
