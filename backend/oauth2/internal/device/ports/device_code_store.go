package ports

import (
	"context"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
)

type DeviceCodeStore interface {
	Save(ctx context.Context, rec *oauthdomain.DeviceAuthorization) error
	FindByDeviceCodeHash(ctx context.Context, hash string) (*oauthdomain.DeviceAuthorization, error)
	FindByUserCode(ctx context.Context, userCode string) (*oauthdomain.DeviceAuthorization, error)
	Update(ctx context.Context, rec *oauthdomain.DeviceAuthorization) error
	Exchange(ctx context.Context, deviceCodeHash string) (*oauthdomain.DeviceAuthorization, error)
	// DeleteAllForSub は anonymize cascade から呼ばれる。
	// 対象 sub に既に紐付いた DeviceAuthorization を物理削除する。
	DeleteAllForSub(ctx context.Context, sub string) error
}
