package db_memory_test

import (
	"context"
	"sync"
	"testing"
	"time"

	approvaldb "github.com/ambi/idmagic/backend/oauth2/approval/db_memory"
	approvaldomain "github.com/ambi/idmagic/backend/oauth2/approval/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

// REQ-OAUTH2-043: a pending request accepts exactly one account decision.
func TestApprovalRequestStoreDecideIsCompareAndSet(t *testing.T) {
	t.Parallel()
	store := approvaldb.NewApprovalRequestStore()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	id, err := approvaldomain.NewApprovalRequestID()
	if err != nil {
		t.Fatal(err)
	}
	rec := &approvaldomain.ApprovalRequest{
		ID: id, TenantID: "tenant-a", ClientID: "client-a", UserID: "alice",
		Scopes: []string{"openid"}, State: spec.ApprovalPending,
		AuthReqIDHash: approvaldomain.HashAuthReqID("secret"), IntervalSeconds: 5,
		RequestedAt: now, ExpiresAt: now.Add(time.Minute),
	}
	ctx := tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{ID: "tenant-a"}, "", "")
	if err := store.Save(ctx, rec); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	results := make(chan *approvaldomain.ApprovalRequest, 2)
	for _, event := range []spec.ApprovalRequestEvent{spec.ApprovalEventApprove, spec.ApprovalEventDeny} {
		wg.Go(func() {
			got, decideErr := store.Decide(ctx, id, "alice", event, now.Add(time.Second))
			if decideErr != nil {
				t.Errorf("Decide: %v", decideErr)
			}
			results <- got
		})
	}
	wg.Wait()
	close(results)
	succeeded := 0
	for got := range results {
		if got != nil {
			succeeded++
		}
	}
	if succeeded != 1 {
		t.Fatalf("successful decisions = %d, want 1", succeeded)
	}
}
