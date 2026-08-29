package domain

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
)

func collectUserCSV(t *testing.T, input io.Reader, schema UserCSVSchema, policy idmdomain.CSVTransferPolicy) ([]string, []idmdomain.CSVRow, error) {
	t.Helper()
	reader, err := idmdomain.NewCSVReader(input, schema.Accepts, policy)
	if err != nil {
		return nil, nil, err
	}
	var rows []idmdomain.CSVRow
	for {
		record, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return reader.Header(), rows, err
		}
		if record.Error != nil {
			return reader.Header(), rows, record.Error
		}
		rows = append(rows, *record.Row)
	}
	return reader.Header(), rows, nil
}

// scenario: 管理者はエクスポートしたユーザー CSV を安全に再適用できる
func TestParseUserCSVAcceptsPermutedPartialHeadersAndPreservesPresence(t *testing.T) {
	schema, err := NewUserCSVSchema([]UserAttributeDef{{
		Key: "cost_code", Type: idmdomain.AttributeTypeString, Visibility: idmdomain.AttrVisibilityPrivate,
	}})
	if err != nil {
		t.Fatal(err)
	}

	header, rows, csvErr := collectUserCSV(t, strings.NewReader("custom:cost_code,email,id\nEngineering,,user-1\n"), schema, idmdomain.DefaultCSVTransferPolicy())
	if csvErr != nil {
		t.Fatalf("ParseUserCSV() error = %+v", csvErr)
	}
	if got, want := header, []string{"custom:cost_code", "email", "id"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("header = %v, want %v", got, want)
	}
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
	row := rows[0]
	if cell, ok := row.Cell("email"); !ok || !cell.Present || cell.Raw != "" {
		t.Fatalf("email cell = %+v, present=%v; want present empty", cell, ok)
	}
	if _, ok := row.Cell("preferred_username"); ok {
		t.Fatal("preferred_username must be absent when its header is absent")
	}
	if id, code := UserCSVIdentifierOf(row); code != "" || id.ID != "user-1" || id.PreferredUsername != "" {
		t.Fatalf("identifier = %+v, code=%q", id, code)
	}
}

// scenario: 管理者は組み込み拡張属性 (department など) を CSV エクスポート/インポートの
// 対象に含められる (models.DataExportColumn, wi-352)。builtin defs は attr:<key> で
// tenant custom 属性の custom:<key> と区別する。
func TestParseUserCSVResolvesBuiltinAttributeAsAttrPrefixedColumn(t *testing.T) {
	defs := append(BuiltinUserAttributeDefs(), UserAttributeDef{
		Key: "cost_code", Type: idmdomain.AttributeTypeString, Visibility: idmdomain.AttrVisibilityPrivate,
	})
	schema, err := NewUserCSVSchema(defs)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := schema.Column("attr:department"); !ok {
		t.Fatal("expected builtin attribute department to resolve as attr:department")
	}
	if _, ok := schema.Column("custom:department"); ok {
		t.Fatal("builtin attribute department must not also resolve as custom:department")
	}
	if _, ok := schema.Column("custom:cost_code"); !ok {
		t.Fatal("expected tenant custom attribute cost_code to resolve as custom:cost_code")
	}

	header, rows, csvErr := collectUserCSV(t, strings.NewReader("id,attr:department\nuser-1,Engineering\n"), schema, idmdomain.DefaultCSVTransferPolicy())
	if csvErr != nil {
		t.Fatalf("ParseUserCSV() error = %+v", csvErr)
	}
	if got, want := header, []string{"id", "attr:department"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("header = %v, want %v", got, want)
	}
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
	if cell, ok := rows[0].Cell("attr:department"); !ok || cell.Raw != "Engineering" {
		t.Fatalf("attr:department cell = %+v", cell)
	}
}

func TestParseUserCSVRejectsUnknownDuplicateAndSecretHeaders(t *testing.T) {
	schema, err := NewUserCSVSchema(nil)
	if err != nil {
		t.Fatal(err)
	}
	for name, input := range map[string]string{
		"unknown":   "id,unknown\nuser-1,x\n",
		"duplicate": "id,email,email\nuser-1,a@example.com,b@example.com\n",
		"password":  "preferred_username,password\nalice,secret\n",
		"hash":      "preferred_username,password_hash\nalice,hash\n",
	} {
		t.Run(name, func(t *testing.T) {
			_, _, got := collectUserCSV(t, strings.NewReader(input), schema, idmdomain.DefaultCSVTransferPolicy())
			var csvErr *idmdomain.CSVError
			if !errors.As(got, &csvErr) || csvErr.Code != idmdomain.CSVErrorInvalidHeader || csvErr.Row != 1 {
				t.Fatalf("error = %+v, want row-1 invalid_header", got)
			}
		})
	}
}

func TestParseUserCSVAttributeCellUsesCanonicalTypes(t *testing.T) {
	dateDef := UserAttributeDef{Key: "hire_date", Type: idmdomain.AttributeTypeDate}
	numberDef := UserAttributeDef{Key: "score", Type: idmdomain.AttributeTypeNumber}
	booleanDef := UserAttributeDef{Key: "enabled", Type: idmdomain.AttributeTypeBoolean}
	arrayDef := UserAttributeDef{Key: "regions", Type: idmdomain.AttributeTypeStringArray, MultiValued: true}

	for _, tc := range []struct {
		name string
		raw  string
		def  UserAttributeDef
		ok   bool
	}{
		{name: "date", raw: "2026-08-10", def: dateDef, ok: true},
		{name: "bad date", raw: "2026-8-10", def: dateDef},
		{name: "number", raw: "12.5", def: numberDef, ok: true},
		{name: "noncanonical number", raw: "12.50", def: numberDef},
		{name: "boolean", raw: "true", def: booleanDef, ok: true},
		{name: "noncanonical boolean", raw: "TRUE", def: booleanDef},
		{name: "array", raw: `["apac","emea"]`, def: arrayDef, ok: true},
		{name: "bad array", raw: `["apac",1]`, def: arrayDef},
	} {
		t.Run(tc.name, func(t *testing.T) {
			value, shouldClear, err := ParseUserCSVAttributeCell(tc.raw, tc.def)
			if tc.ok {
				if err != nil || shouldClear {
					t.Fatalf("value=%+v clear=%v error=%v", value, shouldClear, err)
				}
				if err := ValidateAttributeValue(value, tc.def); err != nil {
					t.Fatalf("ValidateAttributeValue() = %v", err)
				}
				return
			}
			if !errors.Is(err, ErrInvalidUserCSVCell) {
				t.Fatalf("error=%v, want ErrInvalidUserCSVCell", err)
			}
		})
	}

	if _, shouldClear, err := ParseUserCSVAttributeCell("", UserAttributeDef{Key: "department", Type: idmdomain.AttributeTypeString}); err != nil || !shouldClear {
		t.Fatalf("optional empty string: clear=%v error=%v, want clear", shouldClear, err)
	}
	if _, _, err := ParseUserCSVAttributeCell("", UserAttributeDef{Key: "department", Type: idmdomain.AttributeTypeString, Required: true}); !errors.Is(err, ErrInvalidUserCSVCell) {
		t.Fatalf("required empty error=%v, want ErrInvalidUserCSVCell", err)
	}
}

func TestValidateUserCSVTargetsRejectsDuplicateIdentifiersAndFinalUsernames(t *testing.T) {
	schema, err := NewUserCSVSchema(nil)
	if err != nil {
		t.Fatal(err)
	}
	_, rows, csvErr := collectUserCSV(t, strings.NewReader("id,preferred_username\nuser-1,alice\nuser-1,alice-2\nuser-2,alice\n,\n"), schema, idmdomain.DefaultCSVTransferPolicy())
	if csvErr != nil {
		t.Fatal(csvErr)
	}
	errs := ValidateUserCSVTargets(rows)
	want := []idmdomain.CSVErrorCode{"duplicate_target", "duplicate_username", "missing_identifier"}
	if len(errs) != len(want) {
		t.Fatalf("errors=%+v, want codes %v", errs, want)
	}
	for i := range want {
		if errs[i].Code != want[i] {
			t.Fatalf("errors[%d]=%+v, want code %q", i, errs[i], want[i])
		}
	}
}

func TestUserCSVStreamsTenThousandRowsWithinDefaultPolicy(t *testing.T) {
	policy := idmdomain.DefaultCSVTransferPolicy()
	var out strings.Builder
	writer, err := idmdomain.NewCSVWriter(&out, []string{"id", "preferred_username", "email"}, policy)
	if err != nil {
		t.Fatal(err)
	}
	for i := range 10_000 {
		if err := writer.WriteRow([]string{fmt.Sprintf("user-%d", i), fmt.Sprintf("user-%d", i), fmt.Sprintf("user-%d@example.com", i)}); err != nil {
			t.Fatalf("row %d: %v", i, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if out.Len() >= policy.MaxBytes {
		t.Fatalf("fixture bytes=%d exceeds policy=%d", out.Len(), policy.MaxBytes)
	}
	schema, _ := NewUserCSVSchema(nil)
	reader, err := idmdomain.NewCSVReader(strings.NewReader(out.String()), schema.Accepts, policy)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for {
		record, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil || record.Error != nil {
			t.Fatalf("row %d: record=%+v error=%v", count+2, record, err)
		}
		count++
	}
	if count != 10_000 {
		t.Fatalf("rows=%d, want 10000", count)
	}
}

func TestCSVTransferPolicyEnforcesInjectedBoundaries(t *testing.T) {
	schema, _ := NewUserCSVSchema(nil)
	t.Run("rows", func(t *testing.T) {
		policy := idmdomain.DefaultCSVTransferPolicy()
		policy.MaxRows = 2
		_, _, err := collectUserCSV(t, strings.NewReader("id\na\nb\nc\n"), schema, policy)
		var csvErr *idmdomain.CSVError
		if !errors.As(err, &csvErr) || csvErr.Code != idmdomain.CSVErrorTooManyRows {
			t.Fatalf("error=%v", err)
		}
	})
	t.Run("bytes", func(t *testing.T) {
		input := "id\nuser-1\n"
		policy := idmdomain.DefaultCSVTransferPolicy()
		policy.MaxBytes = len(input) - 1
		_, _, err := collectUserCSV(t, strings.NewReader(input), schema, policy)
		var csvErr *idmdomain.CSVError
		if !errors.As(err, &csvErr) || csvErr.Code != idmdomain.CSVErrorCSVTooLarge {
			t.Fatalf("error=%v", err)
		}
	})
	t.Run("field", func(t *testing.T) {
		policy := idmdomain.DefaultCSVTransferPolicy()
		policy.MaxFieldBytes = 20
		_, _, err := collectUserCSV(t, strings.NewReader("id,name\nuser-1,123456789012345678901\n"), schema, policy)
		var csvErr *idmdomain.CSVError
		if !errors.As(err, &csvErr) || csvErr.Code != idmdomain.CSVErrorFieldTooLarge {
			t.Fatalf("error=%v", err)
		}
	})
}

func TestUserImportPlanCountsActions(t *testing.T) {
	plan := UserImportPlan{Rows: []UserImportRowPlan{
		{Row: 2, Action: UserImportCreate},
		{Row: 3, Action: UserImportUpdate},
		{Row: 4, Action: UserImportUnchanged},
		{Row: 5, Action: UserImportRejected, Error: &idmdomain.CSVError{Code: "invalid_email"}},
	}}
	if plan.TotalRows() != 4 || plan.CreatedRows() != 1 || plan.UpdatedRows() != 1 || plan.UnchangedRows() != 1 || plan.RejectedRows() != 1 {
		t.Fatalf("unexpected counts: %+v", plan)
	}
}
