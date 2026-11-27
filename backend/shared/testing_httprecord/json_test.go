package testing_httprecord

import (
	"strings"
	"testing"
)

func TestCompareJSON(t *testing.T) {
	tests := []struct {
		name     string
		recorded string
		got      string
		want     string // 空なら食い違いがないこと
	}{
		{"欄の順序と空白は問わない", `{"a":1,"b":"x"}`, `{ "b": "x", "a": 1 }`, ""},
		{"値の違い", `{"a":1}`, `{"a":2}`, "body = "},
		{"欄の欠落", `{"a":1,"salt_length":"hash"}`, `{"a":1}`, "body = "},
		{"記録にない欄", `{"a":1}`, `{"a":1,"b":2}`, "body = "},
		{"本文のない記録に本文がある", ``, `{"a":1}`, "recorded none"},
		{"記録した本文がない", `{"a":1}`, ``, "body is empty"},
		{"両方とも本文がない", ``, ``, ""},
		{"JSON でない本文", `{"a":1}`, `a=1`, "not JSON"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			diffs := strings.Join(CompareJSON([]byte(tc.recorded), []byte(tc.got)), "\n")
			if tc.want == "" && diffs != "" {
				t.Fatalf("CompareJSON() = %q, want no difference", diffs)
			}
			if tc.want != "" && !strings.Contains(diffs, tc.want) {
				t.Fatalf("CompareJSON() = %q, want a line containing %q", diffs, tc.want)
			}
		})
	}
}
