package spec

import (
	"strings"
	"sync"
)

// InteractiveSessionScope marks an operation that no API access token may reach,
// whatever its scopes. It is the value the RFC 6750 challenge advertises for
// those operations, matching the step-up sentinel the account API already uses.
const InteractiveSessionScope = "interactive_session"

// Operation is runtime metadata derived from the TypeSpec HTTP contract.
type Operation struct {
	Method string
	Path   string
	// ApiTokenScopes are the ApiTokenScope values that let an API access token
	// reach this operation; holding any one of them is enough. A nil slice means
	// the contract declares nothing, which callers must treat as fail-closed.
	ApiTokenScopes []string
	Deprecated     bool
}

// RuntimeContract is the small generated subset of the language-independent
// contract that runtime endpoints need for self-description.
type RuntimeContract struct {
	Operations   map[string]Operation
	Deprecations map[string]Deprecation

	routesOnce sync.Once
	routes     map[routeKey]Operation
}

type routeKey struct{ Method, Path string }

// Deprecation carries the dates required for RFC 9745 and RFC 8594 headers.
// The key in RuntimeContract.Deprecations is a TypeSpec operation name.
type Deprecation struct {
	Since  string
	Sunset string
}

// CurrentRuntimeContract returns the contract compiled into this build.
func CurrentRuntimeContract() *RuntimeContract {
	return &RuntimeContract{Operations: generatedOperations, Deprecations: map[string]Deprecation{}}
}

func (c *RuntimeContract) Operation(name string) (Operation, bool) {
	if c == nil {
		return Operation{}, false
	}
	operation, ok := c.Operations[name]
	return operation, ok
}

// OperationForRoute resolves the contract operation a router matched, keyed by
// the HTTP method and the matched route template. Path parameter names differ
// between the contract (`{user_id}`) and the router (`:sub`), so both sides are
// reduced to the same positional shape before matching.
func (c *RuntimeContract) OperationForRoute(method, routePath string) (Operation, bool) {
	if c == nil {
		return Operation{}, false
	}
	c.routesOnce.Do(func() {
		c.routes = make(map[routeKey]Operation, len(c.Operations))
		for _, operation := range c.Operations {
			c.routes[routeKey{operation.Method, NormalizeRoutePath(operation.Path)}] = operation
		}
	})
	operation, ok := c.routes[routeKey{strings.ToUpper(method), NormalizeRoutePath(routePath)}]
	return operation, ok
}

// NormalizeRoutePath replaces every path parameter with a single placeholder so
// that contract paths and router templates compare by position, not by the name
// each side happens to give a segment.
func NormalizeRoutePath(path string) string {
	segments := strings.Split(path, "/")
	for i, segment := range segments {
		if strings.HasPrefix(segment, ":") ||
			(strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}")) {
			segments[i] = "{}"
		}
	}
	return strings.Join(segments, "/")
}
