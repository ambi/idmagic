package ports

import (
	"context"
	"errors"
	"io"
)

var ErrUserCSVArtifactNotFound = errors.New("user CSV artifact not found")

// UserCSVArtifact is immutable payload metadata computed by the store. SHA256
// is an integrity binding for server-owned preview/apply jobs, not a client
// signature.
type UserCSVArtifact struct {
	Ref      string `json:"ref"`
	TenantID string `json:"-"`
	SHA256   string `json:"sha256"`
	ByteSize int64  `json:"byte_size"`
}

// UserCSVArtifactStore commits an artifact only when write returns nil. The
// implementation computes digest and size while streaming the payload.
type UserCSVArtifactStore interface {
	PutUserCSVArtifact(ctx context.Context, tenantID string, write func(io.Writer) error) (UserCSVArtifact, error)
	OpenUserCSVArtifact(ctx context.Context, tenantID, ref string) (io.ReadCloser, UserCSVArtifact, error)
	PutUserCSVArtifactPages(ctx context.Context, tenantID string, write func(emit func([]byte) error) error) (UserCSVArtifact, error)
	ReadUserCSVArtifactPage(ctx context.Context, tenantID, ref string, page int) ([]byte, UserCSVArtifact, error)
}
