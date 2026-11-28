package db_memory

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"sync"
	"time"

	idmports "github.com/ambi/idmagic/backend/idmanagement/internal/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

type CSVArtifactStore struct {
	mu      sync.RWMutex
	byScope map[string]storedCSVArtifact
}

type storedCSVArtifact struct {
	metadata  idmports.CSVArtifact
	content   []byte
	pages     [][]byte
	createdAt time.Time
}

// DeleteCSVArtifactsCreatedBefore は cutoff より前に保存した成果物を、テナントをまたいで消す。
func (s *CSVArtifactStore) DeleteCSVArtifactsCreatedBefore(_ context.Context, cutoff time.Time) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var deleted int64
	for key, stored := range s.byScope {
		if stored.createdAt.Before(cutoff) {
			delete(s.byScope, key)
			deleted++
		}
	}
	return deleted, nil
}

func (s *CSVArtifactStore) PutCSVArtifactPages(_ context.Context, tenantID string, createdAt time.Time, write func(emit func([]byte) error) error) (idmports.CSVArtifact, error) {
	var pages [][]byte
	digest := sha256.New()
	var size int64
	emit := func(page []byte) error {
		copied := append([]byte(nil), page...)
		pages = append(pages, copied)
		_, _ = digest.Write(copied)
		size += int64(len(copied))
		return nil
	}
	if err := write(emit); err != nil {
		return idmports.CSVArtifact{}, err
	}
	ref, err := spec.NewUUIDv4()
	if err != nil {
		return idmports.CSVArtifact{}, err
	}
	metadata := idmports.CSVArtifact{Ref: ref, TenantID: tenantID, SHA256: hex.EncodeToString(digest.Sum(nil)), ByteSize: size, CreatedAt: createdAt.UTC().Truncate(time.Microsecond)}
	s.mu.Lock()
	s.byScope[tenantID+"\x00"+ref] = storedCSVArtifact{metadata: metadata, pages: pages, createdAt: metadata.CreatedAt}
	s.mu.Unlock()
	return metadata, nil
}

func (s *CSVArtifactStore) ReadCSVArtifactPage(_ context.Context, tenantID, ref string, page int) ([]byte, idmports.CSVArtifact, error) {
	s.mu.RLock()
	stored, ok := s.byScope[tenantID+"\x00"+ref]
	s.mu.RUnlock()
	if !ok || page < 0 || page >= len(stored.pages) {
		return nil, idmports.CSVArtifact{}, idmports.ErrCSVArtifactNotFound
	}
	return append([]byte(nil), stored.pages[page]...), stored.metadata, nil
}

func NewCSVArtifactStore() *CSVArtifactStore {
	return &CSVArtifactStore{byScope: map[string]storedCSVArtifact{}}
}

func (s *CSVArtifactStore) CSVArtifactUsage(_ context.Context, tenantID string) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var size int64
	for _, stored := range s.byScope {
		if stored.metadata.TenantID == tenantID {
			size += stored.metadata.ByteSize
		}
	}
	return size, nil
}

func (s *CSVArtifactStore) PutCSVArtifact(_ context.Context, tenantID string, createdAt time.Time, write func(io.Writer) error) (idmports.CSVArtifact, error) {
	var content bytes.Buffer
	if err := write(&content); err != nil {
		return idmports.CSVArtifact{}, err
	}
	ref, err := spec.NewUUIDv4()
	if err != nil {
		return idmports.CSVArtifact{}, err
	}
	digest := sha256.Sum256(content.Bytes())
	metadata := idmports.CSVArtifact{
		Ref: ref, TenantID: tenantID, SHA256: hex.EncodeToString(digest[:]), ByteSize: int64(content.Len()), CreatedAt: createdAt.UTC().Truncate(time.Microsecond),
	}
	s.mu.Lock()
	s.byScope[tenantID+"\x00"+ref] = storedCSVArtifact{metadata: metadata, content: append([]byte(nil), content.Bytes()...), createdAt: metadata.CreatedAt}
	s.mu.Unlock()
	return metadata, nil
}

func (s *CSVArtifactStore) OpenCSVArtifact(_ context.Context, tenantID, ref string) (io.ReadCloser, idmports.CSVArtifact, error) {
	s.mu.RLock()
	stored, ok := s.byScope[tenantID+"\x00"+ref]
	s.mu.RUnlock()
	if !ok {
		return nil, idmports.CSVArtifact{}, idmports.ErrCSVArtifactNotFound
	}
	return io.NopCloser(bytes.NewReader(stored.content)), stored.metadata, nil
}

var _ idmports.CSVArtifactStore = (*CSVArtifactStore)(nil)
