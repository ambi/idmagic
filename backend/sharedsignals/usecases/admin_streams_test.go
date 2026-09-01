package usecases_test

// 主要ユースケース追跡: REQ-SHAREDSIGNALS-006。

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
	ssmemory "github.com/ambi/idmagic/backend/sharedsignals/db_memory"
	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
	ssusecases "github.com/ambi/idmagic/backend/sharedsignals/usecases"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

func newAdminStreamDeps(t *testing.T) (ssusecases.AdminStreamDeps, *[]spec.DomainEvent) {
	t.Helper()
	events := &[]spec.DomainEvent{}
	deps := ssusecases.AdminStreamDeps{
		StreamRepo:            ssmemory.NewSsfStreamRepository(),
		TransmitterConfigRepo: ssmemory.NewSsfTransmitterConfigRepository(),
		ReceiverConfigRepo:    ssmemory.NewSsfReceiverConfigRepository(),
		DeliveryRepo:          ssmemory.NewSecurityEventDeliveryRepository(),
		Emit:                  func(e spec.DomainEvent) error { *events = append(*events, e); return nil },
	}
	return deps, events
}

func adminStreamTestCtx() context.Context {
	return tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{ID: "tenant-a"}, "", "")
}

// TestRegisterSsfTransmitterStream_ValidatesInput — RED: 必須項目の欠落を
// usecase 層で pre-check し、ドメイン Validate() まで到達させない (4xx マッピング
// のための established convention)。
func TestRegisterSsfTransmitterStream_ValidatesInput(t *testing.T) {
	ctx := adminStreamTestCtx()
	deps, _ := newAdminStreamDeps(t)
	now := time.Now().UTC()

	cases := []struct {
		name    string
		in      ssusecases.RegisterSsfTransmitterStreamInput
		wantErr error
	}{
		{"empty event_types", ssusecases.RegisterSsfTransmitterStreamInput{DeliveryEndpoint: "https://receiver.example", Audience: "aud"}, ssusecases.ErrEventTypesEmpty},
		{"invalid event_types", ssusecases.RegisterSsfTransmitterStreamInput{EventTypes: []ssdomain.CaepEventType{"bogus"}, DeliveryEndpoint: "https://receiver.example", Audience: "aud"}, ssusecases.ErrEventTypeInvalid},
		{"non-https endpoint", ssusecases.RegisterSsfTransmitterStreamInput{EventTypes: []ssdomain.CaepEventType{ssdomain.CaepEventTypeSessionRevoked}, DeliveryEndpoint: "http://receiver.example", Audience: "aud"}, ssusecases.ErrDeliveryEndpointInvalid},
		{"missing audience", ssusecases.RegisterSsfTransmitterStreamInput{EventTypes: []ssdomain.CaepEventType{ssdomain.CaepEventTypeSessionRevoked}, DeliveryEndpoint: "https://receiver.example"}, ssusecases.ErrAudienceRequired},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ssusecases.RegisterSsfTransmitterStream(ctx, deps, tc.in, now); !errors.Is(err, tc.wantErr) {
				t.Fatalf("error = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

// TestRegisterSsfTransmitterStream_Succeeds — RED: 成功時に SsfStream +
// SsfTransmitterConfig を保存し SsfStreamRegistered を emit する。
func TestRegisterSsfTransmitterStream_Succeeds(t *testing.T) {
	ctx := adminStreamTestCtx()
	deps, events := newAdminStreamDeps(t)
	now := time.Now().UTC()
	authz := "Bearer secret"

	stream, err := ssusecases.RegisterSsfTransmitterStream(ctx, deps, ssusecases.RegisterSsfTransmitterStreamInput{
		EventTypes: []ssdomain.CaepEventType{ssdomain.CaepEventTypeSessionRevoked}, DeliveryEndpoint: "https://receiver.example/hook",
		Audience: "aud", DeliveryAuthorization: &authz,
	}, now)
	if err != nil {
		t.Fatalf("RegisterSsfTransmitterStream: %v", err)
	}
	if stream.Direction != ssdomain.SsfStreamDirectionTransmit || stream.Status != ssdomain.SsfStreamStatusEnabled {
		t.Fatalf("unexpected stream: %+v", stream)
	}
	config, err := deps.TransmitterConfigRepo.FindByStream(ctx, "tenant-a", stream.ID)
	if err != nil {
		t.Fatalf("FindByStream: %v", err)
	}
	if config == nil || config.DeliveryEndpoint != "https://receiver.example/hook" || config.MaxDeliveryAttempts != ssdomain.DefaultMaxDeliveryAttempts {
		t.Fatalf("unexpected config: %+v", config)
	}
	if len(*events) != 1 || (*events)[0].EventType() != "SsfStreamRegistered" {
		t.Fatalf("expected SsfStreamRegistered, got %+v", *events)
	}
}

// TestRegisterSsfReceiverStream_ValidatesInputAndSucceeds — RED: 必須項目 pre-check
// と成功時の SsfReceiverConfig 保存を確認する。
func TestRegisterSsfReceiverStream_ValidatesInputAndSucceeds(t *testing.T) {
	ctx := adminStreamTestCtx()
	deps, events := newAdminStreamDeps(t)
	now := time.Now().UTC()

	if _, err := ssusecases.RegisterSsfReceiverStream(ctx, deps, ssusecases.RegisterSsfReceiverStreamInput{
		EventTypes: []ssdomain.CaepEventType{ssdomain.CaepEventTypeSessionRevoked}, TrustedIssuer: "https://issuer.example",
	}, now); !errors.Is(err, ssusecases.ErrJWKSSourceRequired) {
		t.Fatalf("expected ErrJWKSSourceRequired, got %v", err)
	}

	jwksURI := "https://issuer.example/jwks"
	stream, err := ssusecases.RegisterSsfReceiverStream(ctx, deps, ssusecases.RegisterSsfReceiverStreamInput{
		EventTypes: []ssdomain.CaepEventType{ssdomain.CaepEventTypeSessionRevoked}, TrustedIssuer: "https://issuer.example",
		JWKSURI: &jwksURI, AcceptedAudiences: []string{"aud-1"},
	}, now)
	if err != nil {
		t.Fatalf("RegisterSsfReceiverStream: %v", err)
	}
	if stream.Direction != ssdomain.SsfStreamDirectionReceive {
		t.Fatalf("unexpected direction: %+v", stream)
	}
	config, err := deps.ReceiverConfigRepo.FindByStream(ctx, "tenant-a", stream.ID)
	if err != nil {
		t.Fatalf("FindByStream: %v", err)
	}
	if config == nil || config.TrustedIssuer != "https://issuer.example" {
		t.Fatalf("unexpected config: %+v", config)
	}
	if len(*events) != 1 || (*events)[0].EventType() != "SsfStreamRegistered" {
		t.Fatalf("expected SsfStreamRegistered, got %+v", *events)
	}
}

// TestUpdateSsfStream_OmittedEventTypesKeepsCurrent — RED: event_types 省略時は
// 現状維持で no-op (event を emit しない)。
func TestUpdateSsfStream_OmittedEventTypesKeepsCurrent(t *testing.T) {
	ctx := adminStreamTestCtx()
	deps, events := newAdminStreamDeps(t)
	now := time.Now().UTC()
	stream, err := ssusecases.RegisterSsfTransmitterStream(ctx, deps, ssusecases.RegisterSsfTransmitterStreamInput{
		EventTypes: []ssdomain.CaepEventType{ssdomain.CaepEventTypeSessionRevoked}, DeliveryEndpoint: "https://receiver.example", Audience: "aud",
	}, now)
	if err != nil {
		t.Fatalf("seed stream: %v", err)
	}
	*events = (*events)[:0]

	got, err := ssusecases.UpdateSsfStream(ctx, deps, stream.ID, ssusecases.UpdateSsfStreamInput{}, now)
	if err != nil {
		t.Fatalf("UpdateSsfStream: %v", err)
	}
	if len(got.EventTypes) != 1 || got.EventTypes[0] != ssdomain.CaepEventTypeSessionRevoked {
		t.Fatalf("event_types changed unexpectedly: %+v", got.EventTypes)
	}
	if len(*events) != 0 {
		t.Fatalf("expected no event for a no-op update, got %+v", *events)
	}

	updated, err := ssusecases.UpdateSsfStream(ctx, deps, stream.ID, ssusecases.UpdateSsfStreamInput{
		EventTypes: &[]ssdomain.CaepEventType{ssdomain.CaepEventTypeCredentialChange},
	}, now)
	if err != nil {
		t.Fatalf("UpdateSsfStream: %v", err)
	}
	if len(updated.EventTypes) != 1 || updated.EventTypes[0] != ssdomain.CaepEventTypeCredentialChange {
		t.Fatalf("event_types not updated: %+v", updated.EventTypes)
	}
	if len(*events) != 1 || (*events)[0].EventType() != "SsfStreamUpdated" {
		t.Fatalf("expected SsfStreamUpdated, got %+v", *events)
	}
}

// TestDisableEnableSsfStream_IsIdempotent — RED: 既に目的の状態なら no-op
// (event を再 emit しない)。
func TestDisableEnableSsfStream_IsIdempotent(t *testing.T) {
	ctx := adminStreamTestCtx()
	deps, events := newAdminStreamDeps(t)
	now := time.Now().UTC()
	stream, err := ssusecases.RegisterSsfTransmitterStream(ctx, deps, ssusecases.RegisterSsfTransmitterStreamInput{
		EventTypes: []ssdomain.CaepEventType{ssdomain.CaepEventTypeSessionRevoked}, DeliveryEndpoint: "https://receiver.example", Audience: "aud",
	}, now)
	if err != nil {
		t.Fatalf("seed stream: %v", err)
	}
	*events = (*events)[:0]

	if _, err := ssusecases.DisableSsfStream(ctx, deps, stream.ID, now); err != nil {
		t.Fatalf("DisableSsfStream: %v", err)
	}
	if _, err := ssusecases.DisableSsfStream(ctx, deps, stream.ID, now); err != nil {
		t.Fatalf("DisableSsfStream (idempotent): %v", err)
	}
	if len(*events) != 1 || (*events)[0].EventType() != "SsfStreamDisabled" {
		t.Fatalf("expected exactly one SsfStreamDisabled, got %+v", *events)
	}

	if _, err := ssusecases.EnableSsfStream(ctx, deps, stream.ID, now); err != nil {
		t.Fatalf("EnableSsfStream: %v", err)
	}
	if len(*events) != 2 || (*events)[1].EventType() != "SsfStreamEnabled" {
		t.Fatalf("expected SsfStreamEnabled, got %+v", *events)
	}
}

// TestDeleteSsfStream_CascadesTransmitterConfig — RED: stream 削除時に付随する
// SsfTransmitterConfig も削除する。
func TestDeleteSsfStream_CascadesTransmitterConfig(t *testing.T) {
	ctx := adminStreamTestCtx()
	deps, events := newAdminStreamDeps(t)
	now := time.Now().UTC()
	stream, err := ssusecases.RegisterSsfTransmitterStream(ctx, deps, ssusecases.RegisterSsfTransmitterStreamInput{
		EventTypes: []ssdomain.CaepEventType{ssdomain.CaepEventTypeSessionRevoked}, DeliveryEndpoint: "https://receiver.example", Audience: "aud",
	}, now)
	if err != nil {
		t.Fatalf("seed stream: %v", err)
	}
	*events = (*events)[:0]

	if err := ssusecases.DeleteSsfStream(ctx, deps, stream.ID, now); err != nil {
		t.Fatalf("DeleteSsfStream: %v", err)
	}
	if got, _ := deps.StreamRepo.FindByID(ctx, "tenant-a", stream.ID); got != nil {
		t.Fatalf("expected stream deleted, got %+v", got)
	}
	if got, _ := deps.TransmitterConfigRepo.FindByStream(ctx, "tenant-a", stream.ID); got != nil {
		t.Fatalf("expected transmitter config cascade-deleted, got %+v", got)
	}
	if len(*events) != 1 || (*events)[0].EventType() != "SsfStreamDeleted" {
		t.Fatalf("expected SsfStreamDeleted, got %+v", *events)
	}

	if err := ssusecases.DeleteSsfStream(ctx, deps, stream.ID, now); !errors.Is(err, ssusecases.ErrStreamNotFound) {
		t.Fatalf("expected ErrStreamNotFound on re-delete, got %v", err)
	}
}

// TestGetSsfStream_CrossTenantIsNotFound — RED: 別テナントの stream は
// ErrStreamNotFound として扱う (存在確認情報の漏洩を避ける)。
func TestGetSsfStream_CrossTenantIsNotFound(t *testing.T) {
	ctx := adminStreamTestCtx()
	deps, _ := newAdminStreamDeps(t)
	now := time.Now().UTC()
	stream, err := ssusecases.RegisterSsfTransmitterStream(ctx, deps, ssusecases.RegisterSsfTransmitterStreamInput{
		EventTypes: []ssdomain.CaepEventType{ssdomain.CaepEventTypeSessionRevoked}, DeliveryEndpoint: "https://receiver.example", Audience: "aud",
	}, now)
	if err != nil {
		t.Fatalf("seed stream: %v", err)
	}

	otherTenantCtx := tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{ID: "tenant-b"}, "", "")
	if _, err := ssusecases.GetSsfStream(otherTenantCtx, deps, stream.ID); !errors.Is(err, ssusecases.ErrStreamNotFound) {
		t.Fatalf("expected ErrStreamNotFound across tenants, got %v", err)
	}
}

// TestListSecurityEventDeliveries_ReturnsStreamDeliveries — RED: stream の
// 配送状況一覧を返し、未知の stream_id は ErrStreamNotFound。
func TestListSecurityEventDeliveries_ReturnsStreamDeliveries(t *testing.T) {
	ctx := adminStreamTestCtx()
	deps, _ := newAdminStreamDeps(t)
	now := time.Now().UTC()
	stream, err := ssusecases.RegisterSsfTransmitterStream(ctx, deps, ssusecases.RegisterSsfTransmitterStreamInput{
		EventTypes: []ssdomain.CaepEventType{ssdomain.CaepEventTypeSessionRevoked}, DeliveryEndpoint: "https://receiver.example", Audience: "aud",
	}, now)
	if err != nil {
		t.Fatalf("seed stream: %v", err)
	}
	if err := deps.DeliveryRepo.Save(ctx, &ssdomain.SecurityEventDelivery{
		ID: "d1", TenantID: "tenant-a", StreamID: stream.ID, SetJTI: "jti-1",
		Set: ssdomain.SecurityEventToken{
			JTI: "jti-1", Issuer: "https://idp.example", Audience: "aud", IssuedAt: now,
			Event: ssdomain.CaepEvent{
				EventType:      ssdomain.CaepEventTypeSessionRevoked,
				Subject:        ssdomain.SsfSubject{SubjectType: ssdomain.SsfSubjectTypeAgent, TenantID: "tenant-a", PrincipalID: "agent_1"},
				EventTimestamp: now, InitiatingEntity: ssdomain.InitiatingEntityAdmin,
			},
			Compact: "a.b.c",
		},
		Status: ssdomain.SecurityEventDeliveryStatusPending, CreatedAt: now,
	}); err != nil {
		t.Fatalf("seed delivery: %v", err)
	}

	deliveries, err := ssusecases.ListSecurityEventDeliveries(ctx, deps, stream.ID)
	if err != nil {
		t.Fatalf("ListSecurityEventDeliveries: %v", err)
	}
	if len(deliveries) != 1 || deliveries[0].ID != "d1" {
		t.Fatalf("unexpected deliveries: %+v", deliveries)
	}

	if _, err := ssusecases.ListSecurityEventDeliveries(ctx, deps, "no-such-stream"); !errors.Is(err, ssusecases.ErrStreamNotFound) {
		t.Fatalf("expected ErrStreamNotFound, got %v", err)
	}
}
