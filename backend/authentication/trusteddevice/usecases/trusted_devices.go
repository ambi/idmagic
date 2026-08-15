// Package usecases は信頼済みデバイスの発行・評価・失効を持つ (wi-91)。
// 発行はログインで本物の第二要素が成立した直後だけ、評価はサインインポリシーが MFA を
// 要求していてセッションがまだ第二要素を持たない時だけ行う。
package usecases

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/ambi/idmagic/backend/authentication/trusteddevice/domain"
	"github.com/ambi/idmagic/backend/authentication/trusteddevice/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// AMRTrustedDevice は「要素を提示したのではなく端末が記憶されていた」ことを表す
// AMR 値。RFC 8176 の登録値ではなく、rc と同じくこのアプリケーション固有の拡張値である。
const AMRTrustedDevice = "tdev"

// RememberableFactors は記憶の起点として認める第二要素。復旧コード (rc) は要素を失った
// ときの経路であり、その状況の端末を長期の信頼に足るものとして扱えないので含めない。
var RememberableFactors = []string{"otp", "webauthn"}

// ErrTrustedDeviceNotFound は対象のデバイスが存在しない、または本人のものでない。
var ErrTrustedDeviceNotFound = errors.New("trusted device not found")

// Deps は信頼済みデバイスの use case が要る依存。Repo が nil なら機能全体を無効として
// 扱い、発行も評価も静かに行わない (配線の無い環境で MFA を弱めない)。
type Deps struct {
	Repo ports.TrustedDeviceRepository
	Emit func(spec.DomainEvent)
}

// Rememberable は factor が記憶の起点として認められるかを返す。
func Rememberable(factor string) bool { return slices.Contains(RememberableFactors, factor) }

// Issue は新しい信頼済みデバイスを発行し、cookie に載せる平文の資格情報を返す。
// maxAge が 0 以下ならテナントが機能を無効にしているので発行せず空文字を返す。
func Issue(
	ctx context.Context,
	deps Deps,
	tenantID, userID, factor, userAgent string,
	maxAge time.Duration,
	now time.Time,
) (string, error) {
	if deps.Repo == nil || maxAge <= 0 || !Rememberable(factor) {
		return "", nil
	}
	device, cookie, err := domain.NewTrustedDevice(
		tenantID, userID, domain.DeviceLabel(userAgent), maxAge, now,
	)
	if err != nil {
		return "", err
	}
	if err := device.Validate(); err != nil {
		return "", err
	}
	if err := deps.Repo.Save(ctx, device); err != nil {
		return "", err
	}
	if deps.Emit != nil {
		deps.Emit(&domain.TrustedDeviceRegistered{
			At: now, TenantID: tenantID, UserID: userID,
			DeviceID: device.ID, Factor: factor, ExpiresAt: device.ExpiresAt,
		})
	}
	return cookie, nil
}

// EvaluationResult は cookie 照合の結果。Trusted が false なら第二要素を要求する。
// RotatedCookie は照合に成功したときの新しい cookie 値で、呼び出し側が再発行する。
type EvaluationResult struct {
	Trusted       bool
	DeviceID      string
	RotatedCookie string
}

// Evaluate は提示された cookie を照合し、有効なら verifier を回転させて新しい cookie を
// 返す。テナント (tenantID)、ユーザー、絶対期限、idle 期限、失効のいずれかが一致しない
// 場合は fail-closed で Trusted=false を返す。cookie の形が不正でもエラーにはしない
// (改竄された cookie は「信頼できない端末」であって、リクエスト全体の失敗ではない)。
func Evaluate(
	ctx context.Context,
	deps Deps,
	tenantID, userID, cookie string,
	maxAge time.Duration,
	now time.Time,
) (EvaluationResult, error) {
	if deps.Repo == nil || maxAge <= 0 || cookie == "" {
		return EvaluationResult{}, nil
	}
	// 改竄された cookie は「信頼できない端末」であって、リクエスト全体の失敗ではない。
	selector, verifier, ok := domain.ParseCookie(cookie)
	if !ok {
		return EvaluationResult{}, nil
	}
	device, err := deps.Repo.FindBySelector(ctx, tenantID, selector)
	if err != nil {
		return EvaluationResult{}, err
	}
	// UserID の照合は、盗んだ cookie を別アカウントのログインへ持ち込む経路を塞ぐ。
	if device == nil || device.UserID != userID || !device.Active(now) {
		return EvaluationResult{}, nil
	}
	if !device.VerifierMatches(verifier) {
		return EvaluationResult{}, nil
	}
	rotated, err := device.Rotate(now)
	if err != nil {
		return EvaluationResult{}, err
	}
	if err := deps.Repo.Save(ctx, device); err != nil {
		return EvaluationResult{}, err
	}
	return EvaluationResult{
		Trusted: true, DeviceID: device.ID,
		RotatedCookie: domain.FormatCookie(device.Selector, rotated),
	}, nil
}

// ListActive は本人の有効な信頼済みデバイスを最終利用時刻の降順で返す。絶対期限は
// リポジトリ側で、idle 期限はここで切る。
func ListActive(
	ctx context.Context,
	deps Deps,
	tenantID, userID string,
	now time.Time,
) ([]*domain.TrustedDevice, error) {
	if deps.Repo == nil {
		return []*domain.TrustedDevice{}, nil
	}
	devices, err := deps.Repo.ListActiveByUser(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}
	out := make([]*domain.TrustedDevice, 0, len(devices))
	for _, device := range devices {
		if device.Active(now) {
			out = append(out, device)
		}
	}
	return out, nil
}

// RevokeOne は本人の信頼済みデバイス 1 件を失効させる。既に失効済みの対象への再送は
// idempotent に成功し、最初の失効時刻を保持する。他人のデバイスは「存在しない」とする。
func RevokeOne(
	ctx context.Context,
	deps Deps,
	tenantID, userID, deviceID string,
	reason spec.TrustedDeviceRevokeReason,
	now time.Time,
) error {
	if deps.Repo == nil {
		return ErrTrustedDeviceNotFound
	}
	device, err := deps.Repo.FindByID(ctx, tenantID, userID, deviceID)
	if err != nil {
		return err
	}
	if device == nil {
		return ErrTrustedDeviceNotFound
	}
	// 既に失効済みなら Revoke は no-op になり、最初の失効時刻を保持したまま成功する。
	if device.RevokedAt != nil {
		return nil
	}
	device.Revoke(reason, now)
	if err := deps.Repo.Save(ctx, device); err != nil {
		return err
	}
	emitRevoked(deps, device, now)
	return nil
}

// RevokeAllForUser は対象ユーザーの信頼済みデバイスをすべて失効させる。資格情報が変わる
// 操作 (パスワード、認証要素、認証器のリセット、アカウント無効化、全セッション失効) から
// 呼ぶ。Repo が未配線なら no-op。
func RevokeAllForUser(
	ctx context.Context,
	deps Deps,
	tenantID, userID string,
	reason spec.TrustedDeviceRevokeReason,
	now time.Time,
) error {
	if deps.Repo == nil {
		return nil
	}
	revoked, err := deps.Repo.RevokeAllForUser(ctx, tenantID, userID, reason, now)
	if err != nil {
		return err
	}
	for _, device := range revoked {
		emitRevoked(deps, device, now)
	}
	return nil
}

func emitRevoked(deps Deps, device *domain.TrustedDevice, now time.Time) {
	if deps.Emit == nil || device == nil {
		return
	}
	reason := spec.TrustedDeviceSelfRevoke
	if device.RevokeReason != nil {
		reason = *device.RevokeReason
	}
	deps.Emit(&domain.TrustedDeviceRevoked{
		At: now, TenantID: device.TenantID, UserID: device.UserID,
		DeviceID: device.ID, Reason: reason,
	})
}
