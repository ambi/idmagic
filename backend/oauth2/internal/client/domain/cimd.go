package domain

// Client ID Metadata Documents (CIMD, draft-ietf-oauth-client-id-metadata-document-02).
// Pure URL-shape and document validation; no network I/O.

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"

	"github.com/ambi/idmagic/backend/shared/spec"
)

// IsClientIDMetadataDocumentURL reports whether clientID is shaped as a
// Client ID Metadata Document identifier: an https URL with a non-empty
// path, no userinfo, no fragment, and no single-dot or double-dot path
// segment (draft-ietf-oauth-client-id-metadata-document-02 §3, CIMD02-URL-SHAPE).
// A query is a SHOULD NOT for the client and is accepted.
func IsClientIDMetadataDocumentURL(clientID string) bool {
	parsed, err := url.Parse(clientID)
	if err != nil {
		return false
	}
	return parsed.Scheme == "https" &&
		parsed.Hostname() != "" &&
		parsed.Path != "" &&
		parsed.User == nil &&
		parsed.Fragment == "" &&
		!hasDotSegment(parsed.Path)
}

// hasDotSegment は、パーセント復号済みのパスに要素全体が "." か ".." のものがあるかを返す。
// url.URL.Path は復号済みなので、%2e や %2E%2e の要素もここで捕まえる。文字列比較の上では
// 別の URL でも、取得経路の途中の正規化で同じ経路になりうるので、client_id と取得先の
// 同一性を守る側に倒す。
func hasDotSegment(path string) bool {
	for segment := range strings.SplitSeq(path, "/") {
		if segment == "." || segment == ".." {
			return true
		}
	}
	return false
}

type clientIDMetadataDocument struct {
	ClientID                string   `json:"client_id"`
	ClientName              string   `json:"client_name"`
	RedirectURIs            []string `json:"redirect_uris"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	Scope                   string   `json:"scope"`
	// 以下は 4.1 節が文書に禁じる素材の検査だけに読み、クライアントへは取り込まない。
	// json.RawMessage にするのは、値が null でもプロパティの存在を読むためである。
	ClientSecret          json.RawMessage `json:"client_secret"`
	ClientSecretExpiresAt json.RawMessage `json:"client_secret_expires_at"`
	JWKS                  json.RawMessage `json:"jwks"`
}

// privateJWKMembers は、JWK が秘密鍵か対称鍵を運んでいることを示すメンバーである
// (RFC 7517 §4、RFC 7518 §6.2.2、§6.3.2、§6.4.1、RFC 8037 §2)。
var privateJWKMembers = []string{"d", "p", "q", "dp", "dq", "qi", "oth", "k"}

// rejectCredentialMaterial は、4.1 節が文書に禁じる共有秘密と秘密鍵を含む文書を拒否する
// (CIMD02-CREDENTIAL-MATERIAL)。共有秘密の認証方式は token_endpoint_auth_method の検査が
// 先に拒否する。誤りには秘密の値を載せない。拒否の理由はイベントへ残るためである。
func (doc clientIDMetadataDocument) rejectCredentialMaterial() error {
	if doc.ClientSecret != nil || doc.ClientSecretExpiresAt != nil {
		return errors.New("client id metadata document: client_secret must not be published")
	}
	if doc.JWKS == nil {
		return nil
	}
	var set struct {
		Keys []map[string]json.RawMessage `json:"keys"`
	}
	// 鍵の配列として読めない jwks は、秘密鍵がないことを確かめられない。
	if err := json.Unmarshal(doc.JWKS, &set); err != nil {
		return errors.New("client id metadata document: jwks is not a JWK Set")
	}
	for _, key := range set.Keys {
		for _, member := range privateJWKMembers {
			if _, ok := key[member]; ok {
				return errors.New("client id metadata document: jwks must contain public keys only")
			}
		}
	}
	return nil
}

// ParseClientIDMetadataDocument parses and validates a fetched CIMD document
// against the URL it was fetched from (which is also the resulting client's
// client_id), returning a synthesized OAuth2Client that is never persisted.
// It performs no I/O; the caller is responsible for the SSRF-safe fetch.
//
// MVP only accepts documents that omit token_endpoint_auth_method or declare
// it as "none"; anything else is rejected fail-closed (rejected
// private_key_jwt-via-CIMD for this iteration).
func ParseClientIDMetadataDocument(raw []byte, requestURL string, now time.Time) (*OAuth2Client, error) {
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
	if err := doc.rejectCredentialMaterial(); err != nil {
		return nil, err
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
	now = now.UTC()
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
