package domain_test

import (
	"reflect"
	"testing"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
)

//spec:covers REQ-OAUTH2-098: 同名スコープの変更度と個人データを保守的に集約し、対象リソースの限定と未登録を保持する。
func TestScopeHints(t *testing.T) {
	yes, no := true, false
	none := &oauthdomain.PermissionImpact{Changes: "none", PersonalData: &no}
	reversible := &oauthdomain.PermissionImpact{Changes: "reversible", PersonalData: &yes}
	irreversible := &oauthdomain.PermissionImpact{Changes: "irreversible", PersonalData: &no}
	a := &oauthdomain.McpResourceServer{Resource: "https://a.example", State: oauthdomain.McpResourceServerActive, Scopes: []string{"shared", "unknown", "reversible"}, ScopeHints: map[string]*oauthdomain.PermissionImpact{"shared": none, "unknown": none, "reversible": reversible}}
	b := &oauthdomain.McpResourceServer{Resource: "https://b.example", State: oauthdomain.McpResourceServerActive, Scopes: []string{"shared", "unknown", "reversible"}, ScopeHints: map[string]*oauthdomain.PermissionImpact{"shared": irreversible, "reversible": none}}
	disabled := &oauthdomain.McpResourceServer{State: oauthdomain.McpResourceServerDisabled, Scopes: []string{"shared"}}
	for _, order := range [][]*oauthdomain.McpResourceServer{{a, b, disabled}, {disabled, b, a}} {
		hints := oauthdomain.ScopeHints(order, "")
		if len(hints) != 2 || !reflect.DeepEqual(hints["shared"], irreversible) || !reflect.DeepEqual(hints["reversible"], reversible) {
			t.Fatalf("集約=%+v", hints)
		}
		*hints["shared"].PersonalData = true
		if *irreversible.PersonalData {
			t.Fatal("集約結果が登録値を共有")
		}
	}
	if hints := oauthdomain.ScopeHints([]*oauthdomain.McpResourceServer{a, b}, a.Resource); !reflect.DeepEqual(hints, a.ScopeHints) {
		t.Fatalf("リソースの限定=%+v", hints)
	}
	b.ScopeHints["shared"] = &oauthdomain.PermissionImpact{Changes: "future", PersonalData: &yes}
	if _, ok := oauthdomain.ScopeHints([]*oauthdomain.McpResourceServer{a, b}, "")["shared"]; ok {
		t.Fatal("未知の値を安全と表示")
	}
}
