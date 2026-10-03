package db_memory

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"sync"
	"time"

	idmports "github.com/ambi/idmagic/backend/idmanagement/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

type CSVArtifactStore struct {
	mu      sync.RWMutex
	byScope map[string]storedCSVArtifact
	// Now は保存の時刻を返す。nil なら現在の時刻を使う。テストが作成の時刻を決めるために差し替える。
	Now func() time.Time
}

type storedCSVArtifact struct {
	metadata  idmports.CSVArtifact
	content   []byte
	pages     [][]byte
	createdAt time.Time
}

func (s *CSVArtifactStore) clock() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now().UTC()
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
	s.byScope[tenantID+"\x00"+ref] = storedCSVArtifact{metadata: metadata, pages: pages, createdAt: s.clock()}
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
	s.byScope[tenantID+"\x00"+ref] = storedCSVArtifact{metadata: metadata, content: append([]byte(nil), content.Bytes()...), createdAt: s.clock()}
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
