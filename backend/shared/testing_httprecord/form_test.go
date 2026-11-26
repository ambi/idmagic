package testing_httprecord

import (
	"strings"
	"testing"
)

// 再生は記録した要求とフォームの値が食い違う要求を報告し、記録にない欄は許す。
func TestCompareForm(t *testing.T) {
	recorded := []byte("grant_type=refresh_token&refresh_token=tGzv3JOkF0XG5Qx2TlKWIA")
	if diffs := CompareForm(recorded, []byte("refresh_token=tGzv3JOkF0XG5Qx2TlKWIA&grant_type=refresh_token&scope=x")); len(diffs) != 0 {
		t.Errorf("CompareForm() = %q for a matching form", diffs)
	}
	diffs := CompareForm(recorded, []byte("grant_type=authorization_code"))
	want := []string{
		`grant_type = ["authorization_code"], recorded ["refresh_token"]`,
		`refresh_token = [], recorded ["tGzv3JOkF0XG5Qx2TlKWIA"]`,
	}
	if strings.Join(diffs, "\n") != strings.Join(want, "\n") {
		t.Errorf("CompareForm() = %q, want %q", diffs, want)
	}
}
