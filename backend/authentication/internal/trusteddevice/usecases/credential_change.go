package usecases

import (
	"context"
	"errors"
	"time"

	"github.com/ambi/idmagic/backend/authentication/trusteddevice/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// ErrTrustedDeviceRevokerMissing は、資格情報を変えるユースケースの依存に失効の契約が渡されていない
// ことを表す。失効だけを黙って省いたまま資格情報を変えないよう、ユースケースは最初の作用より前に拒否する。
var ErrTrustedDeviceRevokerMissing = errors.New("trusted device revoker is not wired")

// RequireRevoker は、REQ-AUTHENTICATION-028 の契機となるユースケースが入口で呼び、失効の契約が
// 渡されていなければ ErrTrustedDeviceRevokerMissing を返す。
func RequireRevoker(revoker ports.UserDeviceRevoker) error {
	if revoker == nil {
		return ErrTrustedDeviceRevokerMissing
	}
	return nil
}

// RevokeOnCredentialChange は、資格情報を変えたユースケースが自身の確定の後に呼び、対象の User の
// 記憶済みの端末をすべて失効させる (REQ-AUTHENTICATION-028)。TrustedDeviceRevoked は呼び出し側の
// emit で発行する。失効したすべての端末のイベントを発行してから、発行の最初のエラーを返す。
// emit が nil なら発行を省く。
func RevokeOnCredentialChange(
	ctx context.Context,
	revoker ports.UserDeviceRevoker,
	emit func(spec.DomainEvent) error,
	tenantID, userID string,
	reason spec.TrustedDeviceRevokeReason,
	now time.Time,
) error {
	if err := RequireRevoker(revoker); err != nil {
		return err
	}
	events, err := revoker.RevokeAllForUser(ctx, tenantID, userID, reason, now)
	if err != nil {
		return err
	}
	if emit == nil {
		return nil
	}
	var emitErr error
	for _, event := range events {
		if err := emit(event); err != nil && emitErr == nil {
			emitErr = err
		}
	}
	return emitErr
}

// EmitWithoutError は、失敗を返さない発行先を RevokeOnCredentialChange の emit に合わせる。
func EmitWithoutError(emit func(spec.DomainEvent)) func(spec.DomainEvent) error {
	if emit == nil {
		return nil
	}
	return func(event spec.DomainEvent) error {
		emit(event)
		return nil
	}
}
