package push_http

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	clientdomain "github.com/ambi/idmagic/backend/oauth2/client/domain"
	"github.com/ambi/idmagic/backend/shared/security/safehttp"
)

type Client struct{ httpClient *http.Client }

func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = safehttp.NewClient(safehttp.Config{
			DialTimeout: 2 * time.Second, TLSTimeout: 2 * time.Second,
			RequestTimeout: 10 * time.Second, MaxRedirects: 3,
			ValidateURL: func(raw string) error {
				if !clientdomain.ValidateLogoutURI(raw) {
					return fmt.Errorf("back-channel logout: unsafe target URI")
				}
				return nil
			},
		})
	}
	return &Client{httpClient: httpClient}
}

func NewBackChannelLogoutClient(httpClient *http.Client) *Client { return NewClient(httpClient) }

func (c *Client) Deliver(ctx context.Context, targetURI, logoutToken string) error {
	if !clientdomain.ValidateLogoutURI(targetURI) {
		return fmt.Errorf("back-channel logout: unsafe target URI")
	}
	body := url.Values{"logout_token": []string{logoutToken}}.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURI, strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("back-channel logout: create request: %w", err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("back-channel logout: deliver: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("back-channel logout: unexpected status %d", response.StatusCode)
	}
	return nil
}
