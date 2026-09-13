package usecases_test

// REQ-OAUTH2-041 と REQ-OAUTH2-042 が宣言する拒否の具体例を、承認要求の起票と交換の
// use case 境界で観測する。
//
// 拒否は応答の型だけでなく「拒否が防いだ効果」まで読む。起票の拒否は承認要求が 1 件も
// 残らないこと、交換の拒否はトークンが 1 本も発行されないことが、拒否を書いたうえで
// 続行する実装との差になる。

import (
	"strings"
	"testing"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	approvaldomain "github.com/ambi/idmagic/backend/oauth2/approval/domain"
	approvalusecases "github.com/ambi/idmagic/backend/oauth2/approval/usecases"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// どれも invalid_request で括れてしまうので、種別の取り違えはここでしか見分けられない。
//
//spec:covers EX-OAUTH2-041-02, EX-OAUTH2-041-03, EX-OAUTH2-041-04, EX-OAUTH2-041-05: 起票の形式検査が返すエラー種別を、具体例が名指す区別のまま固定し、承認要求を残さない。
func TestStartApprovalRefusesMalformedRequestWithTheDeclaredError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input approvalusecases.StartApprovalInput
		want  string
	}{
		{"scope を指定しない", approvalusecases.StartApprovalInput{LoginHint: "alice"}, "invalid_scope"},
		{
			"scope が openid を含まない",
			approvalusecases.StartApprovalInput{LoginHint: "alice", Scope: "payments.write"},
			"invalid_scope",
		},
		{"hint が無い", approvalusecases.StartApprovalInput{Scope: "openid"}, "invalid_request"},
		{
			"hint が 2 つある",
			approvalusecases.StartApprovalInput{LoginHint: "alice", IDTokenHint: "token", Scope: "openid"},
			"invalid_request",
		},
		{
			"requested_expiry が非正である",
			approvalusecases.StartApprovalInput{
				LoginHint: "alice", Scope: "openid", RequestedExpiry: new(0),
			},
			"invalid_request",
		},
		{
			"requested_expiry が 600 秒を超える",
			approvalusecases.StartApprovalInput{
				LoginHint: "alice", Scope: "openid", RequestedExpiry: new(601),
			},
			"invalid_request",
		},
		{
			"binding_message が 64 文字を超える",
			approvalusecases.StartApprovalInput{
				LoginHint: "alice", Scope: "openid",
				BindingMessage: strings.Repeat("あ", approvaldomain.MaxBindingMessageLength+1),
			},
			"invalid_binding_message",
		},
		{
			"binding_message が制御文字を含む",
			approvalusecases.StartApprovalInput{
				LoginHint: "alice", Scope: "openid", BindingMessage: "W-\x07123",
			},
			"invalid_binding_message",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			f := newApprovalFixture(t)
			input := test.input
			input.ClientID = "agent-app"
			_, err := approvalusecases.StartApproval(f.ctx, f.startDeps, input, time.Now().UTC())
			if approvalOAuthErrorCode(err) != test.want {
				t.Fatalf("StartApproval() error = %v, want %s", err, test.want)
			}
			assertNoApprovalRequested(t, f)
		})
	}
}

// 3 つを同じ種別へ倒すのは、どの利用者が存在するかを起票元へ漏らさないためである。
//
//spec:covers EX-OAUTH2-041-06, EX-OAUTH2-041-07: 解決できない、別テナントの、非 active な login_hint の起票は unknown_user_id で拒否され、承認要求は残らない。
func TestStartApprovalRefusesUnresolvableLoginHint(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		hint string
	}{
		{"未知の login_hint", "nobody"},
		{"別テナントの User", "bob"},
		{"非 active な User", "carol"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			f := newApprovalFixture(t)
			seedNeighbourUsers(t, f)
			_, err := approvalusecases.StartApproval(f.ctx, f.startDeps, approvalusecases.StartApprovalInput{
				ClientID: "agent-app", LoginHint: test.hint, Scope: "openid",
			}, time.Now().UTC())
			if approvalOAuthErrorCode(err) != "unknown_user_id" {
				t.Fatalf("StartApproval() error = %v, want unknown_user_id", err)
			}
			assertNoApprovalRequested(t, f)
		})
	}
}

// openid を含んでいても通らない。
//
//spec:covers EX-OAUTH2-041-08: クライアントの許可 scope に含まれない scope の起票は invalid_scope で拒否され、承認要求は残らない。
func TestStartApprovalRefusesScopeTheClientMayNotRequest(t *testing.T) {
	t.Parallel()
	f := newApprovalFixture(t)
	_, err := approvalusecases.StartApproval(f.ctx, f.startDeps, approvalusecases.StartApprovalInput{
		ClientID: "agent-app", LoginHint: "alice", Scope: "openid accounts.transfer",
	}, time.Now().UTC())
	if approvalOAuthErrorCode(err) != "invalid_scope" {
		t.Fatalf("StartApproval() error = %v, want invalid_scope", err)
	}
	assertNoApprovalRequested(t, f)
}

// 応答だけでは、記録を Pending のまま放置する実装と区別できない。
//
//spec:covers EX-OAUTH2-042-03: expires_at を過ぎた承認要求の交換は expired_token で拒否され、トークンは発行されず、記録は Expired へ確定する。
func TestApprovalExchangeExpiresTheRequestItRefuses(t *testing.T) {
	t.Parallel()
	f := newApprovalFixture(t)
	t0 := time.Now().UTC()
	started, err := approvalusecases.StartApproval(f.ctx, f.startDeps, approvalusecases.StartApprovalInput{
		ClientID: "agent-app", LoginHint: "alice", Scope: "openid",
		RequestedExpiry: new(300),
	}, t0)
	if err != nil {
		t.Fatal(err)
	}
	records, err := f.store.ListPendingForUser(f.ctx, "alice-id")
	if err != nil || len(records) != 1 {
		t.Fatalf("pending records = %d, err = %v", len(records), err)
	}
	_, err = approvalusecases.ExchangeApproval(f.ctx, f.exchangeDeps, approvalusecases.ExchangeApprovalInput{
		ClientID: "agent-app", AuthReqID: started.AuthReqID,
	}, t0.Add(301*time.Second))
	if approvalOAuthErrorCode(err) != "expired_token" {
		t.Fatalf("ExchangeApproval() error = %v, want expired_token", err)
	}
	if f.issuer.accessCalls != 0 {
		t.Fatalf("token issues = %d, want the refusal to have issued nothing", f.issuer.accessCalls)
	}
	record, err := f.store.FindByID(f.ctx, records[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if record.State != spec.ApprovalExpired {
		t.Fatalf("state = %v, want %v", record.State, spec.ApprovalExpired)
	}
}

// seedNeighbourUsers は「解決できてはならない」利用者を 2 人置く。別テナントの bob と、
// 同じテナントで無効化された carol である。fixture の UserRepo をそのまま使う。
func seedNeighbourUsers(t *testing.T, f approvalFixture) {
	t.Helper()
	repo, ok := f.startDeps.UserRepo.(*usermemory.UserRepository)
	if !ok {
		t.Fatalf("UserRepo = %T, want the in-memory repository", f.startDeps.UserRepo)
	}
	now := time.Now().UTC()
	repo.Seed(&userdomain.User{
		ID: "bob-id", TenantID: "tenant-b", PreferredUsername: "bob",
		CreatedAt: now, UpdatedAt: now,
	})
	repo.Seed(&userdomain.User{
		ID: "carol-id", TenantID: "tenant-a", PreferredUsername: "carol",
		Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusDisabled},
		CreatedAt: now, UpdatedAt: now,
	})
}

// assertNoApprovalRequested は、起票が拒否されたあとに承認要求が 1 件も残っていないことを
// 確かめる。拒否の応答だけを読むテストは、要求を保存してから拒否する実装を通してしまう。
func assertNoApprovalRequested(t *testing.T, f approvalFixture) {
	t.Helper()
	for _, userID := range []string{"alice-id", "bob-id", "carol-id"} {
		records, err := f.store.ListPendingForUser(f.ctx, userID)
		if err != nil {
			t.Fatalf("ListPendingForUser(%s): %v", userID, err)
		}
		if len(records) != 0 {
			t.Fatalf("拒否された起票が %s 宛の承認要求を %d 件残した", userID, len(records))
		}
	}
}
