// Package testing_scimtwin は、下流の SCIM サービスプロバイダーの代わりにテストが使う相手を提供する。
//
// 相手は二つある。記録した往復をそのまま返す再生と、User と Group を保持する状態を持つ模倣である。
// 記録は模倣の忠実さの基準でもあり、模倣が記録と同じ応答を返すことをこのパッケージのテストが確かめる。
// どちらもテストの中だけで使い、製品の経路から import しない。
package testing_scimtwin

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"regexp"
	"testing"
)

//go:embed testdata/rfc7644.json
var rfc7644Recording []byte

// Recording は一つの相手との往復の列と、その出どころである。
// 記録が古くなったときに相手が変わったのか記録が古いのかを区別できるよう、取得日、相手の版、取得方法を必ず持つ。
type Recording struct {
	Counterpart        string     `json:"counterpart"`
	CounterpartVersion string     `json:"counterpart_version"`
	RetrievedAt        string     `json:"retrieved_at"`
	Method             string     `json:"method"`
	Exchanges          []Exchange `json:"exchanges"`
}

// Exchange は一つの要求と、相手が返した応答である。
type Exchange struct {
	Name     string           `json:"name"`
	Source   string           `json:"source"`
	Setup    []SetupResource  `json:"setup,omitempty"`
	Request  RecordedRequest  `json:"request"`
	Response RecordedResponse `json:"response"`
}

// SetupResource は、記録の前から相手が持っていたリソースである。
// 記録は相手の状態を作る往復を含まないことがあるので、模倣との照合の前に同じ状態を置く。
type SetupResource struct {
	Kind        string `json:"kind"`
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
}

// RecordedRequest は相手が受け取った要求である。
type RecordedRequest struct {
	Method  string            `json:"method"`
	Path    string            `json:"path"`
	Query   string            `json:"query,omitempty"`
	Headers map[string]string `json:"headers"`
	Body    json.RawMessage   `json:"body,omitempty"`
}

// RecordedResponse は相手が返した応答である。
type RecordedResponse struct {
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers"`
	Body    json.RawMessage   `json:"body,omitempty"`
}

// RFC7644Recording は RFC 7644 と RFC 7643 の例を転記した記録を返す。
func RFC7644Recording(tb testing.TB) Recording {
	tb.Helper()
	rec, err := decodeRecording(rfc7644Recording)
	if err != nil {
		tb.Fatalf("decode RFC 7644 recording: %v", err)
	}
	return rec
}

func decodeRecording(data []byte) (Recording, error) {
	var rec Recording
	if err := json.Unmarshal(data, &rec); err != nil {
		return Recording{}, err
	}
	if rec.Counterpart == "" || rec.CounterpartVersion == "" || rec.RetrievedAt == "" || rec.Method == "" {
		return Recording{}, fmt.Errorf("recording must state its counterpart, version, retrieval date and method")
	}
	return rec, nil
}

// Exchange は名前で往復を返す。
func (r Recording) Exchange(tb testing.TB, name string) Exchange {
	tb.Helper()
	for _, ex := range r.Exchanges {
		if ex.Name == name {
			return ex
		}
	}
	tb.Fatalf("recording has no exchange %q", name)
	return Exchange{}
}

// resourcePath は `/Users/{id}` と `/Groups/{id}` の識別子の部分を取り出す。
var resourcePath = regexp.MustCompile(`^/(Users|Groups)/([^/]+)$`)

// pathShape は識別子を除いたパスの形を返す。
// 記録の識別子は相手が割り当てたものなので、自分の要求と記録を比べるときは形だけを比べる。
func pathShape(path string) string {
	if m := resourcePath.FindStringSubmatch(path); m != nil {
		return "/" + m[1] + "/{id}"
	}
	return path
}
