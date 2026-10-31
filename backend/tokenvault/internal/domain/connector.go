// Package domain は TokenVault のコネクタ、接続、代理実行の判定を持つ。
package domain

import (
	"errors"
	"fmt"
	"net/netip"
	"net/url"
	"slices"
	"strings"
	"time"
)

// ConnectorStatus はコネクタの状態である。
type ConnectorStatus string

const (
	ConnectorActive   ConnectorStatus = "active"
	ConnectorDisabled ConnectorStatus = "disabled"
)

// TokenEndpointAuthMethod は上流のトークンエンドポイントでのクライアント認証である。
type TokenEndpointAuthMethod string

const (
	AuthMethodClientSecretBasic TokenEndpointAuthMethod = "client_secret_basic"
	AuthMethodClientSecretPost  TokenEndpointAuthMethod = "client_secret_post"
)

const maxExecutionRules = 100

var (
	// ErrInvalidConnector はコネクタの定義が REQ-TOKENVAULT-001 の検証を通らないことを表す。
	ErrInvalidConnector = errors.New("invalid connector")
	allowedMethods      = []string{"GET", "HEAD", "POST", "PUT", "PATCH", "DELETE"}
)

// ExecutionRule はコネクタが代理実行を許す要求の組である。
type ExecutionRule struct {
	ID             string   `json:"id"`
	Method         string   `json:"method"`
	Origin         string   `json:"origin"`
	PathPrefix     string   `json:"path_prefix"`
	RequiredScopes []string `json:"required_scopes"`
}

// Connector は管理者が定義する外部サービスの設定である。
type Connector struct {
	ID                      string
	TenantID                string
	Name                    string
	Status                  ConnectorStatus
	AuthorizationEndpoint   string
	TokenEndpoint           string
	RevocationEndpoint      string
	ClientID                string
	TokenEndpointAuthMethod TokenEndpointAuthMethod
	ClientSecret            string
	AllowedScopes           []string
	Rules                   []ExecutionRule
	Version                 int64
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

// Validate は REQ-TOKENVAULT-001 の定義の検証を行う。
func (c Connector) Validate() error {
	if strings.TrimSpace(c.Name) == "" || c.ClientID == "" || c.ClientSecret == "" {
		return fmt.Errorf("%w: name, client_id and client_secret are required", ErrInvalidConnector)
	}
	if c.Status != ConnectorActive && c.Status != ConnectorDisabled {
		return fmt.Errorf("%w: unknown status", ErrInvalidConnector)
	}
	if c.TokenEndpointAuthMethod != AuthMethodClientSecretBasic && c.TokenEndpointAuthMethod != AuthMethodClientSecretPost {
		return fmt.Errorf("%w: unknown token_endpoint_auth_method", ErrInvalidConnector)
	}
	endpoints := []string{c.AuthorizationEndpoint, c.TokenEndpoint}
	if c.RevocationEndpoint != "" {
		endpoints = append(endpoints, c.RevocationEndpoint)
	}
	for _, endpoint := range endpoints {
		if _, err := parseOutboundURL(endpoint); err != nil {
			return fmt.Errorf("%w: endpoint: %w", ErrInvalidConnector, err)
		}
	}
	if len(c.Rules) > maxExecutionRules {
		return fmt.Errorf("%w: rules must not exceed %d", ErrInvalidConnector, maxExecutionRules)
	}
	return c.validateRules()
}

func (c Connector) validateRules() error {
	seen := make(map[string]struct{}, len(c.Rules))
	for _, rule := range c.Rules {
		if rule.ID == "" {
			return fmt.Errorf("%w: rule id is required", ErrInvalidConnector)
		}
		if _, duplicate := seen[rule.ID]; duplicate {
			return fmt.Errorf("%w: rule id must be unique", ErrInvalidConnector)
		}
		seen[rule.ID] = struct{}{}
		if !slices.Contains(allowedMethods, rule.Method) {
			return fmt.Errorf("%w: unsupported rule method", ErrInvalidConnector)
		}
		if !strings.HasPrefix(rule.PathPrefix, "/") || strings.Contains(rule.PathPrefix, "..") {
			return fmt.Errorf("%w: invalid rule path_prefix", ErrInvalidConnector)
		}
		origin, err := parseOutboundURL(rule.Origin)
		if err != nil || origin.Path != "" || origin.RawQuery != "" || Origin(origin) != rule.Origin {
			return fmt.Errorf("%w: rule origin must be a normalized https origin", ErrInvalidConnector)
		}
		for _, scope := range rule.RequiredScopes {
			if !slices.Contains(c.AllowedScopes, scope) {
				return fmt.Errorf("%w: required_scopes must be a subset of allowed_scopes", ErrInvalidConnector)
			}
		}
	}
	return nil
}

// parseOutboundURL は外向きの URL の静的な条件を確かめる。実際の接続先の検査は外向きの
// クライアントが接続の時点で行う (REQ-TOKENVAULT-010)。
func parseOutboundURL(raw string) (*url.URL, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, errors.New("not a URL")
	}
	if parsed.Scheme != "https" || parsed.Hostname() == "" {
		return nil, errors.New("must be an absolute https URL")
	}
	if parsed.User != nil || parsed.Fragment != "" || strings.HasSuffix(raw, "#") {
		return nil, errors.New("must not contain user information or a fragment")
	}
	if addr, err := netip.ParseAddr(parsed.Hostname()); err == nil && !isPublicLiteral(addr.Unmap()) {
		return nil, errors.New("must not name an internal address")
	}
	if host := strings.ToLower(parsed.Hostname()); host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return nil, errors.New("must not name an internal address")
	}
	return parsed, nil
}

func isPublicLiteral(addr netip.Addr) bool {
	return !addr.IsPrivate() && !addr.IsLoopback() && !addr.IsLinkLocalUnicast() && !addr.IsLinkLocalMulticast() &&
		!addr.IsUnspecified() && !addr.IsMulticast()
}

// Origin は URL の正規化したオリジンを返す。https の既定のポートは省く。
func Origin(u *url.URL) string {
	host := strings.ToLower(u.Hostname())
	if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	if port := u.Port(); port != "" && port != "443" {
		return "https://" + host + ":" + port
	}
	return "https://" + host
}

// MatchRule は要求に一致する最初の規則を返す。パスの接頭辞はセグメントの境界で照合する。
func (c Connector) MatchRule(method string, target *url.URL) (ExecutionRule, bool) {
	origin := Origin(target)
	for _, rule := range c.Rules {
		if rule.Method == method && rule.Origin == origin && pathHasPrefix(target.Path, rule.PathPrefix) {
			return rule, true
		}
	}
	return ExecutionRule{}, false
}

func pathHasPrefix(path, prefix string) bool {
	if path == prefix || prefix == "/" {
		return true
	}
	return strings.HasPrefix(path, strings.TrimSuffix(prefix, "/")+"/")
}
