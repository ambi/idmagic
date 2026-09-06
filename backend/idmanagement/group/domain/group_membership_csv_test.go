package domain

import (
	"errors"
	"io"
	"strings"
	"testing"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
)

func collectGroupMembershipCSV(t *testing.T, document string) ([]idmdomain.CSVRow, error) {
	t.Helper()
	schema := NewGroupMembershipCSVSchema()
	reader, err := idmdomain.NewCSVReader(strings.NewReader(document), schema.Accepts, idmdomain.DefaultCSVTransferPolicy())
	if err != nil {
		return nil, err
	}
	var rows []idmdomain.CSVRow
	for {
		record, err := reader.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return rows, err
		}
		if record.Row != nil {
			rows = append(rows, *record.Row)
		}
	}
	return rows, nil
}

// scenario EX-IDMANAGEMENT-029-03: 機械キーの語彙は閉じており、順序と部分集合は自由。
// 各列の役割まで主張するのは、`source` や `created_at` を書き込み可能にした実装と、
// `membership_state` 以外にも意図を持たせた実装を区別するためである。
func TestGroupMembershipCSVSchemaIsAClosedMachineKeyVocabulary(t *testing.T) {
	schema := NewGroupMembershipCSVSchema()
	want := []struct {
		key  string
		mode GroupMembershipCSVColumnMode
	}{
		{"group_id", GroupMembershipCSVVerification},
		{"group_name", GroupMembershipCSVVerification},
		{"user_id", GroupMembershipCSVIdentity},
		{"preferred_username", GroupMembershipCSVIdentity},
		{"membership_state", GroupMembershipCSVIntent},
		{"source", GroupMembershipCSVReadOnly},
		{"created_at", GroupMembershipCSVReadOnly},
	}
	got := schema.Columns()
	if len(got) != len(want) {
		t.Fatalf("the schema declares %d columns, want %d: %+v", len(got), len(want), got)
	}
	intents := 0
	for i, column := range got {
		if column.Key != want[i].key || column.Mode != want[i].mode {
			t.Fatalf("column %d is %q/%q, want %q/%q", i, column.Key, column.Mode, want[i].key, want[i].mode)
		}
		if column.Mode == GroupMembershipCSVIntent {
			intents++
		}
	}
	if intents != 1 {
		t.Fatalf("the dialect declares %d mutation intents, want exactly 1", intents)
	}
	for _, unknown := range []string{"roles", "membership_type", "id", "custom:cost_center", "password"} {
		if schema.Accepts(unknown) {
			t.Fatalf("the schema accepts %q, which belongs to another dialect", unknown)
		}
	}

	rows, err := collectGroupMembershipCSV(t, "membership_state,user_id\npresent,user-1\n")
	if err != nil {
		t.Fatalf("a reordered subset of the machine keys was refused: %v", err)
	}
	if len(rows) != 1 || rows[0].TrimmedCell("user_id") != "user-1" {
		t.Fatalf("the reordered subset did not parse: %+v", rows)
	}
	if _, present := rows[0].Cell("group_name"); present {
		t.Fatal("a column the file omitted is reported as present")
	}
}

// scenario EX-IDMANAGEMENT-029-04: `membership_state` を欠いたファイルは受理しない。
// 他の列と違い、この列が無いファイルは意図をひとつも表せない。
func TestGroupMembershipCSVSchemaRequiresTheIntentColumn(t *testing.T) {
	schema := NewGroupMembershipCSVSchema()
	if missing := schema.MissingRequiredColumn([]string{"user_id", "preferred_username", "source"}); missing != "membership_state" {
		t.Fatalf("a header without the intent column reported %q as missing, want %q", missing, "membership_state")
	}
	if missing := schema.MissingRequiredColumn([]string{"user_id", "membership_state"}); missing != "" {
		t.Fatalf("a header carrying the intent column reported %q as missing", missing)
	}
	// 識別子の列は必須ではない。行ごとに `user_id` か `preferred_username` の
	// どちらかがあればよく、それは行の判定であってヘッダーの判定ではない。
	if missing := schema.MissingRequiredColumn([]string{"membership_state"}); missing != "" {
		t.Fatalf("a header carrying only the intent column reported %q as missing", missing)
	}
}

// scenario EX-IDMANAGEMENT-029-05: `present|absent` は閉じた集合であり、空セルも
// 未知の値も既知の値へ丸めず拒否する。丸めると、ファイルが表明していない側の
// 意図 (追加か解除か) を勝手に選ぶことになる。
func TestGroupMembershipCSVStateVocabularyIsClosed(t *testing.T) {
	for _, accepted := range []struct {
		raw  string
		want GroupMembershipCSVState
	}{
		{"present", GroupMembershipStatePresent},
		{"absent", GroupMembershipStateAbsent},
	} {
		got, err := ParseGroupMembershipCSVState(accepted.raw)
		if err != nil || got != accepted.want {
			t.Fatalf("ParseGroupMembershipCSVState(%q) = %q, %v; want %q", accepted.raw, got, err, accepted.want)
		}
	}
	for _, refused := range []string{"", " ", "Present", "PRESENT", "present ", "absent\t", "removed", "true", "1", "\x00"} {
		if got, err := ParseGroupMembershipCSVState(refused); err == nil {
			t.Fatalf("ParseGroupMembershipCSVState(%q) = %q accepted a value outside the closed set", refused, got)
		}
	}
}

// scenario EX-IDMANAGEMENT-029-06: 行の識別子は `user_id` を優先し、無ければ
// `preferred_username` にたどる。どちらも無い行は識別できない。
func TestGroupMembershipCSVIdentifierPrefersUserIDAndFallsBackToUsername(t *testing.T) {
	cases := []struct {
		name     string
		cells    map[string]idmdomain.CSVCell
		wantID   string
		wantName string
		wantCode idmdomain.CSVErrorCode
	}{
		{
			name:   "user_id を優先する",
			cells:  map[string]idmdomain.CSVCell{"user_id": {Present: true, Raw: " user-1 "}, "preferred_username": {Present: true, Raw: "alice"}},
			wantID: "user-1", wantName: "alice",
		},
		{
			name:     "user_id が空なら username にたどる",
			cells:    map[string]idmdomain.CSVCell{"user_id": {Present: true, Raw: ""}, "preferred_username": {Present: true, Raw: "alice"}},
			wantName: "alice",
		},
		{
			name:     "どちらも無ければ識別できない",
			cells:    map[string]idmdomain.CSVCell{"membership_state": {Present: true, Raw: "present"}},
			wantCode: "missing_identifier",
		},
		{
			name:     "どちらも空でも識別できない",
			cells:    map[string]idmdomain.CSVCell{"user_id": {Present: true, Raw: "  "}, "preferred_username": {Present: true, Raw: ""}},
			wantCode: "missing_identifier",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			identifier, code := GroupMembershipCSVIdentifierOf(idmdomain.NewCSVRow(2, tc.cells))
			if code != tc.wantCode {
				t.Fatalf("code = %q, want %q", code, tc.wantCode)
			}
			if code != "" {
				return
			}
			if identifier.UserID != tc.wantID || identifier.PreferredUsername != tc.wantName {
				t.Fatalf("identifier = %+v, want {%q %q}", identifier, tc.wantID, tc.wantName)
			}
		})
	}
}

// scenario EX-IDMANAGEMENT-029-01: 拒否行は位置と安定コードだけを運び、セル値を
// 決して載せない。
func TestRejectedGroupMembershipImportRowCarriesNoValue(t *testing.T) {
	plan := RejectedGroupMembershipImportRow(7, "membership_state", "invalid_membership_state")
	if plan.Action != GroupMembershipImportRejected || plan.Row != 7 {
		t.Fatalf("plan = %+v, want a rejected row 7", plan)
	}
	if plan.Error == nil || plan.Error.Column != "membership_state" || plan.Error.Code != "invalid_membership_state" {
		t.Fatalf("error = %+v, want the column and the stable code", plan.Error)
	}
	if plan.UserID != "" {
		t.Fatalf("a rejected row resolved a user: %q", plan.UserID)
	}
}
