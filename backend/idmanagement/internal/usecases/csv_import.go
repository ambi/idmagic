package usecases

import (
	"context"
	"io"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	idmports "github.com/ambi/idmagic/backend/idmanagement/internal/ports"
)

// CSVImportContext は元の入力の作成時刻から実行の残り時間を決める。
// 適用ジョブを新しく作っても期限を延長しない。メタデータの取得だけで本文は読まない。
func CSVImportContext(ctx context.Context, artifacts idmports.CSVArtifactStore, tenantID, ref string, now time.Time) (context.Context, context.CancelFunc, error) {
	started := time.Now()
	reader, artifact, err := artifacts.OpenCSVArtifact(ctx, tenantID, ref)
	if err != nil {
		return nil, nil, err
	}
	if err := reader.Close(); err != nil {
		return nil, nil, err
	}
	if artifact.CreatedAt.Before(now.Add(-CSVArtifactRetention)) {
		return nil, nil, idmports.ErrCSVArtifactNotFound
	}
	// 保存先の分解能である 1 マイクロ秒を加え、境界ちょうどを残す比較とそろえる。
	// メタデータの取得に要した時間も実行期限に含め、遅い保存先で期限を延長しない。
	bounded, cancel := context.WithDeadline(ctx, started.Add(artifact.CreatedAt.Add(CSVArtifactRetention).Sub(now)+time.Microsecond))
	return bounded, cancel, nil
}

// CheckStoredCSVLimits は保存したインポートのファイル全体を、行を計画または確定する前に
// 実効の上限で確かめる。投入はすでに上限で拒否しているので、これが拒否するのは投入の後に
// 上限が下がった場合であり、そのとき先行する行を確定してから超過に気付くことを防ぐ。
func CheckStoredCSVLimits(ctx context.Context, artifacts idmports.CSVArtifactStore, tenantID, ref string, policy idmdomain.CSVTransferPolicy) error {
	reader, _, err := artifacts.OpenCSVArtifact(ctx, tenantID, ref)
	if err != nil {
		return err
	}
	defer func() { _ = reader.Close() }()
	return idmdomain.CheckCSVLimits(csvImportReader{ctx, reader}, policy)
}

type csvImportReader struct {
	ctx context.Context
	io.Reader
}

func (r csvImportReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.Reader.Read(p)
}
