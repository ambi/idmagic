// 管理者向け WorkloadTrustBundle ライフサイクル操作。SCL WorkloadIdentity
// モジュールが所有する admin インターフェース群: ListWorkloadTrustBundles /
// GetWorkloadTrustBundle / RegisterWorkloadTrustBundle / UpdateWorkloadTrustBundle /
// DisableWorkloadTrustBundle / EnableWorkloadTrustBundle / DeleteWorkloadTrustBundle /
// RefreshWorkloadTrustBundleJWKS。
//
// すべての操作は tenantports.TenantID(ctx) のテナント境界に閉じる。issuer / name は
// テナント内で一意 (DB の UNIQUE 制約に加え、backend 非依存の一貫した挙動のため
// usecase 層でも事前チェックする)。
package usecases

import (
	"context"
	"strings"
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
	workloaddomain "github.com/ambi/idmagic/backend/workloadidentity/domain"
	workloadports "github.com/ambi/idmagic/backend/workloadidentity/internal/ports"
)

type AdminWorkloadIdentityDeps struct {
	TrustBundleRepo workloadports.WorkloadTrustBundleRepository
	BindingRepo     workloadports.AgentWorkloadBindingRepository
	// FetchJWKS performs the live JWKS lookup used by RefreshWorkloadTrustBundleJWKS
	// (test/refresh action). Shares the same signature as
	// VerifyWorkloadAttestationDeps.FetchJWKS so both can be wired to the same
	// adapter at the composition root.
	FetchJWKS func(ctx context.Context, bundle *workloaddomain.WorkloadTrustBundle) ([]map[string]any, error)
	Emit      func(spec.DomainEvent)
}

func ListWorkloadTrustBundles(ctx context.Context, deps AdminWorkloadIdentityDeps) ([]*workloaddomain.WorkloadTrustBundle, error) {
	return deps.TrustBundleRepo.ListAll(ctx, tenantports.TenantID(ctx))
}

// GetWorkloadTrustBundle は別テナントの bundle を未存在として扱う。
func GetWorkloadTrustBundle(ctx context.Context, deps AdminWorkloadIdentityDeps, id string) (*workloaddomain.WorkloadTrustBundle, error) {
	bundle, err := deps.TrustBundleRepo.FindByID(ctx, tenantports.TenantID(ctx), id)
	if err != nil {
		return nil, err
	}
	if bundle == nil {
		return nil, ErrTrustBundleNotFound
	}
	return bundle, nil
}

type RegisterWorkloadTrustBundleInput struct {
	Name string
	// Profile は空なら oidc にする。
	Profile                   workloaddomain.WorkloadTrustBundleProfile
	TrustDomain               string
	Issuer                    string
	JWKSURI                   *string
	JWKS                      map[string]any
	AcceptedAudiences         []string
	MaxSubjectTokenTTLSeconds *int
}

func RegisterWorkloadTrustBundle(ctx context.Context, deps AdminWorkloadIdentityDeps, in RegisterWorkloadTrustBundleInput, now time.Time) (*workloaddomain.WorkloadTrustBundle, error) {
	tenantID := tenantports.TenantID(ctx)
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, ErrTrustBundleNameRequired
	}
	profile := in.Profile
	if profile == "" {
		profile = workloaddomain.WorkloadTrustBundleProfileOIDC
	}
	if !profile.Valid() {
		return nil, ErrTrustBundleInvalidProfile
	}
	trustDomain := strings.TrimSpace(in.TrustDomain)
	issuer := strings.TrimSpace(in.Issuer)
	if issuer == "" && profile != workloaddomain.WorkloadTrustBundleProfileSPIFFE {
		return nil, ErrTrustBundleIssuerRequired
	}
	if profile == workloaddomain.WorkloadTrustBundleProfileSPIFFE && !workloaddomain.ValidTrustDomain(trustDomain) {
		return nil, ErrTrustBundleInvalidTrustDomain
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
	if profile == workloaddomain.WorkloadTrustBundleProfileSPIFFE {
		existing, err := deps.TrustBundleRepo.FindSPIFFEByTrustDomain(ctx, tenantID, trustDomain)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			return nil, ErrTrustBundleTrustDomainConflict
		}
	}
	ttl := 3600
	if in.MaxSubjectTokenTTLSeconds != nil && *in.MaxSubjectTokenTTLSeconds > 0 {
		ttl = *in.MaxSubjectTokenTTLSeconds
	}
	id, err := workloaddomain.NewWorkloadTrustBundleID()
	if err != nil {
		return nil, err
	}
	bundle := &workloaddomain.WorkloadTrustBundle{
		ID: id, TenantID: tenantID, Name: name, Profile: profile, TrustDomain: trustDomain,
		Issuer: issuer, JWKSURI: in.JWKSURI, JWKS: in.JWKS, AcceptedAudiences: in.AcceptedAudiences,
		MaxSubjectTokenTTLSeconds: ttl, Status: workloaddomain.WorkloadTrustBundleStatusEnabled, CreatedAt: now,
	}
	if err := bundle.Validate(); err != nil {
		return nil, err
	}
	if err := deps.TrustBundleRepo.Save(ctx, bundle); err != nil {
		return nil, err
	}
	emit(deps.Emit, &workloaddomain.WorkloadTrustBundleConfigured{At: now, TenantID: tenantID, TrustBundleID: bundle.ID, Issuer: bundle.Issuer})
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
// max_subject_token_ttl_seconds を更新する。profile / issuer / trust_domain は不変。
func UpdateWorkloadTrustBundle(ctx context.Context, deps AdminWorkloadIdentityDeps, id string, in UpdateWorkloadTrustBundleInput, now time.Time) (*workloaddomain.WorkloadTrustBundle, error) {
	tenantID := tenantports.TenantID(ctx)
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
	// 最後に取得できた鍵は現在の jwks_uri のものだけを使う (REQ-WORKLOADIDENTITY-018)。
	if updated.JWKSSourceChanged(*bundle) {
		updated.JWKSCached, updated.JWKSCachedAt = nil, nil
	}
	if err := updated.Validate(); err != nil {
		return nil, err
	}
	if err := deps.TrustBundleRepo.Save(ctx, &updated); err != nil {
		return nil, err
	}
	emit(deps.Emit, &workloaddomain.WorkloadTrustBundleUpdated{At: now, TenantID: tenantID, TrustBundleID: bundle.ID})
	return &updated, nil
}

func DisableWorkloadTrustBundle(ctx context.Context, deps AdminWorkloadIdentityDeps, id string, now time.Time) (*workloaddomain.WorkloadTrustBundle, error) {
	return setTrustBundleStatus(ctx, deps, id, workloaddomain.WorkloadTrustBundleStatusDisabled, now)
}

func EnableWorkloadTrustBundle(ctx context.Context, deps AdminWorkloadIdentityDeps, id string, now time.Time) (*workloaddomain.WorkloadTrustBundle, error) {
	return setTrustBundleStatus(ctx, deps, id, workloaddomain.WorkloadTrustBundleStatusEnabled, now)
}

func setTrustBundleStatus(ctx context.Context, deps AdminWorkloadIdentityDeps, id string, status workloaddomain.WorkloadTrustBundleStatus, now time.Time) (*workloaddomain.WorkloadTrustBundle, error) {
	tenantID := tenantports.TenantID(ctx)
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
	if status == workloaddomain.WorkloadTrustBundleStatusDisabled {
		emit(deps.Emit, &workloaddomain.WorkloadTrustBundleDisabled{At: now, TenantID: tenantID, TrustBundleID: bundle.ID})
	} else {
		emit(deps.Emit, &workloaddomain.WorkloadTrustBundleEnabled{At: now, TenantID: tenantID, TrustBundleID: bundle.ID})
	}
	return &updated, nil
}

// DeleteWorkloadTrustBundle は WorkloadTrustBundle を削除し、配下の
// AgentWorkloadBinding を先に cascade 削除する (DB の ON DELETE CASCADE と併せた
// 二重の保証)。
func DeleteWorkloadTrustBundle(ctx context.Context, deps AdminWorkloadIdentityDeps, id string, now time.Time) error {
	tenantID := tenantports.TenantID(ctx)
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
	emit(deps.Emit, &workloaddomain.WorkloadTrustBundleDeleted{At: now, TenantID: tenantID, TrustBundleID: id})
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
	tenantID := tenantports.TenantID(ctx)
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
		fetched := workloaddomain.JWKSSnapshot{Keys: keys, FetchedAt: now}
		if err := deps.TrustBundleRepo.RecordJWKSFetched(ctx, tenantID, id, bundle.JWKSURI, fetched); err != nil {
			return nil, err
		}
		result.JWKSCachedAt = &now
	}
	emit(deps.Emit, &workloaddomain.WorkloadTrustBundleJWKSRefreshed{At: now, TenantID: tenantID, TrustBundleID: id, Reachable: reachable})
	return result, nil
}

func ensureTrustBundleNameAndIssuerAvailable(ctx context.Context, deps AdminWorkloadIdentityDeps, tenantID, name, issuer, excludeID string) error {
	bundles, err := deps.TrustBundleRepo.ListAll(ctx, tenantID)
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
		if issuer != "" && b.Issuer == issuer {
			return ErrTrustBundleIssuerConflict
		}
	}
	return nil
}
