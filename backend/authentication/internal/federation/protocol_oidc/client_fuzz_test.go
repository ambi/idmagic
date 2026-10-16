package protocol_oidc

import (
	"encoding/json"
	"testing"
)

func FuzzUpstreamOIDCMetadata(f *testing.F) {
	f.Add([]byte(`{"keys":[{"kty":"RSA","kid":"key-1","alg":"RS256","n":"AQ","e":"AQAB"}]}`))
	f.Add([]byte(`{"keys":[{"kty":"RSA","kid":"key-1","alg":"none","n":"AQ","e":"AQAB"}]}`))
	f.Add([]byte(`{"keys":"not-an-array"}`))

	f.Fuzz(func(t *testing.T, input []byte) {
		if len(input) > maxResponseBytes {
			return
		}
		var document jwksDocument
		if json.Unmarshal(input, &document) == nil {
			_, _ = document.rsaKeys()
		}
	})
}
