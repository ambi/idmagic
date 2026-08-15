// Package ports はセキュリティ通知が要する永続化の抽象を持つ (wi-90)。
package ports

import (
	"context"
	"time"

	"github.com/ambi/idmagic/backend/authentication/securitynotification/domain"
)

// PreferenceRepository は本人による受信設定の保存先。行が無いことと「すべて有効」は
// 同じ意味なので、Find は見つからない場合に nil を返し、それはエラーではない。
type PreferenceRepository interface {
	Find(ctx context.Context, userID string) (*domain.Preferences, error)
	Save(ctx context.Context, preferences domain.Preferences) error
}

// KnownDevice は過去にサインインしたことのあるブラウザー 1 件。生の User-Agent も IP も
// 持たず、テナントの相関ソルトを効かせたハッシュと表示ラベルだけを持つ。
type KnownDevice struct {
	// UserID は所有者の sub。users.id は全体で一意なので tenant_id は持たない。
	UserID string
	// DeviceHash は SaltedHash(tenant salt, User-Agent) の hex。
	DeviceHash string
	// Label は "Chrome / macOS" のようなブラウザーと OS の系統だけの表示ラベル。
	Label string
	// SeenAt は今回のサインイン時刻。
	SeenAt time.Time
}

// KnownDeviceRepository は既知の端末の記録。Observe が「新しい端末かどうか」の判定と
// 通知の重複排除を兼ねる。
type KnownDeviceRepository interface {
	// Observe は端末を記録し、この呼び出しで行が新しく作られたかどうかを返す。
	// 既に在る行では last_seen_at だけを進め、false を返す。複数のレプリカが同時に
	// 呼んでも、true を返すのはちょうど 1 つである。
	Observe(ctx context.Context, device KnownDevice) (bool, error)
	// DeleteIdleBefore は cutoff より前にしか使われていない行を消す。保持期間の掃除が
	// 呼ぶ。履歴から消えた端末を「既知」と呼び続けないための処理である。
	DeleteIdleBefore(ctx context.Context, cutoff time.Time) (int64, error)
}
