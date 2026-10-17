package tokens_saml

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/beevik/etree"
)

// assertionXML は、組み立てたアサーションの ID を固定の値へ置き換えた XML を返す。ID は乱数で作られる。
func assertionXML(t *testing.T, in AssertionInput) string {
	t.Helper()
	assertion, id, err := BuildAssertion(in)
	if err != nil {
		t.Fatal(err)
	}
	doc := etree.NewDocument()
	doc.SetRoot(assertion)
	out, err := doc.WriteToString()
	if err != nil {
		t.Fatal(err)
	}
	return strings.ReplaceAll(out, id, "_assertion-id")
}

// SP が受け入れるアサーションの XML は、要素の順序、属性、名前空間まで含めて変わってはならない。
// 期待値は testdata の XML ファイルである。
func TestBuildAssertionWritesTheSameXMLInBothVersions(t *testing.T) {
	cases := []struct {
		file    string
		version SAMLVersion
	}{
		{file: "assertion_saml20.xml", version: SAML20},
		{file: "assertion_saml11.xml", version: SAML11},
	}
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			in := sampleInput()
			in.Version = tc.version
			in.InResponseTo = "_request-1"
			in.AuthnContext = AuthnContextPassword
			got := assertionXML(t, in)
			path := filepath.Join("testdata", tc.file)
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if got != string(want) {
				t.Fatalf("assertion XML differs from %s:\n%s", path, got)
			}
		})
	}
}
