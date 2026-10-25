package db_postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"math"
	"os"
	"time"

	"github.com/ambi/idmagic/backend/audit/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	postgres "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type ExportArtifactStore struct{ Pool postgres.DB }

func exportMetadata(row *AuditExportArtifact) ports.ExportArtifact {
	return ports.ExportArtifact{Ref: row.ID, JobID: row.JobID, TenantID: row.TenantID, Events: int(row.Events), ByteSize: row.ByteSize, SHA256: row.Sha256, ExpiresAt: row.ExpiresAt}
}

func (s *ExportArtifactStore) find(ctx context.Context, tenantID, jobID string) (ports.ExportArtifact, error) {
	row, err := New(s.Pool).FindAuditExportArtifact(ctx, FindAuditExportArtifactParams{TenantID: tenantID, JobID: jobID})
	if errors.Is(err, pgx.ErrNoRows) {
		return ports.ExportArtifact{}, ports.ErrExportArtifactNotFound
	}
	if err != nil {
		return ports.ExportArtifact{}, err
	}
	return exportMetadata(row), nil
}

func (s *ExportArtifactStore) Write(ctx context.Context, tenantID, jobID string, write func(io.Writer) (ports.ExportCompletion, error)) (ports.ExportArtifact, error) {
	if existing, err := s.find(ctx, tenantID, jobID); err == nil {
		return existing, nil
	} else if !errors.Is(err, ports.ErrExportArtifactNotFound) {
		return ports.ExportArtifact{}, err
	}
	file, err := os.CreateTemp("", "idmagic-audit-export-*")
	if err != nil {
		return ports.ExportArtifact{}, err
	}
	defer file.Close()           //nolint:errcheck // 一時ファイルは成否にかかわらず閉じる。
	defer os.Remove(file.Name()) //nolint:errcheck // 失敗しても非公開の一時ファイルだけが残る。
	digest := sha256.New()
	completion, err := write(io.MultiWriter(file, digest))
	if err != nil {
		return ports.ExportArtifact{}, err
	}
	stat, err := file.Stat()
	if err != nil {
		return ports.ExportArtifact{}, err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return ports.ExportArtifact{}, err
	}
	ref, err := spec.NewUUIDv4()
	if err != nil {
		return ports.ExportArtifact{}, err
	}
	m := ports.ExportArtifact{Ref: ref, JobID: jobID, TenantID: tenantID, Events: completion.Events, ByteSize: stat.Size(), SHA256: hex.EncodeToString(digest.Sum(nil)), ExpiresAt: completion.ExpiresAt.UTC().Truncate(time.Microsecond)}
	if m.Events < 0 || m.Events > math.MaxInt32 {
		return ports.ExportArtifact{}, errors.New("audit export event count exceeds storage range")
	}
	err = New(s.Pool).CreateAuditExportArtifact(ctx, CreateAuditExportArtifactParams{ID: ref, JobID: jobID, TenantID: tenantID, Events: int32(m.Events), ByteSize: m.ByteSize, Sha256: m.SHA256, ExpiresAt: m.ExpiresAt})
	if err != nil {
		return ports.ExportArtifact{}, err
	}
	// 公開前の失敗は独立した短い期限で掃除し、キャンセル済み ctx に後始末を依存させない。
	defer func() {
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
		defer cancel()
		_ = New(s.Pool).DeleteUnpublishedAuditExport(cleanup, ref)
	}()
	buffer := make([]byte, 64<<10)
	for chunk := int32(0); ; chunk++ {
		n, readErr := io.ReadFull(file, buffer)
		if n > 0 {
			if err := New(s.Pool).InsertAuditExportChunk(ctx, InsertAuditExportChunkParams{ArtifactID: ref, ChunkNumber: chunk, Payload: buffer[:n]}); err != nil {
				return ports.ExportArtifact{}, err
			}
		}
		if errors.Is(readErr, io.EOF) || errors.Is(readErr, io.ErrUnexpectedEOF) {
			break
		}
		if readErr != nil {
			return ports.ExportArtifact{}, readErr
		}
	}
	if completion.BeforePublish != nil {
		if err := completion.BeforePublish(); err != nil {
			return ports.ExportArtifact{}, err
		}
	}
	err = New(s.Pool).PublishAuditExportArtifact(ctx, ref)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return s.find(ctx, tenantID, jobID)
	}
	if err != nil {
		return ports.ExportArtifact{}, err
	}
	return m, nil
}

func (s *ExportArtifactStore) Open(ctx context.Context, tenantID, jobID string) (io.ReadCloser, ports.ExportArtifact, error) {
	m, err := s.find(ctx, tenantID, jobID)
	if err != nil {
		return nil, m, err
	}
	return &exportChunkReader{ctx: ctx, pool: s.Pool, ref: m.Ref, remaining: m.ByteSize}, m, nil
}

func (s *ExportArtifactStore) Expire(ctx context.Context, now time.Time) ([]ports.ExportArtifact, error) {
	rows, err := New(s.Pool).ExpireAuditExportArtifacts(ctx, now)
	if err != nil {
		return nil, err
	}
	var expired []ports.ExportArtifact
	for _, row := range rows {
		expired = append(expired, exportMetadata(row))
	}
	return expired, nil
}

func (s *ExportArtifactStore) Usage(ctx context.Context, tenantID string) (int64, error) {
	return New(s.Pool).AuditExportArtifactUsage(ctx, tenantID)
}

func (s *ExportArtifactStore) TotalBytes(ctx context.Context) (int64, error) {
	return New(s.Pool).AuditExportArtifactTotalBytes(ctx)
}

type exportChunkReader struct {
	ctx       context.Context
	pool      postgres.DB
	ref       string
	next      int32
	current   []byte
	pending   [][]byte
	offset    int
	remaining int64
}

func (r *exportChunkReader) Read(out []byte) (int, error) {
	if r.remaining == 0 {
		return 0, io.EOF
	}
	for r.offset >= len(r.current) {
		if len(r.pending) == 0 {
			chunks, err := New(r.pool).ReadAuditExportChunks(r.ctx, ReadAuditExportChunksParams{ArtifactID: r.ref, ChunkNumber: r.next})
			if err != nil {
				return 0, err
			}
			r.pending = chunks
			if len(r.pending) == 0 {
				return 0, io.ErrUnexpectedEOF
			}
			for range r.pending {
				r.next++
			}
		}
		r.current, r.pending = r.pending[0], r.pending[1:]
		r.offset = 0
	}
	n := copy(out, r.current[r.offset:])
	r.offset += n
	r.remaining -= int64(n)
	return n, nil
}
func (*exportChunkReader) Close() error { return nil }
