package domain

import (
	"testing"
	"time"
)

//spec:covers REQ-DIRECTORYCONNECTOR-007: 配送と復号後の束縛を全て検査し、小さな時計の遅れで正しい要求を拒否しない。
func TestPasswordPayloadRequiresEveryBindingAndALiveDeadline(t *testing.T) {
	now := time.Now().UTC()
	c := PasswordChallenge{ID: "request", TenantID: "tenant", ConnectorID: "connector", KeyID: "key", ExpiresAt: now.Add(5 * time.Second)}
	p := PasswordPayload{ID: c.ID, TenantID: c.TenantID, ConnectorID: c.ConnectorID, KeyID: c.KeyID, ExpiresAt: c.ExpiresAt, Operation: "verify_password", ExternalID: "identity", Password: "credential"}
	if !p.Matches(c, c.ConnectorID, c.KeyID, now.Add(-100*time.Millisecond)) {
		t.Fatal("a small clock skew rejected a live challenge")
	}
	for _, edit := range []func(*PasswordPayload){func(p *PasswordPayload) { p.ID = "other" }, func(p *PasswordPayload) { p.TenantID = "other" }, func(p *PasswordPayload) { p.ConnectorID = "other" }, func(p *PasswordPayload) { p.KeyID = "other" }, func(p *PasswordPayload) { p.ExpiresAt = p.ExpiresAt.Add(time.Second) }, func(p *PasswordPayload) { p.Operation = "write_password" }, func(p *PasswordPayload) { p.Password = "" }, func(p *PasswordPayload) { p.ExternalID = "" }} {
		changed := p
		edit(&changed)
		if changed.Matches(c, c.ConnectorID, c.KeyID, now) {
			t.Fatal("mismatched payload accepted")
		}
	}
	if p.Matches(c, c.ConnectorID, c.KeyID, c.ExpiresAt) {
		t.Fatal("expired challenge accepted")
	}
	if p.Matches(c, c.ConnectorID, c.KeyID, now.Add(-2*time.Minute)) {
		t.Fatal("unbounded future challenge accepted")
	}
}
