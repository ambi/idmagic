package testing_statematrix_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/ambi/idmagic/backend/shared/testing_statematrix"
)

const specification = `# タスク

## 状態遷移

### OtherLifecycle

| State | Kind | Meaning |
|---|---|---|
| Open | initial | 別の状態機械 |

### TaskLifecycle

| State | Kind | Meaning |
|---|---|---|
| Ready | initial | 受理直後 |
| Done | terminal | 完了 |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Ready | TaskDone | input.force == true \|\| expired | Done | TaskDone |

| State | 実行 | 取り消し |
|---|---|---|
| Ready | → Done | 何もしない（期限内）<br>拒否：409 task_expired（期限後） |
| Done | 何もしない | 拒否：404 task_not_found |

## 操作
`

func TestReadReturnsTheNamedMachineWithEveryCell(t *testing.T) {
	machine, err := testing_statematrix.Read(specification, "TaskLifecycle")
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

func TestReadReadsAMachineAtTheEndOfTheDocument(t *testing.T) {
	machine, err := testing_statematrix.Read(strings.TrimSuffix(specification, "## 操作\n"), "TaskLifecycle")
	if err != nil {
		t.Fatal(err)
	}
	if len(machine.Outcomes("Done", "取り消し")) != 1 {
		t.Fatalf("Outcomes(Done, 取り消し)=%+v, want the last row read", machine.Outcomes("Done", "取り消し"))
	}
}

func TestEventForNamesTheTransitionTableEvent(t *testing.T) {
	machine, err := testing_statematrix.Read(specification, "TaskLifecycle")
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

func TestReadRejectsAMissingMachineAndAMissingMatrix(t *testing.T) {
	for _, tc := range []struct{ name, markdown, machine, want string }{
		{"machine", specification, "GoneLifecycle", "GoneLifecycle"},
		{"matrix", specification, "OtherLifecycle", "state matrix"},
		{"outcome", strings.Replace(specification, "| Done | 何もしない |", "| Done | 未定 |", 1), "TaskLifecycle", "未定"},
		{"short row", strings.Replace(specification, "| Done | 何もしない | 拒否：404 task_not_found |", "| Done | 何もしない |", 1), "TaskLifecycle", "no outcome"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := testing_statematrix.Read(tc.markdown, tc.machine)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err=%v, want an error naming %q", err, tc.want)
			}
		})
	}
}
