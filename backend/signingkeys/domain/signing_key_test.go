package domain

import "testing"

func TestKeyUsageValidIncludesXMLFederationSigning(t *testing.T) {
	if !KeyUsageXMLFederationSigning.Valid() {
		t.Fatal("xml_federation_signing must be a valid key usage")
	}
}
