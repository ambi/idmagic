package usecases

// 第二要素 (MFA) の登録状態ユーティリティ (wi-26 / ADR-087)。User.mfa_enrolled は
// 「TOTP factor または WebAuthn credential が存在する」で導出する。recovery code は backup
// 専用で単独の第二要素にはしない。TOTP / WebAuthn の解除後に残存要素へ応じて再計算する。

import (
	"context"
	"time"

	authnports "github.com/ambi/idmagic/backend/authentication/ports"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// hasSecondFactor は sub が有効な TOTP factor または WebAuthn credential を持つかを返す。
// credRepo が nil の場合は WebAuthn を考慮しない (TOTP のみの旧経路との後方互換)。
func hasSecondFactor(
	ctx context.Context,
	mfaRepo authnports.MfaFactorRepository,
	credRepo authnports.WebAuthnCredentialRepository,
	sub string,
) (bool, error) {
	factor, err := mfaRepo.Find(ctx, sub, spec.MfaFactorTOTP)
	if err != nil {
		return false, err
	}
	if factor != nil && factor.Secret != nil && *factor.Secret != "" {
		return true, nil
	}
	if credRepo == nil {
		return false, nil
	}
	credentials, err := credRepo.ListBySub(ctx, sub)
	if err != nil {
		return false, err
	}
	return len(credentials) > 0, nil
}

// syncMfaEnrolled は残存する第二要素に応じて User.mfa_enrolled を再計算し、変化があれば保存する。
func syncMfaEnrolled(
	ctx context.Context,
	userRepo userports.UserRepository,
	mfaRepo authnports.MfaFactorRepository,
	credRepo authnports.WebAuthnCredentialRepository,
	user *userdomain.User,
	now time.Time,
) error {
	enrolled, err := hasSecondFactor(ctx, mfaRepo, credRepo, user.ID)
	if err != nil {
		return err
	}
	if user.MfaEnrolled == enrolled {
		return nil
	}
	user.MfaEnrolled = enrolled
	user.UpdatedAt = now
	return userRepo.Save(ctx, user)
}
