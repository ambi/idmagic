package ports

import (
	"context"
	"errors"
	"time"

	claimdomain "github.com/ambi/idmagic/backend/claimmapping/domain"
	"github.com/ambi/idmagic/backend/saml/domain"
)

var (
	ErrServiceProviderNotFound = errors.New("service provider does not exist")
	// ErrSigningCertificateRequired は、AuthnRequest の署名を必須にした SP に検証できる証明書がないことを表す。
	ErrSigningCertificateRequired = errors.New("authn request signing certificate is required")
)

// ServiceProviderAdministration は、Application のカタログが自身に属する SP を作成、更新、
// 関連付け、削除する操作である。既定値と署名の設定の検査は Saml が保つ。
// IdP プロファイルの規則違反は、保存のエラー（domain.ErrInvalidIDPProfile など）としてそのまま返す。
type ServiceProviderAdministration interface {
	// Find は entityID に一致する SP を返す。存在しなければ (nil, nil)。
	Find(ctx context.Context, tenantID, entityID string) (*domain.SamlServiceProvider, error)
	CreateForApplication(ctx context.Context, in CreateApplicationServiceProviderInput) error
	// Update は指定した項目だけを変える。SP がなければ ErrServiceProviderNotFound を返す。
	Update(ctx context.Context, in UpdateServiceProviderInput) error
	// BindApplication は、カタログの作成後に保存側の関連付けを同期する。SP がなければ何もしない。
	BindApplication(ctx context.Context, tenantID, entityID, applicationID string) error
	// DeleteForApplication は、カタログが所有する SP を Application の削除の後に消す。
	DeleteForApplication(ctx context.Context, tenantID, entityID string) error
}

// CreateApplicationServiceProviderInput の IdP プロファイルと NameID の項目は、空なら Saml の既定値を使う。
type CreateApplicationServiceProviderInput struct {
	TenantID                          string
	EntityID                          string
	DisplayName                       string
	IDPProfileID                      string
	ACSURLs                           []string
	SLOURL                            string
	NameIDFormat                      string
	NameIDSource                      string
	SignResponse                      bool
	WantAuthnRequestsSigned           bool
	AuthnRequestSigningCertificatePEM string
	Now                               time.Time
}

type UpdateServiceProviderInput struct {
	TenantID                          string
	EntityID                          string
	IDPProfileID                      *string
	ACSURLs                           *[]string
	SLOURL                            *string
	Audience                          *string
	NameIDFormat                      *string
	NameIDSource                      *string
	SignAssertion                     *bool
	SignResponse                      *bool
	WantAuthnRequestsSigned           *bool
	AuthnRequestSigningCertificatePEM *string
	Rules                             *[]claimdomain.ClaimMappingRule
	Now                               time.Time
}
