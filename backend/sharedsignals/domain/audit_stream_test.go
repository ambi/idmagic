package domain

import (
	"encoding/json"
	"errors"
	"net/netip"
	"testing"
	"time"
)

//spec:covers REQ-SHAREDSIGNALS-017: https でない配送先、userinfo、非公開アドレスの字句を拒否する。
//spec:covers EX-SHAREDSIGNALS-017-01: メタデータの宛先を拒否する。
func TestValidateDeliveryEndpoint(t *testing.T) {
	cases := []struct {
		name, url string
		ok        bool
	}{
		{"public https", "https://hooks.example.com/in?x=1", true},
		{"http", "http://hooks.example.com/", false},
		{"no scheme", "hooks.example.com", false},
		{"userinfo", "https://user:pw@hooks.example.com/", false},
		{"metadata", "https://169.254.169.254/", false},
		{"loopback", "https://127.0.0.1/", false},
		{"ipv6 loopback", "https://[::1]/", false},
		{"mapped private", "https://[::ffff:10.0.0.1]/", false},
		{"localhost", "https://localhost/", false},
		{"localhost subdomain", "https://a.localhost./", false},
		{"too long", "https://a.example.com/" + string(make([]byte, 2048)), false},
		{"empty", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateDeliveryEndpoint(tc.url); (err == nil) != tc.ok {
				t.Fatalf("ValidateDeliveryEndpoint(%q) = %v, want ok=%v", tc.url, err, tc.ok)
			}
		})
	}
}

//spec:covers REQ-SHAREDSIGNALS-017: 公開アドレスの判定の両側。
func TestIsPublicAddress(t *testing.T) {
	public := []string{"8.8.8.8", "1.1.1.1", "100.63.255.255", "100.128.0.0", "2606:4700::1111"}
	blocked := []string{
		"0.0.0.1", "10.0.0.1", "100.64.0.1", "100.100.1.1", "100.127.255.255", "127.0.0.1", "169.254.169.254",
		"172.16.0.1", "192.0.0.1", "192.0.2.1", "192.168.1.1", "198.18.0.1", "198.51.100.1",
		"203.0.113.1", "224.0.0.1", "240.0.0.1", "255.255.255.255",
		"::", "::1", "::ffff:10.0.0.1", "64:ff9b::a00:1", "100::1", "2001:db8::1", "2002:a00:1::",
		"fc00::1", "fd00:ec2::254", "fe80::1", "ff02::1",
	}
	for _, s := range public {
		if !IsPublicAddress(netip.MustParseAddr(s)) {
			t.Errorf("%s should be public", s)
		}
	}
	for _, s := range blocked {
		if IsPublicAddress(netip.MustParseAddr(s)) {
			t.Errorf("%s should be blocked", s)
		}
	}
}

//spec:covers REQ-SHAREDSIGNALS-016: 本文は許可した識別子だけを含み、個人情報を含まない。
//spec:covers EX-SHAREDSIGNALS-016-01: 個人情報を含めない。
func TestNewAuditSetEventAllowlist(t *testing.T) {
	at := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	event := NewAuditSetEvent("e1", "UserUpdated", "t1", at, map[string]any{
		"userId": "u1", "targetUserId": "u2", "username": "alice", "email": "a@example.com", "ipAddress": "1.2.3.4", "accessToken": "secret",
	})
	raw, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"event_id":"e1","event_type":"UserUpdated","tenant_id":"t1","occurred_at":"2026-01-02T03:04:05Z","actor":{"type":"user","id":"u1"},"target_id":"u2"}`
	if string(raw) != want {
		t.Fatalf("event = %s", raw)
	}
}

//spec:covers REQ-SHAREDSIGNALS-016: sub_id は対象、行為者、テナントの順に選ぶ。
func TestAuditSubjectID(t *testing.T) {
	for want, event := range map[string]AuditSetEvent{
		"target": {TenantID: "tenant", Actor: &AuditSetActor{ID: "actor"}, TargetID: "target"},
		"actor":  {TenantID: "tenant", Actor: &AuditSetActor{ID: "actor"}},
		"tenant": {TenantID: "tenant"},
	} {
		if got := event.SubjectID(); got["id"] != want || got["format"] != "opaque" {
			t.Errorf("SubjectID = %v, want %s", got, want)
		}
	}
}

//spec:covers REQ-SHAREDSIGNALS-015: 配送の結果のイベントは種類の指定にも購読にも入らない。
//spec:covers EX-SHAREDSIGNALS-015-02: 配送の結果のイベントは配送を生まない。
func TestAuditTypeSubscription(t *testing.T) {
	stream := SsfStream{AuditEventTypes: []string{AuditEventWildcard}}
	if !stream.SubscribesAudit("UserCreated") || stream.SubscribesAudit("SecurityEventDeliveryFailed") {
		t.Fatal("wildcard must match all but delivery outcomes")
	}
	got, err := NormalizeAuditEventTypes([]string{"UserCreated", "UserCreated", "LoginFailed"})
	if err != nil || len(got) != 2 {
		t.Fatalf("got %v err %v", got, err)
	}
	for _, bad := range [][]string{{"*", "UserCreated"}, {"SecurityEventTransmitted"}, {"lower"}, {""}} {
		if _, err := NormalizeAuditEventTypes(bad); !errors.Is(err, ErrAuditEventTypesInvalid) {
			t.Errorf("%v should be rejected: %v", bad, err)
		}
	}
}
