// Package usecases holds the small set of generic admin usecase helpers
// shared across the user/group/agent feature usecase packages (wi-254). It
// carries no feature-specific domain knowledge; feature-specific usecases
// live under user/usecases, group/usecases, agent/usecases.
package usecases

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/ambi/idmagic/backend/shared/logging"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
	tenancyusecases "github.com/ambi/idmagic/backend/tenancy/usecases"
)

// ErrInvalidRole は role が空文字列のみの場合に返る。
var ErrInvalidRole = errors.New("role must not be empty")

// ErrUserNotFound は対象 User が存在しない場合に返る (feature 横断で参照される)。
var ErrUserNotFound = errors.New("user not found")

// NormalizeRoles は role スライスをトリム・重複排除・ソートして返す。
func NormalizeRoles(roles []string) ([]string, error) {
	out := make([]string, 0, len(roles))
	for _, role := range roles {
		role = strings.TrimSpace(role)
		if role == "" {
			return nil, ErrInvalidRole
		}
		if !slices.Contains(out, role) {
			out = append(out, role)
		}
	}
	slices.Sort(out)
	return out, nil
}

// NormalizedNow は now が zero-value の場合に time.Now().UTC() を返す。
func NormalizedNow(now time.Time) time.Time {
	if now.IsZero() {
		return time.Now().UTC()
	}
	return now.UTC()
}

// EqualOptionalString は 2 つの *string が (両方 nil) または (両方非 nil で同値) かを返す。
func EqualOptionalString(left, right *string) bool {
	return left == nil && right == nil ||
		left != nil && right != nil && *left == *right
}

// NormalizeDescription は description をトリムし、空文字列なら nil を返す。
func NormalizeDescription(description *string) *string {
	if description == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*description)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

// AdminEmit は sink が nil でなければ event を emit する。
func AdminEmit(sink func(spec.DomainEvent) error, event spec.DomainEvent) error {
	if sink == nil {
		return nil
	}
	return sink(event)
}

// CheckQuotaAndAudit enforces resource's tenant Hard Quota (ADR-134, wi-160)
// via quotaRepo.CheckAndIncrement, shared by the user/group/agent usecase
// packages that all emit through the same error-returning sink signature. A
// rejection also emits QuotaExceeded through sink (SCL objective QuotaAudit);
// the emit is best-effort (logged, not propagated) so an audit-sink hiccup
// never masks the fail-closed rejection itself. nil quotaRepo skips
// enforcement (wiring gaps in tests/tools not yet updated); production
// bootstrap always sets it.
func CheckQuotaAndAudit(
	ctx context.Context, quotaRepo tenantports.QuotaRepository, sink func(spec.DomainEvent) error,
	tenantID, resource string, now time.Time,
) error {
	if quotaRepo == nil {
		return nil
	}
	err := tenancyusecases.CheckQuotaAndIncrement(ctx, quotaRepo, tenantID, resource, 1)
	if qErr, ok := errors.AsType[*tenancydomain.QuotaExceededError](err); ok {
		if emitErr := AdminEmit(sink, &tenancydomain.QuotaExceeded{
			At: now, TenantID: tenantID, Resource: qErr.Resource, HardLimit: true,
		}); emitErr != nil {
			logging.Error(ctx, "quota: failed to emit QuotaExceeded audit event", "error", emitErr, "resource", qErr.Resource)
		}
	}
	return err
}
