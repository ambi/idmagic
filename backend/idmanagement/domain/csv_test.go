package domain

import (
	"errors"
	"strings"
	"testing"
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
