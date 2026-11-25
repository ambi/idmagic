package testing_scimtwin

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"maps"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// Twin は User と Group を保持する SCIM 2.0 のサービスプロバイダーの模倣である。
//
// 記録にない操作の組み合わせ、部分的な失敗、429 の連続を作るために使う。
// 応答の形は記録と一致することをこのパッケージのテストが確かめる。
type Twin struct {
	mu        sync.Mutex
	resources map[string]map[string]map[string]any // 種類 (Users、Groups) → id → 表現
	nextID    int
	faults    []Fault
	// PatchReturnsResource が真なら PATCH に 200 と表現を返し、偽なら 204 を返す。
	// RFC 7644 §3.5.2 はどちらも許し、§3.5.2.1 の例は 204 を返すので、既定は 204 である。
	PatchReturnsResource bool
	// Capabilities は /ServiceProviderConfig が広告する対応可否である。
	Capabilities Capabilities
}

// Capabilities は ServiceProviderConfig の五つの supported である。
type Capabilities struct {
	Patch, Bulk, Filter, Etag, Sort, ChangePassword bool
}

// Fault は次の要求に返す障害である。
type Fault struct {
	// Status は返す状態コードである。0 なら状態コードは正常の処理に任せ、本文だけを壊す。
	Status int
	// RetryAfter は Retry-After ヘッダーの値である。空なら載せない。
	RetryAfter string
	// MalformedBody が真なら、JSON として解釈できない本文を返す。
	MalformedBody bool
}

// NewTwin は空の模倣を作る。
func NewTwin() *Twin {
	return &Twin{
		resources:    map[string]map[string]map[string]any{"Users": {}, "Groups": {}},
		Capabilities: Capabilities{Patch: true, Filter: true},
	}
}

// Start は模倣を httptest.Server として起動する。
func (tw *Twin) Start(tb testing.TB) *httptest.Server {
	tb.Helper()
	server := httptest.NewServer(tw)
	tb.Cleanup(server.Close)
	return server
}

// InjectFaults は次の要求から順に返す障害を積む。
func (tw *Twin) InjectFaults(faults ...Fault) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	tw.faults = append(tw.faults, faults...)
}

// Put は相手の状態として、識別子を指定したリソースを直接置く。
// 下流の状態の準備は相手側の操作であり、IdMagic の入口から作れないので、テストが直接置く。
func (tw *Twin) Put(kind, id string, attrs map[string]any) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	doc := maps.Clone(attrs)
	doc["id"] = id
	tw.resources[kind][id] = doc
}

// Get は保持しているリソースの写しを返す。
func (tw *Twin) Get(kind, id string) (map[string]any, bool) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	doc, ok := tw.resources[kind][id]
	return maps.Clone(doc), ok
}

func (tw *Twin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if len(tw.faults) > 0 {
		fault := tw.faults[0]
		tw.faults = tw.faults[1:]
		if fault.RetryAfter != "" {
			w.Header().Set("Retry-After", fault.RetryAfter)
		}
		if fault.MalformedBody {
			w.Header().Set("Content-Type", "application/scim+json")
			status := fault.Status
			if status == 0 {
				status = http.StatusOK
			}
			w.WriteHeader(status)
			_, _ = w.Write([]byte(`{"id":`))
			return
		}
		if fault.Status != 0 {
			writeError(w, fault.Status, "", "injected fault")
			return
		}
	}

	segments := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	switch {
	case len(segments) == 1 && segments[0] == "ServiceProviderConfig" && r.Method == http.MethodGet:
		tw.serviceProviderConfig(w)
	case len(segments) == 1 && isKind(segments[0]):
		switch r.Method {
		case http.MethodPost:
			tw.create(w, r, segments[0])
		case http.MethodGet:
			tw.query(w, r, segments[0])
		default:
			writeError(w, http.StatusMethodNotAllowed, "", "method not allowed")
		}
	case len(segments) == 2 && isKind(segments[0]):
		tw.resource(w, r, segments[0], segments[1])
	default:
		writeError(w, http.StatusNotFound, "", "unknown endpoint")
	}
}

func isKind(s string) bool { return s == "Users" || s == "Groups" }

func (tw *Twin) serviceProviderConfig(w http.ResponseWriter) {
	c := tw.Capabilities
	writeJSON(w, http.StatusOK, map[string]any{
		"schemas":          []string{"urn:ietf:params:scim:schemas:core:2.0:ServiceProviderConfig"},
		"documentationUri": "http://twin.invalid/help/scim.html",
		"patch":            map[string]any{"supported": c.Patch},
		"bulk":             map[string]any{"supported": c.Bulk, "maxOperations": 0, "maxPayloadSize": 0},
		"filter":           map[string]any{"supported": c.Filter, "maxResults": 100},
		"changePassword":   map[string]any{"supported": c.ChangePassword},
		"sort":             map[string]any{"supported": c.Sort},
		"etag":             map[string]any{"supported": c.Etag},
		"authenticationSchemes": []any{map[string]any{
			"name": "OAuth Bearer Token", "description": "Authentication scheme using the OAuth Bearer Token Standard",
			"specUri": "http://www.rfc-editor.org/info/rfc6750", "documentationUri": "http://twin.invalid/help/oauth.html",
			"type": "oauthbearertoken", "primary": true,
		}},
		"meta": map[string]any{
			"location": "http://twin.invalid/ServiceProviderConfig", "resourceType": "ServiceProviderConfig",
			"created": "2026-01-01T00:00:00Z", "lastModified": "2026-01-01T00:00:00Z", "version": `W/"1"`,
		},
	})
}

// uniqueAttr は種類ごとに重複を 409 にする属性である (RFC 7644 §3.3)。
func uniqueAttr(kind string) string {
	if kind == "Users" {
		return "userName"
	}
	return "displayName"
}

func (tw *Twin) create(w http.ResponseWriter, r *http.Request, kind string) {
	var doc map[string]any
	if err := json.NewDecoder(r.Body).Decode(&doc); err != nil {
		writeError(w, http.StatusBadRequest, "invalidSyntax", "request body is not a JSON object")
		return
	}
	key := uniqueAttr(kind)
	if value, ok := doc[key].(string); ok {
		for _, existing := range tw.resources[kind] {
			if other, _ := existing[key].(string); strings.EqualFold(other, value) {
				writeError(w, http.StatusConflict, "uniqueness", fmt.Sprintf("%s %q is already in use", key, value))
				return
			}
		}
	}
	tw.nextID++
	id := fmt.Sprintf("twin-%d", tw.nextID)
	doc["id"] = id
	tw.resources[kind][id] = doc
	tw.writeResource(w, http.StatusCreated, kind, doc)
}

func (tw *Twin) query(w http.ResponseWriter, r *http.Request, kind string) {
	match := func(map[string]any) bool { return true }
	if filter := r.URL.Query().Get("filter"); filter != "" {
		attr, value, err := parseEqFilter(filter)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalidFilter", err.Error())
			return
		}
		match = func(doc map[string]any) bool {
			for name, v := range doc {
				if strings.EqualFold(name, attr) {
					s, ok := v.(string)
					return ok && strings.EqualFold(s, value)
				}
			}
			return false
		}
	}
	found := []any{}
	for _, doc := range tw.resources[kind] {
		if match(doc) {
			out, _, _ := tw.represent(kind, doc)
			found = append(found, out)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"schemas":      []string{"urn:ietf:params:scim:api:messages:2.0:ListResponse"},
		"totalResults": len(found),
		"Resources":    found,
	})
}

// parseEqFilter は `<attrPath> eq <compValue>` だけを解釈する。compValue は JSON の文字列である
// (RFC 7644 §3.4.2.2、RFC 7159 §7)。Provisioning が送るフィルターはこの形だけなので、
// 模倣はそれ以外を invalidFilter として拒否する。
func parseEqFilter(filter string) (attr, value string, err error) {
	attr, rest, ok := strings.Cut(filter, " ")
	if !ok {
		return "", "", fmt.Errorf("filter %q has no operator", filter)
	}
	op, literal, ok := strings.Cut(rest, " ")
	if !ok || !strings.EqualFold(op, "eq") {
		return "", "", fmt.Errorf("filter %q is not an eq comparison", filter)
	}
	if err := json.Unmarshal([]byte(literal), &value); err != nil {
		return "", "", fmt.Errorf("filter value %s is not a JSON string: %w", literal, err)
	}
	return attr, value, nil
}

func (tw *Twin) resource(w http.ResponseWriter, r *http.Request, kind, id string) {
	doc, ok := tw.resources[kind][id]
	if !ok {
		writeError(w, http.StatusNotFound, "", fmt.Sprintf("Resource %s not found", id))
		return
	}
	switch r.Method {
	case http.MethodGet:
		tw.writeResource(w, http.StatusOK, kind, doc)
	case http.MethodPut:
		var next map[string]any
		if err := json.NewDecoder(r.Body).Decode(&next); err != nil {
			writeError(w, http.StatusBadRequest, "invalidSyntax", "request body is not a JSON object")
			return
		}
		next["id"] = id
		tw.resources[kind][id] = next
		tw.writeResource(w, http.StatusOK, kind, next)
	case http.MethodPatch:
		var patch struct {
			Operations []struct {
				Op    string          `json:"op"`
				Path  string          `json:"path"`
				Value json.RawMessage `json:"value"`
			} `json:"Operations"`
		}
		if err := json.NewDecoder(r.Body).Decode(&patch); err != nil || len(patch.Operations) == 0 {
			writeError(w, http.StatusBadRequest, "invalidSyntax", "request body is not a PatchOp")
			return
		}
		for _, op := range patch.Operations {
			if err := applyPatch(doc, strings.ToLower(op.Op), op.Path, op.Value); err != nil {
				writeError(w, http.StatusBadRequest, "invalidValue", err.Error())
				return
			}
		}
		if tw.PatchReturnsResource {
			tw.writeResource(w, http.StatusOK, kind, doc)
			return
		}
		_, location, version := tw.represent(kind, doc)
		w.Header().Set("Location", location)
		w.Header().Set("ETag", version)
		w.WriteHeader(http.StatusNoContent)
	case http.MethodDelete:
		delete(tw.resources[kind], id)
		w.WriteHeader(http.StatusNoContent)
	default:
		writeError(w, http.StatusMethodNotAllowed, "", "method not allowed")
	}
}

// applyPatch は Provisioning が送る二つの形だけを適用する。対象パスのない replace と、
// members への add と remove である (RFC 7644 §3.5.2.1、§3.5.2.2、§3.5.2.3)。
func applyPatch(doc map[string]any, op, path string, raw json.RawMessage) error {
	switch {
	case path == "" && (op == "replace" || op == "add"):
		var value map[string]any
		if err := json.Unmarshal(raw, &value); err != nil {
			return fmt.Errorf("value of %s without path must be an object", op)
		}
		maps.Copy(doc, value)
		return nil
	case strings.EqualFold(path, "members") && (op == "add" || op == "remove"):
		var members []map[string]any
		if err := json.Unmarshal(raw, &members); err != nil {
			return fmt.Errorf("members value must be an array")
		}
		current, _ := doc["members"].([]any)
		for _, m := range members {
			current = withoutMember(current, m["value"])
			if op == "add" {
				current = append(current, map[string]any{"value": m["value"]})
			}
		}
		doc["members"] = current
		return nil
	default:
		return fmt.Errorf("unsupported operation %s %q", op, path)
	}
}

func withoutMember(members []any, value any) []any {
	out := members[:0:0]
	for _, m := range members {
		if mm, ok := m.(map[string]any); ok && mm["value"] == value {
			continue
		}
		out = append(out, m)
	}
	return out
}

// represent は保持している属性に meta を加えた表現と、Location と ETag の値を返す。
func (tw *Twin) represent(kind string, doc map[string]any) (out map[string]any, location, version string) {
	out = maps.Clone(doc)
	id, _ := doc["id"].(string)
	location = "http://twin.invalid/" + kind + "/" + id
	version = etag(doc)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
	out["meta"] = map[string]any{
		"resourceType": strings.TrimSuffix(kind, "s"), "created": now, "lastModified": now,
		"location": location, "version": version,
	}
	return out, location, version
}

func (tw *Twin) writeResource(w http.ResponseWriter, status int, kind string, doc map[string]any) {
	out, location, version := tw.represent(kind, doc)
	w.Header().Set("Location", location)
	w.Header().Set("ETag", version)
	writeJSON(w, status, out)
}

// etag は表現の内容から弱い ETag を作る。内容が変われば値も変わる。
func etag(doc map[string]any) string {
	sum := fnv.New32a()
	_, _ = fmt.Fprint(sum, doc) // fmt は map のキーを整列して書くので、同じ内容は同じ値になる
	return `W/"` + strconv.FormatUint(uint64(sum.Sum32()), 16) + `"`
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	data, err := json.Marshal(body)
	if err != nil {
		http.Error(w, "twin: encode response: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/scim+json")
	w.WriteHeader(status)
	_, _ = w.Write(data)
}

// writeError は RFC 7644 §3.12 のエラー本文を返す。
func writeError(w http.ResponseWriter, status int, scimType, detail string) {
	body := map[string]any{
		"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:Error"},
		"detail":  detail,
		"status":  strconv.Itoa(status),
	}
	if scimType != "" {
		body["scimType"] = scimType
	}
	writeJSON(w, status, body)
}
