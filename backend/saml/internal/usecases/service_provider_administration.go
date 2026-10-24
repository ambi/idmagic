package usecases

import (
	"context"
	"strings"

	claimdomain "github.com/ambi/idmagic/backend/claimmapping/domain"
	samldomain "github.com/ambi/idmagic/backend/saml/domain"
	samlports "github.com/ambi/idmagic/backend/saml/ports"
)

const defaultNameIDSource = "user_id"

// ServiceProviderAdministration は samlports.ServiceProviderAdministration を SP の保存先で実装する。
type ServiceProviderAdministration struct {
	SPRepo samlports.SamlServiceProviderRepository
}

var _ samlports.ServiceProviderAdministration = ServiceProviderAdministration{}

func (a ServiceProviderAdministration) Find(ctx context.Context, tenantID, entityID string) (*samldomain.SamlServiceProvider, error) {
	return a.SPRepo.FindByEntityID(ctx, tenantID, entityID)
}

func (a ServiceProviderAdministration) CreateForApplication(ctx context.Context, in samlports.CreateApplicationServiceProviderInput) error {
	sp := &samldomain.SamlServiceProvider{
		TenantID: in.TenantID, EntityID: in.EntityID, DisplayName: in.DisplayName,
		IDPProfileID: nonEmpty(in.IDPProfileID, samldomain.DefaultIDPProfileID),
		ACSURLs:      in.ACSURLs, SLOURL: strings.TrimSpace(in.SLOURL),
		ClaimPolicy: claimdomain.ClaimMappingPolicy{NameID: claimdomain.NameIdConfiguration{
			Format: nonEmpty(in.NameIDFormat, samldomain.SamlNameIDFormatPersistent), SourceAttribute: nonEmpty(in.NameIDSource, defaultNameIDSource),
		}},
		SignAssertion: true, SignResponse: in.SignResponse,
		WantAuthnRequestsSigned:           in.WantAuthnRequestsSigned,
		AuthnRequestSigningCertificatePEM: strings.TrimSpace(in.AuthnRequestSigningCertificatePEM),
		CreatedAt:                         in.Now,
	}
	if err := sp.Validate(); err != nil {
		return err
	}
	existing, err := a.SPRepo.FindByEntityID(ctx, in.TenantID, in.EntityID)
	if err != nil {
		return err
	}
	if existing != nil {
		return samlports.ErrServiceProviderAlreadyRegistered
	}
	if sp.WantAuthnRequestsSigned {
		if _, err := samldomain.ParseCertificatePEM(sp.AuthnRequestSigningCertificatePEM); err != nil {
			return samlports.ErrSigningCertificateRequired
		}
	}
	return a.SPRepo.Save(ctx, sp)
}

func (a ServiceProviderAdministration) Update(ctx context.Context, in samlports.UpdateServiceProviderInput) error {
	sp, err := a.SPRepo.FindByEntityID(ctx, in.TenantID, in.EntityID)
	if err != nil {
		return err
	}
	if sp == nil {
		return samlports.ErrServiceProviderNotFound
	}
	if in.ACSURLs != nil {
		sp.ACSURLs = *in.ACSURLs
	}
	if in.IDPProfileID != nil {
		sp.IDPProfileID = strings.TrimSpace(*in.IDPProfileID)
	}
	if in.SLOURL != nil {
		sp.SLOURL = strings.TrimSpace(*in.SLOURL)
	}
	if in.Audience != nil {
		sp.Audience = strings.TrimSpace(*in.Audience)
	}
	if in.NameIDFormat != nil {
		sp.ClaimPolicy.NameID.Format = *in.NameIDFormat
	}
	if in.NameIDSource != nil {
		sp.ClaimPolicy.NameID.SourceAttribute = *in.NameIDSource
	}
	if in.SignAssertion != nil {
		sp.SignAssertion = *in.SignAssertion
	}
	if in.SignResponse != nil {
		sp.SignResponse = *in.SignResponse
	}
	if in.WantAuthnRequestsSigned != nil {
		sp.WantAuthnRequestsSigned = *in.WantAuthnRequestsSigned
	}
	if in.AuthnRequestSigningCertificatePEM != nil {
		sp.AuthnRequestSigningCertificatePEM = strings.TrimSpace(*in.AuthnRequestSigningCertificatePEM)
	}
	if sp.WantAuthnRequestsSigned {
		if _, err := samldomain.ParseCertificatePEM(sp.AuthnRequestSigningCertificatePEM); err != nil {
			return samlports.ErrSigningCertificateRequired
		}
	}
	if in.Rules != nil {
		sp.ClaimPolicy.Rules = *in.Rules
	}
	sp.UpdatedAt = in.Now
	return a.SPRepo.Save(ctx, sp)
}

func (a ServiceProviderAdministration) BindApplication(ctx context.Context, tenantID, entityID, applicationID string) error {
	sp, err := a.SPRepo.FindByEntityID(ctx, tenantID, entityID)
	if err != nil || sp == nil {
		return err
	}
	sp.ApplicationID = applicationID
	return a.SPRepo.Save(ctx, sp)
}

func (a ServiceProviderAdministration) DeleteForApplication(ctx context.Context, tenantID, entityID string) error {
	return a.SPRepo.Delete(ctx, tenantID, entityID)
}

func nonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
