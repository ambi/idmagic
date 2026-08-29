package domain

import (
	"errors"
	"strings"
	"testing"
)

// scenario: エクスポートとインポートは `decode(encode(value)) == value` を共有する
// (REQ-IDMANAGEMENT-007 / REQ-IDMANAGEMENT-027)。数式安全変換は情報を失う
// エスケープではなく、可逆な接頭辞でなければならない。
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

// scenario: シークレットを含むヘッダーはどの CSV 種別でもファイルごと拒否する
// (REQ-IDMANAGEMENT-004 / REQ-IDMANAGEMENT-026)。拒否は observable な誤りを
// 返すだけでなく、1 行も読ませてはならない。
func TestCSVReaderRefusesForbiddenAndUnknownHeaders(t *testing.T) {
	accepts := func(key string) bool { return key == "id" || key == "name" || key == "password" }
	for _, header := range []string{"id,password", "id,unknown", "id,id"} {
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
