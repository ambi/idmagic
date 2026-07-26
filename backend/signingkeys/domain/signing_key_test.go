package domain

import "testing"

func TestKeyUsageValidIncludesXMLFederationSigning(t *testing.T) {
	if !KeyUsageXMLFederationSigning.Valid() {
		t.Fatal("XmlFederationSigning must be a valid key usage")
	}
}
