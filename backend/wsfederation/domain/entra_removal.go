package domain

import (
	"errors"
	"net/url"
	"strings"
)

// EntraDomainRealm は、Microsoft のレルム探索がドメインについて答えた送り先である。
type EntraDomainRealm struct {
	// NamespaceType は "Managed"、"Federated"、"Unknown" のどれかか、Microsoft が加えるほかの値である。
	NamespaceType string
	// AuthURL は、Federated のときに Entra が利用者を送る URL である。WS-Fed では登録した
	// PassiveSignInUri に問い合わせ文字列を付けた形になる。
	AuthURL string
}

// EntraReleaseEvidence は、ドメインの解除を許した根拠である。
type EntraReleaseEvidence string

const (
	// EntraReleasedByRealmDiscovery は、解除の時点のレルム探索が切り戻しを答えたことを表す。
	EntraReleasedByRealmDiscovery EntraReleaseEvidence = "realm_discovery"
	// EntraReleasedByObservation は、探索で確かめられず、取り込んだ観測記録が切り戻しを示したことを表す。
	EntraReleasedByObservation EntraReleaseEvidence = "observation"

	entraManagedNamespaceType   = "Managed"
	entraFederatedNamespaceType = "Federated"
)

var (
	// ErrEntraFederationStillActive は、レルム探索が、Entra がまだドメインの利用者を IdMagic へ送ると
	// 答えたことを表す。
	ErrEntraFederationStillActive = errors.New("entra federation: Entra still sends the domain to IdMagic")
	// ErrEntraRollbackUnconfirmed は、探索でも観測記録でも切り戻しを確かめられないことを表す。
	ErrEntraRollbackUnconfirmed = errors.New("entra federation: the rollback in Entra is not confirmed")
)

// WithoutDomain は、domain の組を外した定型設定を返す。受け取った定型設定は変えない。last は残る組が
// ないこと、ok は domain の組があったことを表す。サブドメインの組は別の組なので残す。
func (p EntraFederationProfile) WithoutDomain(domain string) (out EntraFederationProfile, last, ok bool) {
	domains := make([]EntraDomainFederation, 0, len(p.Domains))
	for _, d := range p.Domains {
		if d.Domain == domain {
			ok = true
			continue
		}
		domains = append(domains, d)
	}
	out = p
	out.Domains = domains
	return out, len(domains) == 0, ok
}

// DecideEntraRelease は、レルム探索の答え（失敗や無効なら nil）と観測記録から、ドメインの解除を許す根拠を
// 返す。探索が IdMagic を指せば ErrEntraFederationStillActive、どちらでも確かめられなければ
// ErrEntraRollbackUnconfirmed を返す。
func DecideEntraRelease(realm *EntraDomainRealm, record *EntraFederationObservationRecord, expected EntraExpectedSettings) (EntraReleaseEvidence, error) {
	if realm != nil {
		namespaceType := strings.TrimSpace(realm.NamespaceType)
		switch {
		case strings.EqualFold(namespaceType, entraManagedNamespaceType):
			return EntraReleasedByRealmDiscovery, nil
		case strings.EqualFold(namespaceType, entraFederatedNamespaceType):
			destination, err := url.Parse(strings.TrimSpace(realm.AuthURL))
			if err != nil || destination.Hostname() == "" || (destination.Scheme != "http" && destination.Scheme != "https") {
				// 読めない送り先を「別の IdP」と解釈しない。探索の形式が変わっても観測記録へ戻す。
				break
			}
			if sameEndpoint(destination, expected.PassiveSignInURI) {
				// 探索は解除の時点の送り先なので、取り込んだ観測より新しい。観測記録を問わずに拒否する。
				return "", ErrEntraFederationStillActive
			}
			return EntraReleasedByRealmDiscovery, nil
		}
		// Unknown やほかの値は、Microsoft が情報を絞った場合にも返り得るので、解除の根拠にしない。
	}
	if observationShowsRelease(record, expected) {
		return EntraReleasedByObservation, nil
	}
	return "", ErrEntraRollbackUnconfirmed
}

// sameEndpoint は、Entra が送る URL が、問い合わせ文字列を除いて IdMagic の受動のエンドポイントと同じかを返す。
// ホストは大文字と小文字を区別せず、スキームとパスは区別する。
func sameEndpoint(got *url.URL, passive string) bool {
	want, err := url.Parse(passive)
	if err != nil || want.Host == "" {
		return false
	}
	return strings.EqualFold(got.Scheme, want.Scheme) && strings.EqualFold(got.Host, want.Host) && got.Path == want.Path
}

// observationShowsRelease は、観測記録が今の期待値と比べたものであり、Entra がドメインを IdMagic へ
// 送らない状態（Managed で構成なし、または IdMagic を指さない構成が一つ）を示すかを返す。
func observationShowsRelease(record *EntraFederationObservationRecord, expected EntraExpectedSettings) bool {
	if record == nil || record.ExpectedFingerprint != expected.Fingerprint() {
		return false
	}
	observed := record.Observation
	if expected.EntraTenantID != "" && !strings.EqualFold(expected.EntraTenantID, strings.TrimSpace(observed.EntraTenantID)) {
		return false
	}
	for _, config := range observed.FederationConfigurations {
		if pointsToIdMagic(config, expected) {
			return false
		}
	}
	authenticationType := strings.TrimSpace(observed.AuthenticationType)
	switch {
	case strings.EqualFold(authenticationType, entraManagedNamespaceType):
		return len(observed.FederationConfigurations) == 0
	case strings.EqualFold(authenticationType, entraFederatedAuthenticationType):
		return len(observed.FederationConfigurations) == 1
	}
	return false
}

// pointsToIdMagic は、構成が IdMagic の IssuerUri か、Entra が利用者を送るエンドポイントのどれかを持つかを返す。
// 一つでも残れば、Entra はそのドメインの認証の一部を IdMagic へ送る。
func pointsToIdMagic(config EntraObservedFederationConfiguration, expected EntraExpectedSettings) bool {
	same := func(observed, ours string) bool { return ours != "" && observed == ours }
	return same(config.IssuerURI, expected.IssuerURI) || same(config.PassiveSignInURI, expected.PassiveSignInURI) ||
		same(config.ActiveSignInURI, expected.ActiveSignInURI) || same(config.MetadataExchangeURI, expected.MetadataExchangeURI)
}
