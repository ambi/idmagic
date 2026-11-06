package domain

import (
	"strings"
	"time"

	z "github.com/Oudwins/zog"

	"github.com/ambi/idmagic/backend/shared/spec"
)

// WorkloadTrustBundle はテナントが登録する外部 attestation 発行者の信頼設定。
// trust domain・issuer・JWKS 取得元 (または inline JWKS)・受理する
// audience・受理する外部 SVID の最大 TTL を束ねる。issuer はテナント内で一意
// (repository 層で強制する)。
type WorkloadTrustBundle struct {
	ID                        string
	TenantID                  string
	Name                      string
	Profile                   WorkloadTrustBundleProfile
	TrustDomain               string
	Issuer                    string
	JWKSURI                   *string
	JWKS                      map[string]any
	AcceptedAudiences         []string
	MaxSubjectTokenTTLSeconds int
	Status                    WorkloadTrustBundleStatus
	CreatedAt                 time.Time
	UpdatedAt                 *time.Time
	JWKSCachedAt              *time.Time
}

var workloadTrustBundleSchema = z.Struct(z.Shape{
	"ID":                        spec.Chars(1, spec.LengthHandle).Required(),
	"TenantID":                  z.String().Min(1).Required(),
	"Name":                      spec.Chars(1, spec.LengthName).Required(),
	"TrustDomain":               spec.Chars(1, spec.LengthDNSName).Required(),
	"AcceptedAudiences":         z.Slice(z.String().Min(1)).Min(1).Required(),
	"MaxSubjectTokenTTLSeconds": z.Int().GT(0).Required(),
	"Profile": z.StringLike[WorkloadTrustBundleProfile]().TestFunc(
		func(value *WorkloadTrustBundleProfile, _ z.Ctx) bool { return value.Valid() },
		z.Message("workload trust bundle profile is not in enum"),
	).Required(),
	"Status": z.StringLike[WorkloadTrustBundleStatus]().TestFunc(
		func(value *WorkloadTrustBundleStatus, _ z.Ctx) bool { return value.Valid() },
		z.Message("workload trust bundle status is not in enum"),
	).Required(),
	"CreatedAt": z.Time().Required(),
})

// Validate は構造的妥当性 (docs/modules/workloadidentity/ WorkloadTrustBundle
// constraints) を検証する: issuer は https (spiffe では省略可)、spiffe の
// trust_domain は SPIFFE のトラストドメインの構文、jwks_uri / jwks の少なくとも一方が
// 必須、accepted_audiences は非空、max_subject_token_ttl_seconds は正。
func (b WorkloadTrustBundle) Validate() error {
	if err := spec.Validate(workloadTrustBundleSchema, &b); err != nil {
		return err
	}
	if b.IsSPIFFE() && !ValidTrustDomain(b.TrustDomain) {
		return ErrInvalidTrustDomain
	}
	if (b.Issuer != "" || !b.IsSPIFFE()) && !strings.HasPrefix(b.Issuer, "https://") {
		return errInvalidIssuer
	}
	if b.JWKSURI == nil && b.JWKS == nil {
		return errMissingJWKSSource
	}
	return nil
}

// IsEnabled は WorkloadTrustBundle が交換に使える状態かを返す (WorkloadTrustBundleLifecycle)。
func (b WorkloadTrustBundle) IsEnabled() bool {
	return b.Status == WorkloadTrustBundleStatusEnabled
}

// IsSPIFFE は SPIFFE の JWT-SVID の規則を掛ける信頼設定かを返す。
func (b WorkloadTrustBundle) IsSPIFFE() bool {
	return b.Profile == WorkloadTrustBundleProfileSPIFFE
}

// NewWorkloadTrustBundleID は不変の WorkloadTrustBundle 識別子 (UUID v4) を生成する。
func NewWorkloadTrustBundleID() (string, error) {
	return spec.NewUUIDv4()
}
