package spec

import (
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	z "github.com/Oudwins/zog"
)

type lengthSubject struct {
	Name              string
	Description       string
	PreferredUsername string
}

func lengthSchema() *z.StructSchema {
	return z.Struct(z.Shape{
		"Name":              Chars(1, LengthName).Required(),
		"Description":       CharsAtMost(LengthDescription),
		"PreferredUsername": CharsAtMost(LengthName),
	})
}

func validateLength(subject lengthSubject) error {
	return Validate(lengthSchema(), &subject)
}

// zog の Max は UTF-8 バイト数を数えるため、この 3 例はいずれも byte 単位では
// 上限を超える。コードポイントで数えている限り通らなければならない。
func TestCharsCountsCodePointsNotBytes(t *testing.T) {
	for _, tc := range []struct {
		label string
		unit  string
	}{
		{"ASCII 1 バイト", "a"},
		{"日本語 3 バイト", "あ"},
		{"絵文字 4 バイト", "\U0001F642"},
	} {
		t.Run(tc.label, func(t *testing.T) {
			name := strings.Repeat(tc.unit, LengthName)
			if err := validateLength(lengthSubject{Name: name}); err != nil {
				t.Fatalf("%d code points (%d bytes) rejected: %v", LengthName, len(name), err)
			}
		})
	}
}

func TestCharsRejectsOneCodePointOverTheLimit(t *testing.T) {
	for _, tc := range []struct {
		label string
		unit  string
	}{
		{"ASCII", "a"},
		{"日本語", "あ"},
		{"絵文字", "\U0001F642"},
	} {
		t.Run(tc.label, func(t *testing.T) {
			err := validateLength(lengthSubject{Name: strings.Repeat(tc.unit, LengthName+1)})
			if err == nil {
				t.Fatal("expected the value one code point over the limit to be rejected")
			}
			if !strings.Contains(err.Error(), "at most 100 characters") {
				t.Fatalf("message does not name the limit: %v", err)
			}
		})
	}
}

// 結合文字は正規化しない限り 1 書記素あたり複数のコードポイントを占める。
// 上限はコードポイント数であって書記素数ではない、という約束を固定する。
func TestCharsCountsCombiningMarksSeparately(t *testing.T) {
	// U+304B "か" に結合濁点 U+3099 を付けた NFD 形。1 書記素 2 コードポイント。
	const decomposed = "が"
	value := strings.Repeat(decomposed, LengthName/2)
	if got := utf8.RuneCountInString(value); got != LengthName {
		t.Fatalf("fixture is %d code points, want %d", got, LengthName)
	}
	if err := validateLength(lengthSubject{Name: value}); err != nil {
		t.Fatalf("exactly %d code points rejected: %v", LengthName, err)
	}
	// 書記素で数えれば 50 個しかないが、上限が数えるのはコードポイントである。
	if err := validateLength(lengthSubject{Name: value + "゙"}); err == nil {
		t.Fatalf("expected %d code points to be rejected", LengthName+1)
	}
}

func TestCharsRejectsEmptyWhenAMinimumIsSet(t *testing.T) {
	if err := validateLength(lengthSubject{Name: ""}); err == nil {
		t.Fatal("expected an empty required name to be rejected")
	}
	// 上限だけのフィールドは空文字列を未設定として受ける。
	if err := validateLength(lengthSubject{Name: "ok", Description: ""}); err != nil {
		t.Fatalf("empty optional description rejected: %v", err)
	}
}

func TestLengthViolationIsTyped(t *testing.T) {
	err := validateLength(lengthSubject{Name: strings.Repeat("a", LengthName+1)})
	var lengthErr *LengthError
	if !errors.As(err, &lengthErr) {
		t.Fatalf("length violation is not a *LengthError: %T %v", err, err)
	}
	if !lengthErr.IsFieldLengthViolation() {
		t.Fatal("IsFieldLengthViolation returned false")
	}
}

// 長さ以外の検証失敗まで 422 になると、保存済みデータの破損がサーバの不具合では
// なく利用者入力の誤りに見えてしまう。型を分けたままにする。
func TestNonLengthViolationIsNotTyped(t *testing.T) {
	schema := z.Struct(z.Shape{"Name": z.String().OneOf([]string{"allowed"}).Required()})
	subject := struct{ Name string }{Name: "other"}
	err := Validate(schema, &subject)
	var lengthErr *LengthError
	if err == nil || errors.As(err, &lengthErr) {
		t.Fatalf("expected a plain error, got %T %v", err, err)
	}
}

func TestValidationMessageUsesWireFieldNames(t *testing.T) {
	err := validateLength(lengthSubject{
		Name:              "ok",
		PreferredUsername: strings.Repeat("a", LengthName+1),
	})
	if err == nil || !strings.Contains(err.Error(), "preferred_username:") {
		t.Fatalf("message does not use the wire field name: %v", err)
	}
}

func TestSnakeCase(t *testing.T) {
	for input, want := range map[string]string{
		"Name":              "name",
		"PreferredUsername": "preferred_username",
		"OIDCScope":         "oidc_scope",
		"URL":               "url",
		"ClientID":          "client_id",
		"TenantID":          "tenant_id",
		"BodyHTML":          "body_html",
		"":                  "",
	} {
		if got := snakeCase(input); got != want {
			t.Errorf("snakeCase(%q) = %q, want %q", input, got, want)
		}
	}
}
