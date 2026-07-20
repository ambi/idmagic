package client_scim

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/ambi/idmagic/backend/provisioning/domain"
)

// newSafeHTTPClient builds an http.Client whose DialContext re-resolves the
// target host and rejects non-public IPs at connect time (DNS-rebinding safe),
// and whose CheckRedirect re-validates each redirect target
// (backend/shared/security/tokens_jose.JWKResolver precedent, ADR-128 §配送・信頼性
// SSRF/leak 対策). Production code must go through NewClient, which uses this;
// tests inject a plain http.Client (e.g. httptest.Server.Client()) instead.
func newSafeHTTPClient() *http.Client {
	resolver := net.DefaultResolver
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, err
			}
			ips, err := safeIPsWithResolver(ctx, resolver, host)
			if err != nil {
				return nil, err
			}
			return (&net.Dialer{Timeout: 3 * time.Second}).DialContext(ctx, network, net.JoinHostPort(ips[0].String(), port))
		},
		TLSHandshakeTimeout: 3 * time.Second,
	}
	return &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return errors.New("provisioning/scim: too many redirects")
			}
			return domain.ValidateOutboundBaseURL(req.URL.Scheme + "://" + req.URL.Host)
		},
	}
}

// safeIPs resolves host and rejects it if any resolved address is not a public
// unicast address (loopback, private, link-local, CGNAT, multicast, unspecified).
func safeIPs(ctx context.Context, host string) ([]net.IP, error) {
	return safeIPsWithResolver(ctx, net.DefaultResolver, host)
}

func safeIPsWithResolver(ctx context.Context, resolver *net.Resolver, host string) ([]net.IP, error) {
	if ip := net.ParseIP(host); ip != nil {
		if !isPublicIP(ip) {
			return nil, fmt.Errorf("provisioning/scim: %s resolves to a non-public address", host)
		}
		return []net.IP{ip}, nil
	}
	addresses, err := resolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("provisioning/scim: resolve host: %w", err)
	}
	out := make([]net.IP, 0, len(addresses))
	for _, address := range addresses {
		if !isPublicIP(address.IP) {
			return nil, fmt.Errorf("provisioning/scim: %s resolves to a non-public address", host)
		}
		out = append(out, address.IP)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("provisioning/scim: host %s has no addresses", host)
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
