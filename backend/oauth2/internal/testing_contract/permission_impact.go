package testing_contract

import (
	"context"
	"reflect"
	"testing"
	"time"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
)

type ImpactFixture struct {
	Types                             oauthports.AuthorizationDetailTypeRepository
	Servers                           oauthports.McpResourceServerRepository
	TenantID, OtherTenantID, ServerID string
	Now                               time.Time
}

func RunPermissionImpact(t *testing.T, f ImpactFixture) {
	t.Helper()
	ctx := context.Background()
	no := false
	impact := &oauthdomain.PermissionImpact{Changes: "irreversible", PersonalData: &no}
	detail := &oauthdomain.AuthorizationDetailType{TenantID: f.TenantID, Type: "payment", DisplayTemplate: "payment", State: oauthdomain.DetailTypeEnabled, Schema: oauthdomain.AuthorizationDetailsSchema{Rules: []oauthdomain.AuthorizationDetailFieldRule{{Name: "actions", Semantics: oauthdomain.DetailFieldSet}}}, CreatedAt: f.Now, UpdatedAt: f.Now, Impact: impact}
	if err := f.Types.Save(ctx, detail); err != nil {
		t.Fatal(err)
	}
	got, err := f.Types.FindByType(ctx, f.TenantID, detail.Type)
	if err != nil || got == nil || !reflect.DeepEqual(got.Impact, impact) {
		t.Fatalf("種類の保存=%+v err=%v", got, err)
	}
	got.Impact.Changes = "none"
	*got.Impact.PersonalData = true
	got, err = f.Types.FindByType(ctx, f.TenantID, detail.Type)
	if err != nil || !reflect.DeepEqual(got.Impact, impact) {
		t.Fatalf("種類の複製=%+v err=%v", got, err)
	}
	if leaked, err := f.Types.FindByType(ctx, f.OtherTenantID, detail.Type); err != nil || leaked != nil {
		t.Fatalf("種類のテナント分離=%+v %v", leaked, err)
	}
	server := &oauthdomain.McpResourceServer{TenantID: f.TenantID, ID: f.ServerID, Resource: "https://impact.example/mcp", Name: "Impact", Scopes: []string{"pay"}, ScopeHints: map[string]*oauthdomain.PermissionImpact{"pay": impact}, State: oauthdomain.McpResourceServerActive, CreatedAt: f.Now, UpdatedAt: f.Now, AccessRules: []oauthdomain.ResourceAccessRule{{ResourceType: "idmagic_mcp_operation", Relation: "invoke", Scopes: []string{"pay"}, Tool: &oauthdomain.ResourceAccessTool{ID: "tool", Name: "pay", DisplayName: "Pay", Impact: impact}}}}
	if err := f.Servers.Save(ctx, server); err != nil {
		t.Fatal(err)
	}
	check := func() {
		t.Helper()
		got, err := f.Servers.FindByID(ctx, f.TenantID, f.ServerID)
		if err != nil || got == nil || !reflect.DeepEqual(got.ScopeHints, server.ScopeHints) || !reflect.DeepEqual(got.AccessRules[0].Tool.Impact, impact) {
			t.Fatalf("リソースとツールの保存=%+v err=%v", got, err)
		}
		got.ScopeHints["pay"].Changes = "none"
		*got.AccessRules[0].Tool.Impact.PersonalData = true
	}
	check()
	check()
	if leaked, err := f.Servers.FindByID(ctx, f.OtherTenantID, f.ServerID); err != nil || leaked != nil {
		t.Fatalf("リソースのテナント分離=%+v %v", leaked, err)
	}
	before := server.UpdatedAt
	server.UpdatedAt = before.Add(time.Second)
	server.ScopeHints = nil
	server.AccessRules[0].Tool.Impact = nil
	if saved, err := f.Servers.SaveIfUnchanged(ctx, server, before); err != nil || !saved {
		t.Fatalf("影響度の解除=%v %v", saved, err)
	}
	gotServer, err := f.Servers.FindByID(ctx, f.TenantID, f.ServerID)
	if err != nil || gotServer == nil || len(gotServer.ScopeHints) != 0 || gotServer.AccessRules[0].Tool.Impact != nil {
		t.Fatalf("解除の保存=%+v %v", gotServer, err)
	}
	detail.Impact = nil
	if err := f.Types.Save(ctx, detail); err != nil {
		t.Fatal(err)
	}
	got, err = f.Types.FindByType(ctx, f.TenantID, detail.Type)
	if err != nil || got == nil || got.Impact != nil {
		t.Fatalf("種類の解除=%+v %v", got, err)
	}
}
