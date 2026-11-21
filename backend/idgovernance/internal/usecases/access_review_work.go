package usecases

import (
	"context"
	"errors"
	"time"

	d "github.com/ambi/idmagic/backend/idgovernance/internal/domain"
	notification "github.com/ambi/idmagic/backend/shared/notification/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// 保存のロックを外して作用を行う。プロセスが停止しても、占有の期限後に再実行できる。
const accessReviewWorkLease = 2 * time.Minute

func claimReviewWork(ctx context.Context, deps AccessReviewDeps, review *d.AccessReview, key string, now time.Time, eligible func(*d.AccessReview, time.Time) bool) (string, error) {
	token, err := spec.NewUUIDv4()
	if err != nil {
		return "", err
	}
	claimed := false
	_, err = deps.Reviews.UpdateReview(ctx, review.TenantID, review.ID, func(current *d.AccessReview) error {
		at := reviewNow(deps, now)
		if !eligible(current, at) {
			return nil
		}
		if lease := current.WorkLeases[key]; lease.Until.After(at) {
			return nil
		}
		if current.WorkLeases == nil {
			current.WorkLeases = map[string]d.AccessReviewWorkLease{}
		}
		current.WorkLeases[key] = d.AccessReviewWorkLease{Token: token, Until: at.Add(accessReviewWorkLease)}
		claimed = true
		return nil
	})
	if err != nil || !claimed {
		return "", err
	}
	return token, nil
}

func advanceAccessReview(ctx context.Context, deps AccessReviewDeps, review *d.AccessReview, now time.Time) error {
	now = reviewNow(deps, now)
	current, err := deps.Reviews.UpdateReview(ctx, review.TenantID, review.ID, func(r *d.AccessReview) error { r.Close(now); return nil })
	if err != nil {
		return err
	}
	if current.Status == "completed" {
		return nil
	}
	if current.Status == "open" {
		return notifyAccessReview(ctx, deps, current, now)
	}
	var failures []error
	for _, item := range current.Items {
		if item.Outcome != "" {
			continue
		}
		if item.Decision == "retain" {
			_, err := deps.Reviews.UpdateReview(ctx, current.TenantID, current.ID, func(r *d.AccessReview) error {
				for i := range r.Items {
					if r.Items[i].ID == item.ID {
						r.Items[i].Outcome = "retained"
					}
				}
				return nil
			})
			if err != nil {
				failures = append(failures, err)
			}
			continue
		}
		key := "apply:" + item.ID
		token, err := claimReviewWork(ctx, deps, current, key, now, func(r *d.AccessReview, _ time.Time) bool {
			for _, candidate := range r.Items {
				if candidate.ID == item.ID {
					return r.Status == "applying" && candidate.Outcome == ""
				}
			}
			return false
		})
		if err != nil {
			failures = append(failures, err)
			continue
		}
		if token == "" {
			continue
		}
		// 作用の上限は占有より短くし、遅い下流が同じ項目の実行を重ね続けることを抑える。
		workCtx, cancel := context.WithTimeout(ctx, accessReviewWorkLease/2)
		applyErr := revokeReviewItem(workCtx, deps, current, &item, reviewNow(deps, now))
		cancel()
		_, err = deps.Reviews.UpdateReview(ctx, current.TenantID, current.ID, func(r *d.AccessReview) error {
			if r.WorkLeases[key].Token != token {
				return nil
			}
			delete(r.WorkLeases, key)
			for i := range r.Items {
				if r.Items[i].ID == item.ID {
					if applyErr != nil {
						r.Items[i].ErrorCode = "apply_failed"
					} else {
						r.Items[i].Outcome, r.Items[i].ErrorCode = "revoked", ""
					}
				}
			}
			return nil
		})
		if err != nil {
			failures = append(failures, err)
		}
	}
	completed := false
	_, err = deps.Reviews.UpdateReview(ctx, current.TenantID, current.ID, func(r *d.AccessReview) error {
		if r.Status != "applying" {
			return nil
		}
		for _, item := range r.Items {
			if item.Outcome == "" {
				return nil
			}
		}
		r.Status = "completed"
		completed = true
		return nil
	})
	if err != nil {
		failures = append(failures, err)
	} else if completed {
		emitReviewEvent(deps, current.TenantID, "access-review", current.PlanID, current.ID, "", "review_completed", reviewNow(deps, now))
	}
	return errors.Join(failures...)
}

func notifyAccessReview(ctx context.Context, deps AccessReviewDeps, review *d.AccessReview, now time.Time) error {
	seen := map[string]bool{}
	var failures []error
	for _, item := range review.Items {
		if item.Decision != "" || seen[item.ReviewerID] {
			continue
		}
		seen[item.ReviewerID] = true
		phase := "start"
		if !reviewNow(deps, now).Before(review.StartedAt.Add(review.DueAt.Sub(review.StartedAt) / 2)) {
			phase = "reminder"
		}
		key := "notify:" + phase + ":" + item.ReviewerID
		token, err := claimReviewWork(ctx, deps, review, key, now, func(r *d.AccessReview, at time.Time) bool {
			if r.Status != "open" || !at.Before(r.DueAt) || r.Notified[key] {
				return false
			}
			for _, candidate := range r.Items {
				if candidate.ReviewerID == item.ReviewerID && candidate.Decision == "" {
					return true
				}
			}
			return false
		})
		if err != nil {
			failures = append(failures, err)
			continue
		}
		if token == "" {
			continue
		}
		workCtx, cancel := context.WithTimeout(ctx, accessReviewWorkLease/2)
		ok := sendAccessReviewNotification(workCtx, deps, review, item.ReviewerID)
		cancel()
		_, err = deps.Reviews.UpdateReview(ctx, review.TenantID, review.ID, func(r *d.AccessReview) error {
			if r.WorkLeases[key].Token != token {
				return nil
			}
			delete(r.WorkLeases, key)
			if r.Notified == nil {
				r.Notified = map[string]bool{}
			}
			r.Notified[key] = ok
			if r.FailedNotifications == nil {
				r.FailedNotifications = map[string]bool{}
			}
			r.FailedNotifications[key] = !ok
			r.NotificationFailed = false
			for _, failed := range r.FailedNotifications {
				if failed {
					r.NotificationFailed = true
					break
				}
			}
			return nil
		})
		if err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}

func reviewNow(deps AccessReviewDeps, tick time.Time) time.Time {
	if deps.Now != nil {
		if current := deps.Now().UTC(); current.After(tick) {
			return current
		}
	}
	return tick
}

func sendAccessReviewNotification(ctx context.Context, deps AccessReviewDeps, review *d.AccessReview, reviewer string) bool {
	u, err := reviewUser(ctx, deps, review.TenantID, reviewer)
	if err != nil || u == nil || u.Email == nil || deps.Notifier == nil {
		return false
	}
	return deps.Notifier.Notify(ctx, notification.Notification{TenantID: review.TenantID, To: *u.Email, Key: notification.TemplateKeyAccessReviewRequest, RecipientLocale: u.LocaleAttribute(), Vars: map[string]string{"user_display_name": u.DisplayName(), "review_name": review.Name, "due_at": review.DueAt.UTC().Format(time.RFC3339), "default_decision": review.DefaultDecision}})
}
