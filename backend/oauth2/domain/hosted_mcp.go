package domain

import (
	"errors"
	"net/netip"
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/ambi/idmagic/backend/shared/security/safehttp"
)

// HostedMcpConfiguration は資源登録が所有するマネージド型の上流設定である。
// 資格情報は保持せず、発行元か Vault の接続を参照する。
type HostedMcpConfiguration struct {
	UpstreamURL    string   `json:"upstream_url"`
	Authentication string   `json:"authentication"`
	ConnectorID    string   `json:"connector_id,omitempty"`
	AllowedOrigins []string `json:"allowed_origins,omitempty"`
}

func (h HostedMcpConfiguration) Validate() error {
	if err := ValidateMcpUpstreamURL(h.UpstreamURL); err != nil {
		return err
	}
	switch h.Authentication {
	case "token_exchange":
		if h.ConnectorID != "" {
			return errors.New("connector_id is only valid for token_vault")
		}
	case "token_vault":
		if strings.TrimSpace(h.ConnectorID) == "" {
			return errors.New("connector_id is required for token_vault")
		}
	default:
		return errors.New("authentication must be token_exchange or token_vault")
	}
	for _, origin := range h.AllowedOrigins {
		u, err := url.Parse(origin)
		if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" {
			return errors.New("allowed_origins must contain HTTP origins without paths or credentials")
		}
	}
	return nil
}

// ValidateMcpUpstreamURL は DNS に触れず、解釈が一意な HTTPS 宛先だけを許す。
// ホスト名の全アドレスは、HTTP クライアントが接続する時点で別に検査する。
func ValidateMcpUpstreamURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Opaque != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || strings.ContainsAny(u.Hostname(), "%\\") {
		return errors.New("upstream_url must be an HTTPS URL without credentials, query, or fragment")
	}
	if ip, err := netip.ParseAddr(u.Hostname()); err == nil && !safehttp.IsPublicIP(ip.AsSlice()) {
		return errors.New("upstream_url must not use a non-public IP address")
	}
	if port := u.Port(); port != "" {
		n, err := strconv.Atoi(port)
		if err != nil || n < 1 || n > 65535 {
			return errors.New("upstream_url has an invalid port")
		}
	}
	if u.EscapedPath() != u.Path || strings.Contains(u.Path, "\\") || strings.Contains(u.Path, "//") || (u.Path != "" && u.Path != path.Clean(u.Path) && u.Path != path.Clean(u.Path)+"/") {
		return errors.New("upstream_url must not have an ambiguous path")
	}
	return nil
}
