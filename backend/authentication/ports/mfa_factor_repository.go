package ports

import (
	"context"

	"github.com/ambi/idmagic/backend/authentication/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

type MfaFactorRepository interface {
	ListBySub(ctx context.Context, sub string) ([]*domain.MfaFactor, error)
	Find(ctx context.Context, sub string, factorType spec.MfaFactorType) (*domain.MfaFactor, error)
	Save(ctx context.Context, factor *domain.MfaFactor) error
	Delete(ctx context.Context, sub string, factorType spec.MfaFactorType) error
	// DeleteAllForSub は ADR-036 の anonymize cascade から呼ばれる。
	// 対象 sub の MFA factor をすべて物理削除する。
	DeleteAllForSub(ctx context.Context, sub string) error
}
