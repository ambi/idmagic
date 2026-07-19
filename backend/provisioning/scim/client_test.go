package scim

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/provisioning/domain"
	"github.com/ambi/idmagic/backend/provisioning/ports"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return &Client{HTTPClient: server.Client(), BaseURL: server.URL, BearerToken: "test-token"}
}

// simpleRule builds a single attribute-source AttributeMappingRule mapping
// sourceKey directly to targetPath, for tests that only care about wire
// behavior (method, status handling) rather than mapping semantics (covered by
// TestBuildResource_* in mapping_test.go).
func simpleRule(targetPath, sourceKey string) domain.AttributeMappingRule {
	return domain.AttributeMappingRule{TargetPath: targetPath, SourceKind: domain.SourceKindAttribute, SourceKey: sourceKey, ApplyOn: domain.ApplyCreateAndUpdate}
}

func TestClient_Discover_ParsesCapabilities(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ServiceProviderConfig" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization header = %q, want Bearer test-token", got)
		}
		w.Header().Set("Content-Type", "application/scim+json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"patch":  map[string]any{"supported": true},
			"bulk":   map[string]any{"supported": false},
			"filter": map[string]any{"supported": true},
			"etag":   map[string]any{"supported": false},
			"sort":   map[string]any{"supported": true},
		})
	})
	caps, err := client.Discover(context.Background())
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if !caps.SupportsPatch || caps.SupportsBulk || !caps.SupportsFilter || caps.SupportsEtag || !caps.SupportsSort {
		t.Errorf("Discover() capabilities = %+v, want patch/filter/sort=true bulk/etag=false", caps)
	}
}

func TestClient_CreateUser_ReturnsRemoteID(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/Users" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["userName"] != "alice" {
			t.Errorf("request body userName = %v, want alice", body["userName"])
		}
		w.Header().Set("ETag", `"v1"`)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "remote-1", "userName": "alice"})
	})
	remoteID, etag, err := client.CreateUser(context.Background(), []domain.AttributeMappingRule{simpleRule("userName", "username")}, map[string]any{"username": "alice"})
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if remoteID != "remote-1" {
		t.Errorf("CreateUser() remoteID = %q, want remote-1", remoteID)
	}
	if etag == nil || *etag != `"v1"` {
		t.Errorf("CreateUser() etag = %v, want \"v1\"", etag)
	}
}

func TestClient_CreateUser_ReturnsConflictErrorOn409(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]any{"detail": "userName already exists"})
	})
	_, _, err := client.CreateUser(context.Background(), []domain.AttributeMappingRule{simpleRule("userName", "username")}, map[string]any{"username": "alice"})
	var conflict *ports.ConflictError
	if !ports.AsConflictError(err, &conflict) {
		t.Fatalf("CreateUser() error = %v, want *ports.ConflictError", err)
	}
}

func TestClient_UpdateUser_UsesPatchWhenSupported(t *testing.T) {
	var gotMethod string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		if r.URL.Path != "/Users/remote-1" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if r.Method == http.MethodPatch {
			ops, ok := body["Operations"].([]any)
			if !ok || len(ops) != 1 {
				t.Errorf("PATCH body = %+v, want single replace Operation", body)
			}
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "remote-1"})
	})
	if _, err := client.UpdateUser(context.Background(), "remote-1", []domain.AttributeMappingRule{simpleRule("active", "active")}, map[string]any{"active": false}, true); err != nil {
		t.Fatalf("UpdateUser() error = %v", err)
	}
	if gotMethod != http.MethodPatch {
		t.Errorf("UpdateUser() with supportsPatch=true used method %q, want PATCH", gotMethod)
	}
}

func TestClient_UpdateUser_FallsBackToPutWhenPatchUnsupported(t *testing.T) {
	var gotMethod string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "remote-1"})
	})
	if _, err := client.UpdateUser(context.Background(), "remote-1", []domain.AttributeMappingRule{simpleRule("active", "active")}, map[string]any{"active": false}, false); err != nil {
		t.Fatalf("UpdateUser() error = %v", err)
	}
	if gotMethod != http.MethodPut {
		t.Errorf("UpdateUser() with supportsPatch=false used method %q, want PUT", gotMethod)
	}
}

func TestClient_DeleteUser_TreatsNotFoundAsIdempotentSuccess(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	if err := client.DeleteUser(context.Background(), "remote-missing"); err != nil {
		t.Errorf("DeleteUser() on already-deleted resource = %v, want nil (idempotent)", err)
	}
}

func TestClient_DeleteUser_Success(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/Users/remote-1" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	if err := client.DeleteUser(context.Background(), "remote-1"); err != nil {
		t.Errorf("DeleteUser() error = %v", err)
	}
}

func TestClient_RetryAfter_ParsedFrom429(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
	})
	_, _, err := client.CreateUser(context.Background(), []domain.AttributeMappingRule{simpleRule("userName", "username")}, map[string]any{"username": "alice"})
	var retryable *ports.RetryableError
	if !ports.AsRetryableError(err, &retryable) {
		t.Fatalf("CreateUser() error = %v, want *ports.RetryableError", err)
	}
	if retryable.RetryAfter != 30*time.Second {
		t.Errorf("RetryableError.RetryAfter = %v, want 30s", retryable.RetryAfter)
	}
}

func TestClient_SearchUserByAttribute_FindsExisting(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Users" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("filter"); got != `userName eq "alice"` {
			t.Errorf("filter query = %q, want userName eq \"alice\"", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"totalResults": 1,
			"Resources":    []any{map[string]any{"id": "remote-1"}},
		})
	})
	remoteID, found, err := client.SearchUserByAttribute(context.Background(), "userName", "alice")
	if err != nil {
		t.Fatalf("SearchUserByAttribute() error = %v", err)
	}
	if !found || remoteID != "remote-1" {
		t.Errorf("SearchUserByAttribute() = (%q, %v), want (remote-1, true)", remoteID, found)
	}
}

func TestClient_SearchUserByAttribute_NotFound(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"totalResults": 0, "Resources": []any{}})
	})
	_, found, err := client.SearchUserByAttribute(context.Background(), "userName", "bob")
	if err != nil {
		t.Fatalf("SearchUserByAttribute() error = %v", err)
	}
	if found {
		t.Error("SearchUserByAttribute() found = true, want false")
	}
}

func TestClient_CreateGroup_ReturnsRemoteID(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/Groups" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "remote-group-1"})
	})
	remoteID, _, err := client.CreateGroup(context.Background(), []domain.AttributeMappingRule{simpleRule("displayName", "display_name")}, map[string]any{"display_name": "Engineers"})
	if err != nil {
		t.Fatalf("CreateGroup() error = %v", err)
	}
	if remoteID != "remote-group-1" {
		t.Errorf("CreateGroup() remoteID = %q, want remote-group-1", remoteID)
	}
}

func TestClient_PatchGroupMembers_SendsAddOperation(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/Groups/remote-group-1" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		ops, _ := body["Operations"].([]any)
		if len(ops) != 1 {
			t.Fatalf("PATCH body = %+v, want 1 operation", body)
		}
		op, _ := ops[0].(map[string]any)
		if op["op"] != "add" || op["path"] != "members" {
			t.Errorf("operation = %+v, want op=add path=members", op)
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "remote-group-1"})
	})
	if err := client.PatchGroupMembers(context.Background(), "remote-group-1", "add", []string{"remote-user-1"}); err != nil {
		t.Errorf("PatchGroupMembers() error = %v", err)
	}
}
