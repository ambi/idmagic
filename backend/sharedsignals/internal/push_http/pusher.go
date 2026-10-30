// Package push_http implements ports.SecurityEventPusher: SSF push-based
// delivery of a signed Security Event Token to an external receiver's
// admin-configured delivery_endpoint (RFC 8417 / SSF push-based delivery).
package push_http

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"

	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
	ssports "github.com/ambi/idmagic/backend/sharedsignals/ports"
)

var errNonPublicDestination = errors.New("delivery_endpoint does not resolve to a public address")

// HTTPSecurityEventPusher POSTs the SET's compact serialization to the
// receiver's delivery_endpoint. 接続のたびに解決先を検査し、検査したアドレスへだけ接続する
// (REQ-SHAREDSIGNALS-017)。環境のプロキシを使うと検査がプロキシへ向かい、接続の再利用は次の要求の
// 解決先を検査しないので、どちらも使わない。リダイレクトには従わない。
type HTTPSecurityEventPusher struct {
	// Client is exported so tests can inject httptest.Server.Client()
	// directly, bypassing the public-IP dial restriction against loopback
	// test servers (backend/provisioning/client_scim.Client.HTTPClient
	// precedent). NewHTTPSecurityEventPusher sets it to the SSRF-safe
	// client; production code should always go through that constructor.
	Client *http.Client
	lookup func(ctx context.Context, host string) ([]netip.Addr, error)
}

// NewHTTPSecurityEventPusher builds the production pusher.
func NewHTTPSecurityEventPusher() *HTTPSecurityEventPusher {
	return newPusher(func(ctx context.Context, host string) ([]netip.Addr, error) {
		return net.DefaultResolver.LookupNetIP(ctx, "ip", host)
	})
}

func newPusher(lookup func(ctx context.Context, host string) ([]netip.Addr, error)) *HTTPSecurityEventPusher {
	p := &HTTPSecurityEventPusher{lookup: lookup}
	p.Client = &http.Client{
		Transport: &http.Transport{
			Proxy:               nil,
			DialContext:         p.dial,
			TLSHandshakeTimeout: 3 * time.Second,
			DisableKeepAlives:   true,
		},
		Timeout: 10 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	return p
}

var _ ssports.SecurityEventPusher = (*HTTPSecurityEventPusher)(nil)

func (p *HTTPSecurityEventPusher) dial(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	addrs, err := p.safeAddrs(ctx, host)
	if err != nil {
		return nil, err
	}
	dialer := &net.Dialer{Timeout: 3 * time.Second}
	var lastErr error
	for _, addr := range addrs {
		conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(addr.String(), port))
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

// safeAddrs は解決先を返す。一つでも非公開なら拒否し、公開と非公開を混ぜた応答で検査を抜けさせない。
func (p *HTTPSecurityEventPusher) safeAddrs(ctx context.Context, host string) ([]netip.Addr, error) {
	if addr, err := netip.ParseAddr(host); err == nil {
		if !ssdomain.IsPublicAddress(addr) {
			return nil, errNonPublicDestination
		}
		return []netip.Addr{addr}, nil
	}
	addrs, err := p.lookup(ctx, host)
	if err != nil {
		return nil, errors.New("delivery_endpoint could not be resolved")
	}
	if len(addrs) == 0 {
		return nil, errNonPublicDestination
	}
	for _, addr := range addrs {
		if !ssdomain.IsPublicAddress(addr) {
			return nil, errNonPublicDestination
		}
	}
	return addrs, nil
}

// CheckDestination は配送先の字句と解決先を検査する。登録時に使う。
func (p *HTTPSecurityEventPusher) CheckDestination(ctx context.Context, endpoint string) error {
	if err := ssdomain.ValidateDeliveryEndpoint(endpoint); err != nil {
		return err
	}
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return ssdomain.ErrDeliveryEndpointNotAllowed
	}
	if _, err := p.safeAddrs(ctx, parsed.Hostname()); err != nil {
		return fmt.Errorf("%w: %s", ssdomain.ErrDeliveryEndpointNotAllowed, err.Error())
	}
	return nil
}

// Push POSTs compactSET to endpoint per SSF push-based delivery:
// Content-Type: application/secevent+jwt, body = the raw compact JWT.
// authorization, if non-empty, is sent as the Authorization header verbatim
// (SsfTransmitterConfig.DeliveryAuthorization, e.g. "Bearer <token>"). Any
// non-2xx response or transport failure is returned as an error; the caller
// (usecases.ProcessDueDeliveries) turns that into retry/backoff/dead-letter
// state. エラーは配送の last_error として管理者へ見えるので、内部のアドレスを含めない。
func (p *HTTPSecurityEventPusher) Push(ctx context.Context, endpoint, authorization, compactSET string) error {
	if err := validateDeliveryEndpoint(endpoint); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(compactSET))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/secevent+jwt")
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	resp, err := p.Client.Do(req)
	if err != nil {
		return fmt.Errorf("push security event token: %w", sanitizeTransportError(err))
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("push security event token: receiver returned status %d", resp.StatusCode)
	}
	return nil
}

func sanitizeTransportError(err error) error {
	if errors.Is(err, errNonPublicDestination) {
		return errNonPublicDestination
	}
	var timeout interface{ Timeout() bool }
	if errors.As(err, &timeout) && timeout.Timeout() {
		return errors.New("the receiver timed out")
	}
	return errors.New("the receiver could not be reached")
}

func validateDeliveryEndpoint(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return errors.New("delivery_endpoint: the URL is invalid")
	}
	if parsed.Scheme != "https" {
		return errors.New("delivery_endpoint: https is required")
	}
	if parsed.Hostname() == "" || parsed.User != nil {
		return errors.New("delivery_endpoint: invalid authority or userinfo")
	}
	return nil
}
