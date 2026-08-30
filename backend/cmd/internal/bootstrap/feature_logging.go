package bootstrap

import (
	"context"

	"github.com/ambi/idmagic/backend/shared/logging"
)

// LogFeatureWarnings は純粋な解決結果を、各プロセスの起動ログという効果へ接続する。
func LogFeatureWarnings(ctx context.Context, resolution FeatureResolution) {
	for _, warning := range resolution.Warnings {
		logging.Warn(ctx, "runtime feature requires operator attention",
			"feature_id", warning.ID, "maturity", warning.Maturity, "reason", warning.Reason)
	}
}
