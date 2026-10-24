package domain

import (
	"fmt"
	"net/netip"
	"strings"
)

// Principal は検証済みの KDC の名前であり、メールアドレスとは別の識別子である。
type Principal struct {
	Realm string
	Name  string
}

type Binding struct {
	Principal string `json:"principal"`
	UserID    string `json:"user_id"`
}

// Config の対応付けは管理者の確認を信頼し、利用者の属性から推測しない。
type Config struct {
	TenantID         string    `json:"tenant_id"`
	Realm            string    `json:"realm"`
	ServicePrincipal string    `json:"service_principal"`
	KeytabFile       string    `json:"keytab_file"`
	CertificateFile  string    `json:"certificate_file"`
	IntranetCIDRs    []string  `json:"intranet_cidrs"`
	Bindings         []Binding `json:"bindings"`
}

func (c Config) Validate() error {
	if c.TenantID == "" || strings.TrimSpace(c.TenantID) != c.TenantID ||
		c.Realm == "" || strings.ContainsAny(c.Realm, " /@\t\r\n") ||
		!strings.HasPrefix(c.ServicePrincipal, "HTTP/") ||
		strings.Count(c.ServicePrincipal, "/") != 1 || len(c.ServicePrincipal) <= 5 ||
		strings.ContainsAny(c.ServicePrincipal, " @\t\r\n") || c.KeytabFile == "" || c.CertificateFile == "" {
		return fmt.Errorf("negotiate: tenant, realm, HTTP SPN, keytab and certificate are required")
	}
	for _, value := range c.IntranetCIDRs {
		if _, err := netip.ParsePrefix(value); err != nil {
			return fmt.Errorf("negotiate: invalid intranet CIDR")
		}
	}
	seen := make(map[string]bool)
	for _, binding := range c.Bindings {
		if !ValidName(binding.Principal) || binding.UserID == "" || seen[binding.Principal] {
			return fmt.Errorf("negotiate: invalid or duplicate principal binding")
		}
		seen[binding.Principal] = true
	}
	return nil
}

func ValidName(name string) bool {
	return name != "" && !strings.ContainsAny(name, " /@\\\t\r\n\x00") && !strings.HasSuffix(name, "$")
}

func (c Config) AllowsIP(value string) bool {
	address, err := netip.ParseAddr(value)
	if err != nil {
		return false
	}
	for _, cidr := range c.IntranetCIDRs {
		prefix, err := netip.ParsePrefix(cidr)
		if err == nil && prefix.Contains(address.Unmap()) {
			return true
		}
	}
	return false
}

func (c Config) UserID(principal Principal) string {
	if principal.Realm != c.Realm || !ValidName(principal.Name) {
		return ""
	}
	for _, binding := range c.Bindings {
		if binding.Principal == principal.Name {
			return binding.UserID
		}
	}
	return ""
}
