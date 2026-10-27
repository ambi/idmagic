package testing_statematrix_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/ambi/idmagic/backend/shared/testing_statematrix"
)

const fixture = "testdata/TaskLifecycle.statemachine.json"

func TestLoadReturnsTheMachineWithEveryCell(t *testing.T) {
	machine, err := testing_statematrix.Load(fixture)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"Ready", "Done"}; !reflect.DeepEqual(machine.States, want) {
		t.Fatalf("States=%v, want %v", machine.States, want)
	}
	if machine.Initial != "Ready" {
		t.Fatalf("Initial=%q, want Ready", machine.Initial)
	}
	if want := []string{"実行", "取り消し"}; !reflect.DeepEqual(machine.Operations, want) {
		t.Fatalf("Operations=%v, want %v", machine.Operations, want)
	}
	for _, tc := range []struct {
		state, operation string
		want             []testing_statematrix.Outcome
	}{
		{"Ready", "実行", []testing_statematrix.Outcome{{Kind: testing_statematrix.Transition, To: "Done"}}},
		{"Ready", "取り消し", []testing_statematrix.Outcome{
			{Kind: testing_statematrix.NoOp, Condition: "期限内"},
			{Kind: testing_statematrix.Refusal, Refusal: "409 task_expired", Condition: "期限後"},
		}},
		{"Done", "実行", []testing_statematrix.Outcome{{Kind: testing_statematrix.NoOp}}},
		{"Done", "取り消し", []testing_statematrix.Outcome{{Kind: testing_statematrix.Refusal, Refusal: "404 task_not_found"}}},
	} {
		if got := machine.Outcomes(tc.state, tc.operation); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("Outcomes(%s, %s)=%+v, want %+v", tc.state, tc.operation, got, tc.want)
		}
	}
}

func TestEventForNamesTheTransitionTableEvent(t *testing.T) {
	machine, err := testing_statematrix.Load(fixture)
	if err != nil {
		t.Fatal(err)
	}
	if event, ok := machine.EventFor("Ready", "Done"); !ok || event != "TaskDone" {
		t.Fatalf("EventFor(Ready, Done)=%q, %v, want TaskDone, true", event, ok)
	}
	if _, ok := machine.EventFor("Done", "Ready"); ok {
		t.Fatal("EventFor(Done, Ready) found a transition the table does not list")
	}
	if want := []string{"TaskDone"}; !reflect.DeepEqual(machine.Events(), want) {
		t.Fatalf("Events()=%v, want %v", machine.Events(), want)
	}
}

// 一次情報の形の誤りは `check-spec` も拒否するが、テストが誤った一次情報から予測を始めないよう、
// 読み取りでも予測に要る形を確かめる。
func TestLoadRejectsAMalformedSource(t *testing.T) {
	data, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	source := string(data)
	for _, tc := range []struct{ name, source, want string }{
		{"missing file", "", "no such file"},
		{"unknown field", strings.Replace(source, `"operations"`, `"extra": 1, "operations"`, 1), "extra"},
		{"unknown kind", strings.Replace(source, `"kind": "noop" }]`, `"kind": "skip" }]`, 1), "skip"},
		{"missing cell", strings.Replace(source, `"取り消し": [{ "kind": "refusal", "response": "404 task_not_found" }]`, `"取り消し": []`, 1), "Done × 取り消し"},
		{"missing row", strings.Replace(source, `"Done": {`, `"Gone": {`, 1), "Done"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "machine.statemachine.json")
			if tc.source != "" {
				if err := os.WriteFile(path, []byte(tc.source), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			_, err := testing_statematrix.Load(path)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err=%v, want an error naming %q", err, tc.want)
			}
		})
	}
}
