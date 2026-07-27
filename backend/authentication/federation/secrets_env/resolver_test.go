package secrets_env

import (
	"context"
	"testing"
)

func TestResolverOnlyAcceptsEnvironmentReferences(t *testing.T) {
	resolver := Resolver{Lookup: func(name string) (string, bool) {
		if name == "IDP_CLIENT_SECRET" {
			return "secret-value", true
		}
		return "", false
	}}

	value, err := resolver.Resolve(context.Background(), "env:IDP_CLIENT_SECRET")
	if err != nil || value != "secret-value" {
		t.Fatalf("Resolve() = %q, %v", value, err)
	}
	for _, reference := range []string{"secret-value", "env:", "env:bad-name", "env:MISSING"} {
		if _, err := resolver.Resolve(context.Background(), reference); err == nil {
			t.Fatalf("Resolve(%q) succeeded, want error", reference)
		}
	}
}
