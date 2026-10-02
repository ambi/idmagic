package domain

import (
	"errors"
	"testing"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
)

//spec:covers EX-IDMANAGEMENT-038-01: 型ごとの正規の字句形のセルを値として読み、同じ字句形で書き戻すこと。
func TestAttributeCellReadsAndWritesTheCanonicalForm(t *testing.T) {
	cases := []struct {
		attrType idmdomain.AttributeType
		raw      string
	}{
		{idmdomain.AttributeTypeString, " spaced value "},
		{idmdomain.AttributeTypeDate, "2026-02-28"},
		{idmdomain.AttributeTypeNumber, "1.5"},
		{idmdomain.AttributeTypeNumber, "1e+21"},
		{idmdomain.AttributeTypeBoolean, "false"},
		{idmdomain.AttributeTypeStringArray, `["a","b"]`},
	}
	for _, tc := range cases {
		value, shouldClear, err := ParseAttributeCell(tc.raw, tc.attrType, false)
		if err != nil || shouldClear {
			t.Fatalf("%s %q: shouldClear=%v err=%v, want a value", tc.attrType, tc.raw, shouldClear, err)
		}
		formatted, err := FormatAttributeCell(value)
		if err != nil || formatted != tc.raw {
			t.Fatalf("%s %q: formatted=%q err=%v, want the same lexical form", tc.attrType, tc.raw, formatted, err)
		}
	}
	value, _, _ := ParseAttributeCell("1.5", idmdomain.AttributeTypeNumber, false)
	if value.Number == nil || *value.Number != 1.5 {
		t.Fatalf("number=%v, want 1.5", value.Number)
	}
}

//spec:covers EX-IDMANAGEMENT-038-02: 正規でない字句形のセルを型ごとに拒否すること。
func TestAttributeCellRejectsNonCanonicalForms(t *testing.T) {
	cases := []struct {
		attrType idmdomain.AttributeType
		raw      string
	}{
		{idmdomain.AttributeTypeNumber, "1.0"},
		{idmdomain.AttributeTypeNumber, "01"},
		{idmdomain.AttributeTypeNumber, "NaN"},
		{idmdomain.AttributeTypeDate, "2026-2-28"},
		{idmdomain.AttributeTypeDate, "2026-02-30"},
		{idmdomain.AttributeTypeBoolean, "TRUE"},
		{idmdomain.AttributeTypeStringArray, `["a", "b"]`},
		{idmdomain.AttributeTypeStringArray, `null`},
	}
	for _, tc := range cases {
		if _, _, err := ParseAttributeCell(tc.raw, tc.attrType, false); !errors.Is(err, ErrInvalidAttributeCell) {
			t.Fatalf("%s %q: err=%v, want ErrInvalidAttributeCell", tc.attrType, tc.raw, err)
		}
	}
}

//spec:covers EX-IDMANAGEMENT-038-03: 任意の属性の空のセルを属性を消す意味として読み、必須の属性の空のセルを拒否すること。
func TestAttributeCellEmptyClearsOptionalAndRejectsRequired(t *testing.T) {
	if _, shouldClear, err := ParseAttributeCell("", idmdomain.AttributeTypeString, false); err != nil || !shouldClear {
		t.Fatalf("任意の属性: shouldClear=%v err=%v, want shouldClear", shouldClear, err)
	}
	if _, shouldClear, err := ParseAttributeCell("", idmdomain.AttributeTypeString, true); !errors.Is(err, ErrInvalidAttributeCell) || shouldClear {
		t.Fatalf("必須の属性: shouldClear=%v err=%v, want ErrInvalidAttributeCell", shouldClear, err)
	}
}
