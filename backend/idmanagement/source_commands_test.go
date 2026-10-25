package idmanagement_test

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"reflect"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/idmanagement"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy/testing_tenant"
)

type authorityNotifications struct{ calls int }

func (n *authorityNotifications) NotifyUserMutation(context.Context, string, string, userports.ProvisioningTrigger, time.Time) error {
	n.calls++
	return nil
}

//spec:covers REQ-IDMANAGEMENT-091: 公開の取り込み操作はローカル属性の追加、変更と削除を無作用で拒否し、保持したままの上流属性の変更は受け付ける。
func TestSourceCommandsRespectAttributeWriteAuthority(t *testing.T) {
	ctx := testing_tenant.Default(context.Background())
	m := idmanagement.NewMemoryModule()
	now := time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC)
	var defs []userdomain.UserAttributeDef
	if err := json.Unmarshal([]byte(`[
		{"key":"approval","type":"string","visibility":"admin_readable","write_authority":"local"},
		{"key":"cost_center","type":"string","visibility":"admin_readable"}
	]`), &defs); err != nil {
		t.Fatal(err)
	}
	if err := m.AttrSchemaRepo.Save(ctx, &userdomain.TenantUserAttributeSchema{TenantID: "default", Attributes: defs, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	var events []spec.DomainEvent
	notifications := &authorityNotifications{}
	m.ProvisioningNotifier = notifications
	commands := m.SourceCommands(idmanagement.LifecycleDeps{AttrSchemaRepo: m.AttrSchemaRepo}, "scim", func(e spec.DomainEvent) error {
		events = append(events, e)
		return nil
	})
	value := func(s string) userdomain.AttributeValue {
		return userdomain.AttributeValue{Type: idmdomain.AttributeTypeString, String: new(s)}
	}
	input := &userdomain.User{
		TenantID: "default", PreferredUsername: "alice", CreatedAt: now, UpdatedAt: now,
		Attributes: map[string]userdomain.AttributeValue{"approval": value("approved")},
	}
	if got, err := commands.CreateUser(ctx, input); !errors.Is(err, idmanagement.ErrSourceInvalidAttribute) || got != nil {
		t.Fatalf("local creation = %+v, %v; want no user and attribute refusal", got, err)
	}
	if saved, err := m.UserRepo.FindByUsername(ctx, "default", "alice"); err != nil || saved != nil || len(events) != 0 || notifications.calls != 0 {
		t.Fatalf("refused creation left state or effects: user=%+v err=%v events=%d notifications=%d", saved, err, len(events), notifications.calls)
	}
	input.Attributes = map[string]userdomain.AttributeValue{"cost_center": value("100")}
	user, err := commands.CreateUser(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	// ローカルの値は管理者が設定したものとして保存し、上流の更新後に読み直す。
	user.Attributes["approval"] = value("reviewed")
	if err := m.UserRepo.Save(ctx, user); err != nil {
		t.Fatal(err)
	}
	for _, remove := range []bool{false, true} {
		before, err := m.UserRepo.FindBySub(ctx, user.ID)
		if err != nil {
			t.Fatal(err)
		}
		next := *before
		next.Attributes = maps.Clone(before.Attributes)
		next.Attributes["approval"] = value("injected")
		if remove {
			delete(next.Attributes, "approval")
		}
		next.Attributes["cost_center"] = value("200")
		eventCount, notificationCount := len(events), notifications.calls
		if got, err := commands.UpdateUser(ctx, &next); !errors.Is(err, idmanagement.ErrSourceInvalidAttribute) || got != nil {
			t.Fatalf("remove=%v: update=%+v, %v; want refusal", remove, got, err)
		}
		after, err := m.UserRepo.FindBySub(ctx, user.ID)
		if err != nil || !reflect.DeepEqual(before, after) || len(events) != eventCount || notifications.calls != notificationCount {
			t.Fatalf("remove=%v: refused update changed state or effects: before=%+v after=%+v err=%v", remove, before, after, err)
		}
	}
	user.Attributes = maps.Clone(user.Attributes)
	user.Attributes["cost_center"] = value("300")
	updated, err := commands.UpdateUser(ctx, user)
	if err != nil || updated == nil || *updated.Attributes["approval"].String != "reviewed" || *updated.Attributes["cost_center"].String != "300" {
		t.Fatalf("preserving local attribute: updated=%+v err=%v", updated, err)
	}
}
