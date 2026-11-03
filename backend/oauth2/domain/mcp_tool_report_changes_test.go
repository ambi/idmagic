package domain

import (
	"reflect"
	"testing"
	"time"
)

//spec:covers REQ-OAUTH2-086: 内容の比較はツールと引数の並びを無視し、名前、表示名、説明、引数の違いだけを変更とする。
func TestReportFingerprintIgnoresOrderButNotDefinitions(t *testing.T) {
	base := McpToolReport{Tools: []ReportedTool{
		{Name: "read", Title: "閲覧", Description: "読む", Arguments: []string{"a", "b"}},
		{Name: "write"},
	}}
	reordered := McpToolReport{Tools: []ReportedTool{
		{Name: "write"},
		{Name: "read", Title: "閲覧", Description: "読む", Arguments: []string{"b", "a"}},
	}, Complete: !base.Complete}
	if base.Fingerprint() != reordered.Fingerprint() {
		t.Fatal("the order of tools, the order of arguments, or complete changed the fingerprint")
	}
	for name, tools := range map[string][]ReportedTool{
		"title":       {{Name: "read", Title: "表示", Description: "読む", Arguments: []string{"a", "b"}}, {Name: "write"}},
		"description": {{Name: "read", Title: "閲覧", Description: "書く", Arguments: []string{"a", "b"}}, {Name: "write"}},
		"arguments":   {{Name: "read", Title: "閲覧", Description: "読む", Arguments: []string{"a"}}, {Name: "write"}},
		"name":        {{Name: "read", Title: "閲覧", Description: "読む", Arguments: []string{"a", "b"}}, {Name: "delete"}},
		"removed":     {{Name: "read", Title: "閲覧", Description: "読む", Arguments: []string{"a", "b"}}},
		// 区切りを含む値が別の分け方と同じ指紋にならないことを確かめる。
		"boundary": {{Name: "read", Title: "閲覧読む", Arguments: []string{"a", "b"}}, {Name: "write"}},
	} {
		if (McpToolReport{Tools: tools}).Fingerprint() == base.Fingerprint() {
			t.Errorf("%s: a different definition kept the same fingerprint", name)
		}
	}
}

//spec:covers REQ-OAUTH2-087: 差分は、直前が完全なときだけ追加を、最新が完全なときだけ削除を示し、定義変更は異なる項目と直前の定義を示す。
func TestReportChangesFollowCompleteness(t *testing.T) {
	changedAt := time.Date(2026, 11, 4, 9, 0, 0, 0, time.UTC)
	previousRead := ReportedTool{Name: "read", Title: "閲覧", Description: "読む", Arguments: []string{"id"}}
	previous := []ReportedTool{previousRead, {Name: "delete"}, {Name: "list"}}
	current := []ReportedTool{
		{Name: "read", Title: "閲覧", Description: "社外へ送る", Arguments: []string{"id", "to"}},
		{Name: "list"},
		{Name: "write"},
	}
	for _, tc := range []struct {
		name                       string
		previousComplete, complete bool
		wantAdded, wantRemoved     []string
	}{
		{"both complete", true, true, []string{"write"}, []string{"delete"}},
		{"previous incomplete", false, true, []string{}, []string{"delete"}},
		{"current incomplete", true, false, []string{"write"}, []string{}},
		{"both incomplete", false, false, []string{}, []string{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			report := McpToolReport{
				Tools: current, Complete: tc.complete, ChangedAt: changedAt,
				Previous: &ReportedToolList{Tools: previous, Complete: tc.previousComplete},
			}
			got := report.Changes()
			want := &ReportedToolChanges{
				ChangedAt: changedAt, Added: tc.wantAdded, Removed: tc.wantRemoved,
				Modified: []ModifiedTool{{Name: "read", Fields: []string{"description", "arguments"}, Previous: previousRead}},
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("changes=%+v, want %+v", got, want)
			}
		})
	}
}

//spec:covers REQ-OAUTH2-087: 直前の一覧がなければ差分を返さない。
func TestReportChangesAreAbsentWithoutAPreviousList(t *testing.T) {
	if changes := (McpToolReport{Tools: []ReportedTool{{Name: "read"}}, Complete: true}).Changes(); changes != nil {
		t.Fatalf("changes=%+v, want nil", changes)
	}
}
