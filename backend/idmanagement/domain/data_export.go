package domain

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
)

// DataExportTargetKind is the resource type a CSV export targets. Column
// definitions are closed to a per-kind allowlist so unreviewed attributes never
// leak (spec/contexts/identity-management.yaml models.DataExportTargetKind).
type DataExportTargetKind string

const (
	ExportTargetUser            DataExportTargetKind = "user"
	ExportTargetGroup           DataExportTargetKind = "group"
	ExportTargetGroupMembership DataExportTargetKind = "group_membership"
)

func (k DataExportTargetKind) Valid() bool {
	_, ok := ColumnsForTarget(k)
	return ok
}

// DataExportStatus mirrors the DataExportLifecycle states
// (spec/contexts/identity-management.yaml states.DataExportLifecycle).
type DataExportStatus string

const (
	ExportStatusQueued    DataExportStatus = "queued"
	ExportStatusRunning   DataExportStatus = "running"
	ExportStatusSucceeded DataExportStatus = "succeeded"
	ExportStatusFailed    DataExportStatus = "failed"
	ExportStatusCanceled  DataExportStatus = "canceled"
	ExportStatusExpired   DataExportStatus = "expired"
)

// DataExportColumn is one allowlisted column for an export target. PII marks
// columns whose values are personal data (email/name), which callers select
// explicitly and whose request/download are audited. Sensitive values
// (password_hash, secrets, tokens, recovery/MFA secrets) are never present in
// any allowlist.
type DataExportColumn struct {
	Key   string
	Label string
	PII   bool
}

// exportColumns is the authoritative per-target column allowlist. Adding a new
// exportable attribute is an explicit edit here, reviewed against 's
// sensitive-value exclusion.
var exportColumns = map[DataExportTargetKind][]DataExportColumn{
	ExportTargetUser: {
		{Key: "id", Label: "ID"},
		{Key: "preferred_username", Label: "Preferred username"},
		{Key: "email", Label: "Email", PII: true},
		{Key: "name", Label: "Name", PII: true},
		{Key: "given_name", Label: "Given name", PII: true},
		{Key: "family_name", Label: "Family name", PII: true},
		{Key: "email_verified", Label: "Email verified"},
		{Key: "mfa_enrolled", Label: "MFA enrolled"},
		{Key: "status", Label: "Status"},
		{Key: "roles", Label: "Roles"},
		{Key: "required_actions", Label: "Required actions"},
		{Key: "created_at", Label: "Created at"},
		{Key: "updated_at", Label: "Updated at"},
	},
	ExportTargetGroup: {
		{Key: "id", Label: "ID"},
		{Key: "name", Label: "Name"},
		{Key: "description", Label: "Description"},
		{Key: "membership_type", Label: "Membership type"},
		{Key: "roles", Label: "Roles"},
		{Key: "created_at", Label: "Created at"},
		{Key: "updated_at", Label: "Updated at"},
	},
	ExportTargetGroupMembership: {
		{Key: "group_id", Label: "Group ID"},
		{Key: "group_name", Label: "Group name"},
		{Key: "user_id", Label: "User ID"},
		{Key: "preferred_username", Label: "Preferred username"},
		{Key: "source", Label: "Source"},
		{Key: "created_at", Label: "Created at"},
	},
}

// ColumnsForTarget returns the allowlist for kind, or (nil, false) if kind is
// unknown.
func ColumnsForTarget(kind DataExportTargetKind) ([]DataExportColumn, bool) {
	cols, ok := exportColumns[kind]
	return cols, ok
}

var (
	// ErrInvalidExportTarget is returned when the export target kind is unknown.
	ErrInvalidExportTarget = errors.New("data export: invalid target")
	// ErrInvalidExportColumns is returned when requested columns are empty,
	// duplicated, or not a subset of the target's allowlist.
	ErrInvalidExportColumns = errors.New("data export: invalid columns")
)

// ValidateExportColumns checks that keys is a non-empty, duplicate-free subset of
// kind's allowlist. This is the fail-closed gate that keeps sensitive/unreviewed
// attributes out of exports.
func ValidateExportColumns(kind DataExportTargetKind, keys []string) error {
	allowed, ok := ColumnsForTarget(kind)
	if !ok {
		return fmt.Errorf("%w: %q", ErrInvalidExportTarget, kind)
	}
	if len(keys) == 0 {
		return fmt.Errorf("%w: at least one column is required", ErrInvalidExportColumns)
	}
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, c := range allowed {
		allowedSet[c.Key] = struct{}{}
	}
	seen := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		if _, ok := allowedSet[k]; !ok {
			return fmt.Errorf("%w: %q is not allowlisted for target %q", ErrInvalidExportColumns, k, kind)
		}
		if _, dup := seen[k]; dup {
			return fmt.Errorf("%w: %q is duplicated", ErrInvalidExportColumns, k)
		}
		seen[k] = struct{}{}
	}
	return nil
}

// LabelsForColumns returns the display labels for keys (assumed already
// validated against kind).
func LabelsForColumns(kind DataExportTargetKind, keys []string) []string {
	labels := make([]string, 0, len(keys))
	byKey := map[string]string{}
	if cols, ok := ColumnsForTarget(kind); ok {
		for _, c := range cols {
			byKey[c.Key] = c.Label
		}
	}
	for _, k := range keys {
		if label, ok := byKey[k]; ok {
			labels = append(labels, label)
		} else {
			labels = append(labels, k)
		}
	}
	return labels
}

// EscapeCSVField neutralizes CSV formula injection: spreadsheet software
// (Excel / Sheets) interprets a cell as a formula when it begins with '=', '+',
// '-', '@', or a leading TAB / CR / LF. Such values are prefixed with a single
// quote so they render as text. Safety is prioritized over display fidelity.
// RFC 4180 quoting itself is applied separately by EncodeCSVRecords.
func EscapeCSVField(s string) string {
	if s == "" {
		return s
	}
	switch s[0] {
	case '=', '+', '-', '@', '\t', '\r', '\n':
		return "'" + s
	}
	return s
}

// EncodeCSVRecords writes header followed by rows as an RFC 4180 CSV, applying
// EscapeCSVField to every cell (header included) first. The Go encoding/csv
// Writer supplies RFC 4180 quoting for commas, quotes, and embedded newlines.
func EncodeCSVRecords(header []string, rows [][]string) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	if err := w.Write(escapeRecord(header)); err != nil {
		return nil, err
	}
	for _, row := range rows {
		if err := w.Write(escapeRecord(row)); err != nil {
			return nil, err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func escapeRecord(record []string) []string {
	out := make([]string, len(record))
	for i, cell := range record {
		out[i] = EscapeCSVField(cell)
	}
	return out
}
