// Package domain は信頼済みデバイス (remember this device) の集約を持つ (wi-91)。
// 端末は指紋ではなくサーバー発行の selector / verifier で識別する。cookie には
// "selector.verifier" を入れ、保存するのは selector と SHA-256(verifier) だけである。
package domain

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"

	z "github.com/Oudwins/zog"
)

// TrustedDeviceMaxAgeCeilingSeconds は信頼済みデバイスの絶対有効期間に許す上限 (90 日)。
// テナントはこれ以下の値でのみ機能を有効にできる。
const TrustedDeviceMaxAgeCeilingSeconds = 90 * 24 * 60 * 60

// TrustedDeviceIdleMaxAgeSeconds は最終利用からの idle 期限 (30 日)。絶対期限がこれより
// 短いテナントでは絶対期限の側が先に効く。しばらく使われていない端末を、絶対期限を待たずに
// 落とすためにある。
const TrustedDeviceIdleMaxAgeSeconds = 30 * 24 * 60 * 60

// selectorBytes / verifierBytes は cookie 前半 (探索キー) と後半 (秘密) の長さ。
// selector は一意性だけが要件、verifier は総当たりに耐える必要がある。
const (
	selectorBytes = 16
	verifierBytes = 32
)

// TrustedDevice は 1 ブラウザー分の信頼済みデバイス。
type TrustedDevice struct {
	ID       string `json:"id"`
	TenantID string `json:"tenant_id"`
	UserID   string `json:"user_id"`
	// Selector は cookie 前半の非秘密な探索キー。一意で、1 行の特定にだけ使う。
	Selector string `json:"selector"`
	// VerifierHash は cookie 後半の SHA-256 hex。平文は保存せず、利用のたびに回転する。
	VerifierHash string `json:"verifier_hash"`
	// Label は User-Agent から導いたブラウザーと OS の系統だけの表示名。生の
	// User-Agent も IP も保存しない。
	Label        string                          `json:"label,omitempty"`
	CreatedAt    time.Time                       `json:"created_at"`
	LastUsedAt   time.Time                       `json:"last_used_at"`
	ExpiresAt    time.Time                       `json:"expires_at"`
	RevokedAt    *time.Time                      `json:"revoked_at,omitempty"`
	RevokeReason *spec.TrustedDeviceRevokeReason `json:"revoke_reason,omitempty"`
}

// Active は絶対期限と idle 期限の両方を満たし、失効していないかを fail-closed に判定する。
func (d TrustedDevice) Active(now time.Time) bool {
	if d.RevokedAt != nil || !now.Before(d.ExpiresAt) {
		return false
	}
	return now.Before(d.LastUsedAt.Add(d.idleWindow()))
}

// idleWindow は idle 期限の幅。絶対有効期間がこれより短いテナントでは、idle 期限が
// 絶対期限を追い越さないよう絶対有効期間側に丸める。
func (d TrustedDevice) idleWindow() time.Duration {
	idle := time.Duration(TrustedDeviceIdleMaxAgeSeconds) * time.Second
	if absolute := d.ExpiresAt.Sub(d.CreatedAt); absolute > 0 && absolute < idle {
		return absolute
	}
	return idle
}

// Revoke は tombstone として失効させる。最初の失効だけが revoked_at / revoke_reason を
// 確定し、以降の呼び出しは idempotent な no-op になる。
func (d *TrustedDevice) Revoke(reason spec.TrustedDeviceRevokeReason, now time.Time) {
	if d.RevokedAt != nil {
		return
	}
	at := now
	d.RevokedAt = &at
	d.RevokeReason = &reason
}

// VerifierMatches は提示された verifier のハッシュを定数時間で照合する。
func (d TrustedDevice) VerifierMatches(verifier string) bool {
	return subtle.ConstantTimeCompare([]byte(d.VerifierHash), []byte(HashVerifier(verifier))) == 1
}

// Rotate は利用のたびに verifier を差し替え、last_used_at を進める。新しい平文の
// verifier を返し、呼び出し側はそれを cookie として再発行する。
func (d *TrustedDevice) Rotate(now time.Time) (string, error) {
	verifier, err := randomToken(verifierBytes)
	if err != nil {
		return "", err
	}
	d.VerifierHash = HashVerifier(verifier)
	d.LastUsedAt = now
	return verifier, nil
}

var trustedDeviceSchema = z.Struct(z.Shape{
	"ID":           z.String().UUID().Required(),
	"TenantID":     z.String().Required(),
	"UserID":       z.String().Required(),
	"Selector":     z.String().Required(),
	"VerifierHash": z.String().Required(),
	"ExpiresAt":    z.Time().Required(),
}).TestFunc(func(value any, _ z.Ctx) bool {
	device, ok := value.(*TrustedDevice)
	return ok && (device.RevokedAt == nil) == (device.RevokeReason == nil)
}, z.Message("revoked_at and revoke_reason must be set together"))

func (d TrustedDevice) Validate() error {
	return spec.Validate(trustedDeviceSchema, &d)
}

// NewTrustedDevice は新しい信頼済みデバイスと、cookie に載せる平文の資格情報を作る。
// maxAge はテナントの trusted_device_max_age_seconds で、絶対期限の幅になる。
func NewTrustedDevice(
	tenantID, userID, label string,
	maxAge time.Duration,
	now time.Time,
) (*TrustedDevice, string, error) {
	id, err := spec.NewUUIDv4()
	if err != nil {
		return nil, "", err
	}
	selector, err := randomToken(selectorBytes)
	if err != nil {
		return nil, "", err
	}
	verifier, err := randomToken(verifierBytes)
	if err != nil {
		return nil, "", err
	}
	device := &TrustedDevice{
		ID: id, TenantID: tenantID, UserID: userID, Label: label,
		Selector: selector, VerifierHash: HashVerifier(verifier),
		CreatedAt: now, LastUsedAt: now, ExpiresAt: now.Add(maxAge),
	}
	return device, FormatCookie(selector, verifier), nil
}

// FormatCookie は cookie 値 "selector.verifier" を組み立てる。
func FormatCookie(selector, verifier string) string { return selector + "." + verifier }

// ParseCookie は cookie 値を selector と verifier に分ける。区切りが無い、または
// どちらかが空なら ok=false を返す。壊れた cookie は「信頼できない端末」という通常の
// 入力であって異常事態ではないので、error ではなく真偽値で表す。
func ParseCookie(value string) (selector, verifier string, ok bool) {
	selector, verifier, found := strings.Cut(value, ".")
	if !found || selector == "" || verifier == "" {
		return "", "", false
	}
	return selector, verifier, true
}

// HashVerifier は verifier の SHA-256 hex を返す。保存も照合もこの形だけで行う。
func HashVerifier(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return hex.EncodeToString(sum[:])
}

func randomToken(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
