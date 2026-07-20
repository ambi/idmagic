package bootstrap

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"reflect"
	"time"

	claimdomain "github.com/ambi/idmagic/backend/claimmapping/domain"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	samldomain "github.com/ambi/idmagic/backend/saml/domain"
	samlports "github.com/ambi/idmagic/backend/saml/ports"
	seedingdomain "github.com/ambi/idmagic/backend/seeding/domain"
	"github.com/ambi/idmagic/backend/wsfederation/domain"
	wsfederationports "github.com/ambi/idmagic/backend/wsfederation/ports"
	samltoken "github.com/ambi/idmagic/backend/wsfederation/tokens_saml"
)

// newDevFederationSigner は開発用の自己署名 federation 署名証明書から署名器を作る。
// 本番の証明書ライフサイクル・ローテーション・metadata 掲載は後続スライス (ADR-060) で扱う。
func NewDevFederationSigner() (*samltoken.Signer, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("generate federation signing key: %w", err)
	}
	now := time.Now().UTC()
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(now.UnixNano()),
		Subject:      pkix.Name{CommonName: "idmagic federation signing (dev)"},
		NotBefore:    now.Add(-1 * time.Hour),
		NotAfter:     now.Add(365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return nil, fmt.Errorf("create federation signing certificate: %w", err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, fmt.Errorf("parse federation signing certificate: %w", err)
	}
	return samltoken.NewSigner(cert, key)
}

// seedWsFedRelyingParty は WS-Federation passive のデモ用 relying party を投入する。
func SeedWsFedRelyingParty(ctx context.Context, repo wsfederationports.WsFedRelyingPartyRepository, seed seedingdomain.DevelopmentDemoSeed) error {
	now := time.Now().UTC()
	rp := &domain.WsFedRelyingParty{
		TenantID:    tenancydomain.DefaultTenantID,
		Wtrealm:     seed.WsFedRealm,
		DisplayName: seed.WsFedDisplayName,
		ReplyURLs:   seed.WsFedReplyURLs,
		ClaimPolicy: claimdomain.ClaimMappingPolicy{
			NameID: claimdomain.NameIdConfiguration{
				Format:          "urn:oasis:names:tc:SAML:2.0:nameid-format:persistent",
				SourceAttribute: "sub",
			},
			Rules: []claimdomain.ClaimMappingRule{
				{ClaimType: "http://schemas.xmlsoap.org/claims/UPN", Source: claimdomain.ClaimSourceUserAttribute, SourceKey: "preferred_username", Required: true},
				{ClaimType: "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress", Source: claimdomain.ClaimSourceUserAttribute, SourceKey: "email"},
			},
		},
		CreatedAt: now,
	}
	existing, err := repo.FindByWtrealm(ctx, rp.TenantID, rp.Wtrealm)
	if err != nil {
		return err
	}
	if existing == nil {
		return repo.Save(ctx, rp)
	}
	if !sameWsFedRelyingParty(existing, rp) {
		return fmt.Errorf("seed drift at wsfed-rp:%s", rp.Wtrealm)
	}
	return nil
}

// seedSamlServiceProvider は SAML 2.0 Web Browser SSO のデモ用 service provider を投入する。
func SeedSamlServiceProvider(ctx context.Context, repo samlports.SamlServiceProviderRepository, seed seedingdomain.DevelopmentDemoSeed) error {
	now := time.Now().UTC()
	sp := &samldomain.SamlServiceProvider{
		TenantID:    tenancydomain.DefaultTenantID,
		EntityID:    seed.SamlEntityID,
		DisplayName: seed.SamlDisplayName,
		ACSURLs:     seed.SamlACSURLs,
		ClaimPolicy: claimdomain.ClaimMappingPolicy{
			NameID: claimdomain.NameIdConfiguration{
				Format:          samldomain.SamlNameIDFormatPersistent,
				SourceAttribute: "sub",
			},
			Rules: []claimdomain.ClaimMappingRule{
				{ClaimType: "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress", Source: claimdomain.ClaimSourceUserAttribute, SourceKey: "email"},
				{ClaimType: "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name", Source: claimdomain.ClaimSourceUserAttribute, SourceKey: "preferred_username", Required: true},
			},
		},
		SignAssertion: true,
		CreatedAt:     now,
	}
	existing, err := repo.FindByEntityID(ctx, sp.TenantID, sp.EntityID)
	if err != nil {
		return err
	}
	if existing == nil {
		return repo.Save(ctx, sp)
	}
	if !sameSamlServiceProvider(existing, sp) {
		return fmt.Errorf("seed drift at saml-sp:%s", sp.EntityID)
	}
	return nil
}

func sameWsFedRelyingParty(actual, desired *domain.WsFedRelyingParty) bool {
	left, right := *actual, *desired
	left.CreatedAt, right.CreatedAt = time.Time{}, time.Time{}
	return reflect.DeepEqual(left, right)
}

func sameSamlServiceProvider(actual, desired *samldomain.SamlServiceProvider) bool {
	left, right := *actual, *desired
	left.CreatedAt, right.CreatedAt = time.Time{}, time.Time{}
	return reflect.DeepEqual(left, right)
}
