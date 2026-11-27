package issuer_metadata

import (
	"os"
	"testing"

	"github.com/ambi/idmagic/backend/shared/testing_httprecord"
)

// 公開の workload の issuer から実際に取った Discovery の応答の記録である。
// 期待値は OpenID Connect Discovery 1.0 の文面から書く。§4.1 は取得先を issuer に
// /.well-known/openid-configuration を続けた URL とし、§4.3 は応答の issuer が取得に使った issuer と同一であることを求める。
const (
	githubActionsIssuer = "https://token.actions.githubusercontent.com"
	entraIssuer         = "https://login.microsoftonline.com/72f988bf-86f1-41af-91ab-2d7cd011db47/v2.0"
)

func recordedDiscovery(t *testing.T, file string) testing_httprecord.Exchange {
	t.Helper()
	data, err := os.ReadFile("testdata/" + file)
	if err != nil {
		t.Fatal(err)
	}
	rec, err := testing_httprecord.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	return rec.Exchange(t, "discovery")
}

// replayClient は、記録した Discovery の応答を一度だけ返す再生サーバーへ送る Client を返す。
// 再生サーバーは要求の方式、パス、Accept を記録と照合する。
func replayClient(t *testing.T, file string) Client {
	t.Helper()
	server := testing_httprecord.Replay{}.Serve(t, recordedDiscovery(t, file))
	return Client{client: testing_httprecord.RoutedClient(server)}
}

//spec:covers REQ-WORKLOADIDENTITY-015: パスのない実在の issuer（GitHub Actions）の Discovery から jwks_uri の候補を返す。
func TestDiscoverReadsRecordedGitHubActionsMetadata(t *testing.T) {
	uri, err := replayClient(t, "github-actions.json").Discover(t.Context(), githubActionsIssuer)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if uri != "https://token.actions.githubusercontent.com/.well-known/jwks" {
		t.Fatalf("jwks_uri = %q", uri)
	}
}

//spec:covers REQ-WORKLOADIDENTITY-015: パスを持つ実在の issuer（Microsoft Entra ID の v2.0）では、パスの後に /.well-known/openid-configuration を続けて取得する。
func TestDiscoverReadsRecordedEntraMetadataUnderTheIssuerPath(t *testing.T) {
	uri, err := replayClient(t, "entra-id.json").Discover(t.Context(), entraIssuer)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if uri != "https://login.microsoftonline.com/72f988bf-86f1-41af-91ab-2d7cd011db47/discovery/v2.0/keys" {
		t.Fatalf("jwks_uri = %q", uri)
	}
}

// 実在の issuer は末尾のスラッシュを持たない。利用者が末尾にスラッシュを付けて入力すると、
// 取得先は同じでも応答の issuer が入力と同一でないので、§4.3 に従って候補を返さない。
//
//spec:covers REQ-WORKLOADIDENTITY-015: 応答の issuer が入力の issuer と完全に一致しなければ候補を返さない。
func TestDiscoverRejectsRecordedIssuerThatDiffersByATrailingSlash(t *testing.T) {
	if uri, err := replayClient(t, "github-actions.json").Discover(t.Context(), githubActionsIssuer+"/"); err == nil {
		t.Fatalf("Discover(%s/) = %q, want an issuer mismatch", githubActionsIssuer, uri)
	}
}
