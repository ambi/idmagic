package usecases_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	claimdomain "github.com/ambi/idmagic/backend/claimmapping/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/oauth2/token/usecases"
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
			"department":   {Type: idmdomain.AttributeTypeString, String: new("R&D")},
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

// TestUserInfo_ClaimPolicyAddsOverrideClaim covers wi-73's motivating example: an
// Application-level claim_policy can release a SelfReadable attribute (department)
// that no OIDC standard scope exposes, without the caller granting any extra scope.
func TestUserInfo_ClaimPolicyAddsOverrideClaim(t *testing.T) {
	repo := userInfoFixture(t)
	policy := claimdomain.ClaimMappingPolicy{
		NameID: claimdomain.NameIdConfiguration{Format: "sub", SourceAttribute: "user_id"},
		Rules: []claimdomain.ClaimMappingRule{
			{ClaimType: "department", Source: claimdomain.ClaimSourceUserAttribute, SourceKey: "department", Required: true},
		},
	}
	res, err := usecases.UserInfo(context.Background(), repo, nil, usecases.UserInfoInput{
		Scopes: []string{"openid"}, Sub: "user-1", Active: true, ClientID: "c1",
		ResolveAttributeDefs: resolveBuiltin, ClaimPolicy: &policy,
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
	if got["department"] != "R&D" {
		t.Fatalf("expected claim_policy override claim, got %#v", got)
	}
}

// TestUserInfo_ClaimPolicyRejectsPrivateAttribute verifies the fail-closed floor
// rejects a claim_policy rule that targets a Private-visibility attribute,
// even though it is scoped to a single Application.
func TestUserInfo_ClaimPolicyRejectsPrivateAttribute(t *testing.T) {
	repo := userInfoFixture(t)
	policy := claimdomain.ClaimMappingPolicy{
		NameID: claimdomain.NameIdConfiguration{Format: "sub", SourceAttribute: "user_id"},
		Rules: []claimdomain.ClaimMappingRule{
			{ClaimType: "leak", Source: claimdomain.ClaimSourceUserAttribute, SourceKey: "not_a_defined_attribute"},
		},
	}
	_, err := usecases.UserInfo(context.Background(), repo, nil, usecases.UserInfoInput{
		Scopes: []string{"openid"}, Sub: "user-1", Active: true, ClientID: "c1",
		ResolveAttributeDefs: resolveBuiltin, ClaimPolicy: &policy,
	})
	if err == nil {
		t.Fatal("expected fail-closed rejection for undefined attribute source, got nil")
	}
}
