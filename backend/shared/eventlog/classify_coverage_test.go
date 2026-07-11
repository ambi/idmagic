package eventlog

// wi-184 T004 (ADR-094 decision 4): every DomainEvent implementation
// anywhere in the codebase must have a routing decision in classify.go's
// classification map. This scans backend/ for
// `func (...) EventType() string { return "X" }` — the single-literal-return
// form every DomainEvent in this codebase uses — and reconciles the result
// against the map in both directions, so a new event type with no decision,
// or a stale map entry with no matching type, fails CI instead of drifting
// silently.

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"
)

var eventTypeMethodPattern = regexp.MustCompile(`(?s)func\s*\([^)]+\)\s*EventType\(\)\s*string\s*\{\s*return\s*"([^"]+)"`)

// backendRoot resolves the repo's backend/ directory relative to this file,
// independent of the test binary's working directory.
func backendRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve caller for backendRoot")
	}
	// this file: backend/shared/eventlog/classify_coverage_test.go
	return filepath.Join(filepath.Dir(file), "..", "..")
}

// collectDomainEventTypes scans every non-generated, non-test .go file under
// backend/ and returns each EventType() literal found, mapped to one file it
// was found in (for error messages).
func collectDomainEventTypes(t *testing.T) map[string]string {
	t.Helper()
	root := backendRoot(t)
	found := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "sqlcgen" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for _, m := range eventTypeMethodPattern.FindAllSubmatch(data, -1) {
			eventType := string(m[1])
			if _, exists := found[eventType]; !exists {
				rel, relErr := filepath.Rel(root, path)
				if relErr != nil {
					rel = path
				}
				found[eventType] = rel
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk backend/: %v", err)
	}
	return found
}

func TestAllDomainEventTypesAreClassified(t *testing.T) {
	found := collectDomainEventTypes(t)
	if len(found) < 50 {
		// A drastic drop signals the scan itself is broken (wrong root, regex
		// stopped matching), not that the codebase suddenly shrank.
		t.Fatalf("only found %d DomainEvent EventType() implementations; scan is likely broken", len(found))
	}
	var missing []string
	for eventType, path := range found {
		if _, ok := classification[eventType]; !ok {
			missing = append(missing, eventType+" ("+path+")")
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		t.Fatalf(
			"%d DomainEvent type(s) have no ADR-094 classification in classify.go (wi-184 T004): "+
				"add an entry to the classification map (public_integration/audit_only/telemetry):\n%s",
			len(missing), strings.Join(missing, "\n"),
		)
	}
}

func TestClassificationMapHasNoStaleEntries(t *testing.T) {
	found := collectDomainEventTypes(t)
	var stale []string
	for eventType := range classification {
		if _, ok := found[eventType]; !ok {
			stale = append(stale, eventType)
		}
	}
	if len(stale) > 0 {
		sort.Strings(stale)
		t.Fatalf(
			"classification map has %d entr(y/ies) with no matching DomainEvent implementation "+
				"under backend/ (remove or fix the type name): %s",
			len(stale), strings.Join(stale, ", "),
		)
	}
}
