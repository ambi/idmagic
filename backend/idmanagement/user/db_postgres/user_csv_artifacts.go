package db_postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"hash"
	"io"
	"os"

	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"

	"github.com/jackc/pgx/v5"
)

const userCSVArtifactChunkBytes = 64 << 10

type UserCSVArtifactStore struct{ Pool sharedpg.DB }

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

func (s *UserCSVArtifactStore) PutUserCSVArtifact(ctx context.Context, tenantID string, write func(io.Writer) error) (userports.UserCSVArtifact, error) {
	temporary, err := os.CreateTemp("", "idmagic-user-csv-*")
	if err != nil {
		return userports.UserCSVArtifact{}, err
	}
	name := temporary.Name()
	defer os.Remove(name) //nolint:errcheck // best-effort cleanup of private temporary payload
	w := &hashingArtifactWriter{file: temporary, hash: sha256.New()}
	if err := write(w); err != nil {
		_ = temporary.Close()
		return userports.UserCSVArtifact{}, err
	}
	if _, err := temporary.Seek(0, io.SeekStart); err != nil {
		_ = temporary.Close()
		return userports.UserCSVArtifact{}, err
	}
	ref, err := spec.NewUUIDv4()
	if err != nil {
		_ = temporary.Close()
		return userports.UserCSVArtifact{}, err
	}
	metadata := userports.UserCSVArtifact{Ref: ref, TenantID: tenantID, SHA256: hex.EncodeToString(w.hash.Sum(nil)), ByteSize: w.size}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		_ = temporary.Close()
		return userports.UserCSVArtifact{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // commit path makes rollback a no-op
	if _, err := tx.Exec(ctx, `INSERT INTO user_csv_artifacts (id, tenant_id, sha256, byte_size) VALUES ($1, $2, $3, $4)`, ref, tenantID, metadata.SHA256, metadata.ByteSize); err != nil {
		_ = temporary.Close()
		return userports.UserCSVArtifact{}, err
	}
	buffer := make([]byte, userCSVArtifactChunkBytes)
	for chunkNumber := 0; ; chunkNumber++ {
		n, readErr := io.ReadFull(temporary, buffer)
		if n > 0 {
			if _, err := tx.Exec(ctx, `INSERT INTO user_csv_artifact_chunks (artifact_id, chunk_number, payload) VALUES ($1, $2, $3)`, ref, chunkNumber, buffer[:n]); err != nil {
				_ = temporary.Close()
				return userports.UserCSVArtifact{}, err
			}
		}
		if errors.Is(readErr, io.EOF) || errors.Is(readErr, io.ErrUnexpectedEOF) {
			break
		}
		if readErr != nil {
			_ = temporary.Close()
			return userports.UserCSVArtifact{}, readErr
		}
	}
	if err := temporary.Close(); err != nil {
		return userports.UserCSVArtifact{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return userports.UserCSVArtifact{}, err
	}
	return metadata, nil
}

func (s *UserCSVArtifactStore) OpenUserCSVArtifact(ctx context.Context, tenantID, ref string) (io.ReadCloser, userports.UserCSVArtifact, error) {
	var metadata userports.UserCSVArtifact
	metadata.Ref, metadata.TenantID = ref, tenantID
	if err := s.Pool.QueryRow(ctx, `SELECT sha256, byte_size FROM user_csv_artifacts WHERE tenant_id = $1 AND id = $2`, tenantID, ref).Scan(&metadata.SHA256, &metadata.ByteSize); errors.Is(err, pgx.ErrNoRows) {
		return nil, userports.UserCSVArtifact{}, userports.ErrUserCSVArtifactNotFound
	} else if err != nil {
		return nil, userports.UserCSVArtifact{}, err
	}
	rows, err := s.Pool.Query(ctx, `SELECT payload FROM user_csv_artifact_chunks WHERE artifact_id = $1 ORDER BY chunk_number`, ref)
	if err != nil {
		return nil, userports.UserCSVArtifact{}, err
	}
	return &userCSVChunkReader{rows: rows}, metadata, nil
}

func (s *UserCSVArtifactStore) PutUserCSVArtifactPages(ctx context.Context, tenantID string, write func(emit func([]byte) error) error) (userports.UserCSVArtifact, error) {
	ref, err := spec.NewUUIDv4()
	if err != nil {
		return userports.UserCSVArtifact{}, err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return userports.UserCSVArtifact{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // commit path makes rollback a no-op
	placeholder := "0000000000000000000000000000000000000000000000000000000000000000"
	if _, err := tx.Exec(ctx, `INSERT INTO user_csv_artifacts (id, tenant_id, sha256, byte_size) VALUES ($1, $2, $3, 0)`, ref, tenantID, placeholder); err != nil {
		return userports.UserCSVArtifact{}, err
	}
	digest := sha256.New()
	var size int64
	page := 0
	emit := func(payload []byte) error {
		if len(payload) > userCSVArtifactChunkBytes {
			return errors.New("user CSV artifact page exceeds chunk size")
		}
		if _, err := tx.Exec(ctx, `INSERT INTO user_csv_artifact_chunks (artifact_id, chunk_number, payload) VALUES ($1, $2, $3)`, ref, page, payload); err != nil {
			return err
		}
		page++
		_, _ = digest.Write(payload)
		size += int64(len(payload))
		return nil
	}
	if err := write(emit); err != nil {
		return userports.UserCSVArtifact{}, err
	}
	metadata := userports.UserCSVArtifact{Ref: ref, TenantID: tenantID, SHA256: hex.EncodeToString(digest.Sum(nil)), ByteSize: size}
	if _, err := tx.Exec(ctx, `UPDATE user_csv_artifacts SET sha256 = $2, byte_size = $3 WHERE id = $1`, ref, metadata.SHA256, metadata.ByteSize); err != nil {
		return userports.UserCSVArtifact{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return userports.UserCSVArtifact{}, err
	}
	return metadata, nil
}

func (s *UserCSVArtifactStore) ReadUserCSVArtifactPage(ctx context.Context, tenantID, ref string, page int) ([]byte, userports.UserCSVArtifact, error) {
	var metadata userports.UserCSVArtifact
	metadata.Ref, metadata.TenantID = ref, tenantID
	var payload []byte
	err := s.Pool.QueryRow(ctx, `SELECT a.sha256, a.byte_size, c.payload FROM user_csv_artifacts a
        JOIN user_csv_artifact_chunks c ON c.artifact_id = a.id
        WHERE a.tenant_id = $1 AND a.id = $2 AND c.chunk_number = $3`, tenantID, ref, page).Scan(&metadata.SHA256, &metadata.ByteSize, &payload)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, userports.UserCSVArtifact{}, userports.ErrUserCSVArtifactNotFound
	}
	return payload, metadata, err
}

type userCSVChunkReader struct {
	rows    pgx.Rows
	current []byte
	offset  int
}

func (r *userCSVChunkReader) Read(p []byte) (int, error) {
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

func (r *userCSVChunkReader) Close() error {
	r.rows.Close()
	return nil
}

var _ userports.UserCSVArtifactStore = (*UserCSVArtifactStore)(nil)
