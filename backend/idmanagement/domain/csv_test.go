package domain

import (
	"errors"
	"io"
	"strings"
	"testing"
	"testing/iotest"
)

// 数式安全変換は情報を失うエスケープではなく、可逆な接頭辞でなければならない。
// 既にアポストロフィーで始まる値を二重に守らないことと、RFC 4180 の引用を経ても
// カンマ・引用符・改行が戻ることを、同じ 1 本で読む。
//
//spec:covers EX-IDMANAGEMENT-007-02, EX-IDMANAGEMENT-027-02: 危険な先頭文字、既存のアポストロフィー、カンマ、引用符、改行を含む値で decode(encode(value)) が元の値と一致すること。
func TestCSVFormulaSafeCodecIsReversible(t *testing.T) {
	values := []string{
		"plain", "=SUM(A1:A2)", "+1", "-1", "@name", "\tvalue", "\rvalue", "\nvalue",
		"'already", "''twice", "comma,value", `quote"value`, "multi\nline", "日本語",
	}
	for _, value := range values {
		encoded := EncodeCSVCell(value)
		if isCSVFormulaTrigger(encoded) {
			t.Errorf("EncodeCSVCell(%q) = %q remains dangerous", value, encoded)
		}
		if decoded := DecodeCSVCell(encoded); decoded != value {
			t.Errorf("DecodeCSVCell(EncodeCSVCell(%q)) = %q", value, decoded)
		}
	}

	var out strings.Builder
	writer, err := NewCSVWriter(&out, []string{"id", "name"}, DefaultCSVTransferPolicy())
	if err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteRow([]string{"user-1", "comma, quote\" and\nnewline"}); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	accepts := func(key string) bool { return key == "id" || key == "name" }
	reader, err := NewCSVReader(strings.NewReader(out.String()), accepts, DefaultCSVTransferPolicy())
	if err != nil {
		t.Fatal(err)
	}
	record, err := reader.Next()
	if err != nil || record.Row == nil {
		t.Fatalf("next = %+v, %v", record, err)
	}
	cell, _ := record.Row.Cell("name")
	if cell.Raw != "comma, quote\" and\nnewline" {
		t.Fatalf("round-trip cell=%q", cell.Raw)
	}
}

// 拒否は observable な誤りを返すだけでなく、1 行も読ませてはならない。
// `accepts` が `password` を許していても拒否が残ることで、禁止一覧が種別ごとの
// 語彙とは別の防護であることを固定する。
//
//spec:covers EX-IDMANAGEMENT-004-03, EX-IDMANAGEMENT-026-03: 未知の列、重複した列、password / password_hash を含むヘッダーが invalid_header でファイルごと拒否され、1 行も読めないこと。
func TestCSVReaderRefusesForbiddenAndUnknownHeaders(t *testing.T) {
	// `accepts` が秘密の列まで許していても拒否は残る。禁止一覧は種別ごとの語彙とは
	// 別の防護であり、語彙の側を緩めても素通りしてはならない。
	accepts := func(key string) bool {
		return key == "id" || key == "name" || key == "password" || key == "password_hash"
	}
	for _, header := range []string{"id,password", "id,password_hash", "id,unknown", "id,id"} {
		reader, err := NewCSVReader(strings.NewReader(header+"\na,b\n"), accepts, DefaultCSVTransferPolicy())
		var csvErr *CSVError
		if reader != nil {
			t.Fatalf("header %q produced a reader; a refused file must not be readable", header)
		}
		if !errors.As(err, &csvErr) || csvErr.Code != CSVErrorInvalidHeader {
			t.Fatalf("header %q error = %v, want invalid_header", header, err)
		}
	}
}

// 上限の検査は、上限の超過だけをファイル全体の拒否として返す。見出しの語彙と構文の
// 誤りは行の判定に委ねるので、ここでは拒否しない。
func TestCheckCSVLimitsReportsOnlyTheLimitTheFileExceeds(t *testing.T) {
	policy := CSVTransferPolicy{MaxRows: 2, MaxBytes: 64, MaxFieldBytes: 8}
	for _, tc := range []struct {
		name     string
		document string
		want     *CSVError
	}{
		{name: "上限ちょうど", document: "name\nalice\nbob\n"},
		{name: "行数の超過", document: "name\nalice\nbob\ncarol\n", want: &CSVError{Row: 4, Code: CSVErrorTooManyRows}},
		{name: "行の項目長の超過", document: "name,role\nalice,administrator\n", want: &CSVError{Row: 2, Column: "role", Code: CSVErrorFieldTooLarge}},
		{name: "見出しの項目長の超過", document: "department\nsales\n", want: &CSVError{Row: 1, Column: "department", Code: CSVErrorFieldTooLarge}},
		{name: "byte 数の超過", document: "name\n" + strings.Repeat("a\n", 40), want: &CSVError{Row: 1, Code: CSVErrorCSVTooLarge}},
		{name: "見出しの語彙は問わない", document: "password,password\nsecret,secret\n"},
		{name: "構文の誤りの後は読まない", document: "name\na\"b\nbob\ncarol\ndave\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckCSVLimits(strings.NewReader(tc.document), policy)
			if tc.want == nil {
				if err != nil {
					t.Fatalf("err = %v, want nil", err)
				}
				return
			}
			csvErr, ok := errors.AsType[*CSVError](err)
			if !ok || *csvErr != *tc.want {
				t.Fatalf("err = %v, want %v", err, *tc.want)
			}
		})
	}
}

type failingWriter struct{}

var errFailingWriter = errors.New("the artifact store refused the write")

func (failingWriter) Write([]byte) (int, error) { return 0, errFailingWriter }

// 複製は上限内のファイルを 1 byte も欠かさずに書き、上限を超えるファイルを拒否する。
// 構文の誤りで検査が読むのをやめても、残りを書き切る。
func TestCopyCSVWithinPolicyWritesTheWholeFileOrRefusesIt(t *testing.T) {
	policy := CSVTransferPolicy{MaxRows: 2, MaxBytes: 64, MaxFieldBytes: 8}
	for _, document := range []string{"name\nalice\nbob\n", "name\na\"b\nbob\ncarol\ndave\n"} {
		var output strings.Builder
		if err := CopyCSVWithinPolicy(&output, strings.NewReader(document), policy); err != nil || output.String() != document {
			t.Fatalf("copy of %q = (%q, %v), want the whole file", document, output.String(), err)
		}
	}
	for document, want := range map[string]CSVErrorCode{
		"name\nalice\nbob\ncarol\n":          CSVErrorTooManyRows,
		"name\nadministrator\n":              CSVErrorFieldTooLarge,
		"name\n\"" + strings.Repeat("a", 70): CSVErrorCSVTooLarge,
	} {
		err := CopyCSVWithinPolicy(io.Discard, strings.NewReader(document), policy)
		if csvErr, ok := errors.AsType[*CSVError](err); !ok || csvErr.Code != want {
			t.Fatalf("copy of %q err = %v, want %q", document, err, want)
		}
	}
	if err := CopyCSVWithinPolicy(failingWriter{}, strings.NewReader("name\nalice\n"), policy); !errors.Is(err, errFailingWriter) {
		t.Fatalf("err = %v, want the write error", err)
	}
}

// 少しずつ届く入力では、検査が構文の誤りで読むのをやめた時点で、まだ byte 数の上限に
// 達していないことがある。残りを書いた後の byte 数でも上限を確かめる。
func TestCopyCSVWithinPolicyCountsTheBytesAfterASyntaxError(t *testing.T) {
	policy := CSVTransferPolicy{MaxRows: 100, MaxBytes: 64, MaxFieldBytes: 1 << 10}
	head := "name\na\"b\n"
	atLimit := head + strings.Repeat("x", policy.MaxBytes-len(head))

	var output strings.Builder
	if err := CopyCSVWithinPolicy(&output, iotest.OneByteReader(strings.NewReader(atLimit)), policy); err != nil || output.String() != atLimit {
		t.Fatalf("copy of a file at the limit = (%d bytes, %v), want all %d bytes", output.Len(), err, len(atLimit))
	}
	err := CopyCSVWithinPolicy(io.Discard, iotest.OneByteReader(strings.NewReader(atLimit+"x")), policy)
	if csvErr, ok := errors.AsType[*CSVError](err); !ok || csvErr.Code != CSVErrorCSVTooLarge {
		t.Fatalf("copy of a file one byte over the limit err = %v, want csv_too_large", err)
	}
}
