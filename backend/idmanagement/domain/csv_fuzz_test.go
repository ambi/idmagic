package domain

import (
	"errors"
	"io"
	"strings"
	"testing"
)

// FuzzCSVFormulaSafeCodec: 可逆性と数式安全性は種別に依存しないので、方言では
// なく共有基盤で押さえる。oracle は「符号化した値はもはや数式の引き金で始まらず、
// かつ復号すると元の値に一致する」という往復であり、「panic しない」ではない。
func FuzzCSVFormulaSafeCodec(f *testing.F) {
	for _, seed := range []string{
		"", "plain", "=formula", "+1", "-1", "@name", "\tvalue", "\rvalue", "\nvalue",
		"'apostrophe", "''twice", "comma,value", `quote"value`, "multi\nline", "\x00", "日本語",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, value string) {
		encoded := EncodeCSVCell(value)
		if isCSVFormulaTrigger(encoded) {
			t.Fatalf("encoded value remains dangerous: %q", encoded)
		}
		if got := DecodeCSVCell(encoded); got != value {
			t.Fatalf("round-trip mismatch: got %q, want %q", got, value)
		}
	})
}

// FuzzCSVReaderRejectsOrParses: 解析器は任意のバイト列に対して、ポリシー内の
// 正しい CSV として読み切るか、位置と安定コードを持つ CSVError で拒否するかの
// どちらかでなければならない。oracle は「返る誤りは必ず *CSVError であり、
// 受理した行の列数はヘッダーと一致する」という構造的な境界である。
func FuzzCSVReaderRejectsOrParses(f *testing.F) {
	for _, seed := range []string{
		"a,b\n1,2\n", "a,a\n1,2\n", "a\n1,2\n", "\ufeffa\n1\n", "a,b\n\"unterminated\n", "",
	} {
		f.Add(seed)
	}
	accepts := func(key string) bool { return key == "a" || key == "b" }
	policy := CSVTransferPolicy{MaxRows: 16, MaxBytes: 4096, MaxFieldBytes: 64}
	f.Fuzz(func(t *testing.T, document string) {
		reader, err := NewCSVReader(strings.NewReader(document), accepts, policy)
		if err != nil {
			if _, ok := errors.AsType[*CSVError](err); !ok {
				t.Fatalf("header rejection is not a CSVError: %T %v", err, err)
			}
			return
		}
		header := reader.Header()
		for {
			record, err := reader.Next()
			if errors.Is(err, io.EOF) {
				return
			}
			if err != nil {
				if _, ok := errors.AsType[*CSVError](err); !ok {
					t.Fatalf("row rejection is not a CSVError: %T %v", err, err)
				}
				return
			}
			if record.Error != nil {
				continue
			}
			for _, key := range header {
				if _, ok := record.Row.Cell(key); !ok {
					t.Fatalf("accepted row is missing header column %q", key)
				}
			}
		}
	})
}
