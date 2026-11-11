package usecases

import (
	"context"
	"crypto/x509"
	"errors"
	"slices"
	"strings"
	"time"

	feddomain "github.com/ambi/idmagic/backend/wsfederation/domain"
	wsfedports "github.com/ambi/idmagic/backend/wsfederation/ports"
)

// ErrEntraDomainNotConfigured は、指定したドメインを Entra の定型設定で構成していないことを表す。
var ErrEntraDomainNotConfigured = errors.New("entra federation: the domain is not configured")

// EntraSigningCertificates は、Entra に登録すべき現在と次の XML 署名証明書を返す。次がなければ nil である。
type EntraSigningCertificates interface {
	Current(ctx context.Context) (*x509.Certificate, error)
	Next(ctx context.Context) (*x509.Certificate, error)
}

// EntraFederationVerification は、Entra のドメインの期待値と確認の状態を読み、取り込んだ観測を比べて残す。
// 読む操作は RP と観測記録を変えず、取り込みは観測記録だけを書く。
type EntraFederationVerification struct {
	RPRepo       wsfedports.WsFedRelyingPartyRepository
	Observations wsfedports.EntraFederationObservationRepository
	Certificates EntraSigningCertificates
	// Realms は、解除の前に Entra の送り先を Microsoft のレルム探索で確かめる。
	Realms wsfedports.EntraRealmDiscovery
}

// EntraDomainView は、一つのドメインの保存した設定、期待値、確認の状態である。
type EntraDomainView struct {
	RelyingParty *feddomain.WsFedRelyingParty
	Domain       feddomain.EntraDomainFederation
	Expected     feddomain.EntraExpectedSettings
	Verification feddomain.EntraVerification
}

// Domains は、構成済みのドメインをドメインの順で返す。Entra の構成がなければ空を返す。
func (v EntraFederationVerification) Domains(ctx context.Context, tenantID string) ([]EntraDomainView, error) {
	rp, err := v.entraRelyingParty(ctx, tenantID)
	if err != nil || rp == nil {
		return nil, err
	}
	current, next, err := v.certificates(ctx)
	if err != nil {
		return nil, err
	}
	records, err := v.Observations.ListAll(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	byDomain := map[string]*feddomain.EntraFederationObservationRecord{}
	for _, record := range records {
		byDomain[record.Domain] = record
	}
	views := make([]EntraDomainView, 0, len(rp.EntraProfile.Domains))
	for _, domain := range rp.EntraProfile.Domains {
		views = append(views, view(rp, domain, current, next, byDomain[domain.Domain]))
	}
	slices.SortFunc(views, func(a, b EntraDomainView) int { return strings.Compare(a.Domain.Domain, b.Domain.Domain) })
	return views, nil
}

// Domain は、一つのドメインの詳細を返す。構成していなければ ErrEntraDomainNotConfigured を返す。
func (v EntraFederationVerification) Domain(ctx context.Context, tenantID, domainName string) (EntraDomainView, error) {
	rp, err := v.entraRelyingParty(ctx, tenantID)
	if err != nil {
		return EntraDomainView{}, err
	}
	domain, ok := configuredDomain(rp, domainName)
	if !ok {
		return EntraDomainView{}, ErrEntraDomainNotConfigured
	}
	current, next, err := v.certificates(ctx)
	if err != nil {
		return EntraDomainView{}, err
	}
	record, err := v.Observations.Find(ctx, tenantID, domain.Domain)
	if err != nil {
		return EntraDomainView{}, err
	}
	return view(rp, domain, current, next, record), nil
}

// Record は、観測を現在の期待値と比べて最新の記録として残し、記録から見た状態を返す。
// 文書が不正なら feddomain.ErrInvalidEntraObservation を、古い観測なら
// wsfedports.ErrEntraObservationOutdated を返し、どちらも記録を変えない。
func (v EntraFederationVerification) Record(ctx context.Context, tenantID, domainName string, observation feddomain.EntraFederationObservation, now time.Time) (feddomain.EntraVerification, error) {
	current, err := v.Domain(ctx, tenantID, domainName)
	if err != nil {
		return feddomain.EntraVerification{}, err
	}
	if err := observation.Validate(current.Domain.Domain, now); err != nil {
		return feddomain.EntraVerification{}, err
	}
	record := feddomain.NewEntraFederationObservationRecord(tenantID, current.Expected, observation, now)
	if err := v.Observations.Save(ctx, record); err != nil {
		return feddomain.EntraVerification{}, err
	}
	return feddomain.EntraVerificationFor(&record, current.Expected), nil
}

func (v EntraFederationVerification) entraRelyingParty(ctx context.Context, tenantID string) (*feddomain.WsFedRelyingParty, error) {
	rp, err := v.RPRepo.FindByWtrealm(ctx, tenantID, feddomain.EntraRealm)
	if err != nil || rp == nil || rp.EntraProfile == nil {
		return nil, err
	}
	return rp, nil
}

func (v EntraFederationVerification) certificates(ctx context.Context) (*x509.Certificate, *x509.Certificate, error) {
	current, err := v.Certificates.Current(ctx)
	if err != nil {
		return nil, nil, err
	}
	if current == nil {
		return nil, nil, errors.New("entra federation: the signing certificate is unavailable")
	}
	next, err := v.Certificates.Next(ctx)
	if err != nil {
		return nil, nil, err
	}
	return current, next, nil
}

func configuredDomain(rp *feddomain.WsFedRelyingParty, domainName string) (feddomain.EntraDomainFederation, bool) {
	if rp == nil {
		return feddomain.EntraDomainFederation{}, false
	}
	want := strings.ToLower(strings.TrimSpace(domainName))
	for _, domain := range rp.EntraProfile.Domains {
		if domain.Domain == want {
			return domain, true
		}
	}
	return feddomain.EntraDomainFederation{}, false
}

func view(rp *feddomain.WsFedRelyingParty, domain feddomain.EntraDomainFederation, current, next *x509.Certificate, record *feddomain.EntraFederationObservationRecord) EntraDomainView {
	expected := feddomain.ExpectedEntraSettings(*rp.EntraProfile, domain, current, next)
	return EntraDomainView{
		RelyingParty: rp, Domain: domain, Expected: expected,
		Verification: feddomain.EntraVerificationFor(record, expected),
	}
}
