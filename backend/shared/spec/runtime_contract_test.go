package spec

import "testing"

func TestNormalizeRoutePath(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"no params", "/authorize", "/authorize"},
		{"contract-style param", "/api/admin/v1/users/{user_id}", "/api/admin/v1/users/{}"},
		{"router-style param", "/api/admin/v1/users/:user_id", "/api/admin/v1/users/{}"},
		{"multiple params", "/api/admin/v1/groups/{group_id}/members/{user_id}", "/api/admin/v1/groups/{}/members/{}"},
		{"mixed styles normalize the same", "/api/admin/v1/groups/:group_id/members/:user_id", "/api/admin/v1/groups/{}/members/{}"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := NormalizeRoutePath(tc.in); got != tc.want {
				t.Fatalf("NormalizeRoutePath(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestRuntimeContractOperation(t *testing.T) {
	contract := CurrentRuntimeContract()
	if _, ok := contract.Operation("no-such-operation"); ok {
		t.Fatal("expected ok=false for an unknown operation name")
	}
	op, ok := contract.Operation("Authorize")
	if !ok || op.Method != "GET" || op.Path != "/authorize" {
		t.Fatalf("Operation(Authorize) = %+v, %v", op, ok)
	}

	var nilContract *RuntimeContract
	if _, ok := nilContract.Operation("Authorize"); ok {
		t.Fatal("expected ok=false for a nil contract")
	}
}

func TestRuntimeContractOperationForRoute(t *testing.T) {
	contract := CurrentRuntimeContract()

	op, ok := contract.OperationForRoute("get", "/authorize")
	if !ok || op.Method != "GET" {
		t.Fatalf("OperationForRoute(GET /authorize) = %+v, %v", op, ok)
	}

	// The router's path-parameter spelling differs from the contract's; both
	// must resolve to the same operation once normalized.
	op, ok = contract.OperationForRoute("POST", "/api/admin/v1/groups/:group_id/members/:user_id")
	if !ok || op.Method != "POST" || op.Path != "/api/admin/v1/groups/{group_id}/members/{user_id}" {
		t.Fatalf("OperationForRoute(router path) = %+v, %v", op, ok)
	}

	if _, ok := contract.OperationForRoute("GET", "/no/such/route"); ok {
		t.Fatal("expected ok=false for an unmatched route")
	}

	// The route index is built lazily via sync.Once; a second lookup must
	// still resolve correctly from the cached index.
	op2, ok2 := contract.OperationForRoute("GET", "/authorize")
	if !ok2 || op2.Method != "GET" || op2.Path != "/authorize" {
		t.Fatalf("second OperationForRoute lookup diverged: %+v, %v", op2, ok2)
	}

	var nilContract *RuntimeContract
	if _, ok := nilContract.OperationForRoute("GET", "/authorize"); ok {
		t.Fatal("expected ok=false for a nil contract")
	}
}
