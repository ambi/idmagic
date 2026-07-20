package domain

import (
	authorizationdomain "github.com/ambi/idmagic/backend/oauth2/authorization/domain"
	clientdomain "github.com/ambi/idmagic/backend/oauth2/client/domain"
	consentdomain "github.com/ambi/idmagic/backend/oauth2/consent/domain"
	devicedomain "github.com/ambi/idmagic/backend/oauth2/device/domain"
	tokendomain "github.com/ambi/idmagic/backend/oauth2/token/domain"
)

// Kept package-local for the historical root-package contract tests. Feature
// implementations own the corresponding behavior.
const (
	userCodeCharset         = "BCDFGHJKLMNPQRSTVWXZ"
	refreshTokenTTL         = 14 * 24 * 60 * 60 * 1e9
	refreshTokenAbsoluteTTL = 30 * 24 * 60 * 60 * 1e9
)

// Feature domain types are re-exported while callers migrate to feature paths.
type (
	OAuth2Client                      = clientdomain.OAuth2Client
	ClientSecretCredential            = clientdomain.ClientSecretCredential
	TokenEndpointAuthMethod           = clientdomain.TokenEndpointAuthMethod
	FapiProfile                       = clientdomain.FapiProfile
	ClientRegistered                  = clientdomain.ClientRegistered
	AdminOAuth2ClientCreated          = clientdomain.AdminOAuth2ClientCreated
	AdminOAuth2ClientUpdated          = clientdomain.AdminOAuth2ClientUpdated
	AdminOAuth2ClientDeleted          = clientdomain.AdminOAuth2ClientDeleted
	ClientSecretRotated               = clientdomain.ClientSecretRotated
	Consent                           = consentdomain.Consent
	ConsentState                      = consentdomain.ConsentState
	ConsentGrantedEvent               = consentdomain.ConsentGrantedEvent
	ConsentRevokedEvent               = consentdomain.ConsentRevokedEvent
	AuthorizationRequest              = authorizationdomain.AuthorizationRequest
	AuthorizationCodeRecord           = authorizationdomain.AuthorizationCodeRecord
	AuthorizationCodeInput            = authorizationdomain.AuthorizationCodeInput
	AuthorizationDetailType           = authorizationdomain.AuthorizationDetailType
	AuthorizationDetailTypeState      = authorizationdomain.AuthorizationDetailTypeState
	AuthorizationDetailFieldSemantics = authorizationdomain.AuthorizationDetailFieldSemantics
	AuthorizationDetailFieldRule      = authorizationdomain.AuthorizationDetailFieldRule
	AuthorizationDetailsSchema        = authorizationdomain.AuthorizationDetailsSchema
	PromptTokens                      = authorizationdomain.PromptTokens
	AuthorizationRequestPolicy        = authorizationdomain.AuthorizationRequestPolicy
	PARRecord                         = authorizationdomain.PARRecord
	DeviceAuthorization               = devicedomain.DeviceAuthorization
	SenderConstraint                  = tokendomain.SenderConstraint
	RefreshTokenRecord                = tokendomain.RefreshTokenRecord
	AccessTokenClaims                 = tokendomain.AccessTokenClaims
	IDTokenClaims                     = tokendomain.IDTokenClaims
	GeneratedRefreshToken             = tokendomain.GeneratedRefreshToken
)

const (
	AuthMethodClientSecretBasic = clientdomain.AuthMethodClientSecretBasic
	AuthMethodClientSecretPost  = clientdomain.AuthMethodClientSecretPost
	AuthMethodPrivateKeyJwt     = clientdomain.AuthMethodPrivateKeyJwt
	AuthMethodTlsClientAuth     = clientdomain.AuthMethodTlsClientAuth
	AuthMethodNone              = clientdomain.AuthMethodNone
	FapiNone                    = clientdomain.FapiNone
	FapiSecurityProfileV2       = clientdomain.FapiSecurityProfileV2
	ConsentGranted              = consentdomain.ConsentGranted
	ConsentRevoked              = consentdomain.ConsentRevoked
	ConsentExpired              = consentdomain.ConsentExpired
	SenderConstraintDPoP        = tokendomain.SenderConstraintDPoP
	SenderConstraintMTLS        = tokendomain.SenderConstraintMTLS
	DetailTypeEnabled           = authorizationdomain.DetailTypeEnabled
	DetailTypeDisabled          = authorizationdomain.DetailTypeDisabled
	DetailFieldSet              = authorizationdomain.DetailFieldSet
	DetailFieldAtMost           = authorizationdomain.DetailFieldAtMost
	DetailFieldEnum             = authorizationdomain.DetailFieldEnum
	DetailFieldExact            = authorizationdomain.DetailFieldExact
	DeviceCodeTTL               = devicedomain.DeviceCodeTTL
)

var (
	HashClientSecret              = clientdomain.HashClientSecret
	VerifyClientSecret            = clientdomain.VerifyClientSecret
	GenerateAuthorizationCode     = authorizationdomain.GenerateAuthorizationCode
	IsCodeExpired                 = authorizationdomain.IsCodeExpired
	IsCodeRedeemed                = authorizationdomain.IsCodeRedeemed
	VerifyPKCES256                = authorizationdomain.VerifyPKCES256
	ParsePromptTokens             = authorizationdomain.ParsePromptTokens
	NeedsReauthentication         = authorizationdomain.NeedsReauthentication
	ParsePrompt                   = authorizationdomain.ParsePrompt
	ScopeIntersection             = authorizationdomain.ScopeIntersection
	ValidateAgainstType           = authorizationdomain.ValidateAgainstType
	DetailsSubsetOf               = authorizationdomain.DetailsSubsetOf
	DetailTypes                   = authorizationdomain.DetailTypes
	GenerateDeviceCode            = devicedomain.GenerateDeviceCode
	HashDeviceCode                = devicedomain.HashDeviceCode
	GenerateUserCode              = devicedomain.GenerateUserCode
	NormalizeUserCode             = devicedomain.NormalizeUserCode
	IsDeviceExpired               = devicedomain.IsDeviceExpired
	HashRefreshToken              = tokendomain.HashRefreshToken
	GenerateInitialRefreshToken   = tokendomain.GenerateInitialRefreshToken
	RotateRefreshToken            = tokendomain.RotateRefreshToken
	IsRefreshTokenReplay          = tokendomain.IsRefreshTokenReplay
	IsRefreshTokenAbsoluteExpired = tokendomain.IsRefreshTokenAbsoluteExpired
)
