package ports

import (
	"context"
	"time"

	approvaldomain "github.com/ambi/idmagic/backend/oauth2/approval/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// ApprovalRequestStore persists approval decisions and indexes auth_req_id by digest only.
type ApprovalRequestStore interface {
	Save(ctx context.Context, rec *approvaldomain.ApprovalRequest) error
	FindByID(ctx context.Context, id string) (*approvaldomain.ApprovalRequest, error)
	FindByAuthReqIDHash(ctx context.Context, hash string) (*approvaldomain.ApprovalRequest, error)
	// ListPendingForUser returns undecided, unexpired requests for the subject.
	ListPendingForUser(ctx context.Context, userID string) ([]*approvaldomain.ApprovalRequest, error)
	// RecordPoll atomically updates transport polling bookkeeping without changing the decision state.
	RecordPoll(ctx context.Context, authReqIDHash string, now time.Time) (*approvaldomain.ApprovalRequest, bool, error)
	// Expire atomically moves a pending or approved request to Expired.
	Expire(ctx context.Context, authReqIDHash string, now time.Time) (*approvaldomain.ApprovalRequest, error)
	// Decide atomically changes a still-pending, unexpired request owned by userID.
	// A nil result means the compare-and-set condition no longer holds.
	Decide(ctx context.Context, id, userID string, event spec.ApprovalRequestEvent, now time.Time) (*approvaldomain.ApprovalRequest, error)
	// Consume advances only an approved request. Exactly one concurrent caller receives a result.
	Consume(ctx context.Context, authReqIDHash string, now time.Time) (*approvaldomain.ApprovalRequest, error)
	// DeleteAllForSub participates in the user-anonymization cascade.
	DeleteAllForSub(ctx context.Context, sub string) error
}
