package usecases_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/ambi/idmagic/backend/accesscontrol/internal/usecases"
	policyports "github.com/ambi/idmagic/backend/accesscontrol/ports"
)

type principalsByID map[string]bool

func (p principalsByID) IsPrincipalActive(_ context.Context, _, principalType, principalID string) (bool, error) {
	return p[principalType+":"+principalID], nil
}

func relationCheck() policyports.RelationCheck {
	return policyports.RelationCheck{
		TenantID: tenantID, ResourceType: "document", ResourceID: "d1", Relation: "viewer",
		Subject:       policyports.PrincipalRef{Type: "user", ID: "alice"},
		Actors:        []policyports.PrincipalRef{{Type: "agent", ID: "researcher"}},
		GrantedScopes: []string{"docs:read"}, RequiredScopes: []string{"docs:read"},
	}
}

func relationHarness(t *testing.T) *harness {
	t.Helper()
	h := newHarness(t)
	h.deps.Principals = principalsByID{"user:alice": true, "agent:researcher": true}
	h.write(t, viewerTuple("user", "alice"), viewerTuple("agent", "researcher"))
	return h
}

//spec:covers REQ-AUTHORIZATION-004: 主体と行為者の論理積、主体の有効性、スコープ
func TestRelationCheckerComposesTheSameRules(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*policyports.RelationCheck)
		active  principalsByID
		permits bool
		reason  string
	}{
		{name: "双方が関係を持つ", permits: true},
		{name: "無効な主体", active: principalsByID{"agent:researcher": true}, reason: "subject_principal_inactive"},
		{name: "無効な行為者", active: principalsByID{"user:alice": true}},
		{name: "関係のない行為者", mutate: func(c *policyports.RelationCheck) {
			c.Actors = []policyports.PrincipalRef{{Type: "agent", ID: "stranger"}}
		}, active: principalsByID{"user:alice": true, "agent:stranger": true}},
		{name: "スコープの不足", mutate: func(c *policyports.RelationCheck) { c.GrantedScopes = nil }},
		{name: "別の資源", mutate: func(c *policyports.RelationCheck) { c.ResourceID = "d2" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := relationHarness(t)
			if tc.active != nil {
				h.deps.Principals = tc.active
			}
			check := relationCheck()
			if tc.mutate != nil {
				tc.mutate(&check)
			}
			decision, err := usecases.RelationChecker{Deps: h.deps}.CheckRelation(context.Background(), check)
			if err != nil {
				t.Fatal(err)
			}
			if decision.Permitted != tc.permits {
				t.Fatalf("decision=%+v, want permitted=%v", decision, tc.permits)
			}
			if tc.reason != "" && !slices.Contains(decision.Reasons, tc.reason) {
				t.Fatalf("reasons=%v, want %s", decision.Reasons, tc.reason)
			}
		})
	}
}

//spec:covers REQ-AUTHORIZATION-005: モデルの未登録は不許可、ストアの障害はエラー
func TestRelationCheckerFailsClosed(t *testing.T) {
	bare := newBareHarness()
	bare.deps.Principals = principalsByID{"user:alice": true, "agent:researcher": true}
	decision, err := usecases.RelationChecker{Deps: bare.deps}.CheckRelation(context.Background(), relationCheck())
	if err != nil || decision.Permitted || !slices.Contains(decision.Reasons, "authorization_model_not_found") {
		t.Fatalf("モデルの未登録: decision=%+v err=%v", decision, err)
	}

	h := relationHarness(t)
	h.deps.Tuples = failingTuples{h.deps.Tuples}
	decision, err = usecases.RelationChecker{Deps: h.deps}.CheckRelation(context.Background(), relationCheck())
	if !errors.Is(err, errStoreDown) || decision.Permitted {
		t.Fatalf("ストアの障害: decision=%+v err=%v", decision, err)
	}
}
