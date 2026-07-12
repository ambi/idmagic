package usecases

import (
	"context"
	"crypto/rand"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/identitymanagement/domain"
	"github.com/ambi/idmagic/backend/jobs/domain"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

const (
	UserImportMaxBytes      = 1 << 20
	UserImportMaxRows       = 1000
	UserImportMaxFieldBytes = 64 << 10
)

type (
	UserImportRowError struct {
		Row    int    `json:"row"`
		Column string `json:"column"`
		Code   string `json:"code"`
	}
	UserImportResult struct {
		TotalRows    int                  `json:"total_rows"`
		AcceptedRows int                  `json:"accepted_rows"`
		RejectedRows int                  `json:"rejected_rows"`
		Errors       []UserImportRowError `json:"errors,omitempty"`
	}
	UserImportParams struct {
		CSV         string `json:"csv"`
		ActorUserID string `json:"actor_user_id"`
	}
	importRow struct {
		row                   int
		username, email, name string
		roles                 []string
	}
)

func ParseUserImportCSV(input string) ([]importRow, UserImportResult) {
	result := UserImportResult{}
	if len(input) > UserImportMaxBytes {
		result.Errors = append(result.Errors, UserImportRowError{Code: "csv_too_large"})
		result.RejectedRows = 1
		return nil, result
	}
	input = strings.TrimPrefix(input, "\ufeff")
	r := csv.NewReader(strings.NewReader(input))
	r.FieldsPerRecord = -1
	header, err := r.Read()
	if err != nil {
		result.Errors = []UserImportRowError{{Row: 1, Code: "invalid_csv"}}
		result.RejectedRows = 1
		return nil, result
	}
	want := []string{"preferred_username", "email", "name", "roles"}
	if len(header) != len(want) {
		result.Errors = []UserImportRowError{{Row: 1, Code: "invalid_header"}}
		result.RejectedRows = 1
		return nil, result
	}
	for i := range want {
		if header[i] != want[i] {
			result.Errors = []UserImportRowError{{Row: 1, Column: header[i], Code: "invalid_header"}}
			result.RejectedRows = 1
			return nil, result
		}
	}
	var rows []importRow
	seen := map[string]bool{}
	for line := 2; ; line++ {
		rec, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			result.Errors = append(result.Errors, UserImportRowError{Row: line, Code: "invalid_csv"})
			result.RejectedRows++
			continue
		}
		result.TotalRows++
		if result.TotalRows > UserImportMaxRows {
			result.Errors = append(result.Errors, UserImportRowError{Row: line, Code: "too_many_rows"})
			result.RejectedRows++
			break
		}
		bad := ""
		col := ""
		for i, v := range rec {
			if len(v) > UserImportMaxFieldBytes {
				bad = "field_too_large"
				col = want[i]
				break
			}
		}
		if len(rec) != 4 {
			bad = "invalid_column_count"
		}
		username := ""
		if len(rec) > 0 {
			username = strings.TrimSpace(rec[0])
		}
		if bad == "" && username == "" {
			bad = "required"
			col = "preferred_username"
		}
		if bad == "" && seen[username] {
			bad = "duplicate_username"
			col = "preferred_username"
		}
		seen[username] = true
		email := ""
		if len(rec) > 1 {
			email = strings.TrimSpace(rec[1])
		}
		if bad == "" && email != "" && (!strings.Contains(email, "@") || strings.HasPrefix(email, "@") || strings.HasSuffix(email, "@")) {
			bad = "invalid_email"
			col = "email"
		}
		if bad != "" {
			result.Errors = append(result.Errors, UserImportRowError{Row: line, Column: col, Code: bad})
			result.RejectedRows++
			continue
		}
		roles := []string{}
		if len(rec) > 3 && strings.TrimSpace(rec[3]) != "" {
			roles = strings.Split(rec[3], "|")
		}
		rows = append(rows, importRow{line, username, email, strings.TrimSpace(rec[2]), roles})
		result.AcceptedRows++
	}
	return rows, result
}

func UserImportHandler(deps AdminUserDeps, apply bool) func(context.Context, *domain.Job) (json.RawMessage, error) {
	return func(ctx context.Context, job *domain.Job) (json.RawMessage, error) {
		var p UserImportParams
		if err := json.Unmarshal(job.Params, &p); err != nil {
			return nil, err
		}
		rows, result := ParseUserImportCSV(p.CSV)
		if apply {
			ctx = tenancy.WithTenant(ctx, &tenancydomain.Tenant{ID: job.TenantID}, "", "")
			for _, row := range rows {
				password, err := randomImportPassword()
				if err != nil {
					return nil, err
				}
				var email, name *string
				if row.email != "" {
					email = &row.email
				}
				if row.name != "" {
					name = &row.name
				}
				user, err := CreateUser(ctx, deps, CreateUserInput{ActorUserID: p.ActorUserID, PreferredUsername: row.username, Password: password, Email: email, Name: name, Roles: row.roles, Now: time.Now().UTC()})
				if err != nil {
					result.AcceptedRows--
					result.RejectedRows++
					result.Errors = append(result.Errors, UserImportRowError{Row: row.row, Code: importErrorCode(err)})
					continue
				}
				_, _ = SetUserRequiredAction(ctx, deps, p.ActorUserID, user.ID, idmdomain.RequiredActionUpdatePassword, time.Now().UTC())
			}
		}
		return json.Marshal(result)
	}
}

func randomImportPassword() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("A1!%x", b), nil
}

func importErrorCode(err error) string {
	if errors.Is(err, ErrUsernameConflict) {
		return "username_conflict"
	}
	return "invalid_user"
}
