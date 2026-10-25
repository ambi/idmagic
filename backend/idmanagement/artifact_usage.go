package idmanagement

import (
	"context"
	"errors"

	idmports "github.com/ambi/idmagic/backend/idmanagement/internal/ports"
)

// CSVArtifactUsage は確定済みの成果物が占有するバイト数をテナントごとに測る。
func (m Module) CSVArtifactUsage(ctx context.Context, tenantID string) (int64, error) {
	reader, ok := m.CSVArtifacts.(idmports.CSVArtifactUsageReader)
	if !ok {
		return 0, errors.New("CSV artifact usage reader is required")
	}
	return reader.CSVArtifactUsage(ctx, tenantID)
}
