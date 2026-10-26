package usecases_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"slices"
	"testing"
	"time"

	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	groupusecases "github.com/ambi/idmagic/backend/idmanagement/internal/group/usecases"
	idmusecases "github.com/ambi/idmagic/backend/idmanagement/internal/usecases"
	"github.com/ambi/idmagic/backend/tenancy/testing_tenant"
)

type delegationNotifications struct{ calls int }

func (n *delegationNotifications) NotifyGroupMutation(_ context.Context, _, _ string, trigger groupports.ProvisioningTrigger, _ time.Time) error {
	if trigger != groupports.ProvisioningGroupChanged {
		return errors.New("unexpected trigger")
	}
	n.calls++
	return nil
}

//spec:covers REQ-IDMANAGEMENT-093: 委任と取消しの送信 JSON が対象ロール、式のハッシュと変更欄を持ち、不正な承認はイベントも通知も出さない。
func TestGroupRoleDelegationAuditAndNotification(t *testing.T) {
	ctx := testing_tenant.Default(context.Background())
	deps, events := newGroupDeps(t)
	notifier := &delegationNotifications{}
	now := time.Now().UTC()
	g, err := groupusecases.CreateGroup(ctx, deps, groupusecases.CreateGroupInput{ActorUserID: "actor", Name: "upstream", Roles: []string{"catalog:read"}, UpstreamManaged: true, Now: now})
	if err != nil {
		t.Fatal(err)
	}
	deps.ProvisioningNotifier = notifier
	*events = nil
	notifier.calls = 0
	if _, err := groupusecases.SetGroupRoleDelegation(ctx, deps, groupusecases.SetGroupRoleDelegationInput{ActorUserID: "actor", GroupID: g.ID, Roles: []string{"catalog:write"}, Expression: "", Now: now}); !errors.Is(err, idmusecases.ErrInvalidRole) {
		t.Fatalf("invalid approval err=%v", err)
	}
	if len(*events) != 0 || notifier.calls != 0 {
		t.Fatal("invalid approval emitted effects")
	}
	for _, roles := range [][]string{{"catalog:read"}, {}} {
		if _, err := groupusecases.SetGroupRoleDelegation(ctx, deps, groupusecases.SetGroupRoleDelegationInput{ActorUserID: "actor", GroupID: g.ID, Roles: roles, Expression: "", Now: now}); err != nil {
			t.Fatal(err)
		}
		payload, err := json.Marshal((*events)[len(*events)-1])
		if err != nil {
			t.Fatal(err)
		}
		var body struct {
			Roles  *[]string `json:"delegatedRoles"`
			Fields []string  `json:"changedFields"`
			Hash   string    `json:"delegationRuleSha256"`
		}
		if err := json.Unmarshal(payload, &body); err != nil {
			t.Fatal(err)
		}
		digest := sha256.Sum256(nil)
		if body.Roles == nil || !slices.Equal(*body.Roles, roles) || !slices.Equal(body.Fields, []string{"role_delegation"}) || body.Hash != hex.EncodeToString(digest[:]) {
			t.Fatalf("event=%s", payload)
		}
	}
	if len(*events) != 2 || notifier.calls != 2 {
		t.Fatalf("events=%d notifications=%d", len(*events), notifier.calls)
	}
}

//spec:covers REQ-IDMANAGEMENT-092: 上流管理グループの更新で管理ロールを拒否し、保存と作用を行わない。
func TestUpstreamGroupRejectsAdministrativeRoleUpdate(t *testing.T) {
	ctx := testing_tenant.Default(context.Background())
	deps, events := newGroupDeps(t)
	now := time.Now().UTC()
	g, err := groupusecases.CreateGroup(ctx, deps, groupusecases.CreateGroupInput{ActorUserID: "actor", Name: "upstream", UpstreamManaged: true, Now: now})
	if err != nil {
		t.Fatal(err)
	}
	*events = nil
	for _, role := range []string{"admin", "system_admin"} {
		roles := []string{role}
		_, err := groupusecases.UpdateGroup(ctx, deps, groupusecases.UpdateGroupInput{ActorUserID: "actor", ID: g.ID, Roles: &roles, Now: now})
		if !errors.Is(err, idmusecases.ErrInvalidRole) {
			t.Fatalf("update err=%v", err)
		}
	}
	stored, err := deps.GroupRepo.FindByID(ctx, g.TenantID, g.ID)
	if err != nil || len(stored.Roles) != 0 || len(*events) != 0 {
		t.Fatalf("stored=%+v err=%v events=%d", stored, err, len(*events))
	}
}
