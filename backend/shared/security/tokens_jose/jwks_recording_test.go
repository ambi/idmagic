package tokens_jose

import (
	"crypto/rsa"
	"os"
	"slices"
	"testing"

	"github.com/ambi/idmagic/backend/shared/testing_httprecord"
)

// 公開の workload の issuer から実際に取った JWK Set の記録を再生し、取得から鍵の解釈までを確かめる。
// 期待値は RFC 7517 と RFC 7518 の文面から書く。§5 の JWK Set は keys の配列で鍵を運び、
// §4 は解釈できないメンバーを無視することを求め、RFC 7518 §6.3.1 は RSA の n と e を base64url の符号なし整数とする。

// replayedJWKS は記録した JWK Set の応答を一度だけ返す再生サーバーから、jwksPath の鍵を取得する。
//
// 取得の前の SSRF の検査は URL のホストを名前解決するので、取得先のホストは名前解決の要らない
// 文書用のアドレス（RFC 5737）にし、パスだけを記録と照合する。
func replayedJWKS(t *testing.T, file, jwksPath string) []map[string]any {
	t.Helper()
	data, err := os.ReadFile("testdata/" + file)
	if err != nil {
		t.Fatal(err)
	}
	rec, err := testing_httprecord.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	server := testing_httprecord.Replay{}.Serve(t, rec.Exchange(t, "jwks"))
	r := &JWKResolver{cache: map[string]cachedJWKS{}, client: testing_httprecord.RoutedClient(server)}
	uri := "https://192.0.2.1" + jwksPath
	keys, err := r.ResolveJWKSSource(t.Context(), &uri, nil)
	if err != nil {
		t.Fatalf("ResolveJWKSSource: %v", err)
	}
	return keys
}

// requireRSAKeys は、各 kid の鍵を workload の検証と同じ規則で選び、e が AQAB（65537）の 2048 ビットの RSA 鍵として読めることを確かめる。
func requireRSAKeys(t *testing.T, keys []map[string]any, kids []string) {
	t.Helper()
	got := make([]string, 0, len(keys))
	for _, key := range keys {
		kid, _ := key["kid"].(string)
		got = append(got, kid)
	}
	if !slices.Equal(got, kids) {
		t.Fatalf("kids = %q, want %q", got, kids)
	}
	for _, kid := range kids {
		key, err := pickWorkloadJWK(keys, kid)
		if err != nil {
			t.Errorf("pickWorkloadJWK(%s): %v", kid, err)
			continue
		}
		rsaKey, ok := key.(*rsa.PublicKey)
		if !ok {
			t.Errorf("key %s is %T, want *rsa.PublicKey", kid, key)
			continue
		}
		if rsaKey.E != 65537 || rsaKey.N.BitLen() != 2048 {
			t.Errorf("key %s: e = %d, modulus = %d bits, want 65537 and 2048", kid, rsaKey.E, rsaKey.N.BitLen())
		}
	}
}

// GitHub Actions の JWK Set は alg と use を持つ RS256 の鍵で、x5c と x5t を持つ鍵と持たない鍵が混ざる。
func TestJWKResolverReadsRecordedGitHubActionsJWKS(t *testing.T) {
	keys := replayedJWKS(t, "github-actions-jwks.json", "/.well-known/jwks")
	requireRSAKeys(t, keys, []string{
		"cc413527-173f-5a05-976e-9c52b1d7b431",
		"38826b17-6a30-5f9b-b169-8beb8202f723",
		"38E9B30B3A023A1B72309921A69A42FCC496C42C",
		"4F3E9AD8C9A6F5EB3173006F4FA630E28F43DCE9",
	})
}

// Microsoft Entra ID の JWK Set の鍵は alg を持たず、x5c、x5t と、RFC 7517 にない issuer、cloud_instance_name を持つ。
// alg は §4.4 で任意であり、解釈できないメンバーは §4 に従って無視するので、どの鍵も読める。
func TestJWKResolverReadsRecordedEntraJWKSWithoutAlg(t *testing.T) {
	keys := replayedJWKS(t, "entra-id-jwks.json", "/72f988bf-86f1-41af-91ab-2d7cd011db47/discovery/v2.0/keys")
	requireRSAKeys(t, keys, []string{
		"Xqtg30yHerhLh5UO88-4NMblzWA",
		"xH9DJH3N086Ags8JL6IQzeQh1p8",
		"i65MunI1gao7pz6iN-vOMXkb6HM",
		"e1ApdTPqxaLoJi3AWK4InqBnqV4",
		"OnajO4j25jF_mMcNhZW42zC7TdA",
	})
}
