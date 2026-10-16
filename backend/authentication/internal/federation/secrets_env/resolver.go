// Package secrets_env resolves identity-provider secrets from process
// environment references without persisting their values in IdMagic.
package secrets_env

import (
	"context"
	"errors"
	"os"
	"regexp"
	"strings"
)

var environmentName = regexp.MustCompile(`^[A-Z_][A-Z0-9_]*$`)

type Resolver struct {
	Lookup func(string) (string, bool)
}

func (r Resolver) Resolve(_ context.Context, reference string) (string, error) {
	const prefix = "env:"
	if !strings.HasPrefix(reference, prefix) {
		return "", errors.New("secret reference must use the env: scheme")
	}
	name := strings.TrimPrefix(reference, prefix)
	if !environmentName.MatchString(name) {
		return "", errors.New("secret reference contains an invalid environment variable name")
	}
	lookup := r.Lookup
	if lookup == nil {
		lookup = os.LookupEnv
	}
	value, ok := lookup(name)
	if !ok || value == "" {
		return "", errors.New("referenced environment secret is unavailable")
	}
	return value, nil
}
