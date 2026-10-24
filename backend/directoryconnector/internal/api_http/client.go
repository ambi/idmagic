// Package api_http は Sourcing の Connector API（/directory-connector/v1）の HTTP クライアントである。
// 要求ごとに Connector の秘密鍵で新しい JWT を署名する (REQ-DIRECTORYCONNECTOR-005)。
package api_http

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/ambi/idmagic/backend/directoryconnector/internal/domain"
	"github.com/ambi/idmagic/backend/directoryconnector/internal/ports"
)

// tokenLifetime は JWT の exp までの長さである。サーバーは 5 分より先の exp を拒否する。
const tokenLifetime = 2 * time.Minute

const problemTypePrefix = "urn:idmagic:error:"

type endpoint struct {
	base string
	http *http.Client
}

func newEndpoint(realmURL string, client *http.Client) endpoint {
	return endpoint{base: strings.TrimSuffix(realmURL, "/") + "/directory-connector/v1", http: client}
}

// call は body を JSON で送り、2xx なら応答を into へ読む。2xx 以外は ports.APIError を返す。
func (e endpoint) call(ctx context.Context, path, bearer string, body, into any) error {
	var payload io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		payload = bytes.NewReader(raw)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, e.base+path, payload)
	if err != nil {
		return err
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if bearer != "" {
		request.Header.Set("Authorization", "Bearer "+bearer)
	}
	response, err := e.http.Do(request)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < 200 || response.StatusCode > 299 {
		var problem struct {
			Type string `json:"type"`
		}
		_ = json.NewDecoder(io.LimitReader(response.Body, 64<<10)).Decode(&problem)
		return &ports.APIError{Status: response.StatusCode, Type: strings.TrimPrefix(problem.Type, problemTypePrefix)}
	}
	if into == nil {
		return nil
	}
	if err := json.NewDecoder(response.Body).Decode(into); err != nil {
		return fmt.Errorf("decode %s response: %w", path, err)
	}
	return nil
}

// EnrollmentClient は登録だけを行う。登録の前には Connector の鍵がないので JWT を使わない。
type EnrollmentClient struct{ endpoint endpoint }

func NewEnrollmentClient(realmURL string, client *http.Client) *EnrollmentClient {
	return &EnrollmentClient{endpoint: newEndpoint(realmURL, client)}
}

func (c *EnrollmentClient) Enroll(ctx context.Context, token string, jwk map[string]any) (ports.Enrollment, error) {
	var enrollment ports.Enrollment
	err := c.endpoint.call(ctx, "/enrollment", "", map[string]any{"enrollment_token": token, "jwk": jwk}, &enrollment)
	return enrollment, err
}

// Client は登録した鍵で署名した JWT で Connector API を呼ぶ。
type Client struct {
	endpoint   endpoint
	key        *ecdsa.PrivateKey
	enrollment ports.Enrollment
	now        func() time.Time
}

func NewClient(realmURL string, client *http.Client, key *ecdsa.PrivateKey, enrollment ports.Enrollment, now func() time.Time) *Client {
	return &Client{endpoint: newEndpoint(realmURL, client), key: key, enrollment: enrollment, now: now}
}

func (c *Client) assertion() (string, error) {
	now := c.now()
	token := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.RegisteredClaims{
		Issuer: c.enrollment.ConnectorID, Subject: c.enrollment.ConnectorID,
		Audience: jwt.ClaimStrings{c.enrollment.Audience}, ID: uuid.NewString(),
		IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(tokenLifetime)),
	})
	token.Header["kid"] = c.enrollment.KeyID
	return token.SignedString(c.key)
}

func (c *Client) signedCall(ctx context.Context, path string, body, into any) error {
	assertion, err := c.assertion()
	if err != nil {
		return fmt.Errorf("sign connector jwt: %w", err)
	}
	return c.endpoint.call(ctx, path, assertion, body, into)
}

func (c *Client) Heartbeat(ctx context.Context) error {
	return c.signedCall(ctx, "/heartbeat", nil, nil)
}

func (c *Client) StartRun(ctx context.Context, mode domain.Mode) (string, error) {
	var run struct {
		ID string `json:"id"`
	}
	if err := c.signedCall(ctx, "/ingestion-runs", map[string]any{"mode": mode}, &run); err != nil {
		return "", err
	}
	return run.ID, nil
}

func (c *Client) SendBatch(ctx context.Context, runID, batchID string, items []domain.Item) error {
	return c.signedCall(ctx, "/ingestion-runs/"+url.PathEscape(runID)+"/user-batches",
		map[string]any{"batch_id": batchID, "items": items}, nil)
}

func (c *Client) SendComputerBatch(ctx context.Context, runID, batchID string, items []domain.ComputerItem) error {
	return c.signedCall(ctx, "/ingestion-runs/"+url.PathEscape(runID)+"/computer-batches",
		map[string]any{"batch_id": batchID, "items": items}, nil)
}

func (c *Client) CompleteRun(ctx context.Context, runID string, itemCount int) error {
	return c.signedCall(ctx, "/ingestion-runs/"+url.PathEscape(runID)+"/complete", map[string]any{"item_count": itemCount}, nil)
}
