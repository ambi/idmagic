package usecases

// 第二要素 (MFA) の登録状態ユーティリティ (wi-26)。User.mfa_enrolled は
// 「TOTP factor または WebAuthn credential が存在する」で導出する。recovery code は backup
// 専用で単独の第二要素にはしない。TOTP / WebAuthn の解除後に残存要素へ応じて再計算する。

import (
	"context"
	"time"

	totpports "github.com/ambi/idmagic/backend/authentication/totp/ports"
	webauthnports "github.com/ambi/idmagic/backend/authentication/webauthn/ports"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// hasSecondFactor は sub が有効な TOTP factor または WebAuthn credential を持つかを返す。
// credRepo が nil の場合は WebAuthn を考慮しない (TOTP のみの旧経路との後方互換)。
func hasSecondFactor(
	ctx context.Context,
	mfaRepo totpports.MfaFactorRepository,
	credRepo webauthnports.WebAuthnCredentialRepository,
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

// SyncMfaEnrolled は残存する第二要素に応じて User.mfa_enrolled を再計算し、変化があれば
// 行を固定して読み直した最新の User に当てて保存する。user には保存した値を写す。
// totp/webauthn を横断するため mfa (このパッケージ) が所有し、export して webauthn 側からも
// package 境界を越えて呼べるようにする (Phase 2 と同方針)。
func SyncMfaEnrolled(
	ctx context.Context,
	userRepo userports.UserRepository,
	mfaRepo totpports.MfaFactorRepository,
	credRepo webauthnports.WebAuthnCredentialRepository,
	user *userdomain.User,
	now time.Time,
) error {
	enrolled, err := hasSecondFactor(ctx, mfaRepo, credRepo, user.ID)
	if err != nil {
		return err
	}
	updated, err := userRepo.Update(ctx, user.ID, func(latest *userdomain.User) (bool, error) {
		if latest.MfaEnrolled == enrolled {
			return false, nil
		}
		latest.MfaEnrolled = enrolled
		latest.UpdatedAt = now
		return true, nil
	})
	if err != nil {
		return err
	}
	if updated != nil {
		*user = *updated
	}
	return nil
}
