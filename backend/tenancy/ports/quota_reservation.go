package ports

import (
	"context"

	"github.com/ambi/idmagic/backend/shared/logging"
)

// QuotaReservation は、リソースを作成するために Hard Quota の使用量へ足した加算分である。
// 作成が確定するまでは仮の加算であり、Keep を呼ばずに Release すると加算を戻す
// （REQ-TENANCY-050）。作成する側は ReserveQuota の直後に Release を defer し、
// リソースの保存が成功した時点で Keep を呼ぶ。こうすると、保存より前のどの経路で
// 戻っても加算は残らず、保存の後の失敗ではリソースとともに加算が残る。
type QuotaReservation struct {
	repo     QuotaRepository
	tenantID string
	resource string
	delta    int
	kept     bool
}

// ReserveQuota は CheckAndIncrement で加算し、その予約を返す。上限の超過を含む加算の失敗では
// 予約を返さない。repo が nil の構成（上限を強制しないテストや道具）では何もしない予約を返す。
func ReserveQuota(ctx context.Context, repo QuotaRepository, tenantID, resource string, delta int) (*QuotaReservation, error) {
	if repo != nil {
		if err := repo.CheckAndIncrement(ctx, tenantID, resource, delta); err != nil {
			return nil, err
		}
	}
	return &QuotaReservation{repo: repo, tenantID: tenantID, resource: resource, delta: delta}, nil
}

// Keep は、予約したリソースが保存され、加算を使用量に残すことを記録する。
func (r *QuotaReservation) Keep() {
	if r != nil {
		r.kept = true
	}
}

// Release は、Keep の前なら加算を戻す。何度呼んでも一度だけ戻す。減算の失敗はログに残し、
// 作成の結果のエラーを置き換えない。
func (r *QuotaReservation) Release(ctx context.Context) {
	if r == nil || r.kept || r.repo == nil {
		return
	}
	r.kept = true
	if err := r.repo.Decrement(ctx, r.tenantID, r.resource, r.delta); err != nil {
		logging.Error(ctx, "quota: failed to release the reservation of a failed creation",
			"error", err, "tenant_id", r.tenantID, "resource", r.resource)
	}
}
