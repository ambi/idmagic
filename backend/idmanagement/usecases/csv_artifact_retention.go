package usecases

import (
	"context"
	"time"

	idmports "github.com/ambi/idmagic/backend/idmanagement/ports"
)

// CSVArtifactRetention は CSV の成果物を作成から残す期間である。エクスポートの保持期限
// (DataExportTTL) と同じ長さにし、成果物は完了の時点で作るので、期限切れのエクスポートの
// ファイルが期限と同じ時点で消える。
const CSVArtifactRetention = DataExportTTL

// PurgeExpiredCSVArtifacts は、作成から CSVArtifactRetention を過ぎた CSV の成果物を消し、
// 消した件数を返す。境界の時刻ちょうどに作った成果物は残す。
func PurgeExpiredCSVArtifacts(ctx context.Context, purger idmports.CSVArtifactPurger, now time.Time) (int64, error) {
	return purger.DeleteCSVArtifactsCreatedBefore(ctx, now.Add(-CSVArtifactRetention))
}
