package authentication

import (
	"context"
	"errors"
	"time"

	mfausecases "github.com/ambi/idmagic/backend/authentication/internal/mfa/usecases"
	totpdomain "github.com/ambi/idmagic/backend/authentication/totp/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// EnsureSeedPasswordHistory は User の現資格情報を検証し、途中失敗で欠けた初期履歴だけを補う。
func (m Module) EnsureSeedPasswordHistory(ctx context.Context, users userports.UserRepository, userID, password string, now time.Time) error {
	user, err := users.FindBySub(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("seed user does not exist")
	}
	matches, err := m.PasswordHasher.Verify(password, user.PasswordHash)
	if err != nil || !matches {
		return errors.New("seed password does not match current credential")
	}
	history, err := m.PasswordHistoryRepo.Recent(ctx, userID, 24)
	if err != nil {
		return err
	}
	for _, entry := range history {
		if matches, err := m.PasswordHasher.Verify(password, entry.Encoded); err == nil && matches {
			return nil
		}
	}
	if len(history) != 0 {
		return errors.New("seed password history conflicts with current credential")
	}
	return m.PasswordHistoryRepo.Add(ctx, userID, user.PasswordHash, now)
}

func (m Module) CreateSeedTOTP(ctx context.Context, users userports.UserRepository, factor *totpdomain.MfaFactor, emit func(spec.DomainEvent)) error {
	return mfausecases.SeedTOTP(ctx, mfausecases.AccountMfaDeps{
		UserRepo: users, MfaFactorRepo: m.MfaFactorRepo, CredentialRepo: m.WebAuthnCredentialRepo,
		TrustedDevices: m.UserDeviceRevoker(), Emit: emit,
	}, factor)
}
