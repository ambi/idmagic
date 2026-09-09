package spec_test

import (
	"testing"

	"github.com/ambi/idmagic/backend/shared/spec"
)

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

// OIDC-LOGOUT-ENDPOINT: Discovery がテナントの issuer 配下の
// end_session_endpoint を広告することを固定する。
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
