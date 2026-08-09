package db_memory

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"sync"

	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

type UserCSVArtifactStore struct {
	mu      sync.RWMutex
	byScope map[string]storedUserCSVArtifact
}

type storedUserCSVArtifact struct {
	metadata userports.UserCSVArtifact
	content  []byte
	pages    [][]byte
}

func (s *UserCSVArtifactStore) PutUserCSVArtifactPages(_ context.Context, tenantID string, write func(emit func([]byte) error) error) (userports.UserCSVArtifact, error) {
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
		return userports.UserCSVArtifact{}, err
	}
	ref, err := spec.NewUUIDv4()
	if err != nil {
		return userports.UserCSVArtifact{}, err
	}
	metadata := userports.UserCSVArtifact{Ref: ref, TenantID: tenantID, SHA256: hex.EncodeToString(digest.Sum(nil)), ByteSize: size}
	s.mu.Lock()
	s.byScope[tenantID+"\x00"+ref] = storedUserCSVArtifact{metadata: metadata, pages: pages}
	s.mu.Unlock()
	return metadata, nil
}

func (s *UserCSVArtifactStore) ReadUserCSVArtifactPage(_ context.Context, tenantID, ref string, page int) ([]byte, userports.UserCSVArtifact, error) {
	s.mu.RLock()
	stored, ok := s.byScope[tenantID+"\x00"+ref]
	s.mu.RUnlock()
	if !ok || page < 0 || page >= len(stored.pages) {
		return nil, userports.UserCSVArtifact{}, userports.ErrUserCSVArtifactNotFound
	}
	return append([]byte(nil), stored.pages[page]...), stored.metadata, nil
}

func NewUserCSVArtifactStore() *UserCSVArtifactStore {
	return &UserCSVArtifactStore{byScope: map[string]storedUserCSVArtifact{}}
}

func (s *UserCSVArtifactStore) PutUserCSVArtifact(_ context.Context, tenantID string, write func(io.Writer) error) (userports.UserCSVArtifact, error) {
	var content bytes.Buffer
	if err := write(&content); err != nil {
		return userports.UserCSVArtifact{}, err
	}
	ref, err := spec.NewUUIDv4()
	if err != nil {
		return userports.UserCSVArtifact{}, err
	}
	digest := sha256.Sum256(content.Bytes())
	metadata := userports.UserCSVArtifact{
		Ref: ref, TenantID: tenantID, SHA256: hex.EncodeToString(digest[:]), ByteSize: int64(content.Len()),
	}
	s.mu.Lock()
	s.byScope[tenantID+"\x00"+ref] = storedUserCSVArtifact{metadata: metadata, content: append([]byte(nil), content.Bytes()...)}
	s.mu.Unlock()
	return metadata, nil
}

func (s *UserCSVArtifactStore) OpenUserCSVArtifact(_ context.Context, tenantID, ref string) (io.ReadCloser, userports.UserCSVArtifact, error) {
	s.mu.RLock()
	stored, ok := s.byScope[tenantID+"\x00"+ref]
	s.mu.RUnlock()
	if !ok {
		return nil, userports.UserCSVArtifact{}, userports.ErrUserCSVArtifactNotFound
	}
	return io.NopCloser(bytes.NewReader(stored.content)), stored.metadata, nil
}

var _ userports.UserCSVArtifactStore = (*UserCSVArtifactStore)(nil)
