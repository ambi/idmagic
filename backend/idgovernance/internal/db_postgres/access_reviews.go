package db_postgres

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"time"

	igdomain "github.com/ambi/idmagic/backend/idgovernance/internal/domain"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type AccessReviewRepository struct{ Pool sharedpg.DB }

type accessReviewRecord struct {
	Review              *igdomain.AccessReview                    `json:"review"`
	Notified            map[string]bool                           `json:"notified"`
	FailedNotifications map[string]bool                           `json:"failed_notifications"`
	WorkLeases          map[string]igdomain.AccessReviewWorkLease `json:"work_leases"`
}

func decodeReviewPlan(tenant string, data []byte) (*igdomain.AccessReviewPlan, error) {
	var p igdomain.AccessReviewPlan
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	p.TenantID = tenant
	return &p, nil
}

func decodeReview(tenant string, data []byte) (*igdomain.AccessReview, error) {
	var record accessReviewRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, err
	}
	if record.Review == nil {
		return nil, igdomain.ErrAccessReviewInvalid
	}
	record.Review.TenantID = tenant
	record.Review.Notified = record.Notified
	record.Review.FailedNotifications = record.FailedNotifications
	record.Review.WorkLeases = record.WorkLeases
	return record.Review, nil
}

func reviewNext(at *time.Time) pgtype.Timestamptz {
	if at == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *at, Valid: true}
}

func (r *AccessReviewRepository) CreatePlan(ctx context.Context, p *igdomain.AccessReviewPlan) error {
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}
	return New(r.Pool).CreateAccessReviewPlan(ctx, CreateAccessReviewPlanParams{ID: p.ID, TenantID: p.TenantID, Enabled: p.Enabled, NextStartAt: reviewNext(p.NextStartAt), Data: data})
}

func (r *AccessReviewRepository) ListPlans(ctx context.Context, tenant string) ([]*igdomain.AccessReviewPlan, error) {
	rows, err := New(r.Pool).ListAccessReviewPlans(ctx, tenant)
	if err != nil {
		return nil, err
	}
	out := make([]*igdomain.AccessReviewPlan, 0, len(rows))
	for _, row := range rows {
		p, err := decodeReviewPlan(row.TenantID, row.Data)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

func (r *AccessReviewRepository) ReadyPlans(ctx context.Context, now time.Time) ([]*igdomain.AccessReviewPlan, error) {
	rows, err := New(r.Pool).ReadyAccessReviewPlans(ctx, pgtype.Timestamptz{Time: now, Valid: true})
	if err != nil {
		return nil, err
	}
	out := make([]*igdomain.AccessReviewPlan, 0, len(rows))
	for _, row := range rows {
		p, err := decodeReviewPlan(row.TenantID, row.Data)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

func (r *AccessReviewRepository) FindPlan(ctx context.Context, tenant, id string) (*igdomain.AccessReviewPlan, error) {
	if !sharedpg.IsUUIDText(id) {
		return nil, nil
	}
	row, err := New(r.Pool).FindAccessReviewPlan(ctx, FindAccessReviewPlanParams{TenantID: tenant, ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return decodeReviewPlan(row.TenantID, row.Data)
}

func saveReviewPlan(ctx context.Context, q *Queries, p *igdomain.AccessReviewPlan) error {
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}
	return q.SaveAccessReviewPlan(ctx, SaveAccessReviewPlanParams{TenantID: p.TenantID, ID: p.ID, Enabled: p.Enabled, NextStartAt: reviewNext(p.NextStartAt), Data: data})
}

func (r *AccessReviewRepository) UpdatePlan(ctx context.Context, tenant, id string, apply func(*igdomain.AccessReviewPlan) error) (*igdomain.AccessReviewPlan, error) {
	if !sharedpg.IsUUIDText(id) {
		return nil, igdomain.ErrAccessReviewInvalid
	}
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := New(tx)
	row, err := q.LockAccessReviewPlan(ctx, LockAccessReviewPlanParams{TenantID: tenant, ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, igdomain.ErrAccessReviewInvalid
	}
	if err != nil {
		return nil, err
	}
	p, err := decodeReviewPlan(row.TenantID, row.Data)
	if err != nil {
		return nil, err
	}
	if err := apply(p); err != nil {
		return nil, err
	}
	if err := saveReviewPlan(ctx, q, p); err != nil {
		return nil, err
	}
	return p, tx.Commit(ctx)
}

func (r *AccessReviewRepository) CaptureReview(ctx context.Context, p *igdomain.AccessReviewPlan, review *igdomain.AccessReview, next *time.Time) (bool, error) {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := New(tx)
	row, err := q.LockAccessReviewPlan(ctx, LockAccessReviewPlanParams{TenantID: p.TenantID, ID: p.ID})
	if err != nil {
		return false, err
	}
	current, err := decodeReviewPlan(row.TenantID, row.Data)
	if err != nil {
		return false, err
	}
	if current.Revision != p.Revision || !current.Enabled || current.NextStartAt == nil || p.NextStartAt == nil || !current.NextStartAt.Equal(*p.NextStartAt) {
		return false, nil
	}
	data, err := json.Marshal(accessReviewRecord{Review: review, Notified: review.Notified, FailedNotifications: review.FailedNotifications, WorkLeases: review.WorkLeases})
	if err != nil {
		return false, err
	}
	if err := q.InsertAccessReview(ctx, InsertAccessReviewParams{ID: review.ID, TenantID: review.TenantID, PlanID: review.PlanID, Status: review.Status, StartedAt: review.StartedAt, Data: data}); err != nil {
		return false, err
	}
	current.NextStartAt, current.LastError = next, ""
	if err := saveReviewPlan(ctx, q, current); err != nil {
		return false, err
	}
	return true, tx.Commit(ctx)
}

func (r *AccessReviewRepository) ListByPlan(ctx context.Context, tenant, planID string) ([]*igdomain.AccessReview, error) {
	if !sharedpg.IsUUIDText(planID) {
		return []*igdomain.AccessReview{}, nil
	}
	rows, err := New(r.Pool).ListAccessReviews(ctx, ListAccessReviewsParams{TenantID: tenant, PlanID: planID})
	if err != nil {
		return nil, err
	}
	out := make([]*igdomain.AccessReview, 0, len(rows))
	for _, row := range rows {
		v, err := decodeReview(row.TenantID, row.Data)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

func (r *AccessReviewRepository) ListForReviewer(ctx context.Context, tenant, userID string) ([]*igdomain.AccessReview, error) {
	rows, err := New(r.Pool).ListReviewerAccessReviews(ctx, ListReviewerAccessReviewsParams{TenantID: tenant, Column2: userID})
	if err != nil {
		return nil, err
	}
	out := make([]*igdomain.AccessReview, 0, len(rows))
	for _, row := range rows {
		v, err := decodeReview(row.TenantID, row.Data)
		if err != nil {
			return nil, err
		}
		v.Items = slices.DeleteFunc(v.Items, func(i igdomain.AccessReviewItem) bool { return i.ReviewerID != userID })
		out = append(out, v)
	}
	return out, nil
}

func (r *AccessReviewRepository) ActiveReviews(ctx context.Context) ([]*igdomain.AccessReview, error) {
	rows, err := New(r.Pool).ListActiveAccessReviews(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*igdomain.AccessReview, 0, len(rows))
	for _, row := range rows {
		v, err := decodeReview(row.TenantID, row.Data)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

func (r *AccessReviewRepository) UpdateReview(ctx context.Context, tenant, id string, apply func(*igdomain.AccessReview) error) (*igdomain.AccessReview, error) {
	if !sharedpg.IsUUIDText(id) {
		return nil, igdomain.ErrAccessReviewInvalid
	}
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := New(tx)
	row, err := q.LockAccessReview(ctx, LockAccessReviewParams{TenantID: tenant, ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, igdomain.ErrAccessReviewInvalid
	}
	if err != nil {
		return nil, err
	}
	v, err := decodeReview(row.TenantID, row.Data)
	if err != nil {
		return nil, err
	}
	if err := apply(v); err != nil {
		return nil, err
	}
	data, err := json.Marshal(accessReviewRecord{Review: v, Notified: v.Notified, FailedNotifications: v.FailedNotifications, WorkLeases: v.WorkLeases})
	if err != nil {
		return nil, err
	}
	if err := q.SaveAccessReview(ctx, SaveAccessReviewParams{TenantID: tenant, ID: id, Status: v.Status, Data: data}); err != nil {
		return nil, err
	}
	return v, tx.Commit(ctx)
}
