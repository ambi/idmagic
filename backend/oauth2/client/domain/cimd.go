package domain

// Client ID Metadata Documents (CIMD, draft-ietf-oauth-client-id-metadata-document-00).
// Pure URL-shape and document validation; no network I/O.

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"time"

	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"

	"github.com/ambi/idmagic/backend/shared/spec"
)

// IsClientIDMetadataDocumentURL reports whether clientID is shaped as a
// Client ID Metadata Document identifier: an https URL with a non-empty
// path, no userinfo, and no fragment (draft-ietf-oauth-client-id-metadata-
// document-00 §2).
func IsClientIDMetadataDocumentURL(clientID string) bool {
	parsed, err := url.Parse(clientID)
	if err != nil {
		return false
	}
	return parsed.Scheme == "https" &&
		parsed.Hostname() != "" &&
		parsed.Path != "" &&
		parsed.User == nil &&
		parsed.Fragment == ""
}

type clientIDMetadataDocument struct {
	ClientID                string   `json:"client_id"`
	ClientName              string   `json:"client_name"`
	RedirectURIs            []string `json:"redirect_uris"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	Scope                   string   `json:"scope"`
}

// ParseClientIDMetadataDocument parses and validates a fetched CIMD document
// against the URL it was fetched from (which is also the resulting client's
// client_id), returning a synthesized OAuth2Client that is never persisted.
// It performs no I/O; the caller is responsible for the SSRF-safe fetch.
//
// MVP only accepts documents that omit token_endpoint_auth_method or declare
// it as "none"; anything else is rejected fail-closed (rejected
// private_key_jwt-via-CIMD for this iteration).
func ParseClientIDMetadataDocument(raw []byte, requestURL string) (*OAuth2Client, error) {
	var doc clientIDMetadataDocument
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("client id metadata document: invalid JSON: %w", err)
	}
	if doc.ClientID != requestURL {
		return nil, errors.New("client id metadata document: client_id does not match the fetch URL")
	}
	if doc.ClientName == "" {
		return nil, errors.New("client id metadata document: client_name is required")
	}
	if len(doc.RedirectURIs) == 0 {
		return nil, errors.New("client id metadata document: redirect_uris is required")
	}
	if doc.TokenEndpointAuthMethod != "" && doc.TokenEndpointAuthMethod != string(AuthMethodNone) {
		return nil, fmt.Errorf("client id metadata document: unsupported token_endpoint_auth_method %q", doc.TokenEndpointAuthMethod)
	}

	grantTypes := []spec.GrantType{spec.GrantAuthorizationCode}
	if len(doc.GrantTypes) > 0 {
		grantTypes = make([]spec.GrantType, 0, len(doc.GrantTypes))
		for _, g := range doc.GrantTypes {
			grantTypes = append(grantTypes, spec.GrantType(g))
		}
	}
	responseTypes := []spec.ResponseType{spec.ResponseTypeCode}
	if len(doc.ResponseTypes) > 0 {
		responseTypes = make([]spec.ResponseType, 0, len(doc.ResponseTypes))
		for _, r := range doc.ResponseTypes {
			responseTypes = append(responseTypes, spec.ResponseType(r))
		}
	}
	scope := "openid"
	if doc.Scope != "" {
		scope = doc.Scope
	}

	clientName := doc.ClientName
	now := time.Now().UTC()
	client := &OAuth2Client{
		ClientID:                 doc.ClientID,
		ClientName:               &clientName,
		ClientType:               spec.ClientPublic,
		RedirectURIs:             doc.RedirectURIs,
		GrantTypes:               grantTypes,
		ResponseTypes:            responseTypes,
		TokenEndpointAuthMethod:  AuthMethodNone,
		Scope:                    scope,
		IDTokenSignedResponseAlg: signingdomain.SigAlgPS256,
		FapiProfile:              FapiNone,
		CreatedAt:                now,
		UpdatedAt:                now,
	}
	if err := client.Validate(); err != nil {
		return nil, fmt.Errorf("client id metadata document: %w", err)
	}
	return client, nil
}

// ===============================================================
// イベント
// ===============================================================

type ClientIdMetadataDocumentResolved struct {
	At       time.Time `json:"-"`
	TenantID string    `json:"tenantId"`
	ClientID string    `json:"clientId"`
}

func (e *ClientIdMetadataDocumentResolved) EventType() string {
	return "ClientIdMetadataDocumentResolved"
}
func (e *ClientIdMetadataDocumentResolved) OccurredAt() time.Time { return e.At }

type ClientIdMetadataDocumentRejected struct {
	At       time.Time `json:"-"`
	TenantID string    `json:"tenantId"`
	ClientID string    `json:"clientId"`
	Reason   string    `json:"reason"`
}

func (e *ClientIdMetadataDocumentRejected) EventType() string {
	return "ClientIdMetadataDocumentRejected"
}
func (e *ClientIdMetadataDocumentRejected) OccurredAt() time.Time { return e.At }
