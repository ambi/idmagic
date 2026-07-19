package usecases

// WebAuthn / Passkey ceremony (wi-26 / ADR-087)。core ceremony は go-webauthn に委ね、
// この層は spec 型 <-> go-webauthn 型の変換と challenge の永続化のみを担う。credential は
// MfaFactor と別集合 (identity=credential_id) で、1 ユーザーが複数持てる。

import (
	"encoding/base64"
	"errors"
	"time"

	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/go-webauthn/webauthn/protocol"
	gowebauthn "github.com/go-webauthn/webauthn/webauthn"

	"github.com/ambi/idmagic/backend/authentication/domain"
	authnports "github.com/ambi/idmagic/backend/authentication/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

var (
	// ErrWebAuthnNotConfigured は RP ID / origin 未設定で RP を構築できない場合。
	ErrWebAuthnNotConfigured = errors.New("webauthn is not configured")
	// ErrWebAuthnChallengeMissing は challenge が期限切れ / 未発行 / 消費済みの場合。
	ErrWebAuthnChallengeMissing = errors.New("webauthn challenge not found")
	// ErrWebAuthnVerification は attestation / assertion の検証に失敗した場合。
	ErrWebAuthnVerification = errors.New("webauthn verification failed")
	// ErrWebAuthnCredentialCloned は sign_count 逆行で clone が疑われる場合。
	ErrWebAuthnCredentialCloned = errors.New("webauthn credential may be cloned")
	// ErrWebAuthnCredentialNotFound は解除対象 credential が存在しない / 別ユーザーの場合。
	ErrWebAuthnCredentialNotFound = errors.New("webauthn credential not found")
	// ErrWebAuthnNoCredential はログイン時に対象ユーザーの credential が無い場合。
	ErrWebAuthnNoCredential = errors.New("no webauthn credential enrolled")
)

// challenge TTL。WebAuthnPolicy の timeout_seconds に合わせる。
const webAuthnChallengeTTL = 120 * time.Second

// WebAuthnConfig は RP (Relying Party) 設定。RP ID / origins は deployment config 由来。
type WebAuthnConfig struct {
	RPID          string
	RPDisplayName string
	RPOrigins     []string
}

// NewWebAuthn は config から go-webauthn の RP を構築する。RP ID / origin が無ければ
// ErrWebAuthnNotConfigured を返し、呼び出し側は WebAuthn を無効として扱う。
func NewWebAuthn(cfg WebAuthnConfig) (*gowebauthn.WebAuthn, error) {
	if cfg.RPID == "" || len(cfg.RPOrigins) == 0 {
		return nil, ErrWebAuthnNotConfigured
	}
	return gowebauthn.New(&gowebauthn.Config{
		RPID:          cfg.RPID,
		RPDisplayName: cfg.RPDisplayName,
		RPOrigins:     cfg.RPOrigins,
	})
}

// WebAuthnDeps は WebAuthn use case の依存。RP が nil の場合 WebAuthn は未設定として扱う。
type WebAuthnDeps struct {
	RP             *gowebauthn.WebAuthn
	UserRepo       userports.UserRepository
	CredentialRepo authnports.WebAuthnCredentialRepository
	MfaFactorRepo  authnports.MfaFactorRepository
	SessionStore   authnports.WebAuthnSessionStore
	Emit           func(spec.DomainEvent)
}

// webauthnUser は userdomain.User + 登録済み credential を go-webauthn の User interface に適合させる。
// WebAuthnID は user handle として sub (UUID 文字列) の byte 列を使う。
type webauthnUser struct {
	user        *userdomain.User
	credentials []gowebauthn.Credential
}

func (u *webauthnUser) WebAuthnID() []byte   { return []byte(u.user.ID) }
func (u *webauthnUser) WebAuthnName() string { return u.user.PreferredUsername }

func (u *webauthnUser) WebAuthnDisplayName() string {
	if u.user.Name != nil && *u.user.Name != "" {
		return *u.user.Name
	}
	return u.user.PreferredUsername
}

func (u *webauthnUser) WebAuthnCredentials() []gowebauthn.Credential { return u.credentials }

// loadWebAuthnUser は user と登録済み credential をまとめて go-webauthn の User に適合させる。
func loadWebAuthnUser(user *userdomain.User, stored []*domain.WebAuthnCredential) (*webauthnUser, error) {
	credentials := make([]gowebauthn.Credential, 0, len(stored))
	for _, c := range stored {
		converted, err := toWebAuthnCredential(c)
		if err != nil {
			return nil, err
		}
		credentials = append(credentials, converted)
	}
	return &webauthnUser{user: user, credentials: credentials}, nil
}

// toWebAuthnCredential は spec.WebAuthnCredential を go-webauthn の Credential へ変換する。
func toWebAuthnCredential(c *domain.WebAuthnCredential) (gowebauthn.Credential, error) {
	id, err := base64.RawURLEncoding.DecodeString(c.CredentialID)
	if err != nil {
		return gowebauthn.Credential{}, err
	}
	publicKey, err := base64.RawURLEncoding.DecodeString(c.PublicKey)
	if err != nil {
		return gowebauthn.Credential{}, err
	}
	var aaguid []byte
	if c.AAGUID != nil {
		if aaguid, err = base64.RawURLEncoding.DecodeString(*c.AAGUID); err != nil {
			return gowebauthn.Credential{}, err
		}
	}
	transports := make([]protocol.AuthenticatorTransport, 0, len(c.Transports))
	for _, t := range c.Transports {
		transports = append(transports, protocol.AuthenticatorTransport(t))
	}
	return gowebauthn.Credential{
		ID:        id,
		PublicKey: publicKey,
		Transport: transports,
		Flags: gowebauthn.CredentialFlags{
			BackupEligible: c.BackupEligible,
			BackupState:    c.BackupState,
		},
		Authenticator: gowebauthn.Authenticator{
			AAGUID:    aaguid,
			SignCount: c.SignCount,
		},
	}, nil
}

// fromWebAuthnCredential は go-webauthn の Credential を spec.WebAuthnCredential へ変換する。
func fromWebAuthnCredential(userID string, c *gowebauthn.Credential, label *string, now time.Time) *domain.WebAuthnCredential {
	transports := make([]string, 0, len(c.Transport))
	for _, t := range c.Transport {
		transports = append(transports, string(t))
	}
	var aaguid *string
	if len(c.Authenticator.AAGUID) > 0 {
		encoded := base64.RawURLEncoding.EncodeToString(c.Authenticator.AAGUID)
		aaguid = &encoded
	}
	return &domain.WebAuthnCredential{
		CredentialID:   base64.RawURLEncoding.EncodeToString(c.ID),
		UserID:         userID,
		PublicKey:      base64.RawURLEncoding.EncodeToString(c.PublicKey),
		SignCount:      c.Authenticator.SignCount,
		Transports:     transports,
		AAGUID:         aaguid,
		Label:          label,
		BackupEligible: c.Flags.BackupEligible,
		BackupState:    c.Flags.BackupState,
		CreatedAt:      now,
	}
}
