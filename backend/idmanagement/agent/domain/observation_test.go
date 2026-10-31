package domain

import (
	"slices"
	"strings"
	"testing"
	"time"
)

func validObservation(now time.Time) AgentObservation {
	return AgentObservation{Issuer: "https://discovery.example", Subject: "svc", ObservedAt: now}
}

//spec:covers REQ-IDMANAGEMENT-096: 長さ、件数、未来の時刻の境界の両側
func TestAgentObservationValidateBoundaries(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	at := func(length int) *string {
		value := strings.Repeat("あ", length)
		return &value
	}
	cases := []struct {
		name   string
		change func(*AgentObservation)
		valid  bool
	}{
		{"最小", func(*AgentObservation) {}, true},
		{"subject が空", func(o *AgentObservation) { o.Subject = "" }, false},
		{"issuer が空", func(o *AgentObservation) { o.Issuer = "" }, false},
		{"issuer が 512 文字", func(o *AgentObservation) { o.Issuer = *at(512) }, true},
		{"issuer が 513 文字", func(o *AgentObservation) { o.Issuer = *at(513) }, false},
		{"subject が 512 文字", func(o *AgentObservation) { o.Subject = *at(512) }, true},
		{"subject が 513 文字", func(o *AgentObservation) { o.Subject = *at(513) }, false},
		{"表示名が 256 文字", func(o *AgentObservation) { o.DisplayName = at(256) }, true},
		{"表示名が 257 文字", func(o *AgentObservation) { o.DisplayName = at(257) }, false},
		{"登録 ID が 257 文字", func(o *AgentObservation) { o.ExternalRegistrationID = at(257) }, false},
		{"責任者の候補が 257 文字", func(o *AgentObservation) { o.ResponsibleCandidate = at(257) }, false},
		{"資源が 20 件", func(o *AgentObservation) { o.Resources = slices.Repeat([]string{"r"}, 20) }, true},
		{"資源が 21 件", func(o *AgentObservation) { o.Resources = slices.Repeat([]string{"r"}, 21) }, false},
		{"資源の名前が 257 文字", func(o *AgentObservation) { o.Resources = []string{*at(257)} }, false},
		{"資源の名前が空", func(o *AgentObservation) { o.Resources = []string{""} }, false},
		{"時刻がない", func(o *AgentObservation) { o.ObservedAt = time.Time{} }, false},
		{"5 分先", func(o *AgentObservation) { o.ObservedAt = now.Add(5 * time.Minute) }, true},
		{"5 分 1 秒先", func(o *AgentObservation) { o.ObservedAt = now.Add(5*time.Minute + time.Second) }, false},
		{"過去", func(o *AgentObservation) { o.ObservedAt = now.Add(-1000 * time.Hour) }, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			observation := validObservation(now)
			tc.change(&observation)
			if err := observation.Validate(now); (err == nil) != tc.valid {
				t.Fatalf("err=%v, valid=%v", err, tc.valid)
			}
		})
	}
}

func TestAgentObservationNormalize(t *testing.T) {
	blank := "  "
	name := " name "
	observation := AgentObservation{Issuer: " i ", Subject: " s ", DisplayName: &name, ExternalRegistrationID: &blank}
	observation.Normalize()
	if observation.Issuer != "i" || observation.Subject != "s" || *observation.DisplayName != "name" {
		t.Fatalf("got=%+v", observation)
	}
	if observation.ExternalRegistrationID != nil || observation.Resources == nil {
		t.Fatalf("空の任意項目=%+v", observation)
	}
}

//spec:covers REQ-IDMANAGEMENT-098: 候補の数によらず確定せず、複数の候補をすべて返す
func TestMatchObservationNeverConfirmsByItself(t *testing.T) {
	agent := "confirmed"
	evidence := []MatchEvidence{
		{AgentID: "b", Reason: MatchReasonClientBinding},
		{AgentID: "a", Reason: MatchReasonClientBinding},
		{AgentID: "a", Reason: MatchReasonWorkloadBinding},
		{AgentID: "a", Reason: MatchReasonWorkloadBinding},
	}
	if got := MatchObservation(AgentObservation{}, nil); got.State != ObservationStateUnregistered || len(got.Candidates) != 0 {
		t.Errorf("候補なし=%+v", got)
	}
	one := MatchObservation(AgentObservation{}, evidence[:1])
	if one.State != ObservationStateCandidate {
		t.Errorf("一つの候補=%+v", one)
	}
	many := MatchObservation(AgentObservation{}, evidence)
	if many.State != ObservationStateCandidate || len(many.Candidates) != 2 || many.Candidates[0].AgentID != "a" {
		t.Fatalf("複数の候補=%+v", many)
	}
	if !slices.Equal(many.Candidates[0].Reasons, []MatchReason{MatchReasonWorkloadBinding, MatchReasonClientBinding}) {
		t.Errorf("根拠=%v", many.Candidates[0].Reasons)
	}
	if got := MatchObservation(AgentObservation{Excluded: true}, evidence); got.State != ObservationStateExcluded {
		t.Errorf("除外=%+v", got)
	}
	if got := MatchObservation(AgentObservation{ConfirmedAgentID: &agent}, nil); got.State != ObservationStateConfirmed {
		t.Errorf("確認=%+v", got)
	}
}

//spec:covers REQ-IDMANAGEMENT-102: 30 日の境界の両側
func TestAgentObservationStaleBoundary(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	exact := AgentObservation{ObservedAt: now.Add(-ObservationStaleAfter)}
	if exact.Stale(now) {
		t.Error("ちょうど 30 日は古くない")
	}
	over := AgentObservation{ObservedAt: now.Add(-ObservationStaleAfter - time.Second)}
	if !over.Stale(now) {
		t.Error("30 日を過ぎたら古い")
	}
}
