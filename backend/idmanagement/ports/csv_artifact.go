package ports

// 種別に依存しない CSV 成果物ストア。upload payload と結果ページのどちらも同じ
// 不変ストアに置き、CSV 種別ごとに artifact / error のテーブルを増やさない
// (docs/domain/identity-management/design/csv-transfer.md)。

import (
	"context"
	"errors"
	"io"
	"time"
)

var ErrCSVArtifactNotFound = errors.New("CSV artifact not found")

// CSVArtifact はストアが算出する不変なペイロードのメタデータ。SHA256 は
// サーバーが所有する preview/apply ジョブのための完全性バインディングであり、
// クライアントの署名ではない。
type CSVArtifact struct {
	Ref      string `json:"ref"`
	TenantID string `json:"-"`
	SHA256   string `json:"sha256"`
	ByteSize int64  `json:"byte_size"`
}

// CSVArtifactStore は write が nil を返したときにだけ成果物を確定する。
// 実装はペイロードを流しながらダイジェストとサイズを算出する。
// CSVArtifactPurger は保持期限を過ぎた成果物を消す。cutoff より前に作った成果物を、
// 分割したペイロードごと消し、消した件数を返す。テナントをまたいで消すので、Batch の
// 保持期限の削除だけが使う。
type CSVArtifactPurger interface {
	DeleteCSVArtifactsCreatedBefore(ctx context.Context, cutoff time.Time) (int64, error)
}

type CSVArtifactStore interface {
	PutCSVArtifact(ctx context.Context, tenantID string, write func(io.Writer) error) (CSVArtifact, error)
	OpenCSVArtifact(ctx context.Context, tenantID, ref string) (io.ReadCloser, CSVArtifact, error)
	PutCSVArtifactPages(ctx context.Context, tenantID string, write func(emit func([]byte) error) error) (CSVArtifact, error)
	ReadCSVArtifactPage(ctx context.Context, tenantID, ref string, page int) ([]byte, CSVArtifact, error)
}
