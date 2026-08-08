package domain

import "testing"

func TestIsClientIDMetadataDocumentURL(t *testing.T) {
	for _, valid := range []string{
		"https://app.example.com/oauth/client-metadata.json",
		"https://app.example.com:8443/client.json",
	} {
		if !IsClientIDMetadataDocumentURL(valid) {
			t.Errorf("%s rejected, want accepted", valid)
		}
	}
	for _, invalid := range []string{
		"http://app.example.com/client.json",
		"https://app.example.com",
		"https://user:pass@app.example.com/client.json",
		"https://app.example.com/client.json#fragment",
		"not-a-url",
		"my-registered-client-id",
	} {
		if IsClientIDMetadataDocumentURL(invalid) {
			t.Errorf("%s accepted, want rejected", invalid)
		}
	}
}

func TestParseClientIDMetadataDocument_MinimalDocumentAppliesDefaults(t *testing.T) {
	const url = "https://app.example.com/oauth/client-metadata.json"
	doc := []byte(`{
		"client_id": "` + url + `",
		"client_name": "Example MCP Client",
		"redirect_uris": ["http://127.0.0.1:3000/callback"]
	}`)
	client, err := ParseClientIDMetadataDocument(doc, url)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.ClientID != url {
		t.Errorf("ClientID = %q, want %q", client.ClientID, url)
	}
	if client.TokenEndpointAuthMethod != AuthMethodNone {
		t.Errorf("TokenEndpointAuthMethod = %q, want none", client.TokenEndpointAuthMethod)
	}
	if client.Scope != "openid" {
		t.Errorf("Scope = %q, want openid default", client.Scope)
	}
	if len(client.GrantTypes) != 1 || client.GrantTypes[0] != "authorization_code" {
		t.Errorf("GrantTypes = %v, want [authorization_code] default", client.GrantTypes)
	}
	if len(client.ResponseTypes) != 1 || client.ResponseTypes[0] != "code" {
		t.Errorf("ResponseTypes = %v, want [code] default", client.ResponseTypes)
	}
	if client.ApplicationID != "" {
		t.Errorf("ApplicationID = %q, want empty (CIMD clients are never Application-linked)", client.ApplicationID)
	}
	if err := client.Validate(); err != nil {
		t.Errorf("synthesized client failed Validate(): %v", err)
	}
}

func TestParseClientIDMetadataDocument_ExplicitScopeIsPreserved(t *testing.T) {
	const url = "https://app.example.com/oauth/client-metadata.json"
	doc := []byte(`{
		"client_id": "` + url + `",
		"client_name": "Example MCP Client",
		"redirect_uris": ["http://127.0.0.1:3000/callback"],
		"scope": "openid mcp:tools:read"
	}`)
	client, err := ParseClientIDMetadataDocument(doc, url)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.Scope != "openid mcp:tools:read" {
		t.Errorf("Scope = %q, want document-declared value", client.Scope)
	}
}

func TestParseClientIDMetadataDocument_RejectsClientIDMismatch(t *testing.T) {
	const url = "https://app.example.com/oauth/client-metadata.json"
	doc := []byte(`{
		"client_id": "https://attacker.example.com/client.json",
		"client_name": "Example MCP Client",
		"redirect_uris": ["http://127.0.0.1:3000/callback"]
	}`)
	if _, err := ParseClientIDMetadataDocument(doc, url); err == nil {
		t.Fatal("expected error for client_id mismatch")
	}
}

func TestParseClientIDMetadataDocument_RejectsMissingClientName(t *testing.T) {
	const url = "https://app.example.com/oauth/client-metadata.json"
	doc := []byte(`{
		"client_id": "` + url + `",
		"redirect_uris": ["http://127.0.0.1:3000/callback"]
	}`)
	if _, err := ParseClientIDMetadataDocument(doc, url); err == nil {
		t.Fatal("expected error for missing client_name")
	}
}

func TestParseClientIDMetadataDocument_RejectsEmptyRedirectURIs(t *testing.T) {
	const url = "https://app.example.com/oauth/client-metadata.json"
	doc := []byte(`{
		"client_id": "` + url + `",
		"client_name": "Example MCP Client",
		"redirect_uris": []
	}`)
	if _, err := ParseClientIDMetadataDocument(doc, url); err == nil {
		t.Fatal("expected error for empty redirect_uris")
	}
}

func TestParseClientIDMetadataDocument_RejectsUnsupportedAuthMethod(t *testing.T) {
	const url = "https://app.example.com/oauth/client-metadata.json"
	doc := []byte(`{
		"client_id": "` + url + `",
		"client_name": "Example MCP Client",
		"redirect_uris": ["http://127.0.0.1:3000/callback"],
		"token_endpoint_auth_method": "private_key_jwt"
	}`)
	if _, err := ParseClientIDMetadataDocument(doc, url); err == nil {
		t.Fatal("expected error for unsupported token_endpoint_auth_method")
	}
}

func TestParseClientIDMetadataDocument_RejectsInvalidJSON(t *testing.T) {
	if _, err := ParseClientIDMetadataDocument([]byte("not json"), "https://app.example.com/client.json"); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
