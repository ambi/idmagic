package domain

import (
	"errors"
	"fmt"
	"time"
)

const DefaultIDPProfileID = "default"

type IDPProfileMode string

const (
	IDPProfileModeShared    IDPProfileMode = "shared"
	IDPProfileModeDedicated IDPProfileMode = "dedicated"
)

var (
	ErrInvalidIDPProfile              = errors.New("invalid SAML identity provider profile")
	ErrDedicatedIDPProfileCardinality = errors.New("dedicated SAML identity provider profile can be assigned to only one service provider")
	ErrIDPProfileInUse                = errors.New("SAML identity provider profile is in use")
	ErrDefaultIDPProfile              = errors.New("default SAML identity provider profile cannot be changed or deleted")
)

func (m IDPProfileMode) Valid() bool {
	return m == IDPProfileModeShared || m == IDPProfileModeDedicated
}

type SamlIdentityProviderProfile struct {
	TenantID  string         `json:"tenant_id"`
	ProfileID string         `json:"profile_id"`
	Name      string         `json:"name"`
	Mode      IDPProfileMode `json:"mode"`
	IsDefault bool           `json:"is_default"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

func (p SamlIdentityProviderProfile) Validate(boundServiceProviderCount int) error {
	if p.TenantID == "" || p.ProfileID == "" || p.Name == "" || !p.Mode.Valid() {
		return ErrInvalidIDPProfile
	}
	if p.IsDefault != (p.ProfileID == DefaultIDPProfileID) {
		return ErrInvalidIDPProfile
	}
	if p.IsDefault && p.Mode != IDPProfileModeShared {
		return ErrInvalidIDPProfile
	}
	if boundServiceProviderCount < 0 {
		return fmt.Errorf("%w: negative service provider count", ErrInvalidIDPProfile)
	}
	if p.Mode == IDPProfileModeDedicated && boundServiceProviderCount > 1 {
		return ErrDedicatedIDPProfileCardinality
	}
	return nil
}
