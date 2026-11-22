package db_memory

import (
	"context"
	"maps"
	"slices"
	"strings"
	"sync"
	"time"

	igdomain "github.com/ambi/idmagic/backend/idgovernance/internal/domain"
)

type AccessReviewRepository struct {
	mu      sync.Mutex
	plans   map[string]*igdomain.AccessReviewPlan
	reviews map[string]*igdomain.AccessReview
}

func NewAccessReviewRepository() *AccessReviewRepository {
	return &AccessReviewRepository{plans: map[string]*igdomain.AccessReviewPlan{}, reviews: map[string]*igdomain.AccessReview{}}
}

func cloneReviewPlan(p *igdomain.AccessReviewPlan) *igdomain.AccessReviewPlan {
	if p == nil {
		return nil
	}
	c := *p
	c.ResourceIDs = slices.Clone(p.ResourceIDs)
	if p.NextStartAt != nil {
		at := *p.NextStartAt
		c.NextStartAt = &at
	}
	return &c
}

func cloneReview(r *igdomain.AccessReview) *igdomain.AccessReview {
	if r == nil {
		return nil
	}
	c := *r
	c.Items = slices.Clone(r.Items)
	for i := range c.Items {
		if at := c.Items[i].DecidedAt; at != nil {
			value := *at
			c.Items[i].DecidedAt = &value
		}
	}
	c.Notified = maps.Clone(r.Notified)
	c.FailedNotifications = maps.Clone(r.FailedNotifications)
	c.WorkLeases = maps.Clone(r.WorkLeases)
	return &c
}

func (r *AccessReviewRepository) CreatePlan(_ context.Context, p *igdomain.AccessReviewPlan) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.plans[p.ID] != nil {
		return igdomain.ErrAccessReviewConflict
	}
	r.plans[p.ID] = cloneReviewPlan(p)
	return nil
}

func (r *AccessReviewRepository) ListPlans(_ context.Context, tenant string) ([]*igdomain.AccessReviewPlan, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []*igdomain.AccessReviewPlan{}
	for _, p := range r.plans {
		if p.TenantID == tenant {
			out = append(out, cloneReviewPlan(p))
		}
	}
	slices.SortFunc(out, func(a, b *igdomain.AccessReviewPlan) int { return strings.Compare(a.ID, b.ID) })
	return out, nil
}

func (r *AccessReviewRepository) ReadyPlans(_ context.Context, now time.Time) ([]*igdomain.AccessReviewPlan, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []*igdomain.AccessReviewPlan{}
	for _, p := range r.plans {
		if p.Enabled && p.NextStartAt != nil && !p.NextStartAt.After(now) {
			out = append(out, cloneReviewPlan(p))
		}
	}
	return out, nil
}

func (r *AccessReviewRepository) FindPlan(_ context.Context, tenant, id string) (*igdomain.AccessReviewPlan, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p := r.plans[id]
	if p == nil || p.TenantID != tenant {
		return nil, nil
	}
	return cloneReviewPlan(p), nil
}

func (r *AccessReviewRepository) UpdatePlan(_ context.Context, tenant, id string, apply func(*igdomain.AccessReviewPlan) error) (*igdomain.AccessReviewPlan, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p := r.plans[id]
	if p == nil || p.TenantID != tenant {
		return nil, igdomain.ErrAccessReviewInvalid
	}
	c := cloneReviewPlan(p)
	if err := apply(c); err != nil {
		return nil, err
	}
	r.plans[id] = cloneReviewPlan(c)
	return c, nil
}

func (r *AccessReviewRepository) CaptureReview(_ context.Context, p *igdomain.AccessReviewPlan, review *igdomain.AccessReview, next *time.Time) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	current := r.plans[p.ID]
	if current == nil || current.TenantID != p.TenantID || current.Revision != p.Revision || !current.Enabled || current.NextStartAt == nil || p.NextStartAt == nil || !current.NextStartAt.Equal(*p.NextStartAt) {
		return false, nil
	}
	if next != nil {
		value := *next
		next = &value
	}
	current.NextStartAt, current.LastError = next, ""
	r.reviews[review.ID] = cloneReview(review)
	return true, nil
}

func (r *AccessReviewRepository) ListByPlan(_ context.Context, tenant, planID string) ([]*igdomain.AccessReview, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []*igdomain.AccessReview{}
	for _, v := range r.reviews {
		if v.TenantID == tenant && v.PlanID == planID {
			out = append(out, cloneReview(v))
		}
	}
	slices.SortFunc(out, func(a, b *igdomain.AccessReview) int {
		if n := b.StartedAt.Compare(a.StartedAt); n != 0 {
			return n
		}
		return strings.Compare(a.ID, b.ID)
	})
	return out, nil
}

func (r *AccessReviewRepository) ListForReviewer(_ context.Context, tenant, userID string) ([]*igdomain.AccessReview, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []*igdomain.AccessReview{}
	for _, v := range r.reviews {
		if v.TenantID != tenant {
			continue
		}
		c := cloneReview(v)
		c.Items = slices.DeleteFunc(c.Items, func(i igdomain.AccessReviewItem) bool { return i.ReviewerID != userID })
		if len(c.Items) > 0 {
			out = append(out, c)
		}
	}
	slices.SortFunc(out, func(a, b *igdomain.AccessReview) int { return strings.Compare(a.ID, b.ID) })
	return out, nil
}

func (r *AccessReviewRepository) ActiveReviews(_ context.Context) ([]*igdomain.AccessReview, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []*igdomain.AccessReview{}
	for _, v := range r.reviews {
		if v.Status != "completed" {
			out = append(out, cloneReview(v))
		}
	}
	return out, nil
}

func (r *AccessReviewRepository) UpdateReview(_ context.Context, tenant, id string, apply func(*igdomain.AccessReview) error) (*igdomain.AccessReview, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v := r.reviews[id]
	if v == nil || v.TenantID != tenant {
		return nil, igdomain.ErrAccessReviewInvalid
	}
	c := cloneReview(v)
	if err := apply(c); err != nil {
		return nil, err
	}
	r.reviews[id] = cloneReview(c)
	return c, nil
}
