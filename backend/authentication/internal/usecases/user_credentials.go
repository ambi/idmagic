package usecases

import "context"

type credentialDataPurger interface {
	DeleteAllForSub(context.Context, string) error
}

// UserCredentials は、IdManagement の完全削除が要求する認証情報の区間である。
// nil の保存先は既存の部分構成を表す。操作そのものの未配線とは区別する。
type UserCredentials struct {
	PasswordHistory credentialDataPurger
	TOTP            credentialDataPurger
	TrustedDevices  credentialDataPurger
	WebAuthn        credentialDataPurger
	RecoveryCodes   credentialDataPurger
}

func (d UserCredentials) PurgeUserCredentials(ctx context.Context, sub string) error {
	for _, store := range []credentialDataPurger{d.PasswordHistory, d.TOTP, d.TrustedDevices, d.WebAuthn, d.RecoveryCodes} {
		if store != nil {
			if err := store.DeleteAllForSub(ctx, sub); err != nil {
				return err
			}
		}
	}
	return nil
}
