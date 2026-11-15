package spec_test

import (
	"reflect"
	"slices"
	"testing"

	"github.com/ambi/idmagic/backend/shared/spec"
)

//spec:covers REQ-OAUTH2-014, REQ-OAUTH2-030: PS384 は client assertion の受理方式だけに広告する。
func TestDiscoverySeparatesClientAssertionAlgorithms(t *testing.T) {
	doc, err := spec.CurrentRuntimeContract().BuildDiscoveryDocument("https://idp.example/realms/acme")
	if err != nil {
		t.Fatal(err)
	}
	if got := doc["token_endpoint_auth_signing_alg_values_supported"]; !reflect.DeepEqual(got, []string{"PS256", "ES256", "PS384"}) {
		t.Fatalf("client assertion の広告=%v", got)
	}
	for _, field := range []string{"id_token_signing_alg_values_supported", "dpop_signing_alg_values_supported"} {
		algorithms, ok := doc[field].([]string)
		if !ok || slices.Contains(algorithms, "PS384") {
			t.Fatalf("%s に PS384 が混入した: %v", field, doc[field])
		}
	}
}

func TestBuildDiscoveryDocument_AdvertisesClientIDMetadataDocumentSupport(t *testing.T) {
	contract := spec.CurrentRuntimeContract()
	doc, err := contract.BuildDiscoveryDocument("https://idp.example.com")
	if err != nil {
		t.Fatal(err)
	}
	supported, ok := doc["client_id_metadata_document_supported"].(bool)
	if !ok || !supported {
		t.Errorf("client_id_metadata_document_supported = %v, want true", doc["client_id_metadata_document_supported"])
	}
}

// end_session_endpoint を広告することを固定する。
//
//spec:covers OIDC-LOGOUT-ENDPOINT: Discovery がテナントの issuer 配下の
func TestBuildDiscoveryDocumentAdvertisesEndSessionEndpoint(t *testing.T) {
	contract := spec.CurrentRuntimeContract()
	doc, err := contract.BuildDiscoveryDocument("https://idp.example.com/realms/acme")
	if err != nil {
		t.Fatal(err)
	}
	if got := doc["end_session_endpoint"]; got != "https://idp.example.com/realms/acme/end_session" {
		t.Fatalf("end_session_endpoint = %v", got)
	}
}
