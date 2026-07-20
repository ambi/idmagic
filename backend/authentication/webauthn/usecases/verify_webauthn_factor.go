package usecases

// ログイン / step-up 時の WebAuthn assertion 検証 (wi-26 / ADR-087)。challenge は
// challengeKey (pending login session id) にひもづけて発行・消費する。sign_count 逆行は
// clone の疑いとして拒否する。

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	gowebauthn "github.com/go-webauthn/webauthn/webauthn"

	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
	"github.com/ambi/idmagic/backend/authentication/webauthn/domain"
)

const webAuthnLoginKeyPrefix = "login:"

// BeginWebAuthnAssertion はログインの assertion challenge を発行し、challengeKey に保存する。
// 対象ユーザーに credential が無ければ ErrWebAuthnNoCredential。
func BeginWebAuthnAssertion(
	ctx context.Context,
	deps WebAuthnDeps,
	challengeKey string,
	sub string,
) (*protocol.CredentialAssertion, error) {
	if deps.RP == nil {
		return nil, ErrWebAuthnNotConfigured
	}
	user, err := authusecases.LoadSelfUser(ctx, deps.UserRepo, sub)
	if err != nil {
		return nil, err
	}
	stored, err := deps.CredentialRepo.ListBySub(ctx, sub)
	if err != nil {
		return nil, err
	}
	if len(stored) == 0 {
		return nil, ErrWebAuthnNoCredential
	}
	waUser, err := loadWebAuthnUser(user, stored)
	if err != nil {
		return nil, err
	}
	assertion, session, err := deps.RP.BeginLogin(
		waUser,
		gowebauthn.WithUserVerification(protocol.VerificationPreferred),
	)
	if err != nil {
		return nil, err
	}
	if err := deps.SessionStore.Save(ctx, webAuthnLoginKeyPrefix+challengeKey, *session, time.Now().Add(webAuthnChallengeTTL)); err != nil {
		return nil, err
	}
	return assertion, nil
}

// FinishWebAuthnAssertion は assertion を検証し、成功時に sign_count / last_used_at を更新して
// 検証済み credential を返す。clone 検出時は ErrWebAuthnCredentialCloned。
func FinishWebAuthnAssertion(
	ctx context.Context,
	deps WebAuthnDeps,
	challengeKey string,
	sub string,
	body []byte,
	now time.Time,
) (*domain.WebAuthnCredential, error) {
	if deps.RP == nil {
		return nil, ErrWebAuthnNotConfigured
	}
	now = authusecases.NormalizedNow(now)
	user, err := authusecases.LoadSelfUser(ctx, deps.UserRepo, sub)
	if err != nil {
		return nil, err
	}
	session, err := deps.SessionStore.Take(ctx, webAuthnLoginKeyPrefix+challengeKey)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, ErrWebAuthnChallengeMissing
	}
	stored, err := deps.CredentialRepo.ListBySub(ctx, sub)
	if err != nil {
		return nil, err
	}
	if len(stored) == 0 {
		return nil, ErrWebAuthnNoCredential
	}
	waUser, err := loadWebAuthnUser(user, stored)
	if err != nil {
		return nil, err
	}
	parsed, err := protocol.ParseCredentialRequestResponseBody(bytes.NewReader(body))
	if err != nil {
		return nil, errors.Join(ErrWebAuthnVerification, err)
	}
	credential, err := deps.RP.ValidateLogin(waUser, *session, parsed)
	if err != nil {
		return nil, errors.Join(ErrWebAuthnVerification, err)
	}
	if credential.Authenticator.CloneWarning {
		return nil, ErrWebAuthnCredentialCloned
	}
	credentialID := base64.RawURLEncoding.EncodeToString(credential.ID)
	if err := deps.CredentialRepo.UpdateSignCount(ctx, credentialID, credential.Authenticator.SignCount, now); err != nil {
		return nil, err
	}
	return deps.CredentialRepo.FindByCredentialID(ctx, credentialID)
}
