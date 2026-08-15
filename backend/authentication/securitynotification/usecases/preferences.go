package usecases

import (
	"context"
	"errors"
	"time"

	"github.com/ambi/idmagic/backend/authentication/securitynotification/domain"
	"github.com/ambi/idmagic/backend/authentication/securitynotification/ports"
)

// ErrPreferencesUnavailable は受信設定の保存先が配線されていない。取得は既定
// (すべて有効) を返せるが、更新は保存できないので黙って成功にはしない。
var ErrPreferencesUnavailable = errors.New("notification preferences are not configured")

// PreferenceDeps は受信設定の取得 / 更新が要る依存。
type PreferenceDeps struct {
	Repo ports.PreferenceRepository
}

// CategoryPreference は種別 1 件の本人向けの提示。SCL
// `AccountNotificationCategoryPreference` の双子定義。
type CategoryPreference struct {
	Category  domain.Category
	Mandatory bool
	Enabled   bool
}

// GetPreferences は全種別をカタログの並びで返す。行が無い場合も同じ並びで
// 「すべて有効」を返すので、UI は行の有無を意識しない。
func GetPreferences(ctx context.Context, deps PreferenceDeps, userID string) ([]CategoryPreference, error) {
	var stored domain.Preferences
	if deps.Repo != nil {
		found, err := deps.Repo.Find(ctx, userID)
		if err != nil {
			return nil, err
		}
		if found != nil {
			stored = *found
		}
	}
	out := make([]CategoryPreference, 0, len(domain.Categories()))
	for _, category := range domain.Categories() {
		out = append(out, CategoryPreference{
			Category:  category,
			Mandatory: category.Mandatory(),
			Enabled:   stored.Allows(category),
		})
	}
	return out, nil
}

// UpdatePreferences は受信を止める種別を丸ごと置き換える。必須の種別や未知の種別を
// 含む要求は保存の前に拒否し、一部だけを適用することはしない。
func UpdatePreferences(
	ctx context.Context,
	deps PreferenceDeps,
	userID string,
	disabled []domain.Category,
	now time.Time,
) ([]CategoryPreference, error) {
	preferences, err := domain.NewPreferences(userID, disabled, now)
	if err != nil {
		return nil, err
	}
	if deps.Repo == nil {
		return nil, ErrPreferencesUnavailable
	}
	if err := deps.Repo.Save(ctx, preferences); err != nil {
		return nil, err
	}
	return GetPreferences(ctx, deps, userID)
}
