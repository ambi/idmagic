package domain

import (
	"reflect"
	"strings"
	"time"

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
	// JWKSCached と JWKSCachedAt は、現在の JWKSURI から最後に取得できた鍵の集合と取得の
	// 時刻である。取得元に届かない間の検証に使う (REQ-WORKLOADIDENTITY-018)。
	JWKSCached   []map[string]any
	JWKSCachedAt *time.Time
}

// JWKSSnapshot は取得できた鍵の集合と、その取得の時刻である。
type JWKSSnapshot struct {
	Keys      []map[string]any
	FetchedAt time.Time
}

// JWKSStoreInterval は、同じ鍵の集合を保存し直すまでの間隔である。
const JWKSStoreInterval = time.Hour

// StoredJWKS は保存した鍵の集合を返す。なければ nil を返す。
func (b WorkloadTrustBundle) StoredJWKS() *JWKSSnapshot {
	if b.JWKSCached == nil || b.JWKSCachedAt == nil {
		return nil
	}
	return &JWKSSnapshot{Keys: b.JWKSCached, FetchedAt: *b.JWKSCachedAt}
}

// FetchesJWKSURI は、署名の検証の鍵を jwks_uri から取得する信頼設定かを返す。インラインの
// jwks があれば、取得は失敗しないので、鍵を保存しない。
func (b WorkloadTrustBundle) FetchesJWKSURI() bool {
	return b.JWKS == nil && b.JWKSURI != nil && *b.JWKSURI != ""
}

// NeedsJWKSStore は、取得した鍵の集合を保存し直すべきかを返す。保存した集合と異なるか、
// 保存から JWKSStoreInterval を過ぎていれば保存する。
func (b WorkloadTrustBundle) NeedsJWKSStore(fetched JWKSSnapshot) bool {
	stored := b.StoredJWKS()
	if stored == nil || fetched.FetchedAt.Sub(stored.FetchedAt) > JWKSStoreInterval {
		return true
	}
	return !reflect.DeepEqual(stored.Keys, fetched.Keys)
}

// JWKSSourceChanged は、更新後の信頼設定 b が before と異なる jwks_uri を指すかを返す。
func (b WorkloadTrustBundle) JWKSSourceChanged(before WorkloadTrustBundle) bool {
	return !equalOptionalString(b.JWKSURI, before.JWKSURI)
}

func equalOptionalString(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// Validate は構造的妥当性 (docs/modules/workloadidentity/ WorkloadTrustBundle
// constraints) を検証する: issuer は https (spiffe では省略可)、spiffe の
// trust_domain は SPIFFE のトラストドメインの構文、jwks_uri / jwks の少なくとも一方が
// 必須、accepted_audiences は非空、max_subject_token_ttl_seconds は正。
func (b WorkloadTrustBundle) Validate() error {
	if err := spec.All(
		spec.Required("ID", b.ID, spec.Chars(1, spec.LengthHandle)),
		spec.Required("TenantID", b.TenantID),
		spec.Required("Name", b.Name, spec.Chars(1, spec.LengthName)),
		spec.Required("TrustDomain", b.TrustDomain, spec.Chars(1, spec.LengthDNSName)),
		spec.NonEmpty("AcceptedAudiences", b.AcceptedAudiences),
		spec.Required("MaxSubjectTokenTTLSeconds", b.MaxSubjectTokenTTLSeconds, spec.GreaterThan(0)),
		spec.Required("Profile", b.Profile,
			spec.Must(WorkloadTrustBundleProfile.Valid, "workload trust bundle profile is not in enum")),
		spec.Required("Status", b.Status,
			spec.Must(WorkloadTrustBundleStatus.Valid, "workload trust bundle status is not in enum")),
		spec.Required("CreatedAt", b.CreatedAt),
	); err != nil {
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
