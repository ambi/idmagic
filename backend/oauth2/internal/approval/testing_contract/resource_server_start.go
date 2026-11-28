// Package testing_contract defines the shared approval request store contract.
package testing_contract

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	approvaldomain "github.com/ambi/idmagic/backend/oauth2/internal/approval/domain"
	approvalports "github.com/ambi/idmagic/backend/oauth2/internal/approval/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// ResourceServerStartFixture は承認ストアと、その外部キーを満たす利用者とクライアントである。
// Context は保存先のテナントを持ち、NewID は UUID を返す。
type ResourceServerStartFixture struct {
	Store     approvalports.ApprovalRequestStore
	Context   context.Context
	UserID    string
	ClientID  string
	OtherUser string
	NewID     func() string
}

// RunResourceServerStartContract は、評価の起票の重複の排除、件数の上限、直列化、使用権の記録が
// どの保存先でも同じ結果になることを確かめる (REQ-OAUTH2-093)。
func RunResourceServerStartContract(t *testing.T, newFixture func(t *testing.T) ResourceServerStartFixture) {
	t.Helper()
	// PostgreSQL の分解能に合わせ、読み戻した時刻を比べられるようにする。
	now := time.Now().UTC().Truncate(time.Microsecond)
	request := func(f ResourceServerStartFixture, userID, digest string) *approvaldomain.ApprovalRequest {
		server := "mcp-contract"
		return &approvaldomain.ApprovalRequest{
			ID: f.NewID(), ClientID: f.ClientID, UserID: userID, State: spec.ApprovalPending,
			Initiator: approvaldomain.InitiatedByResourceServer, ResourceServerID: &server, OperationDigest: &digest,
			AuthorizationDetails: []spec.AuthorizationDetail{{Type: "payment", Locations: []string{"https://mcp.example/"}, Actions: []string{"payer"}, Identifier: "a1"}},
			RequestedAt:          now, ExpiresAt: now.Add(approvaldomain.ResourceServerApprovalTTL),
		}
	}
	start := func(t *testing.T, f ResourceServerStartFixture, rec *approvaldomain.ApprovalRequest, at time.Time) (*approvaldomain.ApprovalRequest, approvalports.StartOutcome) {
		t.Helper()
		got, outcome, err := f.Store.StartForResourceServer(f.Context, rec, approvaldomain.MaxPendingResourceServerApprovals, at)
		if err != nil {
			t.Fatalf("StartForResourceServer: %v", err)
		}
		return got, outcome
	}

	t.Run("the same operation is started once while it is pending", func(t *testing.T) {
		f := newFixture(t)
		first, outcome := start(t, f, request(f, f.UserID, "op-1"), now)
		if outcome != approvalports.StartCreated || first == nil {
			t.Fatalf("first start: outcome=%v rec=%v", outcome, first)
		}
		again, outcome := start(t, f, request(f, f.UserID, "op-1"), now)
		if outcome != approvalports.StartExisting || again == nil || again.ID != first.ID {
			t.Fatalf("repeated start: outcome=%v rec=%+v, want the first request %s", outcome, again, first.ID)
		}
		other, outcome := start(t, f, request(f, f.UserID, "op-2"), now)
		if outcome != approvalports.StartCreated || other.ID == first.ID {
			t.Fatalf("another operation: outcome=%v rec=%+v", outcome, other)
		}
		otherUser, outcome := start(t, f, request(f, f.OtherUser, "op-1"), now)
		if outcome != approvalports.StartCreated || otherUser.ID == first.ID {
			t.Fatalf("another user: outcome=%v rec=%+v", outcome, otherUser)
		}
		// 期限を過ぎた保留は再利用しない。
		expired, outcome := start(t, f, request(f, f.UserID, "op-1"), first.ExpiresAt)
		if outcome != approvalports.StartCreated || expired.ID == first.ID {
			t.Fatalf("after expiry: outcome=%v rec=%+v", outcome, expired)
		}
		// 判断した承認は保留ではないので、同じ操作を新たに起票する。
		if _, err := f.Store.Decide(f.Context, other.ID, f.UserID, spec.ApprovalEventDeny, now); err != nil {
			t.Fatal(err)
		}
		if _, outcome := start(t, f, request(f, f.UserID, "op-2"), now); outcome != approvalports.StartCreated {
			t.Fatalf("after a decision: outcome=%v", outcome)
		}
	})

	t.Run("a user holds at most the maximum of pending requests", func(t *testing.T) {
		f := newFixture(t)
		for i := range approvaldomain.MaxPendingResourceServerApprovals {
			if _, outcome := start(t, f, request(f, f.UserID, fmt.Sprintf("op-%d", i)), now); outcome != approvalports.StartCreated {
				t.Fatalf("start %d: outcome=%v", i, outcome)
			}
		}
		if rec, outcome := start(t, f, request(f, f.UserID, "op-over"), now); outcome != approvalports.StartLimited || rec != nil {
			t.Fatalf("over the limit: outcome=%v rec=%v", outcome, rec)
		}
		// 上限でも、保留中の同じ操作は返す。
		if _, outcome := start(t, f, request(f, f.UserID, "op-0"), now); outcome != approvalports.StartExisting {
			t.Fatalf("existing at the limit: outcome=%v", outcome)
		}
		if _, outcome := start(t, f, request(f, f.OtherUser, "op-over"), now); outcome != approvalports.StartCreated {
			t.Fatalf("another user at the limit: outcome=%v", outcome)
		}
		pending, err := f.Store.ListPendingForUser(f.Context, f.UserID, now)
		if err != nil || len(pending) != approvaldomain.MaxPendingResourceServerApprovals {
			t.Fatalf("pending=%d err=%v", len(pending), err)
		}
	})

	t.Run("concurrent starts of one operation create one request", func(t *testing.T) {
		f := newFixture(t)
		var wg sync.WaitGroup
		var mu sync.Mutex
		ids, created := map[string]bool{}, 0
		for range 8 {
			wg.Go(func() {
				got, outcome, err := f.Store.StartForResourceServer(f.Context, request(f, f.UserID, "op-race"), approvaldomain.MaxPendingResourceServerApprovals, now)
				if err != nil || got == nil {
					t.Errorf("start: rec=%v err=%v", got, err)
					return
				}
				mu.Lock()
				defer mu.Unlock()
				ids[got.ID] = true
				if outcome == approvalports.StartCreated {
					created++
				}
			})
		}
		wg.Wait()
		if created != 1 || len(ids) != 1 {
			t.Fatalf("created=%d distinct=%d, want one request", created, len(ids))
		}
	})

	t.Run("an approved request records one use and becomes consumed", func(t *testing.T) {
		f := newFixture(t)
		rec, _ := start(t, f, request(f, f.UserID, "op-use"), now)
		if claimed, recorded, err := f.Store.ClaimUse(f.Context, rec.ID, approvaldomain.ApprovalUse{KeyHash: "early", ClaimedAt: now}, now); err != nil || recorded || claimed != nil {
			t.Fatalf("pending request accepted a use: %+v %v %v", claimed, recorded, err)
		}
		if _, err := f.Store.Decide(f.Context, rec.ID, f.UserID, spec.ApprovalEventApprove, now); err != nil {
			t.Fatal(err)
		}
		var wg sync.WaitGroup
		var mu sync.Mutex
		winners := 0
		for _, key := range []string{"a", "b", "c", "d"} {
			wg.Go(func() {
				_, recorded, err := f.Store.ClaimUse(f.Context, rec.ID, approvaldomain.ApprovalUse{KeyHash: key, ClaimedAt: now}, now)
				if err != nil {
					t.Errorf("claim: %v", err)
				}
				if recorded {
					mu.Lock()
					winners++
					mu.Unlock()
				}
			})
		}
		wg.Wait()
		if winners != 1 {
			t.Fatalf("winners=%d, want 1", winners)
		}
		current, err := f.Store.FindByID(f.Context, rec.ID)
		if err != nil || current == nil || current.State != spec.ApprovalConsumed || current.Use == nil || current.ConsumedAt == nil {
			t.Fatalf("after the use: %+v %v", current, err)
		}
		holder, recorded, err := f.Store.ClaimUse(f.Context, rec.ID, approvaldomain.ApprovalUse{KeyHash: current.Use.KeyHash, ClaimedAt: now}, now)
		if err != nil || recorded || holder == nil || holder.Use.KeyHash != current.Use.KeyHash {
			t.Fatalf("holder replay: %+v %v %v", holder, recorded, err)
		}
		if expired, recorded, err := f.Store.ClaimUse(f.Context, rec.ID, approvaldomain.ApprovalUse{KeyHash: "late", ClaimedAt: now}, rec.ExpiresAt); err != nil || recorded || expired != nil {
			t.Fatalf("expired request returned a holder: %+v %v %v", expired, recorded, err)
		}
	})

	t.Run("an approved CIBA request is not used before its exchange", func(t *testing.T) {
		f := newFixture(t)
		id := f.NewID()
		ciba := &approvaldomain.ApprovalRequest{
			ID: id, ClientID: f.ClientID, UserID: f.UserID, Scopes: []string{"openid"}, State: spec.ApprovalPending,
			AuthReqIDHash: approvaldomain.HashAuthReqID(id), IntervalSeconds: 5, RequestedAt: now, ExpiresAt: now.Add(time.Minute),
		}
		if err := f.Store.Save(f.Context, ciba); err != nil {
			t.Fatal(err)
		}
		if _, err := f.Store.Decide(f.Context, id, f.UserID, spec.ApprovalEventApprove, now); err != nil {
			t.Fatal(err)
		}
		if claimed, recorded, err := f.Store.ClaimUse(f.Context, id, approvaldomain.ApprovalUse{KeyHash: "a", ClaimedAt: now}, now); err != nil || recorded || claimed != nil {
			t.Fatalf("approved CIBA request accepted a use: %+v %v %v", claimed, recorded, err)
		}
		// 評価の起票は CIBA の索引に現れない。
		rec, _ := start(t, f, request(f, f.UserID, "op-index"), now)
		if found, err := f.Store.FindByAuthReqIDHash(f.Context, ""); err != nil || found != nil {
			t.Fatalf("empty auth_req_id hash found %+v (%v); started %s", found, err, rec.ID)
		}
	})
}
