package db_memory

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"sync"
	"time"

	auditports "github.com/ambi/idmagic/backend/audit/ports"
)

type (
	ExportArtifactStore struct {
		mu    sync.RWMutex
		byJob map[string]memoryExportArtifact
	}
	memoryExportArtifact struct {
		metadata auditports.ExportArtifact
		content  []byte
	}
)

func NewExportArtifactStore() *ExportArtifactStore {
	return &ExportArtifactStore{byJob: map[string]memoryExportArtifact{}}
}

func (s *ExportArtifactStore) Write(ctx context.Context, tenantID, jobID string, write func(io.Writer) (auditports.ExportCompletion, error)) (auditports.ExportArtifact, error) {
	s.mu.RLock()
	existing, ok := s.byJob[jobID]
	s.mu.RUnlock()
	if ok {
		if existing.metadata.TenantID != tenantID {
			return auditports.ExportArtifact{}, auditports.ErrExportArtifactNotFound
		}
		return existing.metadata, nil
	}
	var content bytes.Buffer
	completion, err := write(&content)
	if err != nil {
		return auditports.ExportArtifact{}, err
	}
	if err := ctx.Err(); err != nil {
		return auditports.ExportArtifact{}, err
	}
	if completion.BeforePublish != nil {
		if err := completion.BeforePublish(); err != nil {
			return auditports.ExportArtifact{}, err
		}
	}
	digest := sha256.Sum256(content.Bytes())
	meta := auditports.ExportArtifact{Ref: jobID, JobID: jobID, TenantID: tenantID, Events: completion.Events, ByteSize: int64(content.Len()), SHA256: hex.EncodeToString(digest[:]), ExpiresAt: completion.ExpiresAt.UTC().Truncate(time.Microsecond)}
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.byJob[jobID]; ok {
		if existing.metadata.TenantID != tenantID {
			return auditports.ExportArtifact{}, auditports.ErrExportArtifactNotFound
		}
		return existing.metadata, nil
	}
	s.byJob[jobID] = memoryExportArtifact{metadata: meta, content: content.Bytes()}
	return meta, nil
}

func (s *ExportArtifactStore) Open(_ context.Context, tenantID, jobID string) (io.ReadCloser, auditports.ExportArtifact, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	stored, ok := s.byJob[jobID]
	if !ok || stored.metadata.TenantID != tenantID {
		return nil, auditports.ExportArtifact{}, auditports.ErrExportArtifactNotFound
	}
	return io.NopCloser(bytes.NewReader(stored.content)), stored.metadata, nil
}

func (s *ExportArtifactStore) Expire(_ context.Context, now time.Time) ([]auditports.ExportArtifact, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var expired []auditports.ExportArtifact
	for id, stored := range s.byJob {
		if !now.Before(stored.metadata.ExpiresAt) {
			expired = append(expired, stored.metadata)
			delete(s.byJob, id)
			if len(expired) == 1000 {
				break
			}
		}
	}
	return expired, nil
}

func (s *ExportArtifactStore) Usage(_ context.Context, tenantID string) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var size int64
	for _, stored := range s.byJob {
		if stored.metadata.TenantID == tenantID {
			size += stored.metadata.ByteSize
		}
	}
	return size, nil
}

func (s *ExportArtifactStore) TotalBytes(_ context.Context) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var size int64
	for _, stored := range s.byJob {
		size += stored.metadata.ByteSize
	}
	return size, nil
}
