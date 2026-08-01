package handlers_http_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

// TestUpdateApplicationOidcConfig_RulesRoundtrip covers wi-73: an admin can add a
// per-application claim release rule to an OIDC client and read it back.
func TestUpdateApplicationOidcConfig_RulesRoundtrip(t *testing.T) {
	e := newApplicationHandler(t)
	csrf, cookie := appCSRF(t, e)

	created := adminJSON(t, e, http.MethodPost, "/api/admin/applications", csrf, cookie, map[string]any{
		"name": "OIDC App", "type": "oidc", "redirect_uris": []string{"https://oidc.example/callback"},
		"client_type": "confidential", "token_endpoint_auth_method": "client_secret_post",
	})
	if created.Code != http.StatusCreated {
		t.Fatalf("create oidc status=%d body=%s", created.Code, created.Body.String())
	}
	var createdBody struct {
		Application struct {
			ApplicationID string `json:"application_id"`
		} `json:"application"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &createdBody); err != nil {
		t.Fatal(err)
	}
	appID := createdBody.Application.ApplicationID

	update := adminJSON(t, e, http.MethodPatch, "/api/admin/applications/"+appID+"/oidc", csrf, cookie, map[string]any{
		"rules": []map[string]any{
			{"claim_type": "department", "source": "user_attribute", "source_key": "department"},
		},
	})
	if update.Code != http.StatusNoContent {
		t.Fatalf("update oidc config status=%d body=%s", update.Code, update.Body.String())
	}

	detail := adminJSON(t, e, http.MethodGet, "/api/admin/applications/"+appID, csrf, cookie, nil)
	if detail.Code != http.StatusOK {
		t.Fatalf("get application status=%d body=%s", detail.Code, detail.Body.String())
	}
	var detailBody struct {
		Oidc struct {
			Rules []struct {
				ClaimType string `json:"claim_type"`
			} `json:"rules"`
		} `json:"oidc"`
	}
	if err := json.Unmarshal(detail.Body.Bytes(), &detailBody); err != nil {
		t.Fatal(err)
	}
	if len(detailBody.Oidc.Rules) != 1 || detailBody.Oidc.Rules[0].ClaimType != "department" {
		t.Fatalf("expected department rule to round-trip, got %+v", detailBody.Oidc.Rules)
	}
}

// TestUpdateApplicationOidcConfig_RejectsUndefinedAttributeSource verifies the
// fail-closed floor (ADR-151, wi-73): a claim rule referencing an attribute key with no
// UserAttributeDef is rejected at write time, not silently accepted.
func TestUpdateApplicationOidcConfig_RejectsUndefinedAttributeSource(t *testing.T) {
	e := newApplicationHandler(t)
	csrf, cookie := appCSRF(t, e)

	created := adminJSON(t, e, http.MethodPost, "/api/admin/applications", csrf, cookie, map[string]any{
		"name": "OIDC App", "type": "oidc", "redirect_uris": []string{"https://oidc.example/callback"},
		"client_type": "confidential", "token_endpoint_auth_method": "client_secret_post",
	})
	if created.Code != http.StatusCreated {
		t.Fatalf("create oidc status=%d body=%s", created.Code, created.Body.String())
	}
	var createdBody struct {
		Application struct {
			ApplicationID string `json:"application_id"`
		} `json:"application"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &createdBody); err != nil {
		t.Fatal(err)
	}
	appID := createdBody.Application.ApplicationID

	update := adminJSON(t, e, http.MethodPatch, "/api/admin/applications/"+appID+"/oidc", csrf, cookie, map[string]any{
		"rules": []map[string]any{
			{"claim_type": "leak", "source": "user_attribute", "source_key": "totally_undefined_attribute"},
		},
	})
	if update.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for undefined attribute source, got status=%d body=%s", update.Code, update.Body.String())
	}
}

// TestUpdateApplicationWsFedConfig_RejectsReservedClaimType verifies the WS-Fed claim
// release editor shares the same fail-closed floor (ADR-151, wi-73).
func TestUpdateApplicationWsFedConfig_RejectsReservedClaimType(t *testing.T) {
	e := newApplicationHandler(t)
	csrf, cookie := appCSRF(t, e)

	created := adminJSON(t, e, http.MethodPost, "/api/admin/applications", csrf, cookie, map[string]any{
		"name": "WsFed App", "type": "wsfed", "wtrealm": "urn:wsfed:claim-release", "reply_urls": []string{"https://wsfed.example/reply"},
	})
	if created.Code != http.StatusCreated {
		t.Fatalf("create wsfed status=%d body=%s", created.Code, created.Body.String())
	}
	var createdBody struct {
		Application struct {
			ApplicationID string `json:"application_id"`
		} `json:"application"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &createdBody); err != nil {
		t.Fatal(err)
	}
	appID := createdBody.Application.ApplicationID

	update := adminJSON(t, e, http.MethodPatch, "/api/admin/applications/"+appID+"/wsfed", csrf, cookie, map[string]any{
		"rules": []map[string]any{
			{"claim_type": "sub", "source": "fixed", "fixed_value": "attacker-controlled"},
		},
	})
	if update.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for reserved claim_type, got status=%d body=%s", update.Code, update.Body.String())
	}
}

// TestUpdateApplicationSamlConfig_RejectsUndefinedAttributeSource verifies the SAML
// claim release editor shares the same fail-closed floor (ADR-151, wi-73).
func TestUpdateApplicationSamlConfig_RejectsUndefinedAttributeSource(t *testing.T) {
	e := newApplicationHandler(t)
	csrf, cookie := appCSRF(t, e)

	created := adminJSON(t, e, http.MethodPost, "/api/admin/applications", csrf, cookie, map[string]any{
		"name": "SAML App", "type": "saml", "entity_id": "https://saml.example/claim-release", "acs_urls": []string{"https://saml.example/acs"},
	})
	if created.Code != http.StatusCreated {
		t.Fatalf("create saml status=%d body=%s", created.Code, created.Body.String())
	}
	var createdBody struct {
		Application struct {
			ApplicationID string `json:"application_id"`
		} `json:"application"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &createdBody); err != nil {
		t.Fatal(err)
	}
	appID := createdBody.Application.ApplicationID

	update := adminJSON(t, e, http.MethodPatch, "/api/admin/applications/"+appID+"/saml", csrf, cookie, map[string]any{
		"rules": []map[string]any{
			{"claim_type": "leak", "source": "user_attribute", "source_key": "totally_undefined_attribute"},
		},
	})
	if update.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for undefined attribute source, got status=%d body=%s", update.Code, update.Body.String())
	}
}
