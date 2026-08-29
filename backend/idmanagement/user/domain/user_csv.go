package domain

// User CSV の方言。転送ポリシー、解析器、直列化器、可逆なセル変換は
// idmdomain の種別非依存な CSV 基盤が持ち、ここには User 固有の列の語彙、
// 属性セルの字句形、行計画の型だけを置く。

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
)

type UserCSVColumnMode string

const (
	UserCSVColumnWritable UserCSVColumnMode = "writable"
	UserCSVColumnReadOnly UserCSVColumnMode = "read_only"
	UserCSVColumnIdentity UserCSVColumnMode = "identity"
)

// UserCSVColumn is a machine-key column definition shared by import and export.
// Attribute is non-nil only for schema-backed attr:<key> (builtin extended
// attribute) or custom:<key> (tenant-defined attribute) columns.
type UserCSVColumn struct {
	Key       string
	Mode      UserCSVColumnMode
	Attribute *UserAttributeDef
}

type UserCSVSchema struct {
	columns map[string]UserCSVColumn
	ordered []UserCSVColumn
}

var builtinUserCSVColumns = []UserCSVColumn{
	{Key: "id", Mode: UserCSVColumnIdentity},
	{Key: "preferred_username", Mode: UserCSVColumnWritable},
	{Key: "name", Mode: UserCSVColumnWritable},
	{Key: "given_name", Mode: UserCSVColumnWritable},
	{Key: "family_name", Mode: UserCSVColumnWritable},
	{Key: "email", Mode: UserCSVColumnWritable},
	{Key: "email_verified", Mode: UserCSVColumnWritable},
	{Key: "roles", Mode: UserCSVColumnWritable},
	{Key: "required_actions", Mode: UserCSVColumnWritable},
	{Key: "mfa_enrolled", Mode: UserCSVColumnReadOnly},
	{Key: "status", Mode: UserCSVColumnReadOnly},
	{Key: "created_at", Mode: UserCSVColumnReadOnly},
	{Key: "updated_at", Mode: UserCSVColumnReadOnly},
}

// builtinUserCSVAttributeKeys is the set of builtin extended attribute keys
// (attributes.go builtinDefs) that resolve to attr:<key> columns instead of
// custom:<key> (wi-352).
var builtinUserCSVAttributeKeys = func() map[string]struct{} {
	set := make(map[string]struct{}, len(builtinDefs))
	for _, d := range builtinDefs {
		set[d.Key] = struct{}{}
	}
	return set
}()

// NewUserCSVSchema builds the CSV column schema from the effective attribute
// defs (builtin extended attributes + tenant custom attributes, as returned
// by EffectiveUserAttributeSchemaReader). Builtin extended attributes resolve
// to attr:<key> columns; tenant-defined attributes resolve to custom:<key>.
func NewUserCSVSchema(defs []UserAttributeDef) (UserCSVSchema, error) {
	columns := make(map[string]UserCSVColumn, len(builtinUserCSVColumns)+len(defs))
	ordered := make([]UserCSVColumn, 0, len(builtinUserCSVColumns)+len(defs))
	for _, column := range builtinUserCSVColumns {
		columns[column.Key] = column
		ordered = append(ordered, column)
	}
	for _, def := range defs {
		if err := def.Validate(); err != nil {
			return UserCSVSchema{}, err
		}
		prefix := "custom:"
		if _, isBuiltin := builtinUserCSVAttributeKeys[def.Key]; isBuiltin {
			prefix = "attr:"
		}
		key := prefix + def.Key
		if _, exists := columns[key]; exists {
			return UserCSVSchema{}, fmt.Errorf("duplicate user CSV column %q", key)
		}
		copyDef := def
		column := UserCSVColumn{Key: key, Mode: UserCSVColumnWritable, Attribute: &copyDef}
		columns[key] = column
		ordered = append(ordered, column)
	}
	return UserCSVSchema{columns: columns, ordered: ordered}, nil
}

func (s UserCSVSchema) Column(key string) (UserCSVColumn, bool) {
	column, ok := s.columns[key]
	return column, ok
}

func (s UserCSVSchema) Columns() []UserCSVColumn {
	return append([]UserCSVColumn(nil), s.ordered...)
}

// Accepts は共有解析器へ渡す、この方言が受理する機械キーの判定である。
func (s UserCSVSchema) Accepts(key string) bool {
	_, ok := s.columns[key]
	return ok
}

type UserCSVIdentifier struct {
	ID                string
	PreferredUsername string
}

// UserCSVIdentifierOf は 1 行から User の識別子を読み出す。行の型は種別非依存
// なので、識別の規則は方言側の関数として持つ。
func UserCSVIdentifierOf(row idmdomain.CSVRow) (UserCSVIdentifier, idmdomain.CSVErrorCode) {
	id := UserCSVIdentifier{
		ID:                row.TrimmedCell("id"),
		PreferredUsername: row.TrimmedCell("preferred_username"),
	}
	if id.ID == "" && id.PreferredUsername == "" {
		return id, "missing_identifier"
	}
	return id, ""
}

// ValidateUserCSVTargets はリポジトリを引かずに判定できる衝突を検出する。
// リポジトリを要する ID と username の食い違いは計画器の責務である。
func ValidateUserCSVTargets(rows []idmdomain.CSVRow) []idmdomain.CSVError {
	seenIDs := map[string]struct{}{}
	seenUsernames := map[string]struct{}{}
	var out []idmdomain.CSVError
	for _, row := range rows {
		identifier, code := UserCSVIdentifierOf(row)
		if code != "" {
			out = append(out, idmdomain.CSVError{Row: row.Number, Code: code})
			continue
		}
		if identifier.ID != "" {
			if _, duplicate := seenIDs[identifier.ID]; duplicate {
				out = append(out, idmdomain.CSVError{Row: row.Number, Column: "id", Code: "duplicate_target"})
				continue
			}
			seenIDs[identifier.ID] = struct{}{}
		}
		if identifier.PreferredUsername != "" {
			if _, duplicate := seenUsernames[identifier.PreferredUsername]; duplicate {
				out = append(out, idmdomain.CSVError{Row: row.Number, Column: "preferred_username", Code: "duplicate_username"})
				continue
			}
			seenUsernames[identifier.PreferredUsername] = struct{}{}
		}
	}
	return out
}

var ErrInvalidUserCSVCell = errors.New("invalid user CSV cell")

// ParseUserCSVAttributeCell applies the canonical lexical form for one
// schema-backed custom column. clear=true represents an optional present-empty
// cell and is distinct from an absent column.
func ParseUserCSVAttributeCell(raw string, def UserAttributeDef) (AttributeValue, bool, error) {
	if raw == "" {
		if def.Required {
			return AttributeValue{}, false, fmt.Errorf("%w: required attribute", ErrInvalidUserCSVCell)
		}
		return AttributeValue{}, true, nil
	}
	switch def.Type {
	case idmdomain.AttributeTypeString:
		value := raw
		return AttributeValue{Type: def.Type, String: &value}, false, nil
	case idmdomain.AttributeTypeDate:
		parsed, err := time.Parse("2006-01-02", raw)
		if err != nil || parsed.Format("2006-01-02") != raw {
			return AttributeValue{}, false, fmt.Errorf("%w: date", ErrInvalidUserCSVCell)
		}
		value := raw
		return AttributeValue{Type: def.Type, Date: &value}, false, nil
	case idmdomain.AttributeTypeNumber:
		value, err := strconv.ParseFloat(raw, 64)
		if err != nil || math.IsNaN(value) || math.IsInf(value, 0) || strconv.FormatFloat(value, 'g', -1, 64) != raw {
			return AttributeValue{}, false, fmt.Errorf("%w: number", ErrInvalidUserCSVCell)
		}
		return AttributeValue{Type: def.Type, Number: &value}, false, nil
	case idmdomain.AttributeTypeBoolean:
		if raw != "true" && raw != "false" {
			return AttributeValue{}, false, fmt.Errorf("%w: boolean", ErrInvalidUserCSVCell)
		}
		value := raw == "true"
		return AttributeValue{Type: def.Type, Boolean: &value}, false, nil
	case idmdomain.AttributeTypeStringArray:
		var value []string
		if err := json.Unmarshal([]byte(raw), &value); err != nil || value == nil {
			return AttributeValue{}, false, fmt.Errorf("%w: string array", ErrInvalidUserCSVCell)
		}
		canonical, err := json.Marshal(value)
		if err != nil || string(canonical) != raw {
			return AttributeValue{}, false, fmt.Errorf("%w: string array", ErrInvalidUserCSVCell)
		}
		return AttributeValue{Type: def.Type, StringArray: value}, false, nil
	default:
		return AttributeValue{}, false, fmt.Errorf("%w: unsupported type", ErrInvalidUserCSVCell)
	}
}

// FormatUserCSVAttributeCell is the inverse lexical projection used by User
// export. Formula protection is applied later by idmdomain.CSVWriter.
func FormatUserCSVAttributeCell(value AttributeValue, def UserAttributeDef) (string, error) {
	if err := ValidateAttributeValue(value, def); err != nil {
		return "", err
	}
	switch def.Type {
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
		return "", fmt.Errorf("%w: unsupported type", ErrInvalidUserCSVCell)
	}
}

type UserImportAction string

const (
	UserImportCreate    UserImportAction = "created"
	UserImportUpdate    UserImportAction = "updated"
	UserImportUnchanged UserImportAction = "unchanged"
	UserImportRejected  UserImportAction = "rejected"
)

type UserImportRowPlan struct {
	Row        int
	Action     UserImportAction
	Identifier UserCSVIdentifier
	Before     *User
	User       *User
	Error      *idmdomain.CSVError
}

type UserImportPlan struct {
	Rows []UserImportRowPlan
}

type UserImportPlanSummary struct {
	TotalRows     int `json:"total_rows"`
	CreatedRows   int `json:"created_rows"`
	UpdatedRows   int `json:"updated_rows"`
	UnchangedRows int `json:"unchanged_rows"`
	RejectedRows  int `json:"rejected_rows"`
}

func (s *UserImportPlanSummary) Observe(row UserImportRowPlan) {
	s.TotalRows++
	switch row.Action {
	case UserImportCreate:
		s.CreatedRows++
	case UserImportUpdate:
		s.UpdatedRows++
	case UserImportUnchanged:
		s.UnchangedRows++
	case UserImportRejected:
		s.RejectedRows++
	}
}

func (p UserImportPlan) count(action UserImportAction) int {
	count := 0
	for _, row := range p.Rows {
		if row.Action == action {
			count++
		}
	}
	return count
}

func (p UserImportPlan) TotalRows() int     { return len(p.Rows) }
func (p UserImportPlan) CreatedRows() int   { return p.count(UserImportCreate) }
func (p UserImportPlan) UpdatedRows() int   { return p.count(UserImportUpdate) }
func (p UserImportPlan) UnchangedRows() int { return p.count(UserImportUnchanged) }
func (p UserImportPlan) RejectedRows() int  { return p.count(UserImportRejected) }
