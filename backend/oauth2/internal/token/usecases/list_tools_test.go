package usecases

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	accessports "github.com/ambi/idmagic/backend/accesscontrol/ports"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
)

type listedToolRelations struct {
	fail string
}

func (r listedToolRelations) CheckRelation(_ context.Context, check accessports.RelationCheck) (accessports.RelationDecision, error) {
	if strings.HasSuffix(check.ResourceID, r.fail) && r.fail != "" {
		return accessports.RelationDecision{}, errors.New("assignment storage unavailable")
	}
	allowed := !strings.HasSuffix(check.ResourceID, "/write")
	for _, scope := range check.RequiredScopes {
		allowed = allowed && slices.Contains(check.GrantedScopes, scope)
	}
	return accessports.RelationDecision{Permitted: allowed}, nil
}

//spec:covers REQ-OAUTH2-099: 一覧は割当とスコープだけを使い、引数の必要な承認、条件と回数の依存を呼ばない。
func TestListedToolsUseAssignmentsWithoutExecutingRules(t *testing.T) {
	rules := []oauthdomain.ResourceAccessRule{}
	for _, name := range []string{"write", "read", "other_scope"} {
		rule := evalRule
		rule.Tool = &oauthdomain.ResourceAccessTool{ID: name, Name: name, DisplayName: name, ResourceIDArgument: "document_id"}
		rule.RequiresApprovedOperation = true
		rule.AuthorizationDetailType = "approval"
		if name == "other_scope" {
			rule.Scopes = []string{"other"}
		}
		rules = append(rules, rule)
	}
	deps := ResourceAccessDeps{Servers: evalServer(t, rules...), Tokens: evalTokens{result: evalAgentToken()}, Relations: listedToolRelations{}}
	in := evalRequest()
	in.ResourceType, in.Relation, in.ResourceID = "idmagic_mcp_method", "access", "tools/list"
	result, err := EvaluateResourceAccess(context.Background(), deps, in, time.Now())
	if err != nil || !result.Permitted || !slices.Equal(result.AllowedTools, []string{"read"}) || result.ApprovedOperation != nil || result.TokenExchangeGrant != "" {
		t.Fatalf("list executed rules or returned unassigned tools: %+v %v", result, err)
	}
	deps.Relations = listedToolRelations{fail: "/other_scope"}
	if _, err := EvaluateResourceAccess(context.Background(), deps, in, time.Now()); err == nil {
		t.Fatal("returned a partial list after an assignment failure")
	}
}
