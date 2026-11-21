package domain

import (
	"errors"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	ErrAccessReviewInvalid  = errors.New("invalid access review request")
	ErrAccessReviewConflict = errors.New("access review conflict")
	ErrAccessReviewDenied   = errors.New("access review denied")
)

type AccessReviewPlanInput struct {
	Name               string    `json:"name"`
	TargetKind         string    `json:"target_kind"`
	ResourceIDs        []string  `json:"resource_ids"`
	ReviewerID         string    `json:"reviewer_id,omitempty"`
	FallbackReviewerID string    `json:"fallback_reviewer_id"`
	DefaultDecision    string    `json:"default_decision"`
	StartsAt           time.Time `json:"starts_at"`
	DurationDays       int       `json:"duration_days"`
	RecurrenceDays     int       `json:"recurrence_days"`
	Enabled            bool      `json:"enabled"`
}

func (p *AccessReviewPlanInput) NormalizeAndValidate() error {
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" || utf8.RuneCountInString(p.Name) > 200 || !slices.Contains([]string{"group_membership", "application_assignment", "agent"}, p.TargetKind) || len(p.ResourceIDs) == 0 || len(p.ResourceIDs) > 50 || p.FallbackReviewerID == "" || (p.TargetKind != "agent" && p.ReviewerID == "") || !validReviewDecision(p.DefaultDecision) || p.StartsAt.IsZero() || p.DurationDays < 1 || p.DurationDays > 90 || p.RecurrenceDays < 0 || p.RecurrenceDays > 365 || (p.RecurrenceDays != 0 && p.RecurrenceDays < p.DurationDays) {
		return ErrAccessReviewInvalid
	}
	seen := map[string]bool{}
	for _, id := range p.ResourceIDs {
		if id == "" || seen[id] {
			return ErrAccessReviewInvalid
		}
		seen[id] = true
	}
	p.ResourceIDs = slices.Clone(p.ResourceIDs)
	p.StartsAt = p.StartsAt.UTC()
	return nil
}

type AccessReviewPlan struct {
	AccessReviewPlanInput
	ID          string     `json:"id"`
	TenantID    string     `json:"-"`
	Revision    int64      `json:"revision"`
	NextStartAt *time.Time `json:"next_start_at,omitempty"`
	LastError   string     `json:"last_error"`
}

type AccessReviewItem struct {
	ID           string     `json:"id"`
	ResourceID   string     `json:"resource_id"`
	ResourceName string     `json:"resource_name"`
	SubjectType  string     `json:"subject_type"`
	SubjectID    string     `json:"subject_id"`
	SubjectName  string     `json:"subject_name"`
	ReviewerID   string     `json:"reviewer_id"`
	Decision     string     `json:"decision"`
	DecidedBy    string     `json:"decided_by"`
	DecidedAt    *time.Time `json:"decided_at,omitempty"`
	Defaulted    bool       `json:"defaulted"`
	Outcome      string     `json:"outcome"`
	ErrorCode    string     `json:"error_code"`
}

type AccessReview struct {
	ID                 string             `json:"id"`
	TenantID           string             `json:"-"`
	PlanID             string             `json:"plan_id"`
	Name               string             `json:"name"`
	TargetKind         string             `json:"target_kind"`
	DefaultDecision    string             `json:"default_decision"`
	Status             string             `json:"status"`
	StartedAt          time.Time          `json:"started_at"`
	DueAt              time.Time          `json:"due_at"`
	NotificationFailed bool               `json:"notification_failed"`
	Items              []AccessReviewItem `json:"items"`
	// 通知の成否は回と送信段階ごとに保存し、成功済みの宛先へ再送しない。
	Notified            map[string]bool                  `json:"-"`
	FailedNotifications map[string]bool                  `json:"-"`
	WorkLeases          map[string]AccessReviewWorkLease `json:"-"`
}

// AccessReviewWorkLease は外向きの作用を占有し、停止後に再開できる期限を持つ。
type AccessReviewWorkLease struct {
	Token string    `json:"token"`
	Until time.Time `json:"until"`
}

func validReviewDecision(decision string) bool { return decision == "retain" || decision == "revoke" }

func (r *AccessReview) Decide(itemID, reviewerID, decision string, now time.Time) (bool, error) {
	if !validReviewDecision(decision) {
		return false, ErrAccessReviewInvalid
	}
	for i := range r.Items {
		item := &r.Items[i]
		if item.ID != itemID {
			continue
		}
		if item.ReviewerID != reviewerID {
			return false, ErrAccessReviewDenied
		}
		if r.Status != "open" || !now.Before(r.DueAt) {
			return false, ErrAccessReviewConflict
		}
		if item.Decision == decision {
			return false, nil
		}
		if item.Decision != "" {
			return false, ErrAccessReviewConflict
		}
		item.Decision, item.DecidedBy, item.DecidedAt = decision, reviewerID, &now
		return true, nil
	}
	return false, ErrAccessReviewInvalid
}

func (r *AccessReview) Close(now time.Time) bool {
	if r.Status != "open" || now.Before(r.DueAt) {
		return false
	}
	r.Status = "applying"
	for i := range r.Items {
		if r.Items[i].Decision == "" {
			r.Items[i].Decision, r.Items[i].Defaulted = r.DefaultDecision, true
		}
	}
	return true
}
