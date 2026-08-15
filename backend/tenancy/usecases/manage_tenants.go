package usecases

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/ambi/idmagic/backend/shared/notification/template"
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
	// ErrUnsupportedDefaultLocale はカタログが同梱翻訳を持たない locale の指定。
	// 通知が空の本文で届くより先に、保存時点で拒否する。
	ErrUnsupportedDefaultLocale = errors.New("tenant default locale has no bundled translation")
)

// UpdateInput はテナント設定の部分更新を表す。nil のフィールドは現状維持。
// PasswordPolicyOverride にゼロ値を渡すと override を解除する (global default 継承)。
type UpdateInput struct {
	DisplayName            *string
	PasswordPolicyOverride *domain.PasswordPolicyOverride
	// MaxDelegationDepth は Token Exchange の act チェーン深さ上限の上書き。
	// nil は現状維持、0 は上書きの解除 (システム既定を継承)、正の値は上書き。
	// システム既定を超える値は ErrPolicyOverrideWeaker で拒否する。
	MaxDelegationDepth *int
	// TrustedDeviceMaxAgeSeconds は信頼済みデバイスの有効期間。nil は現状維持、
	// 0 は機能無効 (上書きの解除ではない)、正の値は有効化。この設定だけは既定が
	// 最も厳しい状態なので緩める方向の値を保存でき、上限超過だけを拒否する。
	TrustedDeviceMaxAgeSeconds *int
	// DefaultLocale は通知の locale 解決の第 2 段。nil は現状維持、
	// 空文字列はシステム既定へ戻す、それ以外は同梱翻訳を持つ locale のみ受け付ける。
	DefaultLocale *string
}

// PolicyFloor holds the product-wide values a password_policy_override must not
// fall below: the lowest MinLength, the highest MaxLength, and the lowest
// HistoryDepth accepted.
type PolicyFloor struct {
	MinLength    int
	MaxLength    int
	HistoryDepth int
}

// System bounds for expiry (max_age_days). Unlike length and history, stricter
// is not safer here: an extremely short rotation period only pushes users toward
// predictable patterns, and amounts to a tenant inflicting a denial of service on
// itself, while an extremely long one makes the setting meaningless. Both
// directions are rejected against a fixed range rather than the baseline values
// (REQ-TENANCY-019).
const (
	PasswordMaxAgeDaysFloor   = 30
	PasswordMaxAgeDaysCeiling = 3650
)

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
// (id) はサーバが採番する。realm の重複は ErrTenantConflict。
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
		EndpointStyle: domain.TenantEndpointStylePath,
		CreatedAt:     normalizeNow(now), UpdatedAt: normalizeNow(now),
	}
	// 新規 realm には DNS ラベル規則と予約語を適用する。Tenant.Validate は
	// 既存 realm を落とさないよう緩いままなので、作成経路で別に検査する。
	if err := domain.ValidateNewRealm(tenant.Realm); err != nil {
		return nil, ErrInvalidTenantID
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
		if normalized != nil {
			if err := enforcePolicyFloor(*normalized, floor); err != nil {
				return nil, err
			}
		}
		updated.PasswordPolicyOverride = normalized
		// Expiry is measured from this, so only an update that touches the policy
		// moves it (REQ-AUTHENTICATION-024). Moving it on a display_name update
		// would extend the grace window without end.
		policyChangedAt := normalizeNow(now)
		updated.PasswordPolicyUpdatedAt = &policyChangedAt
	}
	if input.MaxDelegationDepth != nil {
		depth := *input.MaxDelegationDepth
		switch {
		case depth == 0:
			// 上書きの解除。システム既定へ戻す。
			updated.MaxDelegationDepth = nil
		case depth < 1 || depth > domain.DefaultMaxDelegationDepth:
			// 上書きは厳しい方向にのみ働く。システム既定を超える値は認可の境界を
			// 緩めるので拒否する。1 未満は「すべての交換を拒否」という、
			// 設定として表現するつもりのない状態になるので同じく拒否する。
			return nil, ErrPolicyOverrideWeaker
		default:
			updated.MaxDelegationDepth = &depth
		}
	}
	if input.TrustedDeviceMaxAgeSeconds != nil {
		maxAge := *input.TrustedDeviceMaxAgeSeconds
		switch {
		case maxAge == 0:
			// 機能無効へ戻す。MaxDelegationDepth の 0 と違い「上書きの解除」ではなく、
			// この設定にとって最も厳しい状態そのものである。
			updated.TrustedDeviceMaxAgeSeconds = nil
		case maxAge < 0 || maxAge > domain.TrustedDeviceMaxAgeCeilingSeconds:
			return nil, ErrPolicyOverrideWeaker
		default:
			updated.TrustedDeviceMaxAgeSeconds = &maxAge
		}
	}
	if input.DefaultLocale != nil {
		requested := strings.TrimSpace(*input.DefaultLocale)
		switch {
		case requested == "":
			updated.DefaultLocale = nil
		case template.LocaleSupported(requested):
			updated.DefaultLocale = &requested
		default:
			return nil, ErrUnsupportedDefaultLocale
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
	if o.MaxAgeDays != nil && *o.MaxAgeDays > 0 {
		v := *o.MaxAgeDays
		result.MaxAgeDays = &v
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
	if o.MaxAgeDays != nil &&
		(*o.MaxAgeDays < PasswordMaxAgeDaysFloor || *o.MaxAgeDays > PasswordMaxAgeDaysCeiling) {
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

// SetEndpointStyle はテナントの正規ロケーションを切り替える破壊的操作。
// 通常の Update に混ぜないことで、呼び出し側が issuer / RP ID / cookie scope の変更を
// 明示的に扱うことを強制する。
func SetEndpointStyle(
	ctx context.Context,
	repo tenantports.TenantRepository,
	id string,
	style domain.TenantEndpointStyle,
	baseDomain string,
	now time.Time,
) (*domain.Tenant, error) {
	if err := domain.ValidateEndpointStyleSelectable(style, baseDomain); err != nil {
		return nil, err
	}
	tenant, err := repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if tenant == nil {
		return nil, ErrTenantNotFound
	}
	updated := *tenant
	updated.EndpointStyle = style
	updated.UpdatedAt = normalizeNow(now)
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
