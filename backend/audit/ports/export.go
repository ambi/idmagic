package ports

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/ambi/idmagic/backend/audit/domain"
)

var ErrExportArtifactNotFound = errors.New("audit export artifact not found")

type ExportParams struct {
	Query       AuditEventQuery     `json:"query"`
	NoMatch     bool                `json:"no_match"`
	RequestedBy string              `json:"requested_by"`
	RequestedAt time.Time           `json:"requested_at"`
	Format      string              `json:"format"`
	Policy      domain.ExportPolicy `json:"policy"`
}

type ExportArtifact struct {
	Ref       string    `json:"ref"`
	JobID     string    `json:"job_id"`
	TenantID  string    `json:"tenant_id"`
	Events    int       `json:"events"`
	ByteSize  int64     `json:"byte_size"`
	SHA256    string    `json:"sha256"`
	ExpiresAt time.Time `json:"expires_at"`
}

type ExportCompletion struct {
	Events        int
	ExpiresAt     time.Time
	BeforePublish func() error
}

// ExportArtifactStore は書込みが正常終了した成果物だけを不変に確定する。
// 同じ Job ID は同じ確定済み成果物を返し、write を再実行しない。
type ExportArtifactStore interface {
	Write(ctx context.Context, tenantID, jobID string, write func(io.Writer) (ExportCompletion, error)) (ExportArtifact, error)
	Open(ctx context.Context, tenantID, jobID string) (io.ReadCloser, ExportArtifact, error)
	// Expire は一度に最大 1,000 件を削除し、削除したメタデータを返す。
	Expire(ctx context.Context, now time.Time) ([]ExportArtifact, error)
	Usage(ctx context.Context, tenantID string) (int64, error)
	TotalBytes(ctx context.Context) (int64, error)
}

type ExportMetrics interface {
	Record(operation, outcome string)
	Limit(kind string)
}
