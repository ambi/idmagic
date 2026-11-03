package usecases

import (
	"context"
	"testing"
	"time"

	policyports "github.com/ambi/idmagic/backend/accesscontrol/ports"
	"github.com/ambi/idmagic/backend/oauth2/domain"
)

type operationRelations struct {
	targetPermitted bool
	got             []policyports.RelationCheck
}

func (r *operationRelations) CheckRelation(_ context.Context, check policyports.RelationCheck) (policyports.RelationDecision, error) {
	r.got = append(r.got, check)
	return policyports.RelationDecision{Permitted: check.ResourceType == "idmagic_mcp_operation" || r.targetPermitted}, nil
}

//spec:covers REQ-OAUTH2-083: 操作割当から対象別の拒否を迂回せず、対応のない操作や引数を拒否する。
func TestAssignedOperationAccess(t *testing.T) {
	rule := evalRule
	rule.Tool = &domain.ResourceAccessTool{ID: "operation-" + "read_document", Name: "read_document", DisplayName: "閲覧", ResourceIDArgument: "document_id"}
	deps := ResourceAccessDeps{Servers: evalServer(t, rule), Tokens: evalTokens{result: evalAgentToken()}}
	check := &operationRelations{}
	deps.Relations = check
	in := evalRequest()
	in.ResourceType = "idmagic_mcp_operation"
	in.Relation = "invoke"
	in.ResourceID = "read_document"
	in.OperationParameters = `{"document_id":"d1"}`
	result, err := EvaluateResourceAccess(context.Background(), deps, in, time.Now())
	if err != nil || result.Permitted || len(check.got) != 2 {
		t.Fatalf("target denied: %+v %v checks=%v", result, err, check.got)
	}
	if check.got[0].ResourceID != "rs-1/operation-read_document" || check.got[1].ResourceID != "d1" {
		t.Fatalf("binding=%v", check.got)
	}
	check.targetPermitted = true
	result, err = EvaluateResourceAccess(context.Background(), deps, in, time.Now())
	if err != nil || !result.Permitted {
		t.Fatalf("permit: %+v %v", result, err)
	}
	for _, args := range []string{`{}`, `{"document_id":2}`, `{"document_id":"d1","document_id":"d2"}`} {
		in.OperationParameters = args
		check.got = nil
		result, err = EvaluateResourceAccess(context.Background(), deps, in, time.Now())
		if err != nil || result.Permitted || len(check.got) != 0 {
			t.Fatalf("arguments=%s: %+v %v checks=%v", args, result, err, check.got)
		}
	}
	in = evalRequest()
	check.got = nil
	result, err = EvaluateResourceAccess(context.Background(), deps, in, time.Now())
	if err != nil || result.Permitted || len(check.got) != 0 {
		t.Fatalf("legacy bypass: %+v %v", result, err)
	}
}

//spec:covers REQ-OAUTH2-083: 業務対象の型名から接続準備へ誤分類しない。
func TestToolTargetCannotBecomeProtocolPreparation(t *testing.T) {
	rule := evalRule
	rule.ResourceType = "idmagic_mcp_method"
	rule.Tool = &domain.ResourceAccessTool{ID: "operation-read", Name: "read_document", DisplayName: "閲覧", ResourceIDArgument: "id"}
	check := &operationRelations{}
	deps := ResourceAccessDeps{Servers: evalServer(t, rule), Tokens: evalTokens{result: evalAgentToken()}, Relations: check}
	in := evalRequest()
	in.ResourceType = "idmagic_mcp_operation"
	in.Relation = "invoke"
	in.ResourceID = "read_document"
	in.OperationParameters = `{"id":"tools/list"}`
	result, err := EvaluateResourceAccess(context.Background(), deps, in, time.Now())
	if err != nil || result.Permitted || len(check.got) != 2 || check.got[1].ResourceType != "idmagic_mcp_method" {
		t.Fatalf("decision=%+v err=%v checks=%v", result, err, check.got)
	}
}
