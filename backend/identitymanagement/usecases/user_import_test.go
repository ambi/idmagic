package usecases

import "testing"

func TestParseUserImportCSV(t *testing.T) {
	rows, result := ParseUserImportCSV("preferred_username,email,name,roles\nalice,alice@example.com,Alice,admin|support\nalice,bad,Again,\n")
	if len(rows) != 1 || result.AcceptedRows != 1 || result.RejectedRows != 1 {
		t.Fatalf("rows=%d result=%+v", len(rows), result)
	}
	if result.Errors[0].Code != "duplicate_username" {
		t.Fatalf("error=%+v", result.Errors[0])
	}
}

func TestParseUserImportCSVRejectsPasswordHeader(t *testing.T) {
	_, result := ParseUserImportCSV("preferred_username,email,name,password\nalice,a@example.com,Alice,secret\n")
	if len(result.Errors) != 1 || result.Errors[0].Code != "invalid_header" {
		t.Fatalf("result=%+v", result)
	}
}
