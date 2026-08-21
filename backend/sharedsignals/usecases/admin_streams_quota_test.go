package usecases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
	ssusecases "github.com/ambi/idmagic/backend/sharedsignals/usecases"
	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

func transmitterInput() ssusecases.RegisterSsfTransmitterStreamInput {
	return ssusecases.RegisterSsfTransmitterStreamInput{
		EventTypes:       []ssdomain.CaepEventType{ssdomain.CaepEventTypeSessionRevoked},
		DeliveryEndpoint: "https://receiver.example", Audience: "aud",
	}
}

func receiverInput() ssusecases.RegisterSsfReceiverStreamInput {
	jwksURI := "https://issuer.example/jwks"
	return ssusecases.RegisterSsfReceiverStreamInput{
		EventTypes:    []ssdomain.CaepEventType{ssdomain.CaepEventTypeSessionRevoked},
		TrustedIssuer: "https://issuer.example", JWKSURI: &jwksURI,
		AcceptedAudiences: []string{"https://idp.example"},
	}
}

// TestRegisterSsfStream_HardQuota — RED: REQ-SHAREDSIGNALS-009
// (spec/contexts/sharedsignals/scenarios.md)。SsfStream の登録は Hard Quota を
// 超えると QuotaExceededError で拒否され、stream も付随する設定も作成されない。
// transmitter と receiver は `ssf_streams` という同一の上限を共有し、削除すると
// 利用量が戻る。
func TestRegisterSsfStream_HardQuota(t *testing.T) {
	ctx := adminStreamTestCtx()
	now := time.Now().UTC()

	newDeps := func(t *testing.T, limit int) ssusecases.AdminStreamDeps {
		t.Helper()
		deps, _ := newAdminStreamDeps(t)
		quotaRepo := tenancymemory.NewQuotaRepository()
		if err := quotaRepo.SetQuota(context.Background(), "tenant-a", &tenancydomain.TenantQuota{
			SsfStreams: &limit,
		}); err != nil {
			t.Fatalf("seed quota: %v", err)
		}
		deps.QuotaRepo = quotaRepo
		return deps
	}

	assertQuotaExceeded := func(t *testing.T, err error) {
		t.Helper()
		var quotaErr *tenancydomain.QuotaExceededError
		if !errors.As(err, &quotaErr) {
			t.Fatalf("expected QuotaExceededError, got %v", err)
		}
		if quotaErr.Resource != tenancydomain.ResourceSsfStreams {
			t.Fatalf("resource = %q, want %q", quotaErr.Resource, tenancydomain.ResourceSsfStreams)
		}
	}

	t.Run("TransmitterAtLimitIsRejected", func(t *testing.T) {
		deps := newDeps(t, 1)
		if _, err := ssusecases.RegisterSsfTransmitterStream(ctx, deps, transmitterInput(), now); err != nil {
			t.Fatalf("first registration: %v", err)
		}
		_, err := ssusecases.RegisterSsfTransmitterStream(ctx, deps, transmitterInput(), now)
		assertQuotaExceeded(t, err)
		streams, listErr := deps.StreamRepo.ListAll(ctx, "tenant-a")
		if listErr != nil {
			t.Fatal(listErr)
		}
		if len(streams) != 1 {
			t.Fatalf("expected the rejected stream not to be created, got %d", len(streams))
		}
	})

	t.Run("ReceiverSharesTheSameLimit", func(t *testing.T) {
		deps := newDeps(t, 1)
		if _, err := ssusecases.RegisterSsfTransmitterStream(ctx, deps, transmitterInput(), now); err != nil {
			t.Fatalf("first registration: %v", err)
		}
		_, err := ssusecases.RegisterSsfReceiverStream(ctx, deps, receiverInput(), now)
		assertQuotaExceeded(t, err)
	})

	t.Run("DeleteReleasesTheUsage", func(t *testing.T) {
		deps := newDeps(t, 1)
		stream, err := ssusecases.RegisterSsfTransmitterStream(ctx, deps, transmitterInput(), now)
		if err != nil {
			t.Fatalf("first registration: %v", err)
		}
		if err := ssusecases.DeleteSsfStream(ctx, deps, stream.ID, now); err != nil {
			t.Fatal(err)
		}
		if _, err := ssusecases.RegisterSsfReceiverStream(ctx, deps, receiverInput(), now); err != nil {
			t.Fatalf("expected the freed quota to allow another stream: %v", err)
		}
	})

	t.Run("QuotaExceededIsEmitted", func(t *testing.T) {
		deps, events := newAdminStreamDeps(t)
		limit := 0
		quotaRepo := tenancymemory.NewQuotaRepository()
		if err := quotaRepo.SetQuota(ctx, "tenant-a", &tenancydomain.TenantQuota{SsfStreams: &limit}); err != nil {
			t.Fatal(err)
		}
		deps.QuotaRepo = quotaRepo
		_, err := ssusecases.RegisterSsfTransmitterStream(ctx, deps, transmitterInput(), now)
		assertQuotaExceeded(t, err)
		var emitted bool
		for _, event := range *events {
			if exceeded, ok := event.(*tenancydomain.QuotaExceeded); ok && exceeded.Resource == tenancydomain.ResourceSsfStreams {
				emitted = true
			}
		}
		if !emitted {
			t.Fatalf("expected a QuotaExceeded event for ssf_streams, got %v", *events)
		}
	})

	t.Run("NilQuotaRepoSkipsEnforcement", func(t *testing.T) {
		deps, _ := newAdminStreamDeps(t)
		for range 3 {
			if _, err := ssusecases.RegisterSsfTransmitterStream(ctx, deps, transmitterInput(), now); err != nil {
				t.Fatalf("expected no enforcement without a QuotaRepo: %v", err)
			}
		}
	})
}
