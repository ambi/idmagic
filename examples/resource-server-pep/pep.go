// Package pep is a minimal reference Policy Enforcement Point for a resource server
// protected by IdMagic. It depends only on the public HTTP contract of
// POST /resource-access/evaluate and on the standard library, so it can be copied into
// a resource server that is deployed and upgraded separately from IdMagic.
//
// The one rule that matters: the wrapped handler runs only after a response that says
// decision=true. Every other outcome, including a network failure, runs nothing.
package pep

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

const maxResponseBytes = 1 << 20

// Operation names what a request is about to do.
type Operation struct {
	ResourceType string
	ResourceID   string
	Relation     string
	// Key and Parameters apply to rules with requires_approved_operation. Key must keep the
	// same value across retries of one business request, and Parameters must be a JSON object
	// built from the values the handler will actually use, not copied from the approval.
	Key        string
	Parameters string
}

// ApprovedOperation correlates a permitted use with the human approval. Store it with the
// business result.
type ApprovedOperation struct {
	ApprovalRequestID string `json:"approval_request_id"`
	OperationDigest   string `json:"operation_digest"`
	Replayed          bool   `json:"replayed"`
}

// Decision is the part of the evaluation response a resource server acts on.
type Decision struct {
	Permitted         bool               `json:"decision"`
	Reasons           []string           `json:"reasons"`
	Scope             string             `json:"scope"`
	ApprovedOperation *ApprovedOperation `json:"approved_operation"`
}

// ErrUnavailable means the evaluation did not produce a decision. Treat it as a denial.
var ErrUnavailable = errors.New("access evaluation unavailable")

// Enforcer calls the evaluation endpoint with the resource server's own credentials.
type Enforcer struct {
	EvaluateURL  string
	ClientID     string
	ClientSecret string
	// PublicBaseURL is the URL clients use to reach this resource server. It is the base
	// of the htu a DPoP proof must name, which the server cannot infer behind a proxy.
	PublicBaseURL string
	HTTPClient    *http.Client
}

// Evaluate asks whether the token presented with r may perform op. Do not cache the
// result: a stop takes effect at the next evaluation only if every operation evaluates.
func (e Enforcer) Evaluate(ctx context.Context, r *http.Request, op Operation) (Decision, error) {
	scheme, token, found := strings.Cut(r.Header.Get("Authorization"), " ")
	if !found || token == "" || (!strings.EqualFold(scheme, "Bearer") && !strings.EqualFold(scheme, "DPoP")) {
		return Decision{}, nil
	}
	form := url.Values{
		"token":         {token},
		"resource_type": {op.ResourceType},
		"resource_id":   {op.ResourceID},
		"relation":      {op.Relation},
	}
	if op.Key != "" {
		form.Set("operation_key", op.Key)
	}
	if op.Parameters != "" {
		form.Set("operation_parameters", op.Parameters)
	}
	if proof := r.Header.Get("DPoP"); proof != "" {
		form.Set("dpop_proof", proof)
		form.Set("htm", r.Method)
		form.Set("htu", strings.TrimSuffix(e.PublicBaseURL, "/")+r.URL.Path)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, e.EvaluateURL, strings.NewReader(form.Encode()))
	if err != nil {
		return Decision{}, fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.SetBasicAuth(e.ClientID, e.ClientSecret)
	client := e.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	response, err := client.Do(request)
	if err != nil {
		return Decision{}, fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return Decision{}, fmt.Errorf("%w: status %d", ErrUnavailable, response.StatusCode)
	}
	var decision Decision
	if err := json.NewDecoder(io.LimitReader(response.Body, maxResponseBytes)).Decode(&decision); err != nil {
		return Decision{}, fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	return decision, nil
}

// ExecuteResult is what a resource server knows about one attempt at a business effect.
type ExecuteResult int

const (
	// ExecuteSucceeded means the effect is committed.
	ExecuteSucceeded ExecuteResult = iota
	// ExecuteFailed means the effect certainly did not happen.
	ExecuteFailed
	// ExecuteUnknown means the attempt ended without knowing, for example on a timeout.
	ExecuteUnknown
)

// OperationLedger is the resource server's own idempotency record, committed with the
// business effect. IdMagic cannot make an operation run once without it.
type OperationLedger struct {
	mu      sync.Mutex
	results map[string]ExecuteResult
}

// Run executes effect at most once per key while its result is success or unknown. An
// unknown result is returned again instead of re-executing: only the business system can
// resolve it. A definite failure may run again under the same key.
func (l *OperationLedger) Run(key string, effect func() ExecuteResult) ExecuteResult {
	l.mu.Lock()
	defer l.mu.Unlock()
	if previous, seen := l.results[key]; seen && previous != ExecuteFailed {
		return previous
	}
	if l.results == nil {
		l.results = map[string]ExecuteResult{}
	}
	result := effect()
	l.results[key] = result
	return result
}

// Protect runs next only when the evaluation of the request's operation permits it.
func (e Enforcer) Protect(operation func(*http.Request) Operation, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		decision, err := e.Evaluate(r.Context(), r, operation(r))
		switch {
		case err != nil:
			http.Error(w, "authorization is unavailable", http.StatusServiceUnavailable)
		case !decision.Permitted:
			http.Error(w, "forbidden", http.StatusForbidden)
		default:
			next.ServeHTTP(w, r)
		}
	})
}
