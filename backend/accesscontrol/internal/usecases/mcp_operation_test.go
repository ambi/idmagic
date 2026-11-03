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
	_, err := usecases.PutAuthorizationModel(context.Background(), h.deps, tenantID, []domain.ResourceTypeDefinition{{Name: "idmagic_mcp_operation"}}, now)
	if err == nil {
		t.Fatal("reserved type was published")
	}
}
