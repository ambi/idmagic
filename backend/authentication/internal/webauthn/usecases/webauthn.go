package usecases

// WebAuthn / Passkey ceremony (wi-26)。core ceremony は go-webauthn に委ね、
// この層は spec 型 <-> go-webauthn 型の変換と challenge の永続化のみを担う。credential は
// MfaFactor と別集合 (identity=credential_id) で、1 ユーザーが複数持てる。

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/go-webauthn/webauthn/protocol"
	gowebauthn "github.com/go-webauthn/webauthn/webauthn"

	totpports "github.com/ambi/idmagic/backend/authentication/totp/ports"
	trusteddeviceports "github.com/ambi/idmagic/backend/authentication/trusteddevice/ports"
	"github.com/ambi/idmagic/backend/authentication/webauthn/domain"
	webauthnports "github.com/ambi/idmagic/backend/authentication/webauthn/ports"
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

// RelyingParty は構築済みの WebAuthn の RP である。go-webauthn の型をこのパッケージの外へ
// 出さないために包み、セレモニーの操作はこのパッケージのユースケースだけが行う。
type RelyingParty struct {
	rp *gowebauthn.WebAuthn
}

// NewRelyingParty は config から RP を構築する。RP ID / origin が無ければ
// ErrWebAuthnNotConfigured を返し、呼び出し側は WebAuthn を無効として扱う。
func NewRelyingParty(cfg WebAuthnConfig) (*RelyingParty, error) {
	if cfg.RPID == "" || len(cfg.RPOrigins) == 0 {
		return nil, ErrWebAuthnNotConfigured
	}
	rp, err := gowebauthn.New(&gowebauthn.Config{
		RPID:          cfg.RPID,
		RPDisplayName: cfg.RPDisplayName,
		RPOrigins:     cfg.RPOrigins,
	})
	if err != nil {
		return nil, err
	}
	return &RelyingParty{rp: rp}, nil
}

// WebAuthnDeps は WebAuthn use case の依存。RP が nil の場合 WebAuthn は未設定として扱う。
type WebAuthnDeps struct {
	RP             *RelyingParty
	UserRepo       userports.UserRepository
	CredentialRepo webauthnports.WebAuthnCredentialRepository
	MfaFactorRepo  totpports.MfaFactorRepository
	SessionStore   webauthnports.WebAuthnSessionStore
	// TrustedDevices は資格情報の登録と解除の後に記憶済みの端末をすべて失効させる
	// (REQ-AUTHENTICATION-028)。登録の完了と解除では必須で、ログインとステップアップの儀式は使わない。
	TrustedDevices trusteddeviceports.UserDeviceRevoker
	Emit           func(spec.DomainEvent)
}

// saveCeremonySession は go-webauthn の SessionData を JSON にして保存する。この JSON の形は
// デプロイの前後で読み書きされるため、go-webauthn の JSON タグのまま変えない。
func saveCeremonySession(
	ctx context.Context, store webauthnports.WebAuthnSessionStore, key string, data *gowebauthn.SessionData,
) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return store.Save(ctx, key, webauthnports.WebAuthnSession{Payload: payload}, time.Now().Add(webAuthnChallengeTTL))
}

// takeCeremonySession は保存したセッションを一度だけ取り出して SessionData に戻す。
// 無い、期限切れ、消費済みなら ErrWebAuthnChallengeMissing を返す。
func takeCeremonySession(
	ctx context.Context, store webauthnports.WebAuthnSessionStore, key string,
) (*gowebauthn.SessionData, error) {
	session, err := store.Take(ctx, key)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, ErrWebAuthnChallengeMissing
	}
	var data gowebauthn.SessionData
	if err := json.Unmarshal(session.Payload, &data); err != nil {
		return nil, err
	}
	return &data, nil
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
