package usecases

import (
	"context"
	"strings"

	claimdomain "github.com/ambi/idmagic/backend/claimmapping/domain"
	feddomain "github.com/ambi/idmagic/backend/wsfederation/domain"
	wsfedports "github.com/ambi/idmagic/backend/wsfederation/ports"
)

const (
	defaultNameIDFormat = "urn:oasis:names:tc:SAML:1.1:nameid-format:unspecified"
	defaultNameIDSource = "user_id"
)

// RelyingPartyAdministration は wsfedports.RelyingPartyAdministration を RP の保存先で実装する。
type RelyingPartyAdministration struct {
	RPRepo wsfedports.WsFedRelyingPartyRepository
}

var _ wsfedports.RelyingPartyAdministration = RelyingPartyAdministration{}

func (a RelyingPartyAdministration) Find(ctx context.Context, tenantID, wtrealm string) (*feddomain.WsFedRelyingParty, error) {
	return a.RPRepo.FindByWtrealm(ctx, tenantID, wtrealm)
}

func (a RelyingPartyAdministration) CreateForApplication(ctx context.Context, in wsfedports.CreateApplicationRelyingPartyInput) error {
	rp := &feddomain.WsFedRelyingParty{
		TenantID: in.TenantID, Wtrealm: in.Wtrealm, DisplayName: in.DisplayName, ReplyURLs: in.ReplyURLs,
		ClaimPolicy: claimdomain.ClaimMappingPolicy{NameID: claimdomain.NameIdConfiguration{
			Format: nonEmpty(in.NameIDFormat, defaultNameIDFormat), SourceAttribute: nonEmpty(in.NameIDSource, defaultNameIDSource),
		}},
		CreatedAt: in.Now,
	}
	if err := rp.Validate(); err != nil {
		return err
	}
	existing, err := a.RPRepo.FindByWtrealm(ctx, in.TenantID, in.Wtrealm)
	if err != nil {
		return err
	}
	if existing != nil {
		return wsfedports.ErrRelyingPartyAlreadyRegistered
	}
	return a.RPRepo.Save(ctx, rp)
}

func (a RelyingPartyAdministration) Update(ctx context.Context, in wsfedports.UpdateRelyingPartyInput) error {
	rp, err := a.RPRepo.FindByWtrealm(ctx, in.TenantID, in.Wtrealm)
	if err != nil {
		return err
	}
	if rp == nil {
		return wsfedports.ErrRelyingPartyNotFound
	}
	if in.TokenType != nil && *in.TokenType != "" && !in.TokenType.Valid() {
		return wsfedports.ErrInvalidTokenType
	}
	if in.ReplyURLs != nil {
		rp.ReplyURLs = *in.ReplyURLs
	}
	if in.Audience != nil {
		rp.Audience = strings.TrimSpace(*in.Audience)
	}
	if in.TokenType != nil {
		rp.TokenType = *in.TokenType
	}
	if in.NameIDFormat != nil {
		rp.ClaimPolicy.NameID.Format = *in.NameIDFormat
	}
	if in.NameIDSource != nil {
		rp.ClaimPolicy.NameID.SourceAttribute = *in.NameIDSource
	}
	if in.Rules != nil {
		rp.ClaimPolicy.Rules = *in.Rules
	}
	rp.UpdatedAt = in.Now
	return a.RPRepo.Save(ctx, rp)
}

func (a RelyingPartyAdministration) BindApplication(ctx context.Context, tenantID, wtrealm, applicationID string) error {
	rp, err := a.RPRepo.FindByWtrealm(ctx, tenantID, wtrealm)
	if err != nil || rp == nil {
		return err
	}
	rp.ApplicationID = applicationID
	return a.RPRepo.Save(ctx, rp)
}

func (a RelyingPartyAdministration) DeleteForApplication(ctx context.Context, tenantID, wtrealm string) error {
	return a.RPRepo.Delete(ctx, tenantID, wtrealm)
}

func nonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
