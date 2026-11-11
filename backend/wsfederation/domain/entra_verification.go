package domain

import (
	"bytes"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// EntraObservationFormat は、読み戻しの手順が書き出す観測文書の形式である。
const EntraObservationFormat = "idmagic.entra-federation-observation.v1"

const (
	// maxObservedFederationConfigurations は、一つの観測文書が持てる構成の数である。Entra は
	// ドメインに一つの構成しか持たないので、超える文書は手順の出力ではない。
	maxObservedFederationConfigurations = 4
	// observationClockSkew は、管理者の端末と IdMagic の時刻のずれとして許す幅である。
	observationClockSkew = 5 * time.Minute

	entraFederatedAuthenticationType   = "Federated"
	entraPreferredAuthenticationProto  = "wsFed"
	entraFederatedIdpMfaBehavior       = "acceptIfMfaDoneByFederatedIdp"
	unreadableObservedCertificateValue = "unreadable"
)

var (
	// ErrInvalidEntraTenantID は、Entra のテナント ID が GUID の形でないことを表す。
	ErrInvalidEntraTenantID = errors.New("entra federation: the Entra tenant ID must be a GUID")
	// ErrInvalidEntraObservation は、観測文書が読み戻しの手順の出力の形でないことを表す。
	ErrInvalidEntraObservation = errors.New("entra federation: the observation document is invalid")

	entraTenantIDPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
)

// NormalizeEntraTenantID は、Entra のテナント ID を前後の空白を除いた小文字にする。空は空のまま返し、
// GUID の形でなければ ErrInvalidEntraTenantID を返す。
func NormalizeEntraTenantID(value string) (string, error) {
	id := strings.ToLower(strings.TrimSpace(value))
	if id == "" || entraTenantIDPattern.MatchString(id) {
		return id, nil
	}
	return "", ErrInvalidEntraTenantID
}

// EntraExpectedSettings は、一つのドメインについて Entra に登録されているべき値である。
type EntraExpectedSettings struct {
	Domain              string
	EntraTenantID       string
	IssuerURI           string
	PassiveSignInURI    string
	ActiveSignInURI     string
	MetadataExchangeURI string
	SignOutURI          string
	// SigningCertificate は有効な XML 署名証明書、NextSigningCertificate は準備済みの証明書で、なければ nil である。
	SigningCertificate     *x509.Certificate
	NextSigningCertificate *x509.Certificate
}

// ExpectedEntraSettings は、定型設定に保存したエンドポイントとドメインの組、署名証明書から期待値を作る。
func ExpectedEntraSettings(profile EntraFederationProfile, domain EntraDomainFederation, signing, next *x509.Certificate) EntraExpectedSettings {
	return EntraExpectedSettings{
		Domain: domain.Domain, EntraTenantID: domain.EntraTenantID, IssuerURI: domain.IssuerURI,
		PassiveSignInURI: profile.PassiveLogOnURI, ActiveSignInURI: profile.ActiveLogOnURI,
		MetadataExchangeURI: profile.MetadataExchangeURI, SignOutURI: profile.PassiveLogOnURI + "?wa=wsignout1.0",
		SigningCertificate: signing, NextSigningCertificate: next,
	}
}

// GraphParameters は、Microsoft Graph PowerShell のパラメーター名をキーとする期待値を返す。
// 次の証明書がなければ NextSigningCertificate を含めない。
func (e EntraExpectedSettings) GraphParameters() map[string]string {
	parameters := map[string]string{
		"IssuerUri":                       e.IssuerURI,
		"PassiveSignInUri":                e.PassiveSignInURI,
		"ActiveSignInUri":                 e.ActiveSignInURI,
		"MetadataExchangeUri":             e.MetadataExchangeURI,
		"SignOutUri":                      e.SignOutURI,
		"SigningCertificate":              certificateBase64(e.SigningCertificate),
		"PreferredAuthenticationProtocol": entraPreferredAuthenticationProto,
		"FederatedIdpMfaBehavior":         entraFederatedIdpMfaBehavior,
	}
	if e.NextSigningCertificate != nil {
		parameters["NextSigningCertificate"] = certificateBase64(e.NextSigningCertificate)
	}
	return parameters
}

// Fingerprint は、比べる期待値のすべて（テナント、ドメイン、Graph の設定値、現在と次の証明書）の
// SHA-256 である。構成の変更でも、鍵の準備や交代でも変わる。
func (e EntraExpectedSettings) Fingerprint() string {
	canonical, err := json.Marshal(struct {
		Domain        string            `json:"domain"`
		EntraTenantID string            `json:"entra_tenant_id"`
		Parameters    map[string]string `json:"parameters"`
	}{strings.ToLower(e.Domain), e.EntraTenantID, e.GraphParameters()})
	if err != nil {
		// 文字列だけの値の直列化は失敗しない。
		panic(err)
	}
	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:])
}

// EntraFederationObservation は、Graph PowerShell が読み戻した一つのドメインの登録値である。
type EntraFederationObservation struct {
	Format                   string                                 `json:"format"`
	EntraTenantID            string                                 `json:"entra_tenant_id"`
	Domain                   string                                 `json:"domain"`
	AuthenticationType       string                                 `json:"authentication_type"`
	RetrievedAt              time.Time                              `json:"retrieved_at"`
	FederationConfigurations []EntraObservedFederationConfiguration `json:"federation_configurations"`
}

// EntraObservedFederationConfiguration は、Graph の一つのフェデレーションの構成の値である。
type EntraObservedFederationConfiguration struct {
	IssuerURI                       string `json:"issuer_uri"`
	PassiveSignInURI                string `json:"passive_sign_in_uri"`
	ActiveSignInURI                 string `json:"active_sign_in_uri"`
	MetadataExchangeURI             string `json:"metadata_exchange_uri"`
	SignOutURI                      string `json:"sign_out_uri"`
	SigningCertificate              string `json:"signing_certificate"`
	NextSigningCertificate          string `json:"next_signing_certificate"`
	PreferredAuthenticationProtocol string `json:"preferred_authentication_protocol"`
	FederatedIdpMfaBehavior         string `json:"federated_idp_mfa_behavior"`
}

// DecodeEntraFederationObservation は、未知の項目と必須の項目の欠落を拒否して観測文書を読む。
// 空文字列を許す項目（次の証明書、SignOutUri）も、項目そのものは必須である。
func DecodeEntraFederationObservation(body []byte) (EntraFederationObservation, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	var raw struct {
		Format                   *string           `json:"format"`
		EntraTenantID            *string           `json:"entra_tenant_id"`
		Domain                   *string           `json:"domain"`
		AuthenticationType       *string           `json:"authentication_type"`
		RetrievedAt              *time.Time        `json:"retrieved_at"`
		FederationConfigurations *[]observedConfig `json:"federation_configurations"`
	}
	if err := decoder.Decode(&raw); err != nil {
		return EntraFederationObservation{}, fmt.Errorf("%w: %w", ErrInvalidEntraObservation, err)
	}
	if decoder.More() {
		return EntraFederationObservation{}, fmt.Errorf("%w: trailing data", ErrInvalidEntraObservation)
	}
	if raw.Format == nil || raw.EntraTenantID == nil || raw.Domain == nil || raw.AuthenticationType == nil ||
		raw.RetrievedAt == nil || raw.FederationConfigurations == nil {
		return EntraFederationObservation{}, fmt.Errorf("%w: a required field is missing", ErrInvalidEntraObservation)
	}
	out := EntraFederationObservation{
		Format: *raw.Format, EntraTenantID: *raw.EntraTenantID, Domain: *raw.Domain,
		AuthenticationType: *raw.AuthenticationType, RetrievedAt: raw.RetrievedAt.UTC(),
	}
	for _, c := range *raw.FederationConfigurations {
		config, err := c.complete()
		if err != nil {
			return EntraFederationObservation{}, err
		}
		out.FederationConfigurations = append(out.FederationConfigurations, config)
	}
	return out, nil
}

// observedConfig は、構成の各項目の欠落を空文字列と区別して読むための形である。
type observedConfig struct {
	IssuerURI                       *string `json:"issuer_uri"`
	PassiveSignInURI                *string `json:"passive_sign_in_uri"`
	ActiveSignInURI                 *string `json:"active_sign_in_uri"`
	MetadataExchangeURI             *string `json:"metadata_exchange_uri"`
	SignOutURI                      *string `json:"sign_out_uri"`
	SigningCertificate              *string `json:"signing_certificate"`
	NextSigningCertificate          *string `json:"next_signing_certificate"`
	PreferredAuthenticationProtocol *string `json:"preferred_authentication_protocol"`
	FederatedIdpMfaBehavior         *string `json:"federated_idp_mfa_behavior"`
}

func (c observedConfig) complete() (EntraObservedFederationConfiguration, error) {
	fields := []*string{
		c.IssuerURI, c.PassiveSignInURI, c.ActiveSignInURI, c.MetadataExchangeURI, c.SignOutURI,
		c.SigningCertificate, c.NextSigningCertificate, c.PreferredAuthenticationProtocol, c.FederatedIdpMfaBehavior,
	}
	if slices.Contains(fields, nil) {
		return EntraObservedFederationConfiguration{}, fmt.Errorf("%w: a federation configuration field is missing", ErrInvalidEntraObservation)
	}
	return EntraObservedFederationConfiguration{
		IssuerURI: *c.IssuerURI, PassiveSignInURI: *c.PassiveSignInURI, ActiveSignInURI: *c.ActiveSignInURI,
		MetadataExchangeURI: *c.MetadataExchangeURI, SignOutURI: *c.SignOutURI, SigningCertificate: *c.SigningCertificate,
		NextSigningCertificate: *c.NextSigningCertificate, PreferredAuthenticationProtocol: *c.PreferredAuthenticationProtocol,
		FederatedIdpMfaBehavior: *c.FederatedIdpMfaBehavior,
	}, nil
}

// Validate は、観測文書が domain の読み戻しの出力であり、checkedAt から見て未来の取得でないことを確かめる。
func (o EntraFederationObservation) Validate(domain string, checkedAt time.Time) error {
	switch {
	case o.Format != EntraObservationFormat:
		return fmt.Errorf("%w: format must be %s", ErrInvalidEntraObservation, EntraObservationFormat)
	case strings.TrimSpace(o.EntraTenantID) == "", strings.TrimSpace(o.AuthenticationType) == "":
		return fmt.Errorf("%w: entra_tenant_id and authentication_type must not be empty", ErrInvalidEntraObservation)
	case !strings.EqualFold(strings.TrimSpace(o.Domain), domain):
		return fmt.Errorf("%w: the observation is for another domain", ErrInvalidEntraObservation)
	case len(o.FederationConfigurations) > maxObservedFederationConfigurations:
		return fmt.Errorf("%w: too many federation configurations", ErrInvalidEntraObservation)
	case o.RetrievedAt.IsZero(), o.RetrievedAt.After(checkedAt.Add(observationClockSkew)):
		return fmt.Errorf("%w: retrieved_at is later than the check", ErrInvalidEntraObservation)
	}
	return nil
}

// EntraSettingDifference は、期待値と観測値が異なる一つの項目である。
type EntraSettingDifference struct {
	Field    string `json:"field"`
	Expected string `json:"expected"`
	Observed string `json:"observed"`
}

// CompareEntraFederation は、期待値と観測値を項目ごとに比べ、異なる項目を決まった順で返す。
func CompareEntraFederation(expected EntraExpectedSettings, observed EntraFederationObservation) []EntraSettingDifference {
	var differences []EntraSettingDifference
	differ := func(field, want, got string) {
		differences = append(differences, EntraSettingDifference{Field: field, Expected: want, Observed: got})
	}
	if expected.EntraTenantID != "" && !strings.EqualFold(expected.EntraTenantID, strings.TrimSpace(observed.EntraTenantID)) {
		differ("entra_tenant_id", expected.EntraTenantID, observed.EntraTenantID)
	}
	if !strings.EqualFold(strings.TrimSpace(observed.AuthenticationType), entraFederatedAuthenticationType) {
		differ("authentication_type", entraFederatedAuthenticationType, observed.AuthenticationType)
	}
	if len(observed.FederationConfigurations) != 1 {
		differ("federation_configurations", "1", strconv.Itoa(len(observed.FederationConfigurations)))
		return differences
	}
	config := observed.FederationConfigurations[0]
	for _, item := range []struct{ field, want, got string }{
		{"issuer_uri", expected.IssuerURI, config.IssuerURI},
		{"passive_sign_in_uri", expected.PassiveSignInURI, config.PassiveSignInURI},
		{"active_sign_in_uri", expected.ActiveSignInURI, config.ActiveSignInURI},
		{"metadata_exchange_uri", expected.MetadataExchangeURI, config.MetadataExchangeURI},
		{"sign_out_uri", expected.SignOutURI, config.SignOutURI},
	} {
		if item.want != item.got {
			differ(item.field, item.want, item.got)
		}
	}
	for _, item := range []struct{ field, want, got string }{
		{"preferred_authentication_protocol", entraPreferredAuthenticationProto, config.PreferredAuthenticationProtocol},
		{"federated_idp_mfa_behavior", entraFederatedIdpMfaBehavior, config.FederatedIdpMfaBehavior},
	} {
		if !strings.EqualFold(item.want, strings.TrimSpace(item.got)) {
			differ(item.field, item.want, item.got)
		}
	}
	for _, item := range []struct {
		field string
		want  *x509.Certificate
		got   string
	}{
		{"signing_certificate", expected.SigningCertificate, config.SigningCertificate},
		{"next_signing_certificate", expected.NextSigningCertificate, config.NextSigningCertificate},
	} {
		got, same := compareCertificate(item.want, item.got)
		if !same {
			differ(item.field, CertificateThumbprint(item.want), got)
		}
	}
	return differences
}

// compareCertificate は、観測した base64 から空白を除いて X.509 として読み、DER を比べる。
// 返す文字列は観測値の要約（拇印、空、unreadable）である。
func compareCertificate(want *x509.Certificate, observed string) (string, bool) {
	compact := strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, observed)
	if compact == "" {
		return "", want == nil
	}
	der, err := base64.StdEncoding.DecodeString(compact)
	if err != nil {
		return unreadableObservedCertificateValue, false
	}
	parsed, err := x509.ParseCertificate(der)
	if err != nil {
		return unreadableObservedCertificateValue, false
	}
	return CertificateThumbprint(parsed), want != nil && bytes.Equal(want.Raw, parsed.Raw)
}

// CertificateThumbprint は、DER の SHA-256 の 16 進の小文字である。nil なら空を返す。
func CertificateThumbprint(certificate *x509.Certificate) string {
	if certificate == nil {
		return ""
	}
	sum := sha256.Sum256(certificate.Raw)
	return hex.EncodeToString(sum[:])
}

func certificateBase64(certificate *x509.Certificate) string {
	if certificate == nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(certificate.Raw)
}

// EntraFederationObservationRecord は、一つのドメインの最新の観測と、そのとき比べた結果である。
type EntraFederationObservationRecord struct {
	TenantID                         string
	Domain                           string
	Observation                      EntraFederationObservation
	ExpectedFingerprint              string
	SigningCertificateThumbprint     string
	NextSigningCertificateThumbprint string
	Matched                          bool
	Differences                      []EntraSettingDifference
	CheckedAt                        time.Time
}

// NewEntraFederationObservationRecord は、観測を期待値と比べた記録を作る。
func NewEntraFederationObservationRecord(tenantID string, expected EntraExpectedSettings, observed EntraFederationObservation, checkedAt time.Time) EntraFederationObservationRecord {
	differences := CompareEntraFederation(expected, observed)
	return EntraFederationObservationRecord{
		TenantID: tenantID, Domain: strings.ToLower(expected.Domain), Observation: observed,
		ExpectedFingerprint:              expected.Fingerprint(),
		SigningCertificateThumbprint:     CertificateThumbprint(expected.SigningCertificate),
		NextSigningCertificateThumbprint: CertificateThumbprint(expected.NextSigningCertificate),
		Matched:                          len(differences) == 0,
		Differences:                      differences,
		CheckedAt:                        checkedAt.UTC(),
	}
}

// EntraVerificationStatus は、Entra への反映の確認の状態である。
type EntraVerificationStatus string

const (
	EntraVerificationUnverified EntraVerificationStatus = "unverified"
	EntraVerificationMatched    EntraVerificationStatus = "matched"
	EntraVerificationMismatched EntraVerificationStatus = "mismatched"
	EntraVerificationStale      EntraVerificationStatus = "stale"
)

// EntraVerification は、現在の期待値から見た観測記録の状態である。
type EntraVerification struct {
	Status                           EntraVerificationStatus   `json:"status"`
	RetrievedAt                      *time.Time                `json:"retrieved_at,omitempty"`
	CheckedAt                        *time.Time                `json:"checked_at,omitempty"`
	ObservedEntraTenantID            string                    `json:"observed_entra_tenant_id,omitempty"`
	SigningCertificateThumbprint     string                    `json:"signing_certificate_thumbprint,omitempty"`
	NextSigningCertificateThumbprint string                    `json:"next_signing_certificate_thumbprint,omitempty"`
	Differences                      *[]EntraSettingDifference `json:"differences,omitempty"`
}

// EntraVerificationFor は、記録が今の期待値と比べたものかを指紋で確かめて状態を決める。
// 期待値が変わった記録の差分は、今の一致や不一致の証拠にしないので返さない。
func EntraVerificationFor(record *EntraFederationObservationRecord, expected EntraExpectedSettings) EntraVerification {
	if record == nil {
		return EntraVerification{Status: EntraVerificationUnverified}
	}
	retrieved, checked := record.Observation.RetrievedAt, record.CheckedAt
	out := EntraVerification{
		RetrievedAt: &retrieved, CheckedAt: &checked,
		ObservedEntraTenantID:            record.Observation.EntraTenantID,
		SigningCertificateThumbprint:     record.SigningCertificateThumbprint,
		NextSigningCertificateThumbprint: record.NextSigningCertificateThumbprint,
	}
	switch {
	case record.ExpectedFingerprint != expected.Fingerprint():
		out.Status = EntraVerificationStale
	case record.Matched:
		out.Status = EntraVerificationMatched
		out.Differences = &[]EntraSettingDifference{}
	default:
		out.Status = EntraVerificationMismatched
		differences := record.Differences
		out.Differences = &differences
	}
	return out
}
