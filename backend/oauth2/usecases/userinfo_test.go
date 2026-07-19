package usecases_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/adapters/persistence/memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/oauth2/usecases"
)

func userInfoFixture(t *testing.T) *usermemory.UserRepository {
	t.Helper()
	repo := usermemory.NewUserRepository()
	repo.Seed(&userdomain.User{
		ID: "user-1", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "carol",
		Name: new("Carol Q"), Email: new("carol@example.com"), EmailVerified: true,
		Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive},
		Attributes: map[string]userdomain.AttributeValue{
			"nickname":     {Type: idmdomain.AttributeTypeString, String: new("cici")},
			"phone_number": {Type: idmdomain.AttributeTypeString, String: new("+819012345678")},
		},
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	})
	return repo
}

func resolveBuiltin(_ context.Context, _ string) ([]userdomain.UserAttributeDef, error) {
	return userdomain.BuiltinUserAttributeDefs(), nil
}

func TestUserInfoIncludesAttributeClaimsByScope(t *testing.T) {
	repo := userInfoFixture(t)
	res, err := usecases.UserInfo(context.Background(), repo, nil, usecases.UserInfoInput{
		Scopes: []string{"openid", "profile", "phone"}, Sub: "user-1", Active: true, ClientID: "c1",
		ResolveAttributeDefs: resolveBuiltin,
	})
	if err != nil {
		t.Fatal(err)
	}
	// MarshalJSON が標準 claim と属性 claim をマージすることを確認する。
	raw, err := json.Marshal(res)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got["name"] != "Carol Q" {
		t.Fatalf("standard claim missing: %#v", got)
	}
	if got["nickname"] != "cici" {
		t.Fatalf("nickname claim missing: %#v", got)
	}
	if got["phone_number"] != "+819012345678" {
		t.Fatalf("phone_number claim missing: %#v", got)
	}
}

func TestUserInfoOmitsAttributeClaimsWithoutScope(t *testing.T) {
	repo := userInfoFixture(t)
	res, err := usecases.UserInfo(context.Background(), repo, nil, usecases.UserInfoInput{
		Scopes: []string{"openid"}, Sub: "user-1", Active: true, ClientID: "c1",
		ResolveAttributeDefs: resolveBuiltin,
	})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(res)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if _, ok := got["nickname"]; ok {
		t.Fatalf("nickname leaked without profile scope: %#v", got)
	}
	if _, ok := got["phone_number"]; ok {
		t.Fatalf("phone_number leaked without phone scope: %#v", got)
	}
}
