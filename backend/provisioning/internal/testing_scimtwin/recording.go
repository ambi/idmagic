// Package testing_scimtwin は、下流の SCIM サービスプロバイダーの代わりにテストが使う相手を提供する。
//
// 相手は二つある。記録した往復をそのまま返す再生と、User と Group を保持する状態を持つ模倣である。
// 記録は模倣の忠実さの基準でもあり、模倣が記録と同じ応答を返すことをこのパッケージのテストが確かめる。
// 記録の形式、再生、照合は testing_httprecord の部品を使い、ここには SCIM に固有の読み替えだけを置く。
// どちらもテストの中だけで使い、製品の経路から import しない。
package testing_scimtwin

import (
	_ "embed"
	"encoding/json"
	"regexp"
	"testing"

	"github.com/ambi/idmagic/backend/shared/testing_httprecord"
)

//go:embed testdata/rfc7644.json
var rfc7644Recording []byte

// Recording は一つの相手との往復の列と、その出どころである。
type Recording = testing_httprecord.Recording

// Exchange は一つの要求と、相手が返した応答である。
type Exchange = testing_httprecord.Exchange

// SetupResource は、記録の前から相手が持っていたリソースである。往復の Setup はこの配列を持つ。
// 記録は相手の状態を作る往復を含まないことがあるので、模倣との照合の前に同じ状態を置く。
type SetupResource struct {
	Kind        string `json:"kind"`
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
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
	return testing_httprecord.Decode(data)
}

func setupResources(ex Exchange) ([]SetupResource, error) {
	if len(ex.Setup) == 0 {
		return nil, nil
	}
	var resources []SetupResource
	err := json.Unmarshal(ex.Setup, &resources)
	return resources, err
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
