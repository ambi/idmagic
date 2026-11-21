package server_http_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/idgovernance"
	stack "github.com/ambi/idmagic/backend/shared/http/testing_stack"
	"github.com/ambi/idmagic/backend/shared/spec"
)

//spec:covers REQ-IDGOVERNANCE-020, REQ-IDGOVERNANCE-022, REQ-IDGOVERNANCE-025: 正式な管理 API から準備し、本人の回答を保存し、他人と非管理者の変更を拒否する。
//verify:operation CreateAccessReviewPlan: 公開の入力で計画を保存し、201 の設定と非管理者の 403 を観測する。
//verify:operation ListAccessReviewPlans: 作成した計画が管理用の一覧の本文へ返る。
//verify:operation UpdateAccessReviewPlan: 設定の更新は 200、古い revision は 409 で拒否する。
//verify:operation ListAccessReviews: 作成した計画の各回の対象と最終の判断を本文から読む。
//verify:operation ListMyAccessReviews: 他人の回答を拒否した後も本人の項目が未回答のまま返る。
//verify:operation DecideAccessReview: 本人の判断を保存し、他人の 403 と確定した判断の変更の 409 を観測する。
func TestAccessReviewPublicSetupAndReviewerBoundary(t *testing.T) {
	f := newPublicSetupPath(t, stack.WithAccessReviews())
	reviewer := f.user(t, "reviewer")
	other := f.user(t, "other-reviewer")
	group := f.request(t, http.MethodPost, "/api/admin/v1/groups", map[string]any{"name": "Review group"}, f.admin, http.StatusCreated)
	groupID := setupString(t, group, "id")
	f.request(t, http.MethodPost, "/api/admin/v1/groups/"+groupID+"/members/"+other, nil, f.admin, http.StatusNoContent)
	now := time.Now().UTC()
	input := map[string]any{"name": "Access check", "target_kind": "group_membership", "resource_ids": []string{groupID}, "reviewer_id": reviewer, "fallback_reviewer_id": reviewer, "default_decision": "revoke", "starts_at": now.Format(time.RFC3339Nano), "duration_days": 1, "recurrence_days": 0, "enabled": true}
	created := f.request(t, http.MethodPost, "/api/admin/v1/access-review-plans", input, f.admin, http.StatusCreated)
	planID := setupString(t, created, "id")
	list := f.request(t, http.MethodGet, "/api/admin/v1/access-review-plans", nil, f.admin, http.StatusOK)
	if len(list["plans"].([]any)) != 1 {
		t.Fatalf("plans=%v", list)
	}
	input["expected_revision"] = int64(1)
	f.request(t, http.MethodPut, "/api/admin/v1/access-review-plans/"+planID, input, f.admin, http.StatusOK)
	f.request(t, http.MethodPut, "/api/admin/v1/access-review-plans/"+planID, input, f.admin, http.StatusConflict)
	if err := f.s.IdGovernance.AdvanceAccessReviews(context.Background(), idgovernance.AccessReviewDeps{UserRepo: f.s.Users, GroupRepo: f.s.Groups, Emit: func(spec.DomainEvent) {}}, now); err != nil {
		t.Fatal(err)
	}
	results := f.request(t, http.MethodGet, "/api/admin/v1/access-review-plans/"+planID+"/reviews", nil, f.admin, http.StatusOK)
	review := results["reviews"].([]any)[0].(map[string]any)
	item := review["items"].([]any)[0].(map[string]any)
	client, _ := f.keyClient(t, "openid profile idmagic.account", "authorization_code")
	_, reviewerCookie := f.login(t, "reviewer", client, "openid profile idmagic.account")
	_, otherCookie := f.login(t, "other-reviewer", client, "openid profile idmagic.account")
	path := "/api/account/v1/access-reviews/" + setupString(t, review, "id") + "/items/" + setupString(t, item, "id")
	f.request(t, http.MethodPost, path, map[string]any{"decision": "revoke"}, otherCookie, http.StatusForbidden)
	before := f.request(t, http.MethodGet, "/api/account/v1/access-reviews", nil, reviewerCookie, http.StatusOK)
	beforeItem := before["reviews"].([]any)[0].(map[string]any)["items"].([]any)[0].(map[string]any)
	if beforeItem["decision"] != "" {
		t.Fatalf("other reviewer changed decision: %v", before)
	}
	f.request(t, http.MethodPost, path, map[string]any{"decision": "retain"}, reviewerCookie, http.StatusOK)
	f.request(t, http.MethodPost, path, map[string]any{"decision": "revoke"}, reviewerCookie, http.StatusConflict)
	f.request(t, http.MethodPost, "/api/admin/v1/access-review-plans", input, reviewerCookie, http.StatusForbidden)
	results = f.request(t, http.MethodGet, "/api/admin/v1/access-review-plans/"+planID+"/reviews", nil, f.admin, http.StatusOK)
	result := results["reviews"].([]any)[0].(map[string]any)["items"].([]any)[0].(map[string]any)
	if result["decision"] != "retain" || result["decided_by"] != reviewer {
		t.Fatalf("result=%v", result)
	}
}
