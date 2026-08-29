package db_postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"hash"
	"io"
	"os"

	idmports "github.com/ambi/idmagic/backend/idmanagement/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"

	"github.com/jackc/pgx/v5"
)

const csvArtifactChunkBytes = 64 << 10

type CSVArtifactStore struct{ Pool sharedpg.DB }

type hashingArtifactWriter struct {
	file io.Writer
	hash hash.Hash
	size int64
}

func (w *hashingArtifactWriter) Write(p []byte) (int, error) {
	n, err := w.file.Write(p)
	if n > 0 {
		_, _ = w.hash.Write(p[:n])
		w.size += int64(n)
	}
	return n, err
}

func (s *CSVArtifactStore) PutCSVArtifact(ctx context.Context, tenantID string, write func(io.Writer) error) (idmports.CSVArtifact, error) {
	temporary, err := os.CreateTemp("", "idmagic-csv-*")
	if err != nil {
		return idmports.CSVArtifact{}, err
	}
	name := temporary.Name()
	defer os.Remove(name) //nolint:errcheck // best-effort cleanup of private temporary payload
	w := &hashingArtifactWriter{file: temporary, hash: sha256.New()}
	if err := write(w); err != nil {
		_ = temporary.Close()
		return idmports.CSVArtifact{}, err
	}
	if _, err := temporary.Seek(0, io.SeekStart); err != nil {
		_ = temporary.Close()
		return idmports.CSVArtifact{}, err
	}
	ref, err := spec.NewUUIDv4()
	if err != nil {
		_ = temporary.Close()
		return idmports.CSVArtifact{}, err
	}
	metadata := idmports.CSVArtifact{Ref: ref, TenantID: tenantID, SHA256: hex.EncodeToString(w.hash.Sum(nil)), ByteSize: w.size}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		_ = temporary.Close()
		return idmports.CSVArtifact{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // commit path makes rollback a no-op
	if _, err := tx.Exec(ctx, `INSERT INTO csv_artifacts (id, tenant_id, sha256, byte_size) VALUES ($1, $2, $3, $4)`, ref, tenantID, metadata.SHA256, metadata.ByteSize); err != nil {
		_ = temporary.Close()
		return idmports.CSVArtifact{}, err
	}
	buffer := make([]byte, csvArtifactChunkBytes)
	for chunkNumber := 0; ; chunkNumber++ {
		n, readErr := io.ReadFull(temporary, buffer)
		if n > 0 {
			if _, err := tx.Exec(ctx, `INSERT INTO csv_artifact_chunks (artifact_id, chunk_number, payload) VALUES ($1, $2, $3)`, ref, chunkNumber, buffer[:n]); err != nil {
				_ = temporary.Close()
				return idmports.CSVArtifact{}, err
			}
		}
		if errors.Is(readErr, io.EOF) || errors.Is(readErr, io.ErrUnexpectedEOF) {
			break
		}
		if readErr != nil {
			_ = temporary.Close()
			return idmports.CSVArtifact{}, readErr
		}
	}
	if err := temporary.Close(); err != nil {
		return idmports.CSVArtifact{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return idmports.CSVArtifact{}, err
	}
	return metadata, nil
}

func (s *CSVArtifactStore) OpenCSVArtifact(ctx context.Context, tenantID, ref string) (io.ReadCloser, idmports.CSVArtifact, error) {
	var metadata idmports.CSVArtifact
	metadata.Ref, metadata.TenantID = ref, tenantID
	if err := s.Pool.QueryRow(ctx, `SELECT sha256, byte_size FROM csv_artifacts WHERE tenant_id = $1 AND id = $2`, tenantID, ref).Scan(&metadata.SHA256, &metadata.ByteSize); errors.Is(err, pgx.ErrNoRows) {
		return nil, idmports.CSVArtifact{}, idmports.ErrCSVArtifactNotFound
	} else if err != nil {
		return nil, idmports.CSVArtifact{}, err
	}
	rows, err := s.Pool.Query(ctx, `SELECT payload FROM csv_artifact_chunks WHERE artifact_id = $1 ORDER BY chunk_number`, ref)
	if err != nil {
		return nil, idmports.CSVArtifact{}, err
	}
	return &csvChunkReader{rows: rows}, metadata, nil
}

func (s *CSVArtifactStore) PutCSVArtifactPages(ctx context.Context, tenantID string, write func(emit func([]byte) error) error) (idmports.CSVArtifact, error) {
	ref, err := spec.NewUUIDv4()
	if err != nil {
		return idmports.CSVArtifact{}, err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return idmports.CSVArtifact{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // commit path makes rollback a no-op
	placeholder := "0000000000000000000000000000000000000000000000000000000000000000"
	if _, err := tx.Exec(ctx, `INSERT INTO csv_artifacts (id, tenant_id, sha256, byte_size) VALUES ($1, $2, $3, 0)`, ref, tenantID, placeholder); err != nil {
		return idmports.CSVArtifact{}, err
	}
	digest := sha256.New()
	var size int64
	page := 0
	emit := func(payload []byte) error {
		if len(payload) > csvArtifactChunkBytes {
			return errors.New("CSV artifact page exceeds chunk size")
		}
		if _, err := tx.Exec(ctx, `INSERT INTO csv_artifact_chunks (artifact_id, chunk_number, payload) VALUES ($1, $2, $3)`, ref, page, payload); err != nil {
			return err
		}
		page++
		_, _ = digest.Write(payload)
		size += int64(len(payload))
		return nil
	}
	if err := write(emit); err != nil {
		return idmports.CSVArtifact{}, err
	}
	metadata := idmports.CSVArtifact{Ref: ref, TenantID: tenantID, SHA256: hex.EncodeToString(digest.Sum(nil)), ByteSize: size}
	if _, err := tx.Exec(ctx, `UPDATE csv_artifacts SET sha256 = $2, byte_size = $3 WHERE id = $1`, ref, metadata.SHA256, metadata.ByteSize); err != nil {
		return idmports.CSVArtifact{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return idmports.CSVArtifact{}, err
	}
	return metadata, nil
}

func (s *CSVArtifactStore) ReadCSVArtifactPage(ctx context.Context, tenantID, ref string, page int) ([]byte, idmports.CSVArtifact, error) {
	var metadata idmports.CSVArtifact
	metadata.Ref, metadata.TenantID = ref, tenantID
	var payload []byte
	err := s.Pool.QueryRow(ctx, `SELECT a.sha256, a.byte_size, c.payload FROM csv_artifacts a
        JOIN csv_artifact_chunks c ON c.artifact_id = a.id
        WHERE a.tenant_id = $1 AND a.id = $2 AND c.chunk_number = $3`, tenantID, ref, page).Scan(&metadata.SHA256, &metadata.ByteSize, &payload)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, idmports.CSVArtifact{}, idmports.ErrCSVArtifactNotFound
	}
	return payload, metadata, err
}

type csvChunkReader struct {
	rows    pgx.Rows
	current []byte
	offset  int
}

func (r *csvChunkReader) Read(p []byte) (int, error) {
	for r.offset >= len(r.current) {
		if !r.rows.Next() {
			if err := r.rows.Err(); err != nil {
				return 0, err
			}
			return 0, io.EOF
		}
		if err := r.rows.Scan(&r.current); err != nil {
			return 0, err
		}
		r.offset = 0
	}
	n := copy(p, r.current[r.offset:])
	r.offset += n
	return n, nil
}

func (r *csvChunkReader) Close() error {
	r.rows.Close()
	return nil
}

var _ idmports.CSVArtifactStore = (*CSVArtifactStore)(nil)
