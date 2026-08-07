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
	"net/url"
	"strings"
	"time"

	ssports "github.com/ambi/idmagic/backend/sharedsignals/ports"
)

// HTTPSecurityEventPusher POSTs the SET's compact serialization to the
// receiver's delivery_endpoint. It re-validates the target on every call
// (DNS-rebinding safe) and re-validates redirect targets, mirroring the
// established per-context precedent for outbound admin-configured URLs
// (backend/shared/security/tokens_jose.JWKResolver,
// backend/provisioning/client_scim's transport) rather than sharing one
// central client.
type HTTPSecurityEventPusher struct {
	// Client is exported so tests can inject httptest.Server.Client()
	// directly, bypassing the public-IP dial restriction against loopback
	// test servers (backend/provisioning/client_scim.Client.HTTPClient
	// precedent). NewHTTPSecurityEventPusher sets it to the SSRF-safe
	// client; production code should always go through that constructor.
	Client   *http.Client
	resolver *net.Resolver
}

// NewHTTPSecurityEventPusher builds the production pusher.
func NewHTTPSecurityEventPusher() *HTTPSecurityEventPusher {
	p := &HTTPSecurityEventPusher{resolver: net.DefaultResolver}
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, err
			}
			ips, err := p.safeIPs(ctx, host)
			if err != nil {
				return nil, err
			}
			return (&net.Dialer{Timeout: 3 * time.Second}).DialContext(ctx, network, net.JoinHostPort(ips[0].String(), port))
		},
		TLSHandshakeTimeout: 3 * time.Second,
	}
	p.Client = &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return errors.New("sharedsignals/push_http: too many redirects")
			}
			return validateDeliveryEndpoint(req.URL.String())
		},
	}
	return p
}

var _ ssports.SecurityEventPusher = (*HTTPSecurityEventPusher)(nil)

// Push POSTs compactSET to endpoint per SSF push-based delivery:
// Content-Type: application/secevent+jwt, body = the raw compact JWT.
// authorization, if non-empty, is sent as the Authorization header verbatim
// (SsfTransmitterConfig.DeliveryAuthorization, e.g. "Bearer <token>"). Any
// non-2xx response or transport failure is returned as an error; the caller
// (usecases.ProcessDueDeliveries) turns that into retry/backoff/dead-letter
// state.
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
		return fmt.Errorf("push security event token: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("push security event token: receiver returned status %d", resp.StatusCode)
	}
	return nil
}

func validateDeliveryEndpoint(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("delivery_endpoint: %w", err)
	}
	if parsed.Scheme != "https" {
		return errors.New("delivery_endpoint: https is required")
	}
	if parsed.Hostname() == "" || parsed.User != nil {
		return errors.New("delivery_endpoint: invalid authority or userinfo")
	}
	return nil
}

func (p *HTTPSecurityEventPusher) safeIPs(ctx context.Context, host string) ([]net.IP, error) {
	if ip := net.ParseIP(host); ip != nil {
		if !isPublicIP(ip) {
			return nil, errors.New("delivery_endpoint resolves to a non-public address")
		}
		return []net.IP{ip}, nil
	}
	addresses, err := p.resolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("resolve delivery_endpoint host: %w", err)
	}
	out := make([]net.IP, 0, len(addresses))
	for _, address := range addresses {
		if !isPublicIP(address.IP) {
			return nil, errors.New("delivery_endpoint resolves to a non-public address")
		}
		out = append(out, address.IP)
	}
	if len(out) == 0 {
		return nil, errors.New("delivery_endpoint host has no addresses")
	}
	return out, nil
}

func isPublicIP(ip net.IP) bool {
	return ip != nil &&
		!ip.IsPrivate() &&
		!ip.IsLoopback() &&
		!ip.IsLinkLocalUnicast() &&
		!ip.IsLinkLocalMulticast() &&
		!ip.IsUnspecified() &&
		!ip.IsMulticast() &&
		!strings.HasPrefix(ip.String(), "100.64.")
}
