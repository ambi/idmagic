package domain

import (
	approvaldomain "github.com/ambi/idmagic/backend/oauth2/internal/approval/domain"
	authorizationdomain "github.com/ambi/idmagic/backend/oauth2/internal/authorization/domain"
	clientdomain "github.com/ambi/idmagic/backend/oauth2/internal/client/domain"
	consentdomain "github.com/ambi/idmagic/backend/oauth2/internal/consent/domain"
	devicedomain "github.com/ambi/idmagic/backend/oauth2/internal/device/domain"
	logoutdomain "github.com/ambi/idmagic/backend/oauth2/internal/logout/domain"
	tokendomain "github.com/ambi/idmagic/backend/oauth2/internal/token/domain"
)

// Kept package-local for the historical root-package contract tests. Feature
// implementations own the corresponding behavior.
const (
	userCodeCharset         = "BCDFGHJKLMNPQRSTVWXZ"
	refreshTokenTTL         = 14 * 24 * 60 * 60 * 1e9
	refreshTokenAbsoluteTTL = 30 * 24 * 60 * 60 * 1e9
)

// 機能スライスの型と操作のうち、ほかのモジュール、組み立て地点、公開ポートが使うものを公開する。
type (
	ApprovalRequest                   = approvaldomain.ApprovalRequest
	ClientSession                     = logoutdomain.ClientSession
	LogoutNotification                = logoutdomain.LogoutNotification
	LogoutNotificationState           = logoutdomain.LogoutNotificationState
	OAuth2Client                      = clientdomain.OAuth2Client
	ClientSecretCredential            = clientdomain.ClientSecretCredential
	ClientSecretCredentialStatus      = clientdomain.ClientSecretCredentialStatus
	TokenEndpointAuthMethod           = clientdomain.TokenEndpointAuthMethod
	FapiProfile                       = clientdomain.FapiProfile
	ClientRegistered                  = clientdomain.ClientRegistered
	AdminOAuth2ClientCreated          = clientdomain.AdminOAuth2ClientCreated
	AdminOAuth2ClientUpdated          = clientdomain.AdminOAuth2ClientUpdated
	AdminOAuth2ClientDeleted          = clientdomain.AdminOAuth2ClientDeleted
	ClientSecretRotated               = clientdomain.ClientSecretRotated
	ClientSecretIssued                = clientdomain.ClientSecretIssued
	ClientSecretRevoked               = clientdomain.ClientSecretRevoked
	Consent                           = consentdomain.Consent
	ConsentState                      = consentdomain.ConsentState
	ConsentGrantedEvent               = consentdomain.ConsentGrantedEvent
	ConsentRevokedEvent               = consentdomain.ConsentRevokedEvent
	AgentDelegationGranted            = consentdomain.AgentDelegationGranted
	AgentDelegationRevoked            = consentdomain.AgentDelegationRevoked
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
	ExchangeProof                     = tokendomain.ExchangeProof
	RefreshTokenRecord                = tokendomain.RefreshTokenRecord
	AccessTokenClaims                 = tokendomain.AccessTokenClaims
	IDTokenClaims                     = tokendomain.IDTokenClaims
	GeneratedRefreshToken             = tokendomain.GeneratedRefreshToken
)

const (
	AuthMethodClientSecretBasic   = clientdomain.AuthMethodClientSecretBasic
	AuthMethodClientSecretPost    = clientdomain.AuthMethodClientSecretPost
	AuthMethodPrivateKeyJwt       = clientdomain.AuthMethodPrivateKeyJwt
	AuthMethodTlsClientAuth       = clientdomain.AuthMethodTlsClientAuth
	AuthMethodNone                = clientdomain.AuthMethodNone
	ClientSecretCredentialActive  = clientdomain.ClientSecretCredentialActive
	ClientSecretCredentialExpired = clientdomain.ClientSecretCredentialExpired
	ClientSecretCredentialRevoked = clientdomain.ClientSecretCredentialRevoked
	FapiNone                      = clientdomain.FapiNone
	FapiSecurityProfileV2         = clientdomain.FapiSecurityProfileV2
	ConsentGranted                = consentdomain.ConsentGranted
	ConsentRevoked                = consentdomain.ConsentRevoked
	ConsentExpired                = consentdomain.ConsentExpired
	SenderConstraintDPoP          = tokendomain.SenderConstraintDPoP
	SenderConstraintMTLS          = tokendomain.SenderConstraintMTLS
	DetailTypeEnabled             = authorizationdomain.DetailTypeEnabled
	DetailTypeDisabled            = authorizationdomain.DetailTypeDisabled
	DetailFieldSet                = authorizationdomain.DetailFieldSet
	DetailFieldAtMost             = authorizationdomain.DetailFieldAtMost
	DetailFieldEnum               = authorizationdomain.DetailFieldEnum
	DetailFieldExact              = authorizationdomain.DetailFieldExact
	DeviceCodeTTL                 = devicedomain.DeviceCodeTTL
	ApprovalRequestMaxTTL         = approvaldomain.MaxTTL
	KindBackChannelLogoutDelivery = logoutdomain.KindBackChannelLogoutDelivery
	LogoutNotificationPending     = logoutdomain.LogoutNotificationPending
	LogoutNotificationDelivered   = logoutdomain.LogoutNotificationDelivered
	LogoutNotificationFailed      = logoutdomain.LogoutNotificationFailed
)

var (
	HashAuthReqID                 = approvaldomain.HashAuthReqID
	HashClientSecret              = clientdomain.HashClientSecret
	VerifyClientSecret            = clientdomain.VerifyClientSecret
	GenerateAuthorizationCode     = authorizationdomain.GenerateAuthorizationCode
	IsCodeExpired                 = authorizationdomain.IsCodeExpired
	IsCodeRedeemed                = authorizationdomain.IsCodeRedeemed
	VerifyPKCES256                = authorizationdomain.VerifyPKCES256
	RedirectURIAllowed            = authorizationdomain.RedirectURIAllowed
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
	ExchangeScopes                = tokendomain.ExchangeScopes
	RotateRefreshToken            = tokendomain.RotateRefreshToken
	IsRefreshTokenReplay          = tokendomain.IsRefreshTokenReplay
	IsRefreshTokenAbsoluteExpired = tokendomain.IsRefreshTokenAbsoluteExpired
	PresentationTokenType         = tokendomain.PresentationTokenType
	DecideExchangeBinding         = tokendomain.DecideExchangeBinding
	ReboundExchangeBinding        = tokendomain.ReboundExchangeBinding
)
