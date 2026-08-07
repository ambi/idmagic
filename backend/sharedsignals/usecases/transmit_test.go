package usecases_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
	"github.com/ambi/idmagic/backend/sharedsignals/sign_jose"
	ssusecases "github.com/ambi/idmagic/backend/sharedsignals/usecases"
	signingmemory "github.com/ambi/idmagic/backend/signingkeys/keys_memory"
)

// TestBuildAndSignSecurityEventToken — RED: RFC 8417 の claims (iss/jti/iat/aud/
// events) を組み立て、SigningKeys の既存鍵で PS256 署名した compact JWT を返す
// (ADR-057 決定3/7、既存の JWT 署名鍵管理を再利用する)。
func TestBuildAndSignSecurityEventToken(t *testing.T) {
	ctx := context.Background()
	keyStore, err := signingmemory.NewInMemoryKeyStore()
	if err != nil {
		t.Fatalf("NewInMemoryKeyStore: %v", err)
	}
	signer := &sign_jose.Signer{KeyStore: keyStore}
	reason := ssdomain.RevocationReasonAgentKilled
	event := ssdomain.CaepEvent{
		EventType:        ssdomain.CaepEventTypeSessionRevoked,
		Subject:          ssdomain.SsfSubject{SubjectType: ssdomain.SsfSubjectTypeAgent, TenantID: "tenant-a", PrincipalID: "agent_1"},
		Reason:           &reason,
		EventTimestamp:   time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC),
		InitiatingEntity: ssdomain.InitiatingEntityAdmin,
	}

	set, err := ssusecases.BuildAndSignSecurityEventToken(ctx, signer, "https://idp.example/tenant-a", "https://receiver.example/stream", event)
	if err != nil {
		t.Fatalf("BuildAndSignSecurityEventToken: %v", err)
	}
	if err := set.Validate(); err != nil {
		t.Fatalf("built SET fails Validate: %v", err)
	}
	if set.Issuer != "https://idp.example/tenant-a" || set.Audience != "https://receiver.example/stream" {
		t.Fatalf("unexpected iss/aud: %+v", set)
	}
	if set.JTI == "" || set.Compact == "" {
		t.Fatalf("expected non-empty JTI/Compact: %+v", set)
	}

	parts := strings.Split(set.Compact, ".")
	if len(parts) != 3 {
		t.Fatalf("expected a 3-part compact JWT, got %d parts", len(parts))
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload["iss"] != "https://idp.example/tenant-a" || payload["aud"] != "https://receiver.example/stream" {
		t.Fatalf("payload iss/aud mismatch: %+v", payload)
	}
	if payload["jti"] != set.JTI {
		t.Fatalf("payload jti mismatch: %+v", payload)
	}
	events, ok := payload["events"].(map[string]any)
	if !ok {
		t.Fatalf("expected an events claim map, got %T: %+v", payload["events"], payload)
	}
	eventClaims, ok := events["https://schemas.openid.net/secevent/caep/event-type/session-revoked"].(map[string]any)
	if !ok {
		t.Fatalf("expected the session-revoked event-type URI as key, got %+v", events)
	}
	if eventClaims["reason"] != "AgentKilled" {
		t.Fatalf("expected reason=AgentKilled in event claims, got %+v", eventClaims)
	}
	subject, ok := eventClaims["subject"].(map[string]any)
	if !ok || subject["principal_id"] != "agent_1" {
		t.Fatalf("expected subject.principal_id=agent_1, got %+v", eventClaims)
	}
}

// TestBuildAndSignSecurityEventToken_OmitsReasonWhenNil — reason が nil の CAEP
// イベント (session-revoked 以外のイベント種別を想定) では reason claim を出さない。
func TestBuildAndSignSecurityEventToken_OmitsReasonWhenNil(t *testing.T) {
	ctx := context.Background()
	keyStore, err := signingmemory.NewInMemoryKeyStore()
	if err != nil {
		t.Fatalf("NewInMemoryKeyStore: %v", err)
	}
	signer := &sign_jose.Signer{KeyStore: keyStore}
	event := ssdomain.CaepEvent{
		EventType:        ssdomain.CaepEventTypeAssuranceLevelChange,
		Subject:          ssdomain.SsfSubject{SubjectType: ssdomain.SsfSubjectTypeAgent, TenantID: "tenant-a", PrincipalID: "agent_1"},
		EventTimestamp:   time.Now().UTC(),
		InitiatingEntity: ssdomain.InitiatingEntityPolicy,
	}

	set, err := ssusecases.BuildAndSignSecurityEventToken(ctx, signer, "https://idp.example/tenant-a", "https://receiver.example/stream", event)
	if err != nil {
		t.Fatalf("BuildAndSignSecurityEventToken: %v", err)
	}
	parts := strings.Split(set.Compact, ".")
	payloadBytes, _ := base64.RawURLEncoding.DecodeString(parts[1])
	var payload map[string]any
	_ = json.Unmarshal(payloadBytes, &payload)
	events, ok := payload["events"].(map[string]any)
	if !ok {
		t.Fatalf("expected an events claim map, got %T: %+v", payload["events"], payload)
	}
	eventClaims, ok := events["https://schemas.openid.net/secevent/caep/event-type/assurance-level-change"].(map[string]any)
	if !ok {
		t.Fatalf("expected the assurance-level-change event-type URI as key, got %+v", events)
	}
	if _, present := eventClaims["reason"]; present {
		t.Fatalf("expected no reason claim, got %+v", eventClaims)
	}
}
