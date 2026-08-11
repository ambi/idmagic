package domain

import (
	"bytes"
	"encoding/csv"
	"errors"
	"strings"
	"testing"
)

// TestDataExportTargetKind_Valid: scenario
// "管理者はリソースをフィルタして CSV に安全にエクスポートできる" の対象種別。
func TestDataExportTargetKind_Valid(t *testing.T) {
	for _, k := range []DataExportTargetKind{ExportTargetUser, ExportTargetGroup, ExportTargetGroupMembership} {
		if !k.Valid() {
			t.Errorf("DataExportTargetKind(%q).Valid() = false, want true", k)
		}
	}
	if DataExportTargetKind("password_dump").Valid() {
		t.Error("unknown target kind must be invalid")
	}
}

// TestValidateExportColumns: model DataExportRequest の列は対象種別の
// allowlist の部分集合に限る (scenario extension "allowlist 外の key" → invalid_columns)。
func TestValidateExportColumns(t *testing.T) {
	if err := ValidateExportColumns(ExportTargetUser, []string{"preferred_username", "email"}); err != nil {
		t.Fatalf("valid subset rejected: %v", err)
	}
	if err := ValidateExportColumns(ExportTargetUser, nil); !errors.Is(err, ErrInvalidExportColumns) {
		t.Errorf("empty columns: got %v, want ErrInvalidExportColumns", err)
	}
	// sensitive/allowlist 外の列は常に拒否される (password_hash は allowlist に存在しない)。
	if err := ValidateExportColumns(ExportTargetUser, []string{"preferred_username", "password_hash"}); !errors.Is(err, ErrInvalidExportColumns) {
		t.Errorf("allowlist 外の列: got %v, want ErrInvalidExportColumns", err)
	}
	if err := ValidateExportColumns(ExportTargetUser, []string{"email", "email"}); !errors.Is(err, ErrInvalidExportColumns) {
		t.Errorf("duplicate columns: got %v, want ErrInvalidExportColumns", err)
	}
	if err := ValidateExportColumns(DataExportTargetKind("bogus"), []string{"id"}); !errors.Is(err, ErrInvalidExportTarget) {
		t.Errorf("unknown target: got %v, want ErrInvalidExportTarget", err)
	}
}

// TestColumnsForTarget_NoSensitiveColumns: allowlist に sensitive 値の列を
// 一切含めない。
func TestColumnsForTarget_NoSensitiveColumns(t *testing.T) {
	banned := []string{"password", "password_hash", "secret", "token", "recovery", "mfa_secret", "totp"}
	for _, kind := range []DataExportTargetKind{ExportTargetUser, ExportTargetGroup, ExportTargetGroupMembership} {
		cols, ok := ColumnsForTarget(kind)
		if !ok || len(cols) == 0 {
			t.Fatalf("ColumnsForTarget(%q) empty", kind)
		}
		for _, c := range cols {
			for _, b := range banned {
				if strings.Contains(c.Key, b) {
					t.Errorf("target %q exposes sensitive column %q", kind, c.Key)
				}
			}
		}
	}
}

// TestEscapeCSVField: formula injection 対策。値先頭が = + - @ TAB CR LF の
// いずれかなら安全に前置エスケープする (scenario extension)。
func TestEscapeCSVField(t *testing.T) {
	dangerous := []string{"=1+1", "+1", "-1", "@SUM(A1)", "\t=cmd", "\r=x", "\n=x"}
	for _, in := range dangerous {
		got := EscapeCSVField(in)
		if !strings.HasPrefix(got, "'") {
			t.Errorf("EscapeCSVField(%q) = %q, want leading quote guard", in, got)
		}
	}
	for _, in := range []string{"", "alice", "user@ok.example"[1:], "safe-value", "山田"} {
		if got := EscapeCSVField(in); got != in {
			t.Errorf("EscapeCSVField(%q) = %q, want unchanged", in, got)
		}
	}
}

// TestEncodeCSVRecords_RFC4180AndInjectionSafe: RFC 4180 quoting を使い、
// 各セルは formula injection safe にエスケープされ、パースし直すと危険な
// 先頭文字を持つセルが残らない。
func TestEncodeCSVRecords_RFC4180AndInjectionSafe(t *testing.T) {
	header := []string{"preferred_username", "name"}
	rows := [][]string{
		{"alice", "Alice, A \"quoted\" name\nwith newline"},
		{"=danger", "@evil"},
	}
	out, err := EncodeCSVRecords(header, rows)
	if err != nil {
		t.Fatalf("EncodeCSVRecords: %v", err)
	}
	r := csv.NewReader(bytes.NewReader(out))
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("output is not valid RFC 4180 CSV: %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("got %d records, want 3 (header + 2 rows)", len(records))
	}
	// ヘッダーは維持される。
	if records[0][0] != "preferred_username" {
		t.Errorf("header mangled: %v", records[0])
	}
	// パース後、いかなるセルも危険な先頭文字で始まらない。
	for _, rec := range records[1:] {
		for _, cell := range rec {
			if cell == "" {
				continue
			}
			switch cell[0] {
			case '=', '+', '-', '@', '\t', '\r':
				t.Errorf("cell %q begins with a formula-injection trigger after round-trip", cell)
			}
		}
	}
}
