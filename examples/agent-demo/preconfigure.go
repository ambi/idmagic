package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

// 管理画面と同じ first-party クライアントと scope でログインする。値は管理画面の設定と揃える。
const (
	adminConsoleClientID = "00000000-0000-4000-8000-000000000022"
	adminConsoleScope    = "openid profile idmagic.admin"
	demoConnectionName   = "社内文書 MCP"
	demoAgentName        = "社内文書アシスタント"
	maxLoginRedirects    = 10
)

// preparedItem は、設定済みの開始点が作ったものと、その値の出所を画面へ伝える。
type preparedItem struct {
	Item   string `json:"item"`
	Value  string `json:"value"`
	Origin string `json:"origin"` // generated、fixed、reported、chosen
	Why    string `json:"why"`
	Path   string `json:"path,omitempty"`
}

// adminSession は管理者として通常のログインを経た Bearer トークンで管理 API を呼ぶ。
// 管理者が入力したパスワードはこの呼び出しの間だけ使い、保存しない。
type adminSession struct {
	realm  string
	origin string
	token  string
	http   *http.Client
}

func randomURLToken() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

// loginAsAdmin は管理画面と同じ認可コードフロー（PKCE）でログインし、アクセストークンを得る。
func loginAsAdmin(ctx context.Context, realm, username, password string) (*adminSession, error) {
	target, err := url.Parse(realm)
	if err != nil {
		return nil, err
	}
	origin := target.Scheme + "://" + target.Host
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	verifier, err := randomURLToken()
	if err != nil {
		return nil, err
	}
	state, err := randomURLToken()
	if err != nil {
		return nil, err
	}
	challenge := sha256.Sum256([]byte(verifier))
	redirectURI := realm + "/callback"
	authorize := realm + "/authorize?" + url.Values{
		"response_type": {"code"}, "client_id": {adminConsoleClientID}, "redirect_uri": {redirectURI},
		"scope": {adminConsoleScope}, "state": {state},
		"code_challenge": {base64.RawURLEncoding.EncodeToString(challenge[:])}, "code_challenge_method": {"S256"},
	}.Encode()
	if _, err := follow(ctx, client, authorize, redirectURI); err != nil && !errors.Is(err, errLoginRequired) {
		return nil, err
	}
	var transaction struct {
		CSRFToken string `json:"csrf_token"`
	}
	if err := getJSON(ctx, client, realm+"/api/auth/transaction", &transaction); err != nil || transaction.CSRFToken == "" {
		return nil, fmt.Errorf("ログインの準備ができません: %w", err)
	}
	body, err := json.Marshal(map[string]string{"username": username, "password": password})
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, realm+"/api/auth/login", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", origin)
	request.Header.Set("X-Csrf-Token", transaction.CSRFToken)
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	var flow struct {
		RedirectTo string `json:"redirect_to"`
		Next       string `json:"next"`
	}
	decodeErr := json.NewDecoder(io.LimitReader(response.Body, maxResponse)).Decode(&flow)
	_ = response.Body.Close()
	if response.StatusCode == http.StatusUnauthorized {
		return nil, errors.New("ユーザー名かパスワードが違います。")
	}
	if response.StatusCode != http.StatusOK || decodeErr != nil || flow.RedirectTo == "" {
		return nil, fmt.Errorf("ログインを完了できません（HTTP %d）。管理画面で通常どおりログインできるか確かめてください。", response.StatusCode)
	}
	next, err := url.Parse(flow.RedirectTo)
	if err != nil {
		return nil, err
	}
	code, err := follow(ctx, client, target.ResolveReference(next).String(), redirectURI)
	if err != nil {
		return nil, err
	}
	var token struct {
		AccessToken string `json:"access_token"`
	}
	form := url.Values{"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {redirectURI}, "client_id": {adminConsoleClientID}, "code_verifier": {verifier}}
	tokenRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, realm+"/token", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	tokenRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err := doJSON(client, tokenRequest, &token); err != nil || token.AccessToken == "" {
		return nil, fmt.Errorf("管理画面のトークンを取得できません: %w", err)
	}
	return &adminSession{realm: realm, origin: origin, token: token.AccessToken, http: &http.Client{Timeout: 15 * time.Second}}, nil
}

var errLoginRequired = errors.New("login required")

// follow はリダイレクトを手で追い、戻り先に届いたら認可コードを返す。ログイン画面に着いたら止まる。
func follow(ctx context.Context, client *http.Client, next, redirectURI string) (string, error) {
	for range maxLoginRedirects {
		if strings.HasPrefix(next, redirectURI) {
			location, err := url.Parse(next)
			if err != nil {
				return "", err
			}
			if code := location.Query().Get("code"); code != "" {
				return code, nil
			}
			return "", fmt.Errorf("認可が拒否されました: %s", location.Query().Get("error"))
		}
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, next, http.NoBody)
		if err != nil {
			return "", err
		}
		response, err := client.Do(request)
		if err != nil {
			return "", err
		}
		_ = response.Body.Close()
		location := response.Header.Get("Location")
		if location == "" {
			return "", errLoginRequired
		}
		base, err := url.Parse(next)
		if err != nil {
			return "", err
		}
		resolved, err := base.Parse(location)
		if err != nil {
			return "", err
		}
		next = resolved.String()
	}
	return "", errors.New("リダイレクトが多すぎます。")
}

func getJSON(ctx context.Context, client *http.Client, endpoint string, out any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return err
	}
	return doJSON(client, request, out)
}

func doJSON(client *http.Client, request *http.Request, out any) error {
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(response.Body, maxResponse))
	if err != nil {
		return err
	}
	if response.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(raw)))
	}
	if out == nil || len(raw) == 0 {
		return nil
	}
	return json.Unmarshal(raw, out)
}

// call は管理 API を Bearer トークンで呼ぶ。Bearer はブラウザーの資格情報ではないので CSRF の検査を受けない。
func (s *adminSession) call(ctx context.Context, method, path string, body, out any) error {
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(raw)
	}
	request, err := http.NewRequestWithContext(ctx, method, s.realm+path, reader)
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+s.token)
	request.Header.Set("Content-Type", "application/json")
	if err := doJSON(s.http, request, out); err != nil {
		return fmt.Errorf("%s %s: %w", method, path, err)
	}
	return nil
}

// publicJWK は管理 API へ渡す公開鍵の JWK である。秘密鍵の要素は含めない。
func (d *demo) publicJWK() map[string]any {
	encode := base64.RawURLEncoding.EncodeToString
	return map[string]any{
		"kty": "RSA", "alg": "PS256", "use": "sig",
		"n": encode(d.key.N.Bytes()), "e": encode(big.NewInt(int64(d.key.E)).Bytes()),
	}
}

// preconfigure は管理者が入力した資格情報でログインし、通常の管理と同じ管理 API で、接続先、評価用の
// クライアント、公開するツール、Agent、認証、閲覧の割当を作り、ゲートウェイを起動して接続する。
func (d *demo) preconfigure(ctx context.Context, username, password string) ([]preparedItem, []step, error) {
	d.mu.Lock()
	if d.prepared {
		d.mu.Unlock()
		return nil, nil, errors.New("このデモはすでに設定済みです。やり直すには mise run demo-agent を起動し直します。")
	}
	d.mu.Unlock()
	if strings.TrimSpace(username) == "" || password == "" {
		return nil, nil, errors.New("管理者のユーザー名とパスワードを入力してください。")
	}
	admin, err := loginAsAdmin(ctx, d.realm, strings.TrimSpace(username), password)
	if err != nil {
		return nil, nil, err
	}
	var application struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
	}
	if err := admin.call(ctx, http.MethodPost, "/api/admin/v1/applications", map[string]any{"name": demoConnectionName + " Gateway", "type": "service"}, &application); err != nil {
		return nil, nil, err
	}
	var server struct {
		ID string `json:"id"`
	}
	if err := admin.call(ctx, http.MethodPost, "/api/admin/v1/mcp-resource-servers", map[string]any{"resource": d.resource, "name": demoConnectionName, "client_id": application.ClientID}, &server); err != nil {
		return nil, nil, err
	}
	steps, err := d.startGateway(ctx, connectInput{Catalog: true, GatewayClientID: application.ClientID, GatewaySecret: application.ClientSecret})
	if err != nil {
		return nil, nil, err
	}
	tools := []map[string]any{}
	for _, tool := range demoTools {
		tools = append(tools, map[string]any{"resource_type": "idmagic_mcp_operation", "relation": "invoke", "scopes": []string{"mcp"}, "tool": map[string]any{"name": tool.name, "display_name": tool.title}})
	}
	var published struct {
		AccessRules []struct {
			Tool struct {
				Name                 string `json:"name"`
				AssignmentResourceID string `json:"assignment_resource_id"`
			} `json:"tool"`
		} `json:"access_rules"`
	}
	if err := admin.call(ctx, http.MethodPatch, "/api/admin/v1/mcp-resource-servers/"+server.ID, map[string]any{"name": demoConnectionName, "access_rules": tools}, &published); err != nil {
		return nil, nil, err
	}
	var agent struct {
		ID string `json:"id"`
	}
	if err := admin.call(ctx, http.MethodPost, "/api/admin/v1/agents", map[string]any{"name": demoAgentName, "kind": "autonomous"}, &agent); err != nil {
		return nil, nil, err
	}
	var client struct {
		Client struct {
			ClientID string `json:"client_id"`
		} `json:"client"`
	}
	if err := admin.call(ctx, http.MethodPost, "/api/admin/v1/clients", map[string]any{
		"client_name": demoAgentName, "client_type": "confidential", "redirect_uris": []string{},
		"grant_types":                []string{"client_credentials", "urn:openid:params:grant-type:ciba"},
		"token_endpoint_auth_method": "private_key_jwt", "scope": "openid profile mcp",
		"jwks": map[string]any{"keys": []any{d.publicJWK()}}, "fapi_profile": "none",
	}, &client); err != nil {
		return nil, nil, err
	}
	if err := admin.call(ctx, http.MethodPost, "/api/admin/v1/agents/"+agent.ID+"/credentials", map[string]string{"client_id": client.Client.ClientID}, nil); err != nil {
		return nil, nil, err
	}
	readAssignment := ""
	for _, rule := range published.AccessRules {
		if rule.Tool.Name == "read_document" {
			readAssignment = rule.Tool.AssignmentResourceID
		}
	}
	if err := admin.call(ctx, http.MethodPost, "/api/admin/v1/authorization/relation-tuples", map[string]any{"writes": []map[string]string{{
		"resource_type": "idmagic_mcp_operation", "resource_id": readAssignment, "relation": "invoke", "subject_type": "agent", "subject_id": agent.ID,
	}}}, nil); err != nil {
		return nil, nil, err
	}
	connected, err := d.connect(ctx, connectInput{Catalog: true, AgentClientID: client.Client.ClientID})
	if err != nil {
		return nil, nil, err
	}
	d.mu.Lock()
	d.prepared = true
	d.mu.Unlock()
	items := []preparedItem{
		{Item: "接続先", Value: demoConnectionName + "（" + d.resource + "）", Origin: "fixed", Why: "Agent が接続する Gateway の公開 URL。トークンの宛先になる", Path: "/admin/mcp-resource-servers"},
		{Item: "評価用のクライアント", Value: application.ClientID, Origin: "generated", Why: "Gateway が IdMagic へ判定を問い合わせ、ツールを報告する資格情報。登録と同時に作成した", Path: "/admin/applications"},
		{Item: "公開したツール", Value: "read_document、write_document、delete_document", Origin: "reported", Why: "Gateway が報告した一覧から公開した。名前と表示名は報告の値", Path: "/admin/mcp-resource-servers/" + url.PathEscape(server.ID) + "/edit"},
		{Item: "エージェント", Value: demoAgentName, Origin: "fixed", Why: "ログインした管理者が責任者になる", Path: "/admin/agents/" + url.PathEscape(agent.ID)},
		{Item: "認証用クライアント", Value: client.Client.ClientID, Origin: "generated", Why: "このデモの公開鍵で認証する。秘密鍵はデモのプロセスのメモリだけにある", Path: "/admin/agents/" + url.PathEscape(agent.ID) + "/credentials"},
		{Item: "許可した操作", Value: "文書の閲覧", Origin: "chosen", Why: "まず閲覧だけを許可した。更新と削除は第 3 節で一つずつ追加する", Path: "/admin/agents/" + url.PathEscape(agent.ID) + "/resources"},
	}
	return items, append(steps, connected...), nil
}
