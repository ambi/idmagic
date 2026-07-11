package usecases

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/ambi/idmagic/backend/shared/spec"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

var (
	ErrTenantNotFound       = errors.New("tenant not found")
	ErrTenantConflict       = errors.New("tenant already exists")
	ErrInvalidTenantID      = errors.New("invalid tenant id")
	ErrDefaultTenant        = errors.New("default tenant cannot be disabled")
	ErrDisplayNameEmpty     = errors.New("display name is required")
	ErrPolicyOverrideWeaker = errors.New("password policy override is weaker than the global default")
)

// UpdateInput はテナント設定の部分更新を表す。nil のフィールドは現状維持。
// PasswordPolicyOverride にゼロ値を渡すと override を解除する (global default 継承)。
type UpdateInput struct {
	DisplayName            *string
	PasswordPolicyOverride *domain.PasswordPolicyOverride
}

// PolicyFloor は password_policy_override が下回ってはならない global 値。
// MinLength の最低値、MaxLength の上限値、HistoryDepth の最低値で gating する (WI-17)。
type PolicyFloor struct {
	MinLength    int
	MaxLength    int
	HistoryDepth int
}

func EnsureDefault(ctx context.Context, repo tenantports.TenantRepository, now time.Time) error {
	tenant, err := repo.FindByID(ctx, domain.DefaultTenantID)
	if err != nil {
		return err
	}
	if tenant != nil {
		return nil
	}
	now = normalizeNow(now)
	return repo.Save(ctx, &domain.Tenant{
		ID: domain.DefaultTenantID, Realm: domain.DefaultRealm, DisplayName: "Default",
		Status: domain.TenantStatusActive, CreatedAt: now, UpdatedAt: now,
	})
}

// Create は admin が指定した realm (URL slug) で新規テナントを作成する。不変 UUID キー
// (id) はサーバが採番する (ADR-085)。realm の重複は ErrTenantConflict。
func Create(ctx context.Context, repo tenantports.TenantRepository, realm, displayName string, now time.Time) (*domain.Tenant, error) {
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		return nil, ErrDisplayNameEmpty
	}
	id, err := spec.NewUUIDv4()
	if err != nil {
		return nil, err
	}
	tenant := &domain.Tenant{
		ID: id, Realm: strings.TrimSpace(realm), DisplayName: displayName, Status: domain.TenantStatusActive,
		CreatedAt: normalizeNow(now), UpdatedAt: normalizeNow(now),
	}
	if err := tenant.Validate(); err != nil {
		return nil, ErrInvalidTenantID
	}
	existing, err := repo.FindByRealm(ctx, tenant.Realm)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrTenantConflict
	}
	if err := repo.Save(ctx, tenant); err != nil {
		return nil, err
	}
	return tenant, nil
}

func Update(
	ctx context.Context,
	repo tenantports.TenantRepository,
	id string,
	input UpdateInput,
	floor PolicyFloor,
	now time.Time,
) (*domain.Tenant, error) {
	tenant, err := repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if tenant == nil {
		return nil, ErrTenantNotFound
	}
	updated := *tenant
	if input.DisplayName != nil {
		name := strings.TrimSpace(*input.DisplayName)
		if name == "" {
			return nil, ErrDisplayNameEmpty
		}
		updated.DisplayName = name
	}
	if input.PasswordPolicyOverride != nil {
		normalized := normalizeOverride(*input.PasswordPolicyOverride)
		if normalized == nil {
			updated.PasswordPolicyOverride = nil
		} else {
			if err := enforcePolicyFloor(*normalized, floor); err != nil {
				return nil, err
			}
			updated.PasswordPolicyOverride = normalized
		}
	}
	t := normalizeNow(now)
	updated.UpdatedAt = t
	if err := repo.Save(ctx, &updated); err != nil {
		return nil, err
	}
	return &updated, nil
}

func normalizeOverride(o domain.PasswordPolicyOverride) *domain.PasswordPolicyOverride {
	result := domain.PasswordPolicyOverride{}
	anyOverride := false
	if o.MinLength != nil && *o.MinLength > 0 {
		v := *o.MinLength
		result.MinLength = &v
		anyOverride = true
	}
	if o.MaxLength != nil && *o.MaxLength > 0 {
		v := *o.MaxLength
		result.MaxLength = &v
		anyOverride = true
	}
	if o.HistoryDepth != nil && *o.HistoryDepth > 0 {
		v := *o.HistoryDepth
		result.HistoryDepth = &v
		anyOverride = true
	}
	if !anyOverride {
		return nil
	}
	return &result
}

func enforcePolicyFloor(o domain.PasswordPolicyOverride, floor PolicyFloor) error {
	if o.MinLength != nil && floor.MinLength > 0 && *o.MinLength < floor.MinLength {
		return ErrPolicyOverrideWeaker
	}
	if o.MaxLength != nil && floor.MaxLength > 0 && *o.MaxLength > floor.MaxLength {
		return ErrPolicyOverrideWeaker
	}
	if o.HistoryDepth != nil && floor.HistoryDepth > 0 && *o.HistoryDepth < floor.HistoryDepth {
		return ErrPolicyOverrideWeaker
	}
	return nil
}

func SetDisabled(ctx context.Context, repo tenantports.TenantRepository, id string, disabled bool, now time.Time) (*domain.Tenant, error) {
	if id == domain.DefaultTenantID && disabled {
		return nil, ErrDefaultTenant
	}
	tenant, err := repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if tenant == nil {
		return nil, ErrTenantNotFound
	}
	updated := *tenant
	t := normalizeNow(now)
	updated.UpdatedAt = t
	if disabled {
		updated.Status = domain.TenantStatusDisabled
		updated.DisabledAt = &t
	} else {
		updated.Status = domain.TenantStatusActive
		updated.DisabledAt = nil
	}
	if err := repo.Save(ctx, &updated); err != nil {
		return nil, err
	}
	return &updated, nil
}

func normalizeNow(now time.Time) time.Time {
	if now.IsZero() {
		return time.Now().UTC()
	}
	return now.UTC()
}
