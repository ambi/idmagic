package domain

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
)

// UserCSVTransferPolicy is injected by the composition boundary and shared by
// User export and import. It limits one artifact, not a tenant's population.
type UserCSVTransferPolicy struct {
	MaxRows       int
	MaxBytes      int
	MaxFieldBytes int
}

func DefaultUserCSVTransferPolicy() UserCSVTransferPolicy {
	return UserCSVTransferPolicy{MaxRows: 100_000, MaxBytes: 64 << 20, MaxFieldBytes: 64 << 10}
}

func (p UserCSVTransferPolicy) Validate() error {
	if p.MaxRows <= 0 || p.MaxBytes <= 0 || p.MaxFieldBytes <= 0 {
		return errors.New("user CSV transfer policy values must be positive")
	}
	return nil
}

type UserCSVErrorCode string

const (
	UserCSVErrorInvalidCSV         UserCSVErrorCode = "invalid_csv"
	UserCSVErrorInvalidHeader      UserCSVErrorCode = "invalid_header"
	UserCSVErrorCSVTooLarge        UserCSVErrorCode = "csv_too_large"
	UserCSVErrorTooManyRows        UserCSVErrorCode = "too_many_rows"
	UserCSVErrorFieldTooLarge      UserCSVErrorCode = "field_too_large"
	UserCSVErrorInvalidColumnCount UserCSVErrorCode = "invalid_column_count"
)

// UserCSVError carries stable location and code metadata only. Raw cell values
// deliberately never cross the domain boundary in an error.
type UserCSVError struct {
	Row    int              `json:"row,omitempty"`
	Column string           `json:"column,omitempty"`
	Code   UserCSVErrorCode `json:"code"`
}

func (e UserCSVError) Error() string {
	return fmt.Sprintf("user CSV error: row=%d column=%q code=%q", e.Row, e.Column, e.Code)
}

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

type UserCSVCell struct {
	Present bool
	Raw     string
}

type UserCSVIdentifier struct {
	ID                string
	PreferredUsername string
}

type UserCSVRow struct {
	Number int
	cells  map[string]UserCSVCell
}

func (r UserCSVRow) Cell(key string) (UserCSVCell, bool) {
	cell, ok := r.cells[key]
	return cell, ok
}

func (r UserCSVRow) Identifier() (UserCSVIdentifier, UserCSVErrorCode) {
	id := UserCSVIdentifier{}
	if cell, ok := r.Cell("id"); ok {
		id.ID = strings.TrimSpace(cell.Raw)
	}
	if cell, ok := r.Cell("preferred_username"); ok {
		id.PreferredUsername = strings.TrimSpace(cell.Raw)
	}
	if id.ID == "" && id.PreferredUsername == "" {
		return id, "missing_identifier"
	}
	return id, ""
}

var forbiddenUserCSVHeaders = map[string]struct{}{
	"password":      {},
	"password_hash": {},
	"mfa_secret":    {},
	"token":         {},
	"recovery_code": {},
}

type countingReader struct {
	r io.Reader
	n int64
}

func (r *countingReader) Read(p []byte) (int, error) {
	n, err := r.r.Read(p)
	r.n += int64(n)
	return n, err
}

type UserCSVRecord struct {
	Row   *UserCSVRow
	Error *UserCSVError
}

// UserCSVReader incrementally parses RFC 4180 records. It retains only the
// header and current record; callers own batching and result collection.
type UserCSVReader struct {
	csv      *csv.Reader
	counting *countingReader
	header   []string
	policy   UserCSVTransferPolicy
	rowsRead int
	nextLine int
}

func NewUserCSVReader(input io.Reader, schema UserCSVSchema, policy UserCSVTransferPolicy) (*UserCSVReader, error) {
	if err := policy.Validate(); err != nil {
		return nil, err
	}
	limited := &io.LimitedReader{R: input, N: int64(policy.MaxBytes) + 1}
	counting := &countingReader{r: limited}
	r := csv.NewReader(counting)
	r.FieldsPerRecord = -1
	header, err := r.Read()
	if counting.n > int64(policy.MaxBytes) {
		return nil, &UserCSVError{Row: 1, Code: UserCSVErrorCSVTooLarge}
	}
	if err != nil || len(header) == 0 {
		return nil, &UserCSVError{Row: 1, Code: UserCSVErrorInvalidCSV}
	}
	header[0] = strings.TrimPrefix(header[0], "\ufeff")
	seen := make(map[string]struct{}, len(header))
	for _, key := range header {
		if len(key) > policy.MaxFieldBytes {
			return nil, &UserCSVError{Row: 1, Column: key, Code: UserCSVErrorFieldTooLarge}
		}
		if _, forbidden := forbiddenUserCSVHeaders[key]; forbidden {
			return nil, &UserCSVError{Row: 1, Column: key, Code: UserCSVErrorInvalidHeader}
		}
		if _, ok := schema.Column(key); !ok {
			return nil, &UserCSVError{Row: 1, Column: key, Code: UserCSVErrorInvalidHeader}
		}
		if _, duplicate := seen[key]; duplicate {
			return nil, &UserCSVError{Row: 1, Column: key, Code: UserCSVErrorInvalidHeader}
		}
		seen[key] = struct{}{}
	}
	return &UserCSVReader{csv: r, counting: counting, header: append([]string(nil), header...), policy: policy, nextLine: 2}, nil
}

func (r *UserCSVReader) Header() []string {
	return append([]string(nil), r.header...)
}

func (r *UserCSVReader) Next() (UserCSVRecord, error) {
	line := r.nextLine
	record, err := r.csv.Read()
	if r.counting.n > int64(r.policy.MaxBytes) {
		return UserCSVRecord{}, &UserCSVError{Row: line, Code: UserCSVErrorCSVTooLarge}
	}
	if errors.Is(err, io.EOF) {
		return UserCSVRecord{}, io.EOF
	}
	if err != nil {
		return UserCSVRecord{}, &UserCSVError{Row: line, Code: UserCSVErrorInvalidCSV}
	}
	r.nextLine++
	r.rowsRead++
	if r.rowsRead > r.policy.MaxRows {
		return UserCSVRecord{}, &UserCSVError{Row: line, Code: UserCSVErrorTooManyRows}
	}
	for i, raw := range record {
		if len(raw) > r.policy.MaxFieldBytes {
			column := ""
			if i < len(r.header) {
				column = r.header[i]
			}
			return UserCSVRecord{}, &UserCSVError{Row: line, Column: column, Code: UserCSVErrorFieldTooLarge}
		}
	}
	if len(record) != len(r.header) {
		return UserCSVRecord{Error: &UserCSVError{Row: line, Code: UserCSVErrorInvalidColumnCount}}, nil
	}
	row := &UserCSVRow{Number: line, cells: make(map[string]UserCSVCell, len(r.header))}
	for i, key := range r.header {
		row.cells[key] = UserCSVCell{Present: true, Raw: DecodeUserCSVCell(record[i])}
	}
	return UserCSVRecord{Row: row}, nil
}

// ValidateUserCSVTargets detects collisions that can be decided without a
// repository lookup. Repository-aware ID/username disagreement is a planner
// responsibility in the use-case layer.
func ValidateUserCSVTargets(rows []UserCSVRow) []UserCSVError {
	seenIDs := map[string]struct{}{}
	seenUsernames := map[string]struct{}{}
	var out []UserCSVError
	for _, row := range rows {
		identifier, code := row.Identifier()
		if code != "" {
			out = append(out, UserCSVError{Row: row.Number, Code: code})
			continue
		}
		if identifier.ID != "" {
			if _, duplicate := seenIDs[identifier.ID]; duplicate {
				out = append(out, UserCSVError{Row: row.Number, Column: "id", Code: "duplicate_target"})
				continue
			}
			seenIDs[identifier.ID] = struct{}{}
		}
		if identifier.PreferredUsername != "" {
			if _, duplicate := seenUsernames[identifier.PreferredUsername]; duplicate {
				out = append(out, UserCSVError{Row: row.Number, Column: "preferred_username", Code: "duplicate_username"})
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
// export. Formula protection is applied later by UserCSVWriter.
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

func isUserCSVFormulaTrigger(value string) bool {
	if value == "" {
		return false
	}
	switch value[0] {
	case '=', '+', '-', '@', '\t', '\r', '\n':
		return true
	default:
		return false
	}
}

func EncodeUserCSVCell(value string) string {
	if value != "" && (value[0] == '\'' || isUserCSVFormulaTrigger(value)) {
		return "'" + value
	}
	return value
}

func DecodeUserCSVCell(value string) string {
	if len(value) < 2 || value[0] != '\'' {
		return value
	}
	if value[1] == '\'' || isUserCSVFormulaTrigger(value[1:]) {
		return value[1:]
	}
	return value
}

type policyWriter struct {
	w       io.Writer
	written int
	max     int
}

func (w *policyWriter) Write(p []byte) (int, error) {
	if len(p) > w.max-w.written {
		return 0, &UserCSVError{Code: UserCSVErrorCSVTooLarge}
	}
	n, err := w.w.Write(p)
	w.written += n
	return n, err
}

// UserCSVWriter serializes one record at a time and enforces the same policy as
// UserCSVReader without retaining prior rows.
type UserCSVWriter struct {
	csv       *csv.Writer
	headerLen int
	policy    UserCSVTransferPolicy
	rows      int
}

func NewUserCSVWriter(output io.Writer, header []string, policy UserCSVTransferPolicy) (*UserCSVWriter, error) {
	if err := policy.Validate(); err != nil {
		return nil, err
	}
	if len(header) == 0 {
		return nil, &UserCSVError{Row: 1, Code: UserCSVErrorInvalidHeader}
	}
	seen := map[string]struct{}{}
	for _, key := range header {
		if len(key) > policy.MaxFieldBytes {
			return nil, &UserCSVError{Row: 1, Column: key, Code: UserCSVErrorFieldTooLarge}
		}
		if _, duplicate := seen[key]; duplicate {
			return nil, &UserCSVError{Row: 1, Column: key, Code: UserCSVErrorInvalidHeader}
		}
		seen[key] = struct{}{}
	}
	bounded := &policyWriter{w: output, max: policy.MaxBytes}
	w := &UserCSVWriter{csv: csv.NewWriter(bounded), headerLen: len(header), policy: policy}
	if err := w.writeRecord(header); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *UserCSVWriter) writeRecord(record []string) error {
	encoded := make([]string, len(record))
	for i, value := range record {
		encoded[i] = EncodeUserCSVCell(value)
	}
	if err := w.csv.Write(encoded); err != nil {
		return err
	}
	return nil
}

func (w *UserCSVWriter) WriteRow(record []string) error {
	if len(record) != w.headerLen {
		return &UserCSVError{Row: w.rows + 2, Code: UserCSVErrorInvalidColumnCount}
	}
	if w.rows >= w.policy.MaxRows {
		return &UserCSVError{Row: w.rows + 2, Code: UserCSVErrorTooManyRows}
	}
	for i, value := range record {
		if len(value) > w.policy.MaxFieldBytes {
			return &UserCSVError{Row: w.rows + 2, Column: strconv.Itoa(i), Code: UserCSVErrorFieldTooLarge}
		}
	}
	if err := w.writeRecord(record); err != nil {
		return err
	}
	w.rows++
	return nil
}

func (w *UserCSVWriter) Close() error {
	w.csv.Flush()
	return w.csv.Error()
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
	Error      *UserCSVError
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
