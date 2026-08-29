package db_memory

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"sync"

	idmports "github.com/ambi/idmagic/backend/idmanagement/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

type CSVArtifactStore struct {
	mu      sync.RWMutex
	byScope map[string]storedCSVArtifact
}

type storedCSVArtifact struct {
	metadata idmports.CSVArtifact
	content  []byte
	pages    [][]byte
}

func (s *CSVArtifactStore) PutCSVArtifactPages(_ context.Context, tenantID string, write func(emit func([]byte) error) error) (idmports.CSVArtifact, error) {
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
	metadata := idmports.CSVArtifact{Ref: ref, TenantID: tenantID, SHA256: hex.EncodeToString(digest.Sum(nil)), ByteSize: size}
	s.mu.Lock()
	s.byScope[tenantID+"\x00"+ref] = storedCSVArtifact{metadata: metadata, pages: pages}
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

func (s *CSVArtifactStore) PutCSVArtifact(_ context.Context, tenantID string, write func(io.Writer) error) (idmports.CSVArtifact, error) {
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
		Ref: ref, TenantID: tenantID, SHA256: hex.EncodeToString(digest[:]), ByteSize: int64(content.Len()),
	}
	s.mu.Lock()
	s.byScope[tenantID+"\x00"+ref] = storedCSVArtifact{metadata: metadata, content: append([]byte(nil), content.Bytes()...)}
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
