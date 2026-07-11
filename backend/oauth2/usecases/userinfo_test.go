package usecases_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	idmmemory "github.com/ambi/idmagic/backend/identitymanagement/adapters/persistence/memory"

	idmdomain "github.com/ambi/idmagic/backend/identitymanagement/domain"

	"github.com/ambi/idmagic/backend/oauth2/usecases"
	"github.com/ambi/idmagic/backend/shared/spec"
)

func userInfoFixture(t *testing.T) *idmmemory.UserRepository {
	t.Helper()
	repo := idmmemory.NewUserRepository()
	repo.Seed(&idmdomain.User{
		ID: "user-1", TenantID: spec.DefaultTenantID, PreferredUsername: "carol",
		Name: new("Carol Q"), Email: new("carol@example.com"), EmailVerified: true,
		Lifecycle: idmdomain.UserLifecycle{Status: idmdomain.UserStatusActive},
		Attributes: map[string]idmdomain.AttributeValue{
			"nickname":     {Type: idmdomain.AttributeTypeString, String: new("cici")},
			"phone_number": {Type: idmdomain.AttributeTypeString, String: new("+819012345678")},
		},
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	})
	return repo
}

func resolveBuiltin(_ context.Context, _ string) ([]idmdomain.UserAttributeDef, error) {
	return idmdomain.BuiltinUserAttributeDefs(), nil
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
