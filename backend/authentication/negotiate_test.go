package authentication

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/ambi/idmagic/backend/authentication/internal/negotiate/config_file"
	fixture "github.com/ambi/idmagic/backend/authentication/testing_kerberos"
	"github.com/ambi/idmagic/backend/idmanagement"
)

//spec:covers REQ-AUTHENTICATION-042, REQ-AUTHENTICATION-043: 未設定時は配線を作らず、有効な信頼は診断でき、同じサービスを複数テナントへ割り当てない。
func TestConfigureNegotiateValidatesOptionalTrust(t *testing.T) {
	users := idmanagement.NewMemoryModule().UserRepo
	m := NewMemoryModule(users)
	if fingerprint, err := m.ConfigureNegotiate("", users); err != nil || fingerprint != "" || m.NegotiateSignIn(RouteDeps{}) != nil {
		t.Fatal("unconfigured Kerberos was enabled")
	}
	f := fixture.New(t)
	path := f.WriteConfiguration(t, "tenant", false)
	if fingerprint, err := m.ConfigureNegotiate(path, users); err != nil || len(fingerprint) != 64 {
		t.Fatalf("trust was refused: %v", err)
	}
	withoutUsers := NewMemoryModule(users)
	if _, err := withoutUsers.ConfigureNegotiate(path, nil); err == nil {
		t.Fatal("missing user repository accepted")
	}
	encoded, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var config config_file.Configuration
	if err := json.Unmarshal(encoded, &config); err != nil {
		t.Fatal(err)
	}
	other := config.Tenants[0]
	other.TenantID = "other"
	config.Tenants = append(config.Tenants, other)
	encoded, err = json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, encoded, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := m.ConfigureNegotiate(path, users); err == nil {
		t.Fatal("a Kerberos service was shared across tenants")
	}
	if _, err := m.ConfigureNegotiate(path+".missing", users); err == nil {
		t.Fatal("missing trust file accepted")
	}
}

//spec:covers REQ-AUTHENTICATION-042: 有効化した認証の HTTP にはセッション、イベントと IP の解決を必須とし、配線漏れを起動時に拒否する。
func TestConfiguredNegotiateRequiresHTTPDependencies(t *testing.T) {
	users := idmanagement.NewMemoryModule().UserRepo
	m := NewMemoryModule(users)
	if _, err := m.ConfigureNegotiate(fixture.New(t).WriteConfiguration(t, "tenant", false), users); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if recover() == nil {
			t.Fatal("incomplete HTTP dependencies accepted")
		}
	}()
	m.NegotiateSignIn(RouteDeps{})
}
