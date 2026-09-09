package spec

import "testing"

func TestBuildDiscoveryDocumentAdvertisesOIDCLogout(t *testing.T) {
	doc, err := CurrentRuntimeContract().BuildDiscoveryDocument("https://idp.example")
	if err != nil {
		t.Fatal(err)
	}
	if doc["check_session_iframe"] != "https://idp.example/session/check" {
		t.Fatalf("check_session_iframe=%v", doc["check_session_iframe"])
	}
	for _, field := range []string{"frontchannel_logout_supported", "frontchannel_logout_session_supported", "backchannel_logout_supported", "backchannel_logout_session_supported"} {
		if doc[field] != true {
			t.Fatalf("%s=%v", field, doc[field])
		}
	}
}
