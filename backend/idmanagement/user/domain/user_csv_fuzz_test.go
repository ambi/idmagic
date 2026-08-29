package domain

import (
	"errors"
	"io"
	"strings"
	"testing"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
)

// FuzzUserCSVParser: User 方言の列の語彙で解析器を駆動する。可逆な変換そのものは
// idmdomain の FuzzCSVFormulaSafeCodec が押さえるため、ここで重ねない。oracle は
// 「受理した行はスキーマが受理する機械キーだけを持つ」という構造的な境界である。
func FuzzUserCSVParser(f *testing.F) {
	for _, seed := range []string{
		"plain", "=formula", "'apostrophe", "comma,value", "multi\nline", "\x00",
		"id,email\nuser-1,alice@example.com\n", "id,id\na,b\n",
	} {
		f.Add(seed)
	}
	schema, err := NewUserCSVSchema(nil)
	if err != nil {
		f.Fatal(err)
	}
	f.Fuzz(func(t *testing.T, value string) {
		reader, err := idmdomain.NewCSVReader(strings.NewReader(value), schema.Accepts, idmdomain.DefaultCSVTransferPolicy())
		if err != nil {
			return
		}
		for _, key := range reader.Header() {
			if !schema.Accepts(key) {
				t.Fatalf("accepted header column %q is outside the User CSV schema", key)
			}
		}
		for {
			record, err := reader.Next()
			if errors.Is(err, io.EOF) || err != nil {
				return
			}
			if record.Error != nil {
				continue
			}
			if _, code := UserCSVIdentifierOf(*record.Row); code != "" && code != "missing_identifier" {
				t.Fatalf("unexpected identifier code %q", code)
			}
		}
	})
}
