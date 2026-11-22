package ports

import (
	"context"

	totpdomain "github.com/ambi/idmagic/backend/authentication/totp/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

type MfaFactorRepository interface {
	ListBySub(ctx context.Context, sub string) ([]*totpdomain.MfaFactor, error)
	Find(ctx context.Context, sub string, factorType spec.MfaFactorType) (*totpdomain.MfaFactor, error)
	Save(ctx context.Context, factor *totpdomain.MfaFactor) error
	Delete(ctx context.Context, sub string, factorType spec.MfaFactorType) error
	// DeleteAllForSub は anonymize cascade から呼ばれる。
	// 対象 sub の MFA factor をすべて物理削除する。
	DeleteAllForSub(ctx context.Context, sub string) error
}
