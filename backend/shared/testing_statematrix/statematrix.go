// Package testing_statematrix は、状態機械の一次情報 `testdata/<Machine>.statemachine.json` を読み、
// モデルベースのテストが予測に使える形で返す。
//
// 機能仕様の状態遷移の節の三つの表は同じ一次情報から生成し、テストは文書の Markdown を
// 読まない。形は docs/formats/specification-format.md の「状態遷移」が定め、`mise run check-spec` が
// 一次情報と生成した表の一致を検査する。ここでは予測に要る形だけを確かめ、誤りは見つけ次第エラーにする。
package testing_statematrix

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
)

// OutcomeKind は、状態遷移表（マトリクス形式）の一つの結果の種類である。
type OutcomeKind int

const (
	// Transition は `→ <State>`。その状態へ遷移する。
	Transition OutcomeKind = iota + 1
	// NoOp は `何もしない`。成功を返し、状態もイベントも変えない。
	NoOp
	// Refusal は `拒否：<返すもの>`。拒否し、状態を変えない。
	Refusal
)

// Outcome は、一つのセルが述べる結果の一つである。
type Outcome struct {
	Kind OutcomeKind
	// To は Transition の遷移先。
	To string
	// Refusal は拒否で返すもの（`409 user_pending_deletion` など）。解釈は呼び出し側が決める。
	Refusal string
	// Condition は結果に添えた括弧の条件。状態だけで結果が決まるセルでは空である。
	Condition string
}

// TransitionRow は遷移の表の一行のうち、予測に使う列である。
type TransitionRow struct {
	From, Event, To string
}

// Machine は一つの状態機械の三つの表である。
type Machine struct {
	States      []string
	Initial     string
	Transitions []TransitionRow
	Operations  []string
	cells       map[[2]string][]Outcome
}

// Outcomes は、state の行と operation の列のセルが述べる結果を返す。
func (m Machine) Outcomes(state, operation string) []Outcome {
	return m.cells[[2]string{state, operation}]
}

// EventFor は、from から to への遷移に遷移の表が付けるイベントを返す。
func (m Machine) EventFor(from, to string) (string, bool) {
	for _, row := range m.Transitions {
		if row.From == from && row.To == to {
			return row.Event, true
		}
	}
	return "", false
}

// Events は遷移の表に現れるイベントを、重複を除いて現れた順に返す。
func (m Machine) Events() []string {
	var events []string
	seen := map[string]bool{}
	for _, row := range m.Transitions {
		if !seen[row.Event] {
			seen[row.Event] = true
			events = append(events, row.Event)
		}
	}
	return events
}

// source は一次情報の JSON の形である。表だけに要る列（Meaning、Guard、Effects）も、知らない欄を
// 拒否するために受け取る。
type source struct {
	Name   string `json:"name"`
	States []struct {
		Name    string `json:"name"`
		Kind    string `json:"kind"`
		Meaning string `json:"meaning"`
	} `json:"states"`
	Transitions []struct {
		From    string `json:"from"`
		Event   string `json:"event"`
		Guard   string `json:"guard"`
		To      string `json:"to"`
		Effects string `json:"effects"`
	} `json:"transitions"`
	Operations []string                              `json:"operations"`
	Matrix     map[string]map[string][]sourceOutcome `json:"matrix"`
}

type sourceOutcome struct {
	Kind      string `json:"kind"`
	To        string `json:"to"`
	Response  string `json:"response"`
	Condition string `json:"condition"`
}

// Load は path の状態機械の一次情報を読む。path はテストのパッケージの `testdata` を指す。
func Load(path string) (Machine, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Machine{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var src source
	if err := decoder.Decode(&src); err != nil {
		return Machine{}, fmt.Errorf("%s: %w", path, err)
	}
	machine := Machine{Operations: src.Operations, cells: map[[2]string][]Outcome{}}
	for _, state := range src.States {
		machine.States = append(machine.States, state.Name)
		if state.Kind == "initial" {
			machine.Initial = state.Name
		}
	}
	for _, row := range src.Transitions {
		machine.Transitions = append(machine.Transitions, TransitionRow{From: row.From, Event: row.Event, To: row.To})
	}
	for _, state := range machine.States {
		for _, operation := range machine.Operations {
			outcomes, err := parseCell(src.Matrix[state][operation])
			if err != nil {
				return Machine{}, fmt.Errorf("%s: %s × %s: %w", src.Name, state, operation, err)
			}
			machine.cells[[2]string{state, operation}] = outcomes
		}
	}
	return machine, nil
}

func parseCell(cell []sourceOutcome) ([]Outcome, error) {
	if len(cell) == 0 {
		return nil, fmt.Errorf("the cell gives no outcome")
	}
	outcomes := make([]Outcome, 0, len(cell))
	for _, item := range cell {
		outcome := Outcome{Condition: item.Condition}
		switch item.Kind {
		case "transition":
			outcome.Kind, outcome.To = Transition, item.To
		case "noop":
			outcome.Kind = NoOp
		case "refusal":
			outcome.Kind, outcome.Refusal = Refusal, item.Response
		default:
			return nil, fmt.Errorf("outcome kind %q is not transition, noop, or refusal", item.Kind)
		}
		outcomes = append(outcomes, outcome)
	}
	return outcomes, nil
}
