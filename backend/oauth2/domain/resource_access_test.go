package domain

import "testing"

//spec:covers REQ-OAUTH2-064: requires_approved_operation が明細の種類を必須とすること
func TestApprovedOperationRuleRequiresDetailType(t *testing.T) {
	rule := ResourceAccessRule{ResourceType: "document", Relation: "viewer", Scopes: []string{"s"}, RequiresApprovedOperation: true}
	server := McpResourceServer{Scopes: []string{"s"}, AccessRules: []ResourceAccessRule{rule}}
	if err := server.validateAccessRules(); err == nil {
		t.Fatal("rule without authorization_detail_type was accepted")
	}
	server.AccessRules[0].AuthorizationDetailType = "payment"
	if err := server.validateAccessRules(); err != nil {
		t.Fatal(err)
	}
}
