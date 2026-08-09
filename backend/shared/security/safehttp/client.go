// Package safehttp builds http.Client instances that only reach public IP
// addresses, protecting fetchers that dereference a client-supplied URL
// (JWKS jwks_uri, Client ID Metadata Documents) from SSRF into internal
// networks. Extracted from tokens_jose.JWKResolver so both fetchers share
// one hardened dialer (ADR-155).
package safehttp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"
)

var cgnatNetwork = &net.IPNet{
	IP:   net.IPv4(100, 64, 0, 0),
	Mask: net.CIDRMask(10, 32),
}

// IsPublicIP reports whether ip is routable on the public internet: not
// private, loopback, link-local, multicast, unspecified, or CGNAT
// (100.64.0.0/10).
func IsPublicIP(ip net.IP) bool {
	return ip != nil &&
		!ip.IsPrivate() &&
		!ip.IsLoopback() &&
		!ip.IsLinkLocalUnicast() &&
		!ip.IsLinkLocalMulticast() &&
		!ip.IsUnspecified() &&
		!ip.IsMulticast() &&
		!cgnatNetwork.Contains(ip)
}

// SafeIPs resolves host and returns its addresses, failing if host is
// itself a non-public IP literal or resolves to any non-public address.
// A nil resolver defaults to net.DefaultResolver.
func SafeIPs(ctx context.Context, resolver *net.Resolver, host string) ([]net.IP, error) {
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	if ip := net.ParseIP(host); ip != nil {
		if !IsPublicIP(ip) {
			return nil, errors.New("host resolves to a non-public address")
		}
		return []net.IP{ip}, nil
	}
	addresses, err := resolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("resolve host: %w", err)
	}
	out := make([]net.IP, 0, len(addresses))
	for _, address := range addresses {
		if !IsPublicIP(address.IP) {
			return nil, errors.New("host resolves to a non-public address")
		}
		out = append(out, address.IP)
	}
	if len(out) == 0 {
		return nil, errors.New("host has no addresses")
	}
	return out, nil
}

// Config controls the hardened client built by NewClient.
type Config struct {
	DialTimeout    time.Duration
	TLSTimeout     time.Duration
	RequestTimeout time.Duration
	MaxRedirects   int
	// ValidateURL re-checks scheme/authority on the initial request and on
	// every redirect hop (e.g. https-only, no userinfo, no fragment).
	ValidateURL func(rawURL string) error
}

// NewClient builds an *http.Client whose dialer resolves each host via
// SafeIPs before connecting — so a DNS answer or redirect pointing at a
// private/loopback/link-local address is rejected rather than followed —
// and whose CheckRedirect re-validates each hop with cfg.ValidateURL and
// caps the redirect count.
func NewClient(cfg Config) *http.Client {
	resolver := net.DefaultResolver
	transport := &http.Transport{
		// A proxy would resolve and connect to the final target outside this
		// transport's checked dial path, bypassing its SSRF boundary.
		Proxy: nil,
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, err
			}
			ips, err := SafeIPs(ctx, resolver, host)
			if err != nil {
				return nil, err
			}
			return (&net.Dialer{Timeout: cfg.DialTimeout}).DialContext(
				ctx, network, net.JoinHostPort(ips[0].String(), port),
			)
		},
		TLSHandshakeTimeout: cfg.TLSTimeout,
	}
	return &http.Client{
		Transport: transport,
		Timeout:   cfg.RequestTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= cfg.MaxRedirects {
				return errors.New("too many redirects")
			}
			return cfg.ValidateURL(req.URL.String())
		},
	}
}
