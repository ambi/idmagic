package api_http

import (
	"context"
	"net/http"
	"net/url"

	"github.com/ambi/idmagic/backend/directoryconnector/internal/domain"
)

func (c *Client) RegisterEncryption(ctx context.Context, jwk map[string]any) (string, error) {
	assertion, err := c.assertion()
	if err != nil {
		return "", err
	}
	var response struct {
		KeyID string `json:"key_id"`
	}
	err = c.endpoint.callMethod(ctx, http.MethodPut, "/encryption-capability", assertion, map[string]any{"jwk": jwk}, &response)
	return response.KeyID, err
}

func (c *Client) PollPassword(ctx context.Context) (*domain.PasswordChallenge, error) {
	var challenge domain.PasswordChallenge
	err := c.signedCall(ctx, "/password-challenges", nil, &challenge)
	if err != nil {
		return nil, err
	}
	if challenge.ID == "" {
		return nil, nil //nolint:nilnil // 204 は仕事がない正常な応答である。
	}
	return &challenge, nil
}

func (c *Client) CompletePassword(ctx context.Context, id, result string) error {
	return c.signedCall(ctx, "/password-challenges/"+url.PathEscape(id)+"/result", map[string]string{"result": result}, nil)
}
