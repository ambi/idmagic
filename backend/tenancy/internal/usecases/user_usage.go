package usecases

import (
	"context"
	"errors"

	"github.com/ambi/idmagic/backend/shared/logging"
	"github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/ambi/idmagic/backend/tenancy/ports"
)

// RecountUserUsage は、count が数えた使用量を占める User の件数で User の使用量を置き換え、
// 置き換える前の使用量とずれていれば警告ログへ残す。
func RecountUserUsage(ctx context.Context, repository ports.QuotaRepository, tenantID string, count func(context.Context) (int, error)) error {
	if repository == nil || count == nil {
		return errors.New("user usage recount: repository and count are required")
	}
	var counted int
	previous, err := repository.RecountUsage(ctx, tenantID, domain.ResourceUsers, func(ctx context.Context) (int, error) {
		n, err := count(ctx)
		counted = n
		return n, err
	})
	if err != nil {
		return err
	}
	if previous != counted {
		logging.Warn(ctx, "user usage recounted", "tenant_id", tenantID, "previous_usage", previous, "counted", counted)
	}
	return nil
}
