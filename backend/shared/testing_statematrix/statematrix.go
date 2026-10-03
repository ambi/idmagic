// Package testing_statematrix は、機能仕様の状態遷移の節から一つの状態機械を読み、
// モデルベースのテストが予測に使える形で返す。
//
// テストは表を手で写さず、この読み取りを通して仕様の文書そのものを予測の一次情報にする。
// 書式は SPECIFICATION_FORMAT.md の「状態遷移」が定め、`mise run check-spec` が
// 同じ書式を検査する。ここでは予測に要る形だけを読み、書式の誤りは見つけ次第エラーにする。
package testing_statematrix

import (
	"fmt"
	"regexp"
	"strings"
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

const (
	stateHeader      = "| State | Kind | Meaning |"
	transitionHeader = "| From | Event | Guard | To | Effects |"
)

var outcomePattern = regexp.MustCompile(`^(?:→ ([^\s（]+)|(何もしない)|拒否：(\S.*?))(?:（([^（）]+)）)?$`)

// Read は markdown の `### <name>` の状態機械を読む。状態遷移表（マトリクス形式）がない
// 状態機械は、予測に使えないのでエラーにする。
func Read(markdown, name string) (Machine, error) {
	block, ok := machineBlock(markdown, name)
	if !ok {
		return Machine{}, fmt.Errorf("no state machine named %s", name)
	}
	var machine Machine
	for _, cells := range tableRows(block, func(header string) bool { return header == stateHeader }) {
		machine.States = append(machine.States, cells[0])
		if len(cells) > 1 && cells[1] == "initial" {
			machine.Initial = cells[0]
		}
	}
	for _, cells := range tableRows(block, func(header string) bool { return header == transitionHeader }) {
		if len(cells) < 4 {
			return Machine{}, fmt.Errorf("%s: transition row %v has fewer than four cells", name, cells)
		}
		machine.Transitions = append(machine.Transitions, TransitionRow{From: cells[0], Event: cells[1], To: cells[3]})
	}
	isMatrix := func(header string) bool {
		return strings.HasPrefix(header, "| State |") && header != stateHeader
	}
	header, ok := findHeader(block, isMatrix)
	if !ok {
		return Machine{}, fmt.Errorf("%s has no state matrix", name)
	}
	machine.Operations = splitRow(header)[1:]
	machine.cells = map[[2]string][]Outcome{}
	for _, cells := range tableRows(block, isMatrix) {
		for column, operation := range machine.Operations {
			cell := ""
			if column+1 < len(cells) {
				cell = cells[column+1]
			}
			outcomes, err := parseCell(cell)
			if err != nil {
				return Machine{}, fmt.Errorf("%s: %s × %s: %w", name, cells[0], operation, err)
			}
			machine.cells[[2]string{cells[0], operation}] = outcomes
		}
	}
	return machine, nil
}

func parseCell(cell string) ([]Outcome, error) {
	if cell == "" {
		return nil, fmt.Errorf("the cell gives no outcome")
	}
	var outcomes []Outcome
	for part := range strings.SplitSeq(cell, "<br>") {
		match := outcomePattern.FindStringSubmatch(strings.TrimSpace(part))
		if match == nil {
			return nil, fmt.Errorf("outcome %q is not → <State>, 何もしない, or 拒否：<response>", part)
		}
		outcome := Outcome{Condition: match[4]}
		switch {
		case match[1] != "":
			outcome.Kind, outcome.To = Transition, match[1]
		case match[2] != "":
			outcome.Kind = NoOp
		default:
			outcome.Kind, outcome.Refusal = Refusal, match[3]
		}
		outcomes = append(outcomes, outcome)
	}
	return outcomes, nil
}

// machineBlock は `### <name>` から次の H2 または H3 までを返す。
func machineBlock(markdown, name string) (string, bool) {
	lines := strings.Split(markdown, "\n")
	for start, line := range lines {
		if strings.TrimSpace(line) != "### "+name {
			continue
		}
		end := len(lines)
		for index := start + 1; index < len(lines); index++ {
			if strings.HasPrefix(lines[index], "## ") || strings.HasPrefix(lines[index], "### ") {
				end = index
				break
			}
		}
		return strings.Join(lines[start+1:end], "\n"), true
	}
	return "", false
}

func findHeader(block string, isHeader func(string) bool) (string, bool) {
	for line := range strings.SplitSeq(block, "\n") {
		if row := strings.TrimSpace(line); isHeader(row) {
			return row, true
		}
	}
	return "", false
}

// tableRows は、isHeader に当たる見出し行から始まる表の本文の行を、セルに分けて返す。
func tableRows(block string, isHeader func(string) bool) [][]string {
	var rows [][]string
	inTable := false
	for line := range strings.SplitSeq(block, "\n") {
		row := strings.TrimSpace(line)
		switch {
		case isHeader(row):
			inTable = true
		case !inTable:
		case !strings.HasPrefix(row, "|"):
			inTable = false
		case strings.Trim(row, "|-: ") == "":
		default:
			rows = append(rows, splitRow(row))
		}
	}
	return rows
}

// splitRow は表の一行をセルに分ける。ガードの `\|` はセルの区切りではない。
func splitRow(row string) []string {
	parts := strings.Split(strings.ReplaceAll(row, `\|`, "\x00"), "|")
	if len(parts) < 2 {
		return nil
	}
	cells := make([]string, 0, len(parts)-2)
	for _, part := range parts[1 : len(parts)-1] {
		cells = append(cells, strings.ReplaceAll(strings.TrimSpace(part), "\x00", `\|`))
	}
	return cells
}
