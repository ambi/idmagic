package spec_test

import (
	"testing"

	"github.com/ambi/idmagic/backend/shared/spec"
)

func TestBuildDiscoveryDocument_AdvertisesClientIDMetadataDocumentSupport(t *testing.T) {
	s, err := spec.LoadSCL()
	if err != nil {
		t.Fatal(err)
	}
	doc, err := s.BuildDiscoveryDocument("https://idp.example.com")
	if err != nil {
		t.Fatal(err)
	}
	supported, ok := doc["client_id_metadata_document_supported"].(bool)
	if !ok || !supported {
		t.Errorf("client_id_metadata_document_supported = %v, want true", doc["client_id_metadata_document_supported"])
	}
}
