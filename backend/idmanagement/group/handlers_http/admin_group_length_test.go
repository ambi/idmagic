package handlers_http_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

const groupNameLimit = 100

// 上限はコードポイント数なので、1 文字 3 バイトの日本語でも仕様どおり 100 文字
// 入る。zog の Max はバイト数を数えるため、以前はここが 34 文字で 500 になった。
func TestAdminGroupAPIAcceptsMultibyteNameUpToTheLimit(t *testing.T) {
	for _, tc := range []struct {
		label string
		unit  string
	}{
		{"ASCII", "a"},
		{"日本語", "あ"},
		{"絵文字", "\U0001F642"},
	} {
		t.Run(tc.label, func(t *testing.T) {
			e, _ := newAdminGroupHandler(t)
			csrf, cookie := adminCSRF(t, e)
			name := strings.Repeat(tc.unit, groupNameLimit)
			res := adminJSONRequest(t, e, http.MethodPost, "/api/admin/v1/groups", csrf, cookie,
				map[string]any{"name": name})
			if res.Code != http.StatusCreated {
				t.Fatalf("status=%d body=%s (%d code points, %d bytes)",
					res.Code, res.Body.String(), groupNameLimit, len(name))
			}
		})
	}
}

// 上限超過は解析できた内容の業務規則違反なので 422 を返し、どのフィールドが
// 何文字までかを detail に載せる。以前は素の error が echo まで抜けて 500 だった。
func TestAdminGroupAPIRejectsOverlongNameWith422(t *testing.T) {
	for _, tc := range []struct {
		label string
		unit  string
	}{
		{"ASCII", "a"},
		{"日本語", "あ"},
	} {
		t.Run(tc.label, func(t *testing.T) {
			e, _ := newAdminGroupHandler(t)
			csrf, cookie := adminCSRF(t, e)
			res := adminJSONRequest(t, e, http.MethodPost, "/api/admin/v1/groups", csrf, cookie,
				map[string]any{"name": strings.Repeat(tc.unit, groupNameLimit+1)})
			if res.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
			}
			var problem struct {
				Type   string `json:"type"`
				Status int    `json:"status"`
				Detail string `json:"detail"`
			}
			if err := json.Unmarshal(res.Body.Bytes(), &problem); err != nil {
				t.Fatalf("response is not Problem Details: %v (%s)", err, res.Body.String())
			}
			if !strings.Contains(problem.Detail, "name") ||
				!strings.Contains(problem.Detail, "at most 100 characters") {
				t.Fatalf("detail does not name the field and the limit: %q", problem.Detail)
			}
		})
	}
}

// 説明は 500 コードポイントまで。境界ちょうどと 1 文字超過を両方固定する。
func TestAdminGroupAPIDescriptionBoundary(t *testing.T) {
	const descriptionLimit = 500
	e, _ := newAdminGroupHandler(t)
	csrf, cookie := adminCSRF(t, e)

	atLimit := adminJSONRequest(t, e, http.MethodPost, "/api/admin/v1/groups", csrf, cookie,
		map[string]any{"name": "at-limit", "description": strings.Repeat("あ", descriptionLimit)})
	if atLimit.Code != http.StatusCreated {
		t.Fatalf("at the limit: status=%d body=%s", atLimit.Code, atLimit.Body.String())
	}

	over := adminJSONRequest(t, e, http.MethodPost, "/api/admin/v1/groups", csrf, cookie,
		map[string]any{"name": "over-limit", "description": strings.Repeat("あ", descriptionLimit+1)})
	if over.Code != http.StatusUnprocessableEntity {
		t.Fatalf("one over the limit: status=%d body=%s", over.Code, over.Body.String())
	}
}
