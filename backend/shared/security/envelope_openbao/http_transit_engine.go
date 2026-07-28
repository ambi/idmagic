// Package envelope_openbao: Layer 4 - Adapter Layer (technical shared adapter)
//
// OpenBaoMasterKeyProvider is the production envelope_crypto.MasterKeyProvider
// (ADR-148): it wraps per-tenant DEKs through OpenBao's Vault
// Transit-compatible HTTP API, one transit key per tenant. It mirrors the
// backend/signingkeys/keys_vault HTTP client pattern (token auth, timeout,
// no hand-rolled retry/backoff — a single request per call).
package envelope_openbao

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// HTTPTransitEngine talks to OpenBao's Transit secrets engine (Vault Transit
// wire-compatible): create-if-missing key, encrypt, decrypt, health.
type HTTPTransitEngine struct {
	Address string // e.g. https://openbao.internal:8200
	Token   string // X-Vault-Token (OpenBao keeps the Vault header name)
	Mount   string // transit mount path (default "transit")
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
	status, _, err := e.do(ctx, http.MethodGet, "/v1/"+e.Mount+"/keys/"+name, nil)
	if err != nil {
		return err
	}
	if status == http.StatusOK {
		return nil
	}
	if status != http.StatusNotFound {
		return fmt.Errorf("openbao: unexpected status %d reading key", status)
	}
	_, _, err = e.do(ctx, http.MethodPost, "/v1/"+e.Mount+"/keys/"+name,
		map[string]any{"type": "aes256-gcm96"})
	return err
}

func (e *HTTPTransitEngine) EncryptDataKey(ctx context.Context, name string, plaintext []byte) (string, error) {
	status, body, err := e.do(ctx, http.MethodPost, "/v1/"+e.Mount+"/encrypt/"+name, map[string]any{
		"plaintext": base64.StdEncoding.EncodeToString(plaintext),
	})
	if err != nil {
		return "", err
	}
	if status != http.StatusOK {
		return "", fmt.Errorf("openbao: unexpected status %d encrypting", status)
	}
	var parsed struct {
		Data struct {
			Ciphertext string `json:"ciphertext"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("openbao: decode encrypt response: %w", err)
	}
	if parsed.Data.Ciphertext == "" {
		return "", errors.New("openbao: empty ciphertext in encrypt response")
	}
	return parsed.Data.Ciphertext, nil
}

func (e *HTTPTransitEngine) DecryptDataKey(ctx context.Context, name, ciphertext string) ([]byte, error) {
	status, body, err := e.do(ctx, http.MethodPost, "/v1/"+e.Mount+"/decrypt/"+name, map[string]any{
		"ciphertext": ciphertext,
	})
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("openbao: unexpected status %d decrypting", status)
	}
	var parsed struct {
		Data struct {
			Plaintext string `json:"plaintext"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("openbao: decode decrypt response: %w", err)
	}
	plaintext, err := base64.StdEncoding.DecodeString(parsed.Data.Plaintext)
	if err != nil {
		return nil, fmt.Errorf("openbao: decode decrypt plaintext: %w", err)
	}
	return plaintext, nil
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
