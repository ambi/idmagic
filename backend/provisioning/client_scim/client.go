package client_scim

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/ambi/idmagic/backend/provisioning/domain"
	"github.com/ambi/idmagic/backend/provisioning/ports"
)

var _ ports.ProvisioningTargetClient = (*Client)(nil)

// Client is the outbound SCIM 2.0 wire client for one ProvisioningConnection.
// It sends idempotent, mapping-driven resource documents built by BuildResource
// to a downstream SCIM service provider (spec/contexts/provisioning.yaml §配送・信頼性).
// HTTPClient is injectable so tests can point it at an httptest.Server without
// the SSRF-safe dialer NewClient builds for production use (mirrors
// backend/shared/security/tokens_jose.JWKResolver's ValidateJWKSURI/safeIPs split).
type Client struct {
	HTTPClient  *http.Client
	BaseURL     string
	BearerToken string
}

// NewClient validates baseURL (ValidateOutboundBaseURL) and builds a Client with
// an SSRF-safe transport for production use.
func NewClient(baseURL, bearerToken string) (*Client, error) {
	if err := domain.ValidateOutboundBaseURL(baseURL); err != nil {
		return nil, err
	}
	return &Client{HTTPClient: newSafeHTTPClient(), BaseURL: baseURL, BearerToken: bearerToken}, nil
}

const maxResponseBytes = 1 << 20 // 1 MiB, mirrors backend/shared/security/tokens_jose.maxJWKSBytes

func (c *Client) do(ctx context.Context, method, path string, body any) (*http.Response, []byte, error) {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, nil, err
		}
		reader = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, reader)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Accept", "application/scim+json")
	if body != nil {
		req.Header.Set("Content-Type", "application/scim+json")
	}
	if c.BearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.BearerToken)
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return nil, nil, err
	}
	return resp, data, nil
}

// classify maps a non-2xx SCIM response to the protocol-agnostic error
// taxonomy ports.ConflictError/ports.NotFoundError/ports.RetryableError
// declares (ADR-128 decision 2: the delivery engine usecase must not depend
// on this package).
func classify(resp *http.Response, data []byte) error {
	switch {
	case resp.StatusCode == http.StatusConflict:
		return &ports.ConflictError{Detail: scimErrorDetail(data)}
	case resp.StatusCode == http.StatusNotFound:
		return &ports.NotFoundError{}
	case resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500:
		return &ports.RetryableError{StatusCode: resp.StatusCode, RetryAfter: parseRetryAfter(resp.Header.Get("Retry-After"))}
	default:
		return fmt.Errorf("provisioning/scim: unexpected status %d: %s", resp.StatusCode, scimErrorDetail(data))
	}
}

func scimErrorDetail(data []byte) string {
	var payload struct {
		Detail string `json:"detail"`
	}
	if json.Unmarshal(data, &payload) == nil && payload.Detail != "" {
		return payload.Detail
	}
	return string(data)
}

func parseRetryAfter(header string) time.Duration {
	if header == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(header); err == nil {
		return time.Duration(seconds) * time.Second
	}
	if when, err := http.ParseTime(header); err == nil {
		return time.Until(when)
	}
	return 0
}

// Discover fetches /ServiceProviderConfig and caches nothing itself; callers
// persist the result on ProvisioningConnection.Capabilities
// (spec/contexts/provisioning.yaml interfaces.TestProvisioningConnection).
func (c *Client) Discover(ctx context.Context) (domain.ProvisioningCapabilities, error) {
	resp, data, err := c.do(ctx, http.MethodGet, "/ServiceProviderConfig", nil) //nolint:bodyclose // do() already reads and closes resp.Body before returning
	if err != nil {
		return domain.ProvisioningCapabilities{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return domain.ProvisioningCapabilities{}, classify(resp, data)
	}
	var payload struct {
		Patch  struct{ Supported bool } `json:"patch"`
		Bulk   struct{ Supported bool } `json:"bulk"`
		Filter struct{ Supported bool } `json:"filter"`
		Etag   struct{ Supported bool } `json:"etag"`
		Sort   struct{ Supported bool } `json:"sort"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return domain.ProvisioningCapabilities{}, fmt.Errorf("provisioning/scim: decode ServiceProviderConfig: %w", err)
	}
	return domain.ProvisioningCapabilities{
		SupportsPatch:  payload.Patch.Supported,
		SupportsBulk:   payload.Bulk.Supported,
		SupportsFilter: payload.Filter.Supported,
		SupportsEtag:   payload.Etag.Supported,
		SupportsSort:   payload.Sort.Supported,
	}, nil
}

// CreateUser applies rules against attrs (BuildResource) and POSTs the result to
// /Users, returning the downstream id and etag (etag is nil when the response
// has none, e.g. supports_etag=false). It satisfies ports.ProvisioningTargetClient.
func (c *Client) CreateUser(ctx context.Context, rules []domain.AttributeMappingRule, attrs map[string]any) (remoteID string, etag *string, err error) {
	doc, err := BuildResource(rules, resolverFromMap(attrs), ApplyOnCreate)
	if err != nil {
		return "", nil, err
	}
	return c.createResource(ctx, "/Users", doc)
}

// CreateGroup applies rules against attrs and POSTs the result to /Groups.
func (c *Client) CreateGroup(ctx context.Context, rules []domain.AttributeMappingRule, attrs map[string]any) (remoteID string, etag *string, err error) {
	doc, err := BuildResource(rules, resolverFromMap(attrs), ApplyOnCreate)
	if err != nil {
		return "", nil, err
	}
	return c.createResource(ctx, "/Groups", doc)
}

func resolverFromMap(attrs map[string]any) AttributeResolver {
	return func(key string) (any, bool) {
		v, ok := attrs[key]
		return v, ok
	}
}

func (c *Client) createResource(ctx context.Context, path string, doc map[string]any) (string, *string, error) {
	resp, data, err := c.do(ctx, http.MethodPost, path, doc) //nolint:bodyclose // do() already reads and closes resp.Body before returning
	if err != nil {
		return "", nil, err
	}
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", nil, classify(resp, data)
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(data, &created); err != nil {
		return "", nil, fmt.Errorf("provisioning/scim: decode created resource: %w", err)
	}
	return created.ID, etagFromHeader(resp), nil
}

func etagFromHeader(resp *http.Response) *string {
	if v := resp.Header.Get("ETag"); v != "" {
		return &v
	}
	return nil
}

// UpdateUser applies rules against attrs (BuildResource) as the desired-state
// document for remoteID. When supportsPatch is true it sends a single PATCH
// replace operation (RFC 7644 §3.5.2.1, an operation without "path" replaces
// the listed top-level attributes); otherwise it falls back to PUT (full
// resource replace), per wi-45's PATCH→PUT fallback requirement.
func (c *Client) UpdateUser(ctx context.Context, remoteID string, rules []domain.AttributeMappingRule, attrs map[string]any, supportsPatch bool) (*string, error) {
	doc, err := BuildResource(rules, resolverFromMap(attrs), ApplyOnUpdate)
	if err != nil {
		return nil, err
	}
	return c.updateResource(ctx, "/Users/"+url.PathEscape(remoteID), doc, supportsPatch)
}

// UpdateGroup applies rules against attrs as the desired-state document for a
// Group (excluding members, which PatchGroupMembers manages incrementally).
func (c *Client) UpdateGroup(ctx context.Context, remoteID string, rules []domain.AttributeMappingRule, attrs map[string]any, supportsPatch bool) (*string, error) {
	doc, err := BuildResource(rules, resolverFromMap(attrs), ApplyOnUpdate)
	if err != nil {
		return nil, err
	}
	return c.updateResource(ctx, "/Groups/"+url.PathEscape(remoteID), doc, supportsPatch)
}

func (c *Client) updateResource(ctx context.Context, path string, doc map[string]any, supportsPatch bool) (*string, error) {
	method := http.MethodPut
	body := any(doc)
	if supportsPatch {
		method = http.MethodPatch
		body = map[string]any{
			"schemas":    []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []any{map[string]any{"op": "replace", "value": doc}},
		}
	}
	resp, data, err := c.do(ctx, method, path, body) //nolint:bodyclose // do() already reads and closes resp.Body before returning
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, classify(resp, data)
	}
	return etagFromHeader(resp), nil
}

// PatchGroupMembers sends a single add/remove Operation against a Group's
// members attribute (spec/contexts/provisioning.yaml events.GroupMembershipPushed).
// op must be "add" or "remove".
func (c *Client) PatchGroupMembers(ctx context.Context, remoteGroupID, op string, remoteUserIDs []string) error {
	members := make([]map[string]string, 0, len(remoteUserIDs))
	for _, id := range remoteUserIDs {
		members = append(members, map[string]string{"value": id})
	}
	body := map[string]any{
		"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
		"Operations": []any{map[string]any{
			"op":    op,
			"path":  "members",
			"value": members,
		}},
	}
	resp, data, err := c.do(ctx, http.MethodPatch, "/Groups/"+url.PathEscape(remoteGroupID), body) //nolint:bodyclose // do() already reads and closes resp.Body before returning
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return classify(resp, data)
	}
	return nil
}

// DeleteUser DELETEs /Users/{remoteID}. A 404 is treated as idempotent success
// (the resource is already gone, which is the desired end state).
func (c *Client) DeleteUser(ctx context.Context, remoteID string) error {
	return c.deleteResource(ctx, "/Users/"+url.PathEscape(remoteID))
}

// DeleteGroup DELETEs /Groups/{remoteID}, with the same 404-is-success idempotency.
func (c *Client) DeleteGroup(ctx context.Context, remoteID string) error {
	return c.deleteResource(ctx, "/Groups/"+url.PathEscape(remoteID))
}

func (c *Client) deleteResource(ctx context.Context, path string) error {
	resp, data, err := c.do(ctx, http.MethodDelete, path, nil) //nolint:bodyclose // do() already reads and closes resp.Body before returning
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return classify(resp, data)
	}
	return nil
}

// SearchUserByAttribute resolves a 409 conflict by looking up the existing
// downstream resource via MatchingRule.conflict_match_attribute
// (spec/contexts/provisioning.yaml models.MatchingRule).
func (c *Client) SearchUserByAttribute(ctx context.Context, attribute, value string) (remoteID string, found bool, err error) {
	return c.searchResource(ctx, "/Users", attribute, value)
}

// SearchGroupByAttribute is the Group analogue of SearchUserByAttribute.
func (c *Client) SearchGroupByAttribute(ctx context.Context, attribute, value string) (remoteID string, found bool, err error) {
	return c.searchResource(ctx, "/Groups", attribute, value)
}

func (c *Client) searchResource(ctx context.Context, path, attribute, value string) (string, bool, error) {
	filter := fmt.Sprintf("%s eq %q", attribute, value)
	resp, data, err := c.do(ctx, http.MethodGet, path+"?filter="+url.QueryEscape(filter), nil) //nolint:bodyclose // do() already reads and closes resp.Body before returning
	if err != nil {
		return "", false, err
	}
	if resp.StatusCode != http.StatusOK {
		return "", false, classify(resp, data)
	}
	var list struct {
		TotalResults int `json:"totalResults"`
		Resources    []struct {
			ID string `json:"id"`
		} `json:"Resources"`
	}
	if err := json.Unmarshal(data, &list); err != nil {
		return "", false, fmt.Errorf("provisioning/scim: decode search results: %w", err)
	}
	if list.TotalResults == 0 || len(list.Resources) == 0 {
		return "", false, nil
	}
	return list.Resources[0].ID, true, nil
}
