package client_scim

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/ambi/idmagic/backend/provisioning/domain"
	"github.com/ambi/idmagic/backend/provisioning/ports"
)

// docs/contexts/provisioning/standards.md が宣言する外向き SCIM の準拠範囲のうち、
// 「送らない」と「この範囲までしか送らない」を確かめる証拠を集める。
//
// 個々の操作が正しく送れることは client_test.go が持つ。ここが見るのは、
// 下流が受け取った要求の全体に対して、宣言した範囲の外側が現れないことである。

// recordedRequest は下流が受け取った要求のうち、規範の証拠に必要な部分だけを写す。
type recordedRequest struct {
	Method string
	Path   string
	Query  url.Values
	Header http.Header
	Body   []byte
}

// lifecycleOperationCount は fullLifecycleRequests が通す送出経路の数。
// 否定テストが「そもそも何も送っていないから通った」で成立しないための錘である。
const lifecycleOperationCount = 6

// newRecordingClient は、下流が受け取った要求を順に記録するクライアントを返す。
// 応答はどの capability も真として広告する。宣言した excluded が下流の対応状況に
// よらないことを見るには、下流が「対応している」と言っている状況で確かめる必要がある。
func newRecordingClient(t *testing.T) (*Client, *[]recordedRequest) {
	t.Helper()
	recorded := &[]recordedRequest{}
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		*recorded = append(*recorded, recordedRequest{
			Method: r.Method, Path: r.URL.Path, Query: r.URL.Query(), Header: r.Header.Clone(), Body: body,
		})
		w.Header().Set("Content-Type", "application/scim+json")
		switch {
		case r.URL.Path == "/ServiceProviderConfig":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"patch":  map[string]any{"supported": true},
				"bulk":   map[string]any{"supported": true},
				"filter": map[string]any{"supported": true},
				"etag":   map[string]any{"supported": true},
				"sort":   map[string]any{"supported": true},
			})
		case r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"totalResults": 1,
				"Resources":    []any{map[string]any{"id": "remote-1"}},
			})
		case r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPost:
			w.Header().Set("ETag", `"v1"`)
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "remote-1"})
		default:
			w.Header().Set("ETag", `"v2"`)
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "remote-1"})
		}
	})
	return client, recorded
}

// fullLifecycleRequests は 1 人の User に対する送出経路を 1 度ずつ通し、
// 下流が受け取った要求を返す。PATCH と PUT の両方を通すのは、更新の 2 経路の
// どちらにも宣言した範囲外が現れないことを見るためである。
func fullLifecycleRequests(t *testing.T) []recordedRequest {
	t.Helper()
	client, recorded := newRecordingClient(t)
	ctx := context.Background()
	rules := []domain.AttributeMappingRule{simpleRule("userName", "username")}
	attrs := map[string]any{"username": "alice"}

	if _, err := client.Discover(ctx); err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if _, _, err := client.CreateUser(ctx, rules, attrs); err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if _, err := client.UpdateUser(ctx, "remote-1", rules, attrs, true); err != nil {
		t.Fatalf("UpdateUser(patch) error = %v", err)
	}
	if _, err := client.UpdateUser(ctx, "remote-1", rules, attrs, false); err != nil {
		t.Fatalf("UpdateUser(put) error = %v", err)
	}
	if _, _, err := client.SearchUserByAttribute(ctx, "userName", "alice"); err != nil {
		t.Fatalf("SearchUserByAttribute() error = %v", err)
	}
	if err := client.DeleteUser(ctx, "remote-1"); err != nil {
		t.Fatalf("DeleteUser() error = %v", err)
	}

	if len(*recorded) != lifecycleOperationCount {
		t.Fatalf("下流が受け取った要求 = %d 件, want %d 件", len(*recorded), lifecycleOperationCount)
	}
	return *recorded
}

// RFC7644-OUT-BULK: /Bulk を使わない。下流が bulk.supported を広告していても、
// 配信 1 件につき 1 リソースの要求を送る。
func TestClient_SendsNoBulkRequest(t *testing.T) {
	for _, request := range fullLifecycleRequests(t) {
		if strings.HasPrefix(request.Path, "/Bulk") {
			t.Errorf("%s %s: /Bulk は使わない", request.Method, request.Path)
		}
	}
}

// RFC7644-OUT-SORT: 照会に sortBy と sortOrder を送らない。
func TestClient_SendsNoSortParameters(t *testing.T) {
	for _, request := range fullLifecycleRequests(t) {
		for _, name := range []string{"sortBy", "sortOrder"} {
			if request.Query.Has(name) {
				t.Errorf("%s %s: %s=%q を送っている", request.Method, request.Path, name, request.Query.Get(name))
			}
		}
	}
}

// RFC7644-OUT-ETAG: 条件付き要求を送らない。作成と更新の応答が ETag を返していても、
// 後続の要求の前提条件には使わない。
func TestClient_SendsNoConditionalRequestHeaders(t *testing.T) {
	for _, request := range fullLifecycleRequests(t) {
		for _, name := range []string{"If-Match", "If-None-Match", "If-Unmodified-Since"} {
			if value := request.Header.Get(name); value != "" {
				t.Errorf("%s %s: %s: %s を送っている", request.Method, request.Path, name, value)
			}
		}
	}
}

// RFC7644-OUT-DISCOVERY: 取得するのは /ServiceProviderConfig だけであり、
// /ResourceTypes と /Schemas は取得しない。下流が広告するスキーマに送出内容を合わせない
// という宣言は、そのスキーマを読んでいないことによって支えられている。
func TestClient_DiscoversOnlyServiceProviderConfig(t *testing.T) {
	client, recorded := newRecordingClient(t)
	if _, err := client.Discover(context.Background()); err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	for _, request := range *recorded {
		if request.Path == "/ResourceTypes" || request.Path == "/Schemas" {
			t.Errorf("%s を取得している", request.Path)
		}
	}
	if len(*recorded) != 1 || (*recorded)[0].Path != "/ServiceProviderConfig" {
		t.Errorf("Discover() が送った要求 = %+v, want /ServiceProviderConfig 1 件", *recorded)
	}
}

// RFC7644-OUT-AUTHENTICATION: 認証は接続に保存した資格情報を Authorization: Bearer で
// 提示する 1 方式だけであり、資格情報を得るための別の要求は送らない。
// 送った要求が操作の数と一致することが、トークンエンドポイントを呼んでいないことの証拠になる。
func TestClient_SendsOneAuthenticatedRequestPerOperation(t *testing.T) {
	requests := fullLifecycleRequests(t)
	for _, request := range requests {
		if got := request.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("%s %s: Authorization = %q, want Bearer test-token", request.Method, request.Path, got)
		}
	}
	if len(requests) != lifecycleOperationCount {
		t.Errorf("要求 = %d 件, want %d 件 (資格情報を得るための追加要求があってはならない)", len(requests), lifecycleOperationCount)
	}
}

// RFC7644-OUT-RESOURCE-OPERATIONS: 要求は Accept: application/scim+json を持ち、
// 本文を伴う要求は Content-Type: application/scim+json を持つ。
func TestClient_SendsScimMediaTypes(t *testing.T) {
	for _, request := range fullLifecycleRequests(t) {
		if got := request.Header.Get("Accept"); got != "application/scim+json" {
			t.Errorf("%s %s: Accept = %q, want application/scim+json", request.Method, request.Path, got)
		}
		contentType := request.Header.Get("Content-Type")
		if len(request.Body) > 0 && contentType != "application/scim+json" {
			t.Errorf("%s %s: 本文があるのに Content-Type = %q", request.Method, request.Path, contentType)
		}
		if len(request.Body) == 0 && contentType != "" {
			t.Errorf("%s %s: 本文が無いのに Content-Type = %q", request.Method, request.Path, contentType)
		}
	}
}

// RFC7644-OUT-FILTERING: 組み立てるのは <属性> eq "<値>" という比較 1 つだけで、
// 論理演算子もグループ化も組み立てない。値に空白が含まれていても比較は 1 つのままである。
func TestClient_SearchBuildsSingleEqualityFilter(t *testing.T) {
	client, recorded := newRecordingClient(t)
	if _, _, err := client.SearchUserByAttribute(context.Background(), "userName", "alice smith"); err != nil {
		t.Fatalf("SearchUserByAttribute() error = %v", err)
	}
	if len(*recorded) != 1 {
		t.Fatalf("要求 = %d 件, want 1 件", len(*recorded))
	}
	filter := (*recorded)[0].Query.Get("filter")
	if filter != `userName eq "alice smith"` {
		t.Errorf("filter = %q, want `userName eq \"alice smith\"`", filter)
	}
	for _, operator := range []string{" and ", " or ", " not ", "(", "["} {
		if strings.Contains(filter, operator) {
			t.Errorf("filter = %q: %q を組み立てている", filter, operator)
		}
	}
}

// RFC7644-OUT-ERROR-RESPONSE: 409 / 404 / 429 / 5xx 以外の 2xx でない応答は、
// 再試行しない失敗として扱う。400 を再試行可能と読むと、拒否された本文を
// 上限まで送り続けることになる。
func TestClient_UnknownErrorStatusIsNotRetryable(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"detail": "invalidValue", "scimType": "invalidValue"})
	})
	_, _, err := client.CreateUser(context.Background(), []domain.AttributeMappingRule{simpleRule("userName", "username")}, map[string]any{"username": "alice"})
	if err == nil {
		t.Fatal("CreateUser() on 400 should fail")
	}
	var retryable *ports.RetryableError
	if ports.AsRetryableError(err, &retryable) {
		t.Errorf("CreateUser() on 400 error = %v, 再試行可能として扱ってはならない", err)
	}
	var conflict *ports.ConflictError
	if ports.AsConflictError(err, &conflict) {
		t.Errorf("CreateUser() on 400 error = %v, 衝突として扱ってはならない", err)
	}
	var notFound *ports.NotFoundError
	if ports.AsNotFoundError(err, &notFound) {
		t.Errorf("CreateUser() on 400 error = %v, 消失として扱ってはならない", err)
	}
	if !strings.Contains(err.Error(), "invalidValue") {
		t.Errorf("CreateUser() on 400 error = %v, SCIM エラー本文の detail を含めるべき", err)
	}
}

// RFC7643-OUT-CORE-RESOURCES: RFC 7643 §3 はリソース表現が `schemas` を持つことを
// 要求する。User なら `urn:ietf:params:scim:schemas:core:2.0:User` である。
// 受け取り側が検証する実装なら、欠けていれば作成も置換も 400 で拒否され、
// その拒否は「再試行しない失敗」として dead_letter に落ちる。IdMagic 自身の内向き
// サーバーは `schemas` を読み取り専用属性として無視するので、IdMagic どうしを
// 繋いだ試験では再現しない。
func TestClient_SendsSchemasOnResourceRepresentations(t *testing.T) {
	const userSchemaURN = "urn:ietf:params:scim:schemas:core:2.0:User"
	const patchOpURN = "urn:ietf:params:scim:api:messages:2.0:PatchOp"

	schemasOf := func(t *testing.T, body []byte) []string {
		t.Helper()
		var doc struct {
			Schemas []string `json:"schemas"`
		}
		if err := json.Unmarshal(body, &doc); err != nil {
			t.Fatalf("本文を解釈できない: %v (body=%s)", err, body)
		}
		return doc.Schemas
	}

	for _, request := range fullLifecycleRequests(t) {
		switch {
		case request.Method == http.MethodPost && request.Path == "/Users":
			// 作成のリソース表現。
			if got := schemasOf(t, request.Body); len(got) != 1 || got[0] != userSchemaURN {
				t.Fatalf("POST /Users の schemas = %v, want [%s] (body=%s)", got, userSchemaURN, request.Body)
			}
		case request.Method == http.MethodPut:
			// 置換もリソース表現なので、作成と同じ URN を持つ。
			if got := schemasOf(t, request.Body); len(got) != 1 || got[0] != userSchemaURN {
				t.Fatalf("PUT %s の schemas = %v, want [%s] (body=%s)", request.Path, got, userSchemaURN, request.Body)
			}
		case request.Method == http.MethodPatch:
			// PATCH の本文はメッセージであってリソース表現ではないので、
			// PatchOp の URN を持つ。中の value は部分断片なので `schemas` を持たない。
			if got := schemasOf(t, request.Body); len(got) != 1 || got[0] != patchOpURN {
				t.Fatalf("PATCH %s の schemas = %v, want [%s] (body=%s)", request.Path, got, patchOpURN, request.Body)
			}
			var patch struct {
				Operations []struct {
					Value map[string]any `json:"value"`
				} `json:"Operations"`
			}
			if err := json.Unmarshal(request.Body, &patch); err != nil {
				t.Fatalf("PatchOp を解釈できない: %v (body=%s)", err, request.Body)
			}
			if len(patch.Operations) != 1 {
				t.Fatalf("Operations = %d 件, want 1 (body=%s)", len(patch.Operations), request.Body)
			}
			if _, present := patch.Operations[0].Value["schemas"]; present {
				t.Fatalf("PatchOp の value は部分断片なので schemas を持ってはならない (body=%s)", request.Body)
			}
		}
	}
}
