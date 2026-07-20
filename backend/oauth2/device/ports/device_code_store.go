package ports

import (
	"context"

	"github.com/ambi/idmagic/backend/oauth2/domain"
)

type DeviceCodeStore interface {
	Save(ctx context.Context, rec *domain.DeviceAuthorization) error
	FindByDeviceCodeHash(ctx context.Context, hash string) (*domain.DeviceAuthorization, error)
	FindByUserCode(ctx context.Context, userCode string) (*domain.DeviceAuthorization, error)
	Update(ctx context.Context, rec *domain.DeviceAuthorization) error
	Exchange(ctx context.Context, deviceCodeHash string) (*domain.DeviceAuthorization, error)
	// DeleteAllForSub は ADR-036 の anonymize cascade から呼ばれる。
	// 対象 sub に既に紐付いた DeviceAuthorization を物理削除する。
	DeleteAllForSub(ctx context.Context, sub string) error
}
