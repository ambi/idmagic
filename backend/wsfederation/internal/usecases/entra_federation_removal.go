package usecases

import (
	"context"
	"slices"
	"time"

	feddomain "github.com/ambi/idmagic/backend/wsfederation/domain"
	wsfedports "github.com/ambi/idmagic/backend/wsfederation/ports"
)

// EntraRemovalResult は、ドメインの解除の結果と、発行するイベントである。
type EntraRemovalResult struct {
	Domain feddomain.EntraDomainFederation
	// RemainingDomains は、解除の後に残るドメインをドメインの順で並べる。
	RemainingDomains    []string
	RelyingPartyDeleted bool
	ConfirmedBy         feddomain.EntraReleaseEvidence
	Event               *feddomain.EntraFederationDomainRemoved
}

// Remove は、Entra がもうドメインの利用者を IdMagic へ送らないことを確かめたときだけ、ドメインの組を
// Entra の RP から外す。最後の組なら RP を削除する。署名鍵には触れない。
//
// 構成していなければ探索を呼ばずに ErrEntraDomainNotConfigured を返す。レルム探索は RP の行の固定の外で
// 呼び、判定は固定の中で読み直した RP の期待値に対して行う。確かめられなければ
// feddomain.ErrEntraFederationStillActive か feddomain.ErrEntraRollbackUnconfirmed を返し、RP と観測記録を変えない。
func (v EntraFederationVerification) Remove(ctx context.Context, tenantID, domainName string, now time.Time) (EntraRemovalResult, error) {
	rp, err := v.entraRelyingParty(ctx, tenantID)
	if err != nil {
		return EntraRemovalResult{}, err
	}
	configured, ok := configuredDomain(rp, domainName)
	if !ok {
		return EntraRemovalResult{}, ErrEntraDomainNotConfigured
	}
	current, next, err := v.certificates(ctx)
	if err != nil {
		return EntraRemovalResult{}, err
	}
	// 失敗は「確かめられない」として判定へ渡し、取り込んだ観測記録へ戻る。
	var realm *feddomain.EntraDomainRealm
	if discovered, err := v.Realms.Discover(ctx, configured.Domain); err == nil {
		realm = &discovered
	}

	var result EntraRemovalResult
	err = v.RPRepo.Update(ctx, tenantID, feddomain.EntraRealm, func(rp *feddomain.WsFedRelyingParty) (wsfedports.RelyingPartyOutcome, error) {
		if rp == nil || rp.EntraProfile == nil {
			return wsfedports.RelyingPartyUnchanged, ErrEntraDomainNotConfigured
		}
		domain, ok := configuredDomain(rp, domainName)
		if !ok {
			return wsfedports.RelyingPartyUnchanged, ErrEntraDomainNotConfigured
		}
		record, err := v.Observations.Find(ctx, tenantID, domain.Domain)
		if err != nil {
			return wsfedports.RelyingPartyUnchanged, err
		}
		expected := feddomain.ExpectedEntraSettings(*rp.EntraProfile, domain, current, next)
		evidence, err := feddomain.DecideEntraRelease(realm, record, expected)
		if err != nil {
			return wsfedports.RelyingPartyUnchanged, err
		}
		// RP の保存より先に消すので、保存に失敗するとドメインは構成されたまま未確認へ倒れ、管理者は
		// 解除を再実行する。解除した後のドメインに記録を残さない。
		if err := v.Observations.Delete(ctx, tenantID, domain.Domain); err != nil {
			return wsfedports.RelyingPartyUnchanged, err
		}
		remaining, last, _ := rp.EntraProfile.WithoutDomain(domain.Domain)
		result = EntraRemovalResult{
			Domain: domain, RemainingDomains: domainNames(remaining), RelyingPartyDeleted: last, ConfirmedBy: evidence,
		}
		if last {
			return wsfedports.RelyingPartyDeleted, nil
		}
		rp.EntraProfile = &remaining
		rp.UpdatedAt = now
		return wsfedports.RelyingPartySaved, nil
	})
	if err != nil {
		return EntraRemovalResult{}, err
	}
	result.Event = &feddomain.EntraFederationDomainRemoved{
		At: now, TenantID: tenantID, Domain: result.Domain.Domain, IssuerURI: result.Domain.IssuerURI,
		ConfirmedBy: result.ConfirmedBy, RelyingPartyDeleted: result.RelyingPartyDeleted,
	}
	return result, nil
}

func domainNames(profile feddomain.EntraFederationProfile) []string {
	names := make([]string, 0, len(profile.Domains))
	for _, d := range profile.Domains {
		names = append(names, d.Domain)
	}
	slices.Sort(names)
	return names
}
