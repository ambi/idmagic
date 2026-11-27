package keys_vault

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// HTTPTransitEngine は Vault Transit secrets engine の HTTP API 実装。
// 秘密鍵は Vault 内に留まり、署名は transit/sign 経由で行う。
type HTTPTransitEngine struct {
	Address string // 例: https://vault.internal:8200
	Token   string // X-Vault-Token
	Mount   string // transit mount path (既定 "transit")
	Client  *http.Client
}

func NewHTTPTransitEngine(address, token, mount string) *HTTPTransitEngine {
	if mount == "" {
		mount = "transit"
	}
	return &HTTPTransitEngine{
		Address: strings.TrimSuffix(address, "/"),
		Token:   token,
		Mount:   mount,
		Client:  &http.Client{Timeout: 5 * time.Second},
	}
}

func (e *HTTPTransitEngine) EnsureKey(ctx context.Context, name string) error {
	// 既に存在すれば何もしない。
	status, _, err := e.do(ctx, http.MethodGet, "/v1/"+e.Mount+"/keys/"+name, nil)
	if err != nil {
		return err
	}
	if status == http.StatusOK {
		return nil
	}
	if status != http.StatusNotFound {
		return fmt.Errorf("vault: unexpected status %d reading key", status)
	}
	status, _, err = e.do(ctx, http.MethodPost, "/v1/"+e.Mount+"/keys/"+name,
		map[string]any{"type": "rsa-2048"})
	if err != nil {
		return err
	}
	if !succeeded(status) {
		return fmt.Errorf("vault: unexpected status %d creating key", status)
	}
	return nil
}

func (e *HTTPTransitEngine) RotateKey(ctx context.Context, name string) error {
	status, _, err := e.do(ctx, http.MethodPost, "/v1/"+e.Mount+"/keys/"+name+"/rotate", nil)
	if err != nil {
		return err
	}
	if !succeeded(status) {
		return fmt.Errorf("vault: unexpected status %d rotating key", status)
	}
	return nil
}

// succeeded は、本文の要らない書き込みの成功を表す状態コードかを返す。
// Vault 2.2 の鍵の作成と回転は鍵の情報を載せた 200 を返す。以前の版が返す本文のない 204 も成功とする。
func succeeded(status int) bool {
	return status == http.StatusOK || status == http.StatusNoContent
}

func (e *HTTPTransitEngine) LatestPublicKey(ctx context.Context, name string) (string, int, error) {
	status, body, err := e.do(ctx, http.MethodGet, "/v1/"+e.Mount+"/keys/"+name, nil)
	if err != nil {
		return "", 0, err
	}
	if status != http.StatusOK {
		return "", 0, fmt.Errorf("vault: unexpected status %d reading key", status)
	}
	var parsed struct {
		Data struct {
			LatestVersion int `json:"latest_version"`
			Keys          map[string]struct {
				PublicKey string `json:"public_key"`
			} `json:"keys"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", 0, err
	}
	version := parsed.Data.LatestVersion
	entry, ok := parsed.Data.Keys[fmt.Sprintf("%d", version)]
	if !ok || entry.PublicKey == "" {
		return "", 0, errors.New("vault: latest version has no public key")
	}
	return entry.PublicKey, version, nil
}

func (e *HTTPTransitEngine) Sign(ctx context.Context, name string, version int, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	payload := map[string]any{
		"input":               base64.StdEncoding.EncodeToString(digest),
		"prehashed":           true,
		"signature_algorithm": "pkcs1v15",
		"hash_algorithm":      "sha2-256",
		"key_version":         version,
	}
	if pss, ok := opts.(*rsa.PSSOptions); ok {
		payload["signature_algorithm"] = "pss"
		payload["salt_length"] = pssSaltLength(pss.SaltLength)
	}
	status, body, err := e.do(ctx, http.MethodPost, "/v1/"+e.Mount+"/sign/"+name, payload)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("vault: unexpected status %d signing", status)
	}
	var parsed struct {
		Data struct {
			Signature string `json:"signature"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}
	// Vault の signature は "vault:vN:<base64>" 形式。最後の segment を復号する。
	parts := strings.Split(parsed.Data.Signature, ":")
	if len(parts) != 3 {
		return nil, errors.New("vault: malformed signature")
	}
	return base64.StdEncoding.DecodeString(parts[2])
}

// pssSaltLength は、呼び出し元の PSS の salt 長を Transit の salt_length の値へ写す。
//
// salt_length を省くと Vault は既定の auto で署名し、auto は署名で最大長の salt を使う。
// RFC 7518 §3.5 の PS256 はハッシュ長の salt を求めるので、署名者の指定をそのまま送る。
func pssSaltLength(saltLength int) string {
	switch saltLength {
	case rsa.PSSSaltLengthEqualsHash:
		return "hash"
	case rsa.PSSSaltLengthAuto:
		return "auto"
	default:
		return strconv.Itoa(saltLength)
	}
}

func (e *HTTPTransitEngine) Healthy(ctx context.Context) bool {
	status, _, err := e.do(ctx, http.MethodGet, "/v1/sys/health", nil)
	return err == nil && status == http.StatusOK
}

func (e *HTTPTransitEngine) do(ctx context.Context, method, path string, payload any) (int, []byte, error) {
	var reader io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return 0, nil, err
		}
		reader = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, e.Address+path, reader)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("X-Vault-Token", e.Token)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := e.Client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return 0, nil, err
	}
	return resp.StatusCode, body, nil
}
