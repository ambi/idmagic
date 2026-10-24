package domain

import (
	"bytes"
	"testing"
	"time"
	"unicode/utf8"
)

//spec:covers REQ-AUTHENTICATION-042, REQ-AUTHENTICATION-043: 不完全な信頼設定、曖昧な対応付けと不正な自動試行の条件を起動時に拒否する。
func TestConfigurationRejectsInvalidTrust(t *testing.T) {
	valid := func() Config {
		return Config{TenantID: "tenant", Realm: "EXAMPLE.COM", ServicePrincipal: "HTTP/sso.example.com", KeytabFile: "keytab", CertificateFile: "cert", Bindings: []Binding{{Principal: "alice", UserID: "id"}}}
	}
	if err := valid().Validate(); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"tenant", "tenant-space", "realm", "realm-format", "spn-prefix", "spn-host", "spn-path", "spn-space", "keytab", "cert", "cidr", "duplicate", "user", "machine", "principal"} {
		t.Run(name, func(t *testing.T) {
			config := valid()
			switch name {
			case "tenant":
				config.TenantID = ""
			case "tenant-space":
				config.TenantID = " tenant"
			case "realm":
				config.Realm = ""
			case "realm-format":
				config.Realm = "EXAMPLE@COM"
			case "spn-prefix":
				config.ServicePrincipal = "LDAP/sso.example.com"
			case "spn-host":
				config.ServicePrincipal = "HTTP/"
			case "spn-path":
				config.ServicePrincipal = "HTTP/host/path"
			case "spn-space":
				config.ServicePrincipal = "HTTP/host name"
			case "keytab":
				config.KeytabFile = ""
			case "cert":
				config.CertificateFile = ""
			case "cidr":
				config.IntranetCIDRs = []string{"invalid"}
			case "duplicate":
				config.Bindings = append(config.Bindings, config.Bindings[0])
			case "user":
				config.Bindings[0].UserID = ""
			case "machine":
				config.Bindings[0].Principal = "machine$"
			case "principal":
				config.Bindings[0].Principal = "alice/other"
			}
			if err := config.Validate(); err == nil {
				t.Fatal("invalid trust accepted")
			}
		})
	}
}

//spec:covers REQ-AUTHENTICATION-042: principal の完全一致と明示した対応だけを認め、名前とメールの推測をしない。
func TestMappingUsesExactRealmAndName(t *testing.T) {
	c := Config{Realm: "EXAMPLE.COM", Bindings: []Binding{{Principal: "alice", UserID: "local-id"}}}
	for _, p := range []Principal{{"example.com", "alice"}, {"OTHER.COM", "alice"}, {"EXAMPLE.COM", "Alice"}, {"EXAMPLE.COM", "alice@example.com"}, {"EXAMPLE.COM", "alice$"}, {"EXAMPLE.COM", "unknown"}} {
		if got := c.UserID(p); got != "" {
			t.Fatalf("principal=%+v mapped to %q", p, got)
		}
	}
	if got := c.UserID(Principal{"EXAMPLE.COM", "alice"}); got != "local-id" {
		t.Fatalf("id=%q", got)
	}
}

//spec:covers REQ-AUTHENTICATION-043: 出口 IP は自動試行の条件であり、未設定なら自動試行しない。
func TestAutomaticNetworkPolicyUsesAddresses(t *testing.T) {
	c := Config{IntranetCIDRs: []string{"203.0.113.0/24", "2001:db8::/32"}}
	for _, ip := range []string{"203.0.113.7", "::ffff:203.0.113.7", "2001:db8::1"} {
		if !c.AllowsIP(ip) {
			t.Fatalf("allowed IP %q refused", ip)
		}
	}
	for _, ip := range []string{"192.0.2.1", "invalid", "203.0.113.7:123"} {
		if c.AllowsIP(ip) {
			t.Fatalf("unlisted IP %q accepted", ip)
		}
	}
	if (Config{}).AllowsIP("203.0.113.7") {
		t.Fatal("missing allowlist enabled automatic authentication")
	}
}

//spec:covers REQ-AUTHENTICATION-043: 試行は署名、テナント、継続先と期限のすべてに束縛する。
func TestAttemptBindsTenantTargetAndExpiry(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	secret := bytes.Repeat([]byte{7}, 32)
	a := Attempt{TenantID: "tenant", Target: "target", Expires: now.Add(5 * time.Minute).Unix()}
	value := a.Sign(secret)
	if !ValidAttempt(value, secret, "tenant", "target", now) {
		t.Fatal("valid attempt refused")
	}
	for _, tc := range []struct {
		tenant, target string
		at             time.Time
	}{
		{"other", "target", now}, {"tenant", "other", now}, {"tenant", "target", now.Add(5 * time.Minute)}, {"tenant", "target", now.Add(-time.Second)},
	} {
		if ValidAttempt(value, secret, tc.tenant, tc.target, tc.at) {
			t.Fatalf("attempt accepted %+v", tc)
		}
	}
	if ValidAttempt(value, bytes.Repeat([]byte{8}, 32), "tenant", "target", now) {
		t.Fatal("wrong signing key accepted")
	}
	if ValidAttempt(value+"x", secret, "tenant", "target", now) {
		t.Fatal("tampering accepted")
	}
}

//spec:covers REQ-AUTHENTICATION-043: 任意の入力でパニックせず、署名した試行の往復と異なる継続先への拒否を保つ。
func FuzzAttempt(f *testing.F) {
	f.Add("tenant", "target", "invalid")
	f.Fuzz(func(t *testing.T, tenant, target, raw string) {
		if len(tenant)+len(target) > 500 || !utf8.ValidString(tenant) || !utf8.ValidString(target) {
			t.Skip()
		}
		now := time.Unix(1_800_000_000, 0)
		secret := bytes.Repeat([]byte{3}, 32)
		value := (Attempt{TenantID: tenant, Target: target, Expires: now.Unix() + 300}).Sign(secret)
		if !ValidAttempt(value, secret, tenant, target, now) {
			t.Fatal("signed attempt failed roundtrip")
		}
		if ValidAttempt(value, secret, tenant, target+"x", now) {
			t.Fatal("attempt crossed target")
		}
		_ = ValidAttempt(raw, secret, tenant, target, now)
	})
}
