package usecases

import (
	"context"
	"errors"
	"strings"
	"time"

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
	if err := requireSigningCertificate(sp); err != nil {
		return err
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
	if err := requireSigningCertificate(sp); err != nil {
		return err
	}
	if in.Rules != nil {
		sp.ClaimPolicy.Rules = *in.Rules
	}
	sp.UpdatedAt = in.Now
	return a.SPRepo.Save(ctx, sp)
}

// ErrApplicationOwned は、Application に属する SP を管理 API から上書きまたは削除しようとしたことを表す。
var ErrApplicationOwned = errors.New("service provider is owned by an application")

// RegisterServiceProviderInput は、管理 API の登録が SP を置き換える内容である。SignAssertion が nil なら
// Assertion に署名し、IDPProfileID が空なら既存の SP の現在のプロファイル（新しい SP では default）を使う。
type RegisterServiceProviderInput struct {
	TenantID                          string
	EntityID                          string
	DisplayName                       string
	IDPProfileID                      string
	ACSURLs                           []string
	SLOURL                            string
	Audience                          string
	ClaimPolicy                       claimdomain.ClaimMappingPolicy
	SignAssertion                     *bool
	SignResponse                      bool
	WantAuthnRequestsSigned           bool
	AuthnRequestSigningCertificatePEM string
	Now                               time.Time
}

// Register は管理 API の登録（REQ-SAML-005）で、同じ entityID の SP を作成時刻を保って置き換えるか作る。
// Application に属する SP は ErrApplicationOwned で、検証できる証明書のない署名の要求は
// ErrSigningCertificateRequired で、長さの違反は Validate のエラーで拒否する。IdP プロファイルの
// 規則の違反は保存のエラーとしてそのまま返す。
func (a ServiceProviderAdministration) Register(ctx context.Context, in RegisterServiceProviderInput) (*samldomain.SamlServiceProvider, bool, error) {
	existing, err := a.SPRepo.FindByEntityID(ctx, in.TenantID, in.EntityID)
	if err != nil {
		return nil, false, err
	}
	if existing != nil && existing.ApplicationID != "" {
		return nil, false, ErrApplicationOwned
	}
	signAssertion := true
	if in.SignAssertion != nil {
		signAssertion = *in.SignAssertion
	}
	idpProfileID := strings.TrimSpace(in.IDPProfileID)
	if idpProfileID == "" {
		idpProfileID = samldomain.DefaultIDPProfileID
		if existing != nil {
			idpProfileID = existing.EffectiveIDPProfileID()
		}
	}
	sp := &samldomain.SamlServiceProvider{
		TenantID: in.TenantID, EntityID: in.EntityID, DisplayName: in.DisplayName, IDPProfileID: idpProfileID,
		ACSURLs: in.ACSURLs, SLOURL: strings.TrimSpace(in.SLOURL), Audience: strings.TrimSpace(in.Audience),
		ClaimPolicy: in.ClaimPolicy, SignAssertion: signAssertion, SignResponse: in.SignResponse,
		WantAuthnRequestsSigned:           in.WantAuthnRequestsSigned,
		AuthnRequestSigningCertificatePEM: strings.TrimSpace(in.AuthnRequestSigningCertificatePEM),
		CreatedAt:                         in.Now, UpdatedAt: in.Now,
	}
	if err := requireSigningCertificate(sp); err != nil {
		return nil, false, err
	}
	if err := sp.Validate(); err != nil {
		return nil, false, err
	}
	if existing != nil {
		sp.CreatedAt = existing.CreatedAt
	}
	if err := a.SPRepo.Save(ctx, sp); err != nil {
		return nil, false, err
	}
	return sp, existing == nil, nil
}

// DeleteStandalone は管理 API の削除で、カタログの外の SP を冪等に消す。Application に属する SP は
// ErrApplicationOwned で拒否して残す。
func (a ServiceProviderAdministration) DeleteStandalone(ctx context.Context, tenantID, entityID string) error {
	sp, err := a.SPRepo.FindByEntityID(ctx, tenantID, entityID)
	if err != nil {
		return err
	}
	if sp != nil && sp.ApplicationID != "" {
		return ErrApplicationOwned
	}
	return a.SPRepo.Delete(ctx, tenantID, entityID)
}

// requireSigningCertificate は、AuthnRequest の署名を求める SP が読める証明書を持つことを確かめる。
func requireSigningCertificate(sp *samldomain.SamlServiceProvider) error {
	if !sp.WantAuthnRequestsSigned {
		return nil
	}
	if _, err := samldomain.ParseCertificatePEM(sp.AuthnRequestSigningCertificatePEM); err != nil {
		return samlports.ErrSigningCertificateRequired
	}
	return nil
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
