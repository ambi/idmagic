package usecases

import (
	"context"
	"errors"

	"github.com/ambi/idmagic/backend/shared/logging"
	"github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/ambi/idmagic/backend/tenancy/ports"
)

// RecountUserUsage は、count が数えた使用量を占める User の件数で User の使用量を置き換え、
// 置き換える前の使用量とずれていれば警告ログへ残す (REQ-TENANCY-049)。
func RecountUserUsage(ctx context.Context, repository ports.QuotaRepository, tenantID string, count func(context.Context) (int, error)) error {
	return recountUsage(ctx, repository, tenantID, domain.ResourceUsers, "user usage recounted", count)
}

// RecountActiveSessionUsage は、count が数えた有効なセッションの件数でセッションの使用量を
// 置き換え、置き換える前の使用量とずれていれば警告ログへ残す (REQ-TENANCY-051)。
func RecountActiveSessionUsage(ctx context.Context, repository ports.QuotaRepository, tenantID string, count func(context.Context) (int, error)) error {
	return recountUsage(ctx, repository, tenantID, domain.ResourceActiveSessions, "active session usage recounted", count)
}

func recountUsage(
	ctx context.Context,
	repository ports.QuotaRepository,
	tenantID, resource, driftMessage string,
	count func(context.Context) (int, error),
) error {
	if repository == nil || count == nil {
		return errors.New("usage recount: repository and count are required")
	}
	var counted int
	previous, err := repository.RecountUsage(ctx, tenantID, resource, func(ctx context.Context) (int, error) {
		n, err := count(ctx)
		counted = n
		return n, err
	})
	if err != nil {
		return err
	}
	if previous != counted {
		logging.Warn(ctx, driftMessage, "tenant_id", tenantID, "previous_usage", previous, "counted", counted)
	}
	return nil
}
