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
)

const maxResponseBytes = 1 << 20

// Operation names what a request is about to do.
type Operation struct {
	ResourceType string
	ResourceID   string
	Relation     string
}

// Decision is the part of the evaluation response a resource server acts on.
type Decision struct {
	Permitted bool     `json:"decision"`
	Reasons   []string `json:"reasons"`
	Scope     string   `json:"scope"`
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
