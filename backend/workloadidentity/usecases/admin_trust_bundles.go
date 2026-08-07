// 管理者向け WorkloadTrustBundle ライフサイクル操作 (ADR-053)。SCL WorkloadIdentity
// bounded context が所有する admin インターフェース群: ListWorkloadTrustBundles /
// GetWorkloadTrustBundle / RegisterWorkloadTrustBundle / UpdateWorkloadTrustBundle /
// DisableWorkloadTrustBundle / EnableWorkloadTrustBundle / DeleteWorkloadTrustBundle /
// RefreshWorkloadTrustBundleJWKS。
//
// すべての操作は tenancy.TenantID(ctx) のテナント境界に閉じる。issuer / name は
// テナント内で一意 (DB の UNIQUE 制約に加え、backend 非依存の一貫した挙動のため
// usecase 層でも事前チェックする)。
package usecases

import (
	"context"
	"strings"
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
	"github.com/ambi/idmagic/backend/workloadidentity/domain"
	"github.com/ambi/idmagic/backend/workloadidentity/ports"
)

type AdminWorkloadIdentityDeps struct {
	TrustBundleRepo ports.WorkloadTrustBundleRepository
	BindingRepo     ports.AgentWorkloadBindingRepository
	// FetchJWKS performs the live JWKS lookup used by RefreshWorkloadTrustBundleJWKS
	// (test/refresh action). Shares the same signature as
	// VerifyWorkloadAttestationDeps.FetchJWKS so both can be wired to the same
	// adapter at the composition root.
	FetchJWKS func(ctx context.Context, bundle *domain.WorkloadTrustBundle) ([]map[string]any, error)
	Emit      func(spec.DomainEvent)
}

func ListWorkloadTrustBundles(ctx context.Context, deps AdminWorkloadIdentityDeps) ([]*domain.WorkloadTrustBundle, error) {
	return deps.TrustBundleRepo.ListByTenant(ctx, tenancy.TenantID(ctx))
}

// GetWorkloadTrustBundle は別テナントの bundle を未存在として扱う。
func GetWorkloadTrustBundle(ctx context.Context, deps AdminWorkloadIdentityDeps, id string) (*domain.WorkloadTrustBundle, error) {
	bundle, err := deps.TrustBundleRepo.FindByID(ctx, tenancy.TenantID(ctx), id)
	if err != nil {
		return nil, err
	}
	if bundle == nil {
		return nil, ErrTrustBundleNotFound
	}
	return bundle, nil
}

type RegisterWorkloadTrustBundleInput struct {
	Name                      string
	TrustDomain               string
	Issuer                    string
	JWKSURI                   *string
	JWKS                      map[string]any
	AcceptedAudiences         []string
	MaxSubjectTokenTTLSeconds *int
}

func RegisterWorkloadTrustBundle(ctx context.Context, deps AdminWorkloadIdentityDeps, in RegisterWorkloadTrustBundleInput, now time.Time) (*domain.WorkloadTrustBundle, error) {
	tenantID := tenancy.TenantID(ctx)
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, ErrTrustBundleNameRequired
	}
	issuer := strings.TrimSpace(in.Issuer)
	if issuer == "" {
		return nil, ErrTrustBundleIssuerRequired
	}
	if in.JWKSURI == nil && in.JWKS == nil {
		return nil, ErrTrustBundleMissingJWKS
	}
	if len(in.AcceptedAudiences) == 0 {
		return nil, ErrTrustBundleAudiencesEmpty
	}
	if err := ensureTrustBundleNameAndIssuerAvailable(ctx, deps, tenantID, name, issuer, ""); err != nil {
		return nil, err
	}
	ttl := 3600
	if in.MaxSubjectTokenTTLSeconds != nil && *in.MaxSubjectTokenTTLSeconds > 0 {
		ttl = *in.MaxSubjectTokenTTLSeconds
	}
	id, err := domain.NewWorkloadTrustBundleID()
	if err != nil {
		return nil, err
	}
	bundle := &domain.WorkloadTrustBundle{
		ID: id, TenantID: tenantID, Name: name, TrustDomain: strings.TrimSpace(in.TrustDomain),
		Issuer: issuer, JWKSURI: in.JWKSURI, JWKS: in.JWKS, AcceptedAudiences: in.AcceptedAudiences,
		MaxSubjectTokenTTLSeconds: ttl, Status: domain.WorkloadTrustBundleStatusEnabled, CreatedAt: now,
	}
	if err := bundle.Validate(); err != nil {
		return nil, err
	}
	if err := deps.TrustBundleRepo.Save(ctx, bundle); err != nil {
		return nil, err
	}
	emit(deps.Emit, &domain.WorkloadTrustBundleConfigured{At: now, TenantID: tenantID, TrustBundleID: bundle.ID, Issuer: bundle.Issuer})
	return bundle, nil
}

type UpdateWorkloadTrustBundleInput struct {
	Name                      *string
	JWKSURI                   *string
	JWKS                      map[string]any
	AcceptedAudiences         []string
	MaxSubjectTokenTTLSeconds *int
}

// UpdateWorkloadTrustBundle は name / jwks_uri / jwks / accepted_audiences /
// max_subject_token_ttl_seconds を更新する。issuer / trust_domain は不変。
func UpdateWorkloadTrustBundle(ctx context.Context, deps AdminWorkloadIdentityDeps, id string, in UpdateWorkloadTrustBundleInput, now time.Time) (*domain.WorkloadTrustBundle, error) {
	tenantID := tenancy.TenantID(ctx)
	bundle, err := deps.TrustBundleRepo.FindByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if bundle == nil {
		return nil, ErrTrustBundleNotFound
	}
	updated := *bundle
	changed := false
	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		if name == "" {
			return nil, ErrTrustBundleNameRequired
		}
		if name != bundle.Name {
			if err := ensureTrustBundleNameAndIssuerAvailable(ctx, deps, tenantID, name, bundle.Issuer, bundle.ID); err != nil {
				return nil, err
			}
			updated.Name = name
			changed = true
		}
	}
	if in.JWKSURI != nil {
		updated.JWKSURI = in.JWKSURI
		changed = true
	}
	if in.JWKS != nil {
		updated.JWKS = in.JWKS
		changed = true
	}
	if in.AcceptedAudiences != nil {
		if len(in.AcceptedAudiences) == 0 {
			return nil, ErrTrustBundleAudiencesEmpty
		}
		updated.AcceptedAudiences = in.AcceptedAudiences
		changed = true
	}
	if in.MaxSubjectTokenTTLSeconds != nil {
		if *in.MaxSubjectTokenTTLSeconds <= 0 {
			return nil, ErrTrustBundleInvalidTTL
		}
		updated.MaxSubjectTokenTTLSeconds = *in.MaxSubjectTokenTTLSeconds
		changed = true
	}
	if !changed {
		return &updated, nil
	}
	updated.UpdatedAt = &now
	if err := updated.Validate(); err != nil {
		return nil, err
	}
	if err := deps.TrustBundleRepo.Save(ctx, &updated); err != nil {
		return nil, err
	}
	emit(deps.Emit, &domain.WorkloadTrustBundleUpdated{At: now, TenantID: tenantID, TrustBundleID: bundle.ID})
	return &updated, nil
}

func DisableWorkloadTrustBundle(ctx context.Context, deps AdminWorkloadIdentityDeps, id string, now time.Time) (*domain.WorkloadTrustBundle, error) {
	return setTrustBundleStatus(ctx, deps, id, domain.WorkloadTrustBundleStatusDisabled, now)
}

func EnableWorkloadTrustBundle(ctx context.Context, deps AdminWorkloadIdentityDeps, id string, now time.Time) (*domain.WorkloadTrustBundle, error) {
	return setTrustBundleStatus(ctx, deps, id, domain.WorkloadTrustBundleStatusEnabled, now)
}

func setTrustBundleStatus(ctx context.Context, deps AdminWorkloadIdentityDeps, id string, status domain.WorkloadTrustBundleStatus, now time.Time) (*domain.WorkloadTrustBundle, error) {
	tenantID := tenancy.TenantID(ctx)
	bundle, err := deps.TrustBundleRepo.FindByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if bundle == nil {
		return nil, ErrTrustBundleNotFound
	}
	if bundle.Status == status {
		return bundle, nil
	}
	updated := *bundle
	updated.Status = status
	updated.UpdatedAt = &now
	if err := updated.Validate(); err != nil {
		return nil, err
	}
	if err := deps.TrustBundleRepo.Save(ctx, &updated); err != nil {
		return nil, err
	}
	if status == domain.WorkloadTrustBundleStatusDisabled {
		emit(deps.Emit, &domain.WorkloadTrustBundleDisabled{At: now, TenantID: tenantID, TrustBundleID: bundle.ID})
	} else {
		emit(deps.Emit, &domain.WorkloadTrustBundleEnabled{At: now, TenantID: tenantID, TrustBundleID: bundle.ID})
	}
	return &updated, nil
}

// DeleteWorkloadTrustBundle は WorkloadTrustBundle を削除し、配下の
// AgentWorkloadBinding を先に cascade 削除する (DB の ON DELETE CASCADE と併せた
// 二重の保証)。
func DeleteWorkloadTrustBundle(ctx context.Context, deps AdminWorkloadIdentityDeps, id string, now time.Time) error {
	tenantID := tenancy.TenantID(ctx)
	bundle, err := deps.TrustBundleRepo.FindByID(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if bundle == nil {
		return ErrTrustBundleNotFound
	}
	bindings, err := deps.BindingRepo.ListByTrustBundle(ctx, tenantID, id)
	if err != nil {
		return err
	}
	for _, b := range bindings {
		if err := deps.BindingRepo.Delete(ctx, tenantID, b.ID); err != nil {
			return err
		}
	}
	if err := deps.TrustBundleRepo.Delete(ctx, tenantID, id); err != nil {
		return err
	}
	emit(deps.Emit, &domain.WorkloadTrustBundleDeleted{At: now, TenantID: tenantID, TrustBundleID: id})
	return nil
}

type RefreshWorkloadTrustBundleJWKSResult struct {
	Reachable    bool
	KeyCount     int
	JWKSCachedAt *time.Time
}

// RefreshWorkloadTrustBundleJWKS は管理者操作による即時の JWKS 到達性確認 (設定ミスの
// 事前検知)。成功時は jwks_cached_at を更新する。到達不能でもエラーにはせず
// Reachable=false を返す (呼び出し側が結果を提示できるように)。
func RefreshWorkloadTrustBundleJWKS(ctx context.Context, deps AdminWorkloadIdentityDeps, id string, now time.Time) (*RefreshWorkloadTrustBundleJWKSResult, error) {
	tenantID := tenancy.TenantID(ctx)
	bundle, err := deps.TrustBundleRepo.FindByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if bundle == nil {
		return nil, ErrTrustBundleNotFound
	}
	if deps.FetchJWKS == nil {
		return &RefreshWorkloadTrustBundleJWKSResult{Reachable: false}, nil
	}
	keys, fetchErr := deps.FetchJWKS(ctx, bundle)
	reachable := fetchErr == nil
	result := &RefreshWorkloadTrustBundleJWKSResult{Reachable: reachable, KeyCount: len(keys)}
	if reachable {
		updated := *bundle
		updated.JWKSCachedAt = &now
		if err := deps.TrustBundleRepo.Save(ctx, &updated); err != nil {
			return nil, err
		}
		result.JWKSCachedAt = &now
	}
	emit(deps.Emit, &domain.WorkloadTrustBundleJWKSRefreshed{At: now, TenantID: tenantID, TrustBundleID: id, Reachable: reachable})
	return result, nil
}

func ensureTrustBundleNameAndIssuerAvailable(ctx context.Context, deps AdminWorkloadIdentityDeps, tenantID, name, issuer, excludeID string) error {
	bundles, err := deps.TrustBundleRepo.ListByTenant(ctx, tenantID)
	if err != nil {
		return err
	}
	for _, b := range bundles {
		if b.ID == excludeID {
			continue
		}
		if strings.EqualFold(b.Name, name) {
			return ErrTrustBundleNameConflict
		}
		if b.Issuer == issuer {
			return ErrTrustBundleIssuerConflict
		}
	}
	return nil
}
