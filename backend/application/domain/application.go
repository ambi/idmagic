// Package domain は Application の aggregate 不変条件を所有する (wi-69)。
package domain

import (
	"errors"
)

var (
	ErrNameRequired        = errors.New("application name is required")
	ErrInvalidKind         = errors.New("invalid application kind")
	ErrInvalidStatus       = errors.New("invalid application status")
	ErrWeblinkLaunchURL    = errors.New("weblink application requires launch_url")
	ErrWeblinkHasProtocol  = errors.New("weblink application must not have a protocol")
	ErrProtocolRequired    = errors.New("application protocol is required")
	ErrServiceRequiresOIDC = errors.New("service application requires oidc protocol")
	ErrInvalidProtocolType = errors.New("invalid application protocol type")
	ErrOIDCClientID        = errors.New("oidc protocol requires client_id")
	ErrWsFedWtrealm        = errors.New("wsfed protocol requires wtrealm")
	ErrSAMLEntityID        = errors.New("saml protocol requires entity_id")
)

// ValidateApplication は Application aggregate の不変条件を検証する (wi-69)。
// weblink は launch_url 必須で protocol を持てない。federated は単一 protocol、
// service は OIDC protocol を必須とする。
func ValidateApplication(app *Application) error {
	if app.Name == "" {
		return ErrNameRequired
	}
	if !app.Kind.Valid() {
		return ErrInvalidKind
	}
	if !app.Status.Valid() {
		return ErrInvalidStatus
	}
	if app.Kind == ApplicationWeblink {
		if app.LaunchURL == "" {
			return ErrWeblinkLaunchURL
		}
		if app.Protocol != nil {
			return ErrWeblinkHasProtocol
		}
		return nil
	}
	if app.Protocol == nil {
		return ErrProtocolRequired
	}
	if app.Kind == ApplicationService && app.Protocol.Type != ApplicationProtocolOIDC {
		return ErrServiceRequiresOIDC
	}
	return ValidateProtocol(*app.Protocol)
}

// ValidateProtocol は Application が持つ単一 protocol 参照を検証する。
// oidc は client_id、wsfed は wtrealm を必須とする。
func ValidateProtocol(protocol ApplicationProtocol) error {
	if !protocol.Type.Valid() {
		return ErrInvalidProtocolType
	}
	switch protocol.Type {
	case ApplicationProtocolOIDC:
		if protocol.ClientID == "" {
			return ErrOIDCClientID
		}
	case ApplicationProtocolWsFed:
		if protocol.Wtrealm == "" {
			return ErrWsFedWtrealm
		}
	case ApplicationProtocolSAML:
		if protocol.EntityID == "" {
			return ErrSAMLEntityID
		}
	}
	return nil
}
