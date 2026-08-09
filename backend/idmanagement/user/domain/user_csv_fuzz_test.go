package domain

import (
	"errors"
	"io"
	"strings"
	"testing"
)

func FuzzUserCSVParserAndFormulaSafeCodec(f *testing.F) {
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
		encoded := EncodeUserCSVCell(value)
		if isUserCSVFormulaTrigger(encoded) {
			t.Fatalf("encoded value remains dangerous: %q", encoded)
		}
		if got := DecodeUserCSVCell(encoded); got != value {
			t.Fatalf("round-trip mismatch: got %q, want %q", got, value)
		}
		reader, err := NewUserCSVReader(strings.NewReader(value), schema, DefaultUserCSVTransferPolicy())
		if err != nil {
			return
		}
		for {
			_, err := reader.Next()
			if errors.Is(err, io.EOF) || err != nil {
				return
			}
		}
	})
}
