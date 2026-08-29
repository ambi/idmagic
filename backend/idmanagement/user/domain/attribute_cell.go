package domain

// 属性値 1 件の CSV 上の正規表記。表記を決めるのは属性の型であって CSV の種別では
// ないため、User と Group のどちらの方言もここを通る。値の型 AttributeValue が
// この package にあるので、その表記も同じ場所に置く。

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
)

var ErrInvalidAttributeCell = errors.New("invalid attribute cell")

// ParseAttributeCell applies the canonical lexical form for one schema-backed
// attribute column. clear=true represents an optional present-empty cell and is
// distinct from an absent column.
func ParseAttributeCell(raw string, attrType idmdomain.AttributeType, required bool) (AttributeValue, bool, error) {
	if raw == "" {
		if required {
			return AttributeValue{}, false, fmt.Errorf("%w: required attribute", ErrInvalidAttributeCell)
		}
		return AttributeValue{}, true, nil
	}
	switch attrType {
	case idmdomain.AttributeTypeString:
		value := raw
		return AttributeValue{Type: attrType, String: &value}, false, nil
	case idmdomain.AttributeTypeDate:
		parsed, err := time.Parse("2006-01-02", raw)
		if err != nil || parsed.Format("2006-01-02") != raw {
			return AttributeValue{}, false, fmt.Errorf("%w: date", ErrInvalidAttributeCell)
		}
		value := raw
		return AttributeValue{Type: attrType, Date: &value}, false, nil
	case idmdomain.AttributeTypeNumber:
		value, err := strconv.ParseFloat(raw, 64)
		if err != nil || math.IsNaN(value) || math.IsInf(value, 0) || strconv.FormatFloat(value, 'g', -1, 64) != raw {
			return AttributeValue{}, false, fmt.Errorf("%w: number", ErrInvalidAttributeCell)
		}
		return AttributeValue{Type: attrType, Number: &value}, false, nil
	case idmdomain.AttributeTypeBoolean:
		if raw != "true" && raw != "false" {
			return AttributeValue{}, false, fmt.Errorf("%w: boolean", ErrInvalidAttributeCell)
		}
		value := raw == "true"
		return AttributeValue{Type: attrType, Boolean: &value}, false, nil
	case idmdomain.AttributeTypeStringArray:
		var value []string
		if err := json.Unmarshal([]byte(raw), &value); err != nil || value == nil {
			return AttributeValue{}, false, fmt.Errorf("%w: string array", ErrInvalidAttributeCell)
		}
		canonical, err := json.Marshal(value)
		if err != nil || string(canonical) != raw {
			return AttributeValue{}, false, fmt.Errorf("%w: string array", ErrInvalidAttributeCell)
		}
		return AttributeValue{Type: attrType, StringArray: value}, false, nil
	default:
		return AttributeValue{}, false, fmt.Errorf("%w: unsupported type", ErrInvalidAttributeCell)
	}
}

// FormatAttributeCell is the inverse lexical projection used by export.
// Formula protection is applied later by idmdomain.CSVWriter.
func FormatAttributeCell(value AttributeValue) (string, error) {
	switch value.Type {
	case idmdomain.AttributeTypeString:
		return *value.String, nil
	case idmdomain.AttributeTypeDate:
		return *value.Date, nil
	case idmdomain.AttributeTypeNumber:
		return strconv.FormatFloat(*value.Number, 'g', -1, 64), nil
	case idmdomain.AttributeTypeBoolean:
		return strconv.FormatBool(*value.Boolean), nil
	case idmdomain.AttributeTypeStringArray:
		encoded, err := json.Marshal(value.StringArray)
		return string(encoded), err
	default:
		return "", fmt.Errorf("%w: unsupported type", ErrInvalidAttributeCell)
	}
}
