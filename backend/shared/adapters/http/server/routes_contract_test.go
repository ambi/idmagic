package server

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
)

var routeParameterPattern = regexp.MustCompile(`:[^/]+|\{[^/}]+\}`)

type routeOperation struct {
	method string
	path   string
}

func (o routeOperation) String() string {
	return o.method + " " + o.path
}

func normalizeContractPath(path string) string {
	path = routeParameterPattern.ReplaceAllString(path, "{param}")
	for _, prefix := range []string{"/realms/{param}", "/realms/default"} {
		if path == prefix {
			return "/"
		}
		if strings.HasPrefix(path, prefix+"/") {
			return strings.TrimPrefix(path, prefix)
		}
	}
	return path
}

func operationSet(operations []routeOperation) map[routeOperation]struct{} {
	set := make(map[routeOperation]struct{}, len(operations))
	for _, operation := range operations {
		operation.method = strings.ToUpper(operation.method)
		operation.path = normalizeContractPath(operation.path)
		set[operation] = struct{}{}
	}
	return set
}

func TestNormalizeContractOperations(t *testing.T) {
	t.Parallel()

	got := operationSet([]routeOperation{
		{method: "get", path: "/realms/:tenant_id/api/users/:user_id"},
		{method: "GET", path: "/api/users/{id}"},
		{method: "post", path: "/realms/default/api/users"},
	})
	want := map[routeOperation]struct{}{
		{method: "GET", path: "/api/users/{param}"}: {},
		{method: "POST", path: "/api/users"}:        {},
	}
	if !setsEqual(got, want) {
		t.Fatalf("normalized operations = %v, want %v", sortedOperations(got), sortedOperations(want))
	}
}

func TestAssembledRoutesMatchGeneratedOpenAPI(t *testing.T) {
	e := echo.New()
	Register(e, Deps{})

	runtimeOperations := make([]routeOperation, 0, len(e.Router().Routes()))
	for _, route := range e.Router().Routes() {
		runtimeOperations = append(runtimeOperations, routeOperation{method: route.Method, path: route.Path})
	}

	openAPIOperations, err := loadGeneratedOpenAPIOperations()
	if err != nil {
		t.Fatal(err)
	}
	runtimeSet := operationSet(runtimeOperations)
	openAPISet := operationSet(openAPIOperations)
	runtimeOnly := setDifference(runtimeSet, openAPISet)
	specOnly := setDifference(openAPISet, runtimeSet)
	if len(runtimeOnly) != 0 || len(specOnly) != 0 {
		t.Fatalf(
			"assembled router and generated OpenAPI differ\nruntime-only (%d):\n%s\nspec-only (%d):\n%s",
			len(runtimeOnly), strings.Join(runtimeOnly, "\n"), len(specOnly), strings.Join(specOnly, "\n"),
		)
	}
}

func loadGeneratedOpenAPIOperations() ([]routeOperation, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return nil, fmt.Errorf("resolve contract test source path")
	}
	path := filepath.Join(
		filepath.Dir(filename), "..", "..", "..", "..", "..", "spec", "idmagic.openapi.json",
	)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read generated OpenAPI %s: %w", path, err)
	}
	var document struct {
		Paths map[string]map[string]json.RawMessage `json:"paths"`
	}
	if err := json.Unmarshal(data, &document); err != nil {
		return nil, fmt.Errorf("decode generated OpenAPI %s: %w", path, err)
	}

	httpMethods := map[string]struct{}{
		"get": {}, "put": {}, "post": {}, "delete": {}, "options": {}, "head": {}, "patch": {}, "trace": {},
	}
	operations := make([]routeOperation, 0, len(document.Paths))
	for path, item := range document.Paths {
		for method := range item {
			if _, ok := httpMethods[strings.ToLower(method)]; !ok {
				continue
			}
			operations = append(operations, routeOperation{method: method, path: path})
		}
	}
	return operations, nil
}

func setsEqual(left, right map[routeOperation]struct{}) bool {
	return len(left) == len(right) && len(setDifference(left, right)) == 0
}

func setDifference(left, right map[routeOperation]struct{}) []string {
	difference := make(map[routeOperation]struct{})
	for operation := range left {
		if _, ok := right[operation]; !ok {
			difference[operation] = struct{}{}
		}
	}
	return sortedOperations(difference)
}

func sortedOperations(set map[routeOperation]struct{}) []string {
	operations := make([]string, 0, len(set))
	for operation := range set {
		operations = append(operations, operation.String())
	}
	slices.Sort(operations)
	return operations
}
