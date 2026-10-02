package domain

// CSV の解析器が見出し、行、上限、セルの復号について約束する細部を固定する。
// どれも種別に依存しない共有の規則なので、User と Group の方言を通さずに読む。

import (
	"errors"
	"io"
	"slices"
	"strings"
	"testing"
)

func acceptKeys(keys ...string) func(string) bool {
	return func(key string) bool { return slices.Contains(keys, key) }
}

func csvErrorCode(t *testing.T, err error) CSVErrorCode {
	t.Helper()
	var csvErr *CSVError
	if !errors.As(err, &csvErr) {
		t.Fatalf("err=%v, want *CSVError", err)
	}
	return csvErr.Code
}

//spec:covers EX-IDMANAGEMENT-034-01: UTF-8 の BOM で始まる見出しを、BOM を除いた機械キーとして受け付けること。
func TestCSVHeaderAcceptsAKeyAfterTheByteOrderMark(t *testing.T) {
	reader, err := NewCSVReader(strings.NewReader("\ufeffpreferred_username\nalice\n"), acceptKeys("preferred_username"), DefaultCSVTransferPolicy())
	if err != nil {
		t.Fatalf("BOM 付きの見出しが拒否された: %v", err)
	}
	if header := reader.Header(); len(header) != 1 || header[0] != "preferred_username" {
		t.Fatalf("header=%q, want [preferred_username]", header)
	}
}

//spec:covers EX-IDMANAGEMENT-034-02, EX-IDMANAGEMENT-034-03: 大文字を含む列名、前後に空白を含む列名、秘密情報の列名を invalid_header で拒否し、空のファイルを invalid_csv で拒否すること。
func TestCSVHeaderRejectsAnyNameThatIsNotAnExactMachineKey(t *testing.T) {
	// 秘密情報の列は、方言が受け付けると答えても拒否する。
	accepts := acceptKeys("preferred_username", "mfa_secret", "password", "password_hash", "token", "recovery_code")
	for _, header := range []string{
		"Preferred_Username", " preferred_username", "mfa_secret", "password", "password_hash", "token", "recovery_code",
	} {
		_, err := NewCSVReader(strings.NewReader(header+"\nvalue\n"), accepts, DefaultCSVTransferPolicy())
		if code := csvErrorCode(t, err); code != CSVErrorInvalidHeader {
			t.Fatalf("header %q: code=%q, want invalid_header", header, code)
		}
	}
	_, err := NewCSVReader(strings.NewReader(""), accepts, DefaultCSVTransferPolicy())
	if code := csvErrorCode(t, err); code != CSVErrorInvalidCSV {
		t.Fatalf("空のファイル: code=%q, want invalid_csv", code)
	}
}

//spec:covers EX-IDMANAGEMENT-035-01: 列の数が見出しと異なる行だけを invalid_column_count で拒否し、前後の行を読み続けること。
func TestCSVRowWithTheWrongColumnCountIsRejectedAloneAndReadingContinues(t *testing.T) {
	reader, err := NewCSVReader(strings.NewReader("a,b\n1,2\nonly\n3,4\n"), acceptKeys("a", "b"), DefaultCSVTransferPolicy())
	if err != nil {
		t.Fatal(err)
	}
	var rows []int
	var rejected []CSVError
	for {
		record, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("ファイル全体の解析が止まった: %v", err)
		}
		if record.Error != nil {
			rejected = append(rejected, *record.Error)
			continue
		}
		rows = append(rows, record.Row.Number)
	}
	if len(rejected) != 1 || rejected[0].Row != 3 || rejected[0].Code != CSVErrorInvalidColumnCount {
		t.Fatalf("rejected=%+v, want row 3 invalid_column_count", rejected)
	}
	if len(rows) != 2 || rows[0] != 2 || rows[1] != 4 {
		t.Fatalf("rows=%v, want [2 4]", rows)
	}
}

//spec:covers EX-IDMANAGEMENT-036-01, EX-IDMANAGEMENT-036-02: 保護として付けたアポストロフィーだけを一文字取り除き、値の一部であるアポストロフィーを残すこと。
func TestCSVDecodeRemovesOnlyTheProtectiveApostrophe(t *testing.T) {
	cases := map[string]string{
		"'=SUM(A1)": "=SUM(A1)",
		"''quoted":  "'quoted",
		"'\tvalue":  "\tvalue",
		"'abc":      "'abc",
		"'":         "'",
		"'=":        "=",
		"abc":       "abc",
	}
	for raw, want := range cases {
		if got := DecodeCSVCell(raw); got != want {
			t.Fatalf("DecodeCSVCell(%q)=%q, want %q", raw, got, want)
		}
	}
}

//spec:covers EX-IDMANAGEMENT-037-01, EX-IDMANAGEMENT-037-02: データ行の上限を見出しを除いて数え、上限ちょうどを受け付け、超えた行で too_many_rows として止まること。
func TestCSVRowLimitCountsDataRowsAndAcceptsTheLimitItself(t *testing.T) {
	policy := CSVTransferPolicy{MaxRows: 2, MaxBytes: 1 << 20, MaxFieldBytes: 1 << 10}
	read := func(input string) (int, error) {
		reader, err := NewCSVReader(strings.NewReader(input), acceptKeys("a"), policy)
		if err != nil {
			return 0, err
		}
		count := 0
		for {
			_, err := reader.Next()
			if errors.Is(err, io.EOF) {
				return count, nil
			}
			if err != nil {
				return count, err
			}
			count++
		}
	}
	if count, err := read("a\n1\n2\n"); err != nil || count != 2 {
		t.Fatalf("上限ちょうど: count=%d err=%v, want 2 rows", count, err)
	}
	count, err := read("a\n1\n2\n3\n")
	if code := csvErrorCode(t, err); code != CSVErrorTooManyRows || count != 2 {
		t.Fatalf("上限超過: count=%d code=%q, want 2 rows then too_many_rows", count, code)
	}
}

// デフォルト値は規則の表の値そのものである。値を変えると、同じファイルの受理と拒否が
// 入れ替わる。
//
//spec:covers REQ-IDMANAGEMENT-037: 転送ポリシーのデフォルト値が 100,000 行、64 MiB、64 KiB であること。
func TestDefaultCSVTransferPolicyValues(t *testing.T) {
	policy := DefaultCSVTransferPolicy()
	if policy.MaxRows != 100_000 || policy.MaxBytes != 64<<20 || policy.MaxFieldBytes != 64<<10 {
		t.Fatalf("policy=%+v, want 100000 rows / 64 MiB / 64 KiB", policy)
	}
}

//spec:covers REQ-IDMANAGEMENT-037: 項目の大きさの上限ちょうどの項目を受け付け、一バイト超えた項目を field_too_large で拒否すること。
func TestCSVFieldLimitAcceptsTheLimitItself(t *testing.T) {
	policy := CSVTransferPolicy{MaxRows: 10, MaxBytes: 1 << 20, MaxFieldBytes: 4}
	reader, err := NewCSVReader(strings.NewReader("a\n1234\n12345\n"), acceptKeys("a"), policy)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reader.Next(); err != nil {
		t.Fatalf("上限ちょうどの項目が拒否された: %v", err)
	}
	_, err = reader.Next()
	if code := csvErrorCode(t, err); code != CSVErrorFieldTooLarge {
		t.Fatalf("code=%q, want field_too_large", code)
	}
}

//spec:covers REQ-IDMANAGEMENT-037: 成果物の大きさの上限ちょうどのファイルと見出しの項目を受け付け、一バイト超えたファイルを csv_too_large で拒否すること。
func TestCSVByteLimitAcceptsTheLimitItself(t *testing.T) {
	input := "a\n12\n" // 5 バイト
	exact := CSVTransferPolicy{MaxRows: 10, MaxBytes: len(input), MaxFieldBytes: 1}
	reader, err := NewCSVReader(strings.NewReader(input), acceptKeys("a"), exact)
	if err != nil {
		t.Fatalf("上限ちょうどの見出しが拒否された: %v", err)
	}
	if _, err := reader.Next(); csvErrorCode(t, err) != CSVErrorFieldTooLarge {
		t.Fatalf("前提が壊れている: 2 バイトの項目は field_too_large のはず: %v", err)
	}
	exact.MaxFieldBytes = 2
	reader, err = NewCSVReader(strings.NewReader(input), acceptKeys("a"), exact)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reader.Next(); err != nil {
		t.Fatalf("上限ちょうどのファイルが拒否された: %v", err)
	}
	over := CSVTransferPolicy{MaxRows: 10, MaxBytes: len(input) - 1, MaxFieldBytes: 2}
	// 解析器は先読みするので、超過は見出しの時点で見つかることもある。どちらの位置でも
	// ファイルは拒否される。
	reader, err = NewCSVReader(strings.NewReader(input), acceptKeys("a"), over)
	if err == nil {
		_, err = reader.Next()
	}
	if csvErrorCode(t, err) != CSVErrorCSVTooLarge {
		t.Fatalf("err=%v, want csv_too_large", err)
	}
}

//spec:covers REQ-IDMANAGEMENT-037: 書き出しが行数、項目、成果物の大きさの上限ちょうどを受け付け、超えた行を拒否すること。
func TestCSVWriterAcceptsEachLimitItself(t *testing.T) {
	var out strings.Builder
	// 見出し "a\n" と 2 行 "12\n" で 8 バイト。
	policy := CSVTransferPolicy{MaxRows: 2, MaxBytes: 8, MaxFieldBytes: 2}
	writer, err := NewCSVWriter(&out, []string{"a"}, policy)
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := writer.WriteRow([]string{"12"}); err != nil {
			t.Fatalf("上限ちょうどの行が拒否された: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("上限ちょうどの成果物が拒否された: %v", err)
	}
	if out.String() != "a\n12\n12\n" {
		t.Fatalf("out=%q", out.String())
	}
	if err := writer.WriteRow([]string{"1"}); csvErrorCode(t, err) != CSVErrorTooManyRows {
		t.Fatalf("err=%v, want too_many_rows", err)
	}
	wide, err := NewCSVWriter(&strings.Builder{}, []string{"a"}, policy)
	if err != nil {
		t.Fatal(err)
	}
	if err := wide.WriteRow([]string{"123"}); csvErrorCode(t, err) != CSVErrorFieldTooLarge {
		t.Fatalf("err=%v, want field_too_large", err)
	}
	small, err := NewCSVWriter(&strings.Builder{}, []string{"a"}, CSVTransferPolicy{MaxRows: 10, MaxBytes: 4, MaxFieldBytes: 2})
	if err != nil {
		t.Fatal(err)
	}
	if err := small.WriteRow([]string{"12"}); err != nil {
		t.Fatal(err)
	}
	if err := small.Close(); csvErrorCode(t, err) != CSVErrorCSVTooLarge {
		t.Fatalf("err=%v, want csv_too_large", err)
	}
}

// csv.Writer は 4 KiB ごとに書き出すので、上限の勘定は複数回の書き出しにまたがる。
// 一度で書き終わる小さな成果物では、勘定の誤りが表に出ない。
//
//spec:covers REQ-IDMANAGEMENT-037: 複数回に分けて書き出す成果物でも、大きさの上限を超えた時点で csv_too_large にすること。
func TestCSVWriterCountsBytesAcrossFlushes(t *testing.T) {
	writer, err := NewCSVWriter(&strings.Builder{}, []string{"a"}, CSVTransferPolicy{MaxRows: 1000, MaxBytes: 5000, MaxFieldBytes: 200})
	if err != nil {
		t.Fatal(err)
	}
	row := []string{strings.Repeat("x", 99)}
	var writeErr error
	for i := 0; i < 60 && writeErr == nil; i++ {
		writeErr = writer.WriteRow(row)
	}
	if writeErr == nil {
		writeErr = writer.Close()
	}
	if csvErrorCode(t, writeErr) != CSVErrorCSVTooLarge {
		t.Fatalf("err=%v, want csv_too_large", writeErr)
	}
}
