package usecases_test

import (
	"context"
	"testing"

	"github.com/ambi/idmagic/backend/accesscontrol/internal/domain"
	"github.com/ambi/idmagic/backend/accesscontrol/internal/ports"
	"github.com/ambi/idmagic/backend/accesscontrol/internal/usecases"
	policyports "github.com/ambi/idmagic/backend/accesscontrol/ports"
)

//spec:covers REQ-AUTHORIZATION-016: モデルを発行せず、割当と全行為者、現在の状態、scopeを合成する
func TestMcpOperationWithoutPublishedModel(t *testing.T) {
	h := newBareHarness()
	h.deps.Principals = principalsByID{"agent:reader": true, "agent:parent": true}
	tuple := domain.RelationTuple{Resource: domain.ObjectRef{Type: "idmagic_mcp_operation", ID: "server-a/read_document"}, Relation: "invoke", Subject: domain.SubjectRef{Type: "agent", ID: "reader"}}
	parent := tuple
	parent.Subject.ID = "parent"
	_, err := usecases.WriteRelationTuples(context.Background(), h.deps, tenantID, ports.TupleWrite{Writes: []domain.RelationTuple{tuple, parent}}, now)
	if err != nil {
		t.Fatal(err)
	}
	check := policyports.RelationCheck{TenantID: tenantID, ResourceType: "idmagic_mcp_operation", ResourceID: "server-a/read_document", Relation: "invoke", Subject: policyports.PrincipalRef{Type: "agent", ID: "reader"}, Actors: []policyports.PrincipalRef{{Type: "agent", ID: "parent"}}, GrantedScopes: []string{"docs:read"}, RequiredScopes: []string{"docs:read"}}
	assertDecision := func(want bool) {
		t.Helper()
		got, err := (usecases.RelationChecker{Deps: h.deps}).CheckRelation(context.Background(), check)
		if err != nil || got.Permitted != want {
			t.Fatalf("decision=%+v err=%v want=%v", got, err, want)
		}
	}
	assertDecision(true)
	check.ResourceID = "server-b/read_document"
	assertDecision(false)
	check.ResourceID = "server-a/read_document"
	check.GrantedScopes = nil
	assertDecision(false)
	check.GrantedScopes = []string{"docs:read"}
	h.deps.Principals = principalsByID{"agent:reader": true}
	assertDecision(false)
	h.deps.Principals = principalsByID{"agent:reader": true, "agent:parent": true}
	_, err = usecases.WriteRelationTuples(context.Background(), h.deps, tenantID, ports.TupleWrite{Deletes: []domain.RelationTuple{parent}}, now)
	if err != nil {
		t.Fatal(err)
	}
	assertDecision(false)
}

//spec:covers REQ-AUTHORIZATION-001: 利用者のモデルから組み込み定義を上書きできない
func TestMcpOperationModelCannotBeOverridden(t *testing.T) {
	h := newBareHarness()
	_, err := usecases.PutAuthorizationModel(context.Background(), h.deps, tenantID, 0, []domain.ResourceTypeDefinition{{Name: "idmagic_mcp_operation"}}, now)
	if err == nil {
		t.Fatal("reserved type was published")
	}
}

//spec:covers REQ-AUTHORIZATION-017: 対象の範囲は利用者のモデルなしに組み込みの型へ書け、全対象と指定した対象を区別して判定する。
func TestMcpOperationTargetRangeWithoutPublishedModel(t *testing.T) {
	h := newBareHarness()
	h.deps.Principals = principalsByID{"agent:reader": true}
	subject := domain.SubjectRef{Type: "agent", ID: "reader"}
	anyTarget := domain.RelationTuple{Resource: domain.ObjectRef{Type: "idmagic_mcp_operation", ID: "server-a/op-read"}, Relation: "invoke_any_target", Subject: subject}
	oneTarget := domain.RelationTuple{Resource: domain.ObjectRef{Type: "idmagic_mcp_operation", ID: "server-a/op-read/handbook"}, Relation: "invoke", Subject: subject}
	if _, err := usecases.WriteRelationTuples(context.Background(), h.deps, tenantID, ports.TupleWrite{Writes: []domain.RelationTuple{anyTarget, oneTarget}}, now); err != nil {
		t.Fatal(err)
	}
	decide := func(resourceID, relation string) bool {
		t.Helper()
		got, err := (usecases.RelationChecker{Deps: h.deps}).CheckRelation(context.Background(), policyports.RelationCheck{TenantID: tenantID, ResourceType: "idmagic_mcp_operation", ResourceID: resourceID, Relation: relation, Subject: policyports.PrincipalRef{Type: "agent", ID: "reader"}})
		if err != nil {
			t.Fatal(err)
		}
		return got.Permitted
	}
	if !decide("server-a/op-read", "invoke_any_target") || !decide("server-a/op-read/handbook", "invoke") {
		t.Fatal("written ranges were not permitted")
	}
	if decide("server-a/op-read/salary", "invoke") || decide("server-a/op-read", "invoke") {
		t.Fatal("a range permitted an unassigned target or the operation itself")
	}
	if _, err := usecases.WriteRelationTuples(context.Background(), h.deps, tenantID, ports.TupleWrite{Writes: []domain.RelationTuple{{Resource: anyTarget.Resource, Relation: "owner", Subject: subject}}}, now); err == nil {
		t.Fatal("an undeclared relation on the reserved type was accepted")
	}
}
