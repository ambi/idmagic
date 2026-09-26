package db_postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"hash"
	"io"
	"math"
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
	queries := New(tx)
	if err := queries.InsertCSVArtifact(ctx, InsertCSVArtifactParams{ID: ref, TenantID: tenantID, Sha256: metadata.SHA256, ByteSize: metadata.ByteSize}); err != nil {
		_ = temporary.Close()
		return idmports.CSVArtifact{}, err
	}
	buffer := make([]byte, csvArtifactChunkBytes)
	for chunkNumber := int32(0); ; chunkNumber++ {
		n, readErr := io.ReadFull(temporary, buffer)
		if n > 0 {
			if err := queries.InsertCSVArtifactChunk(ctx, InsertCSVArtifactChunkParams{ArtifactID: ref, ChunkNumber: chunkNumber, Payload: buffer[:n]}); err != nil {
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
	queries := New(s.Pool)
	row, err := queries.FindCSVArtifact(ctx, FindCSVArtifactParams{TenantID: tenantID, ID: ref})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, idmports.CSVArtifact{}, idmports.ErrCSVArtifactNotFound
	}
	if err != nil {
		return nil, idmports.CSVArtifact{}, err
	}
	metadata.SHA256, metadata.ByteSize = row.Sha256, row.ByteSize
	return &csvChunkReader{ctx: ctx, queries: queries, ref: ref}, metadata, nil
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
	queries := New(tx)
	placeholder := "0000000000000000000000000000000000000000000000000000000000000000"
	if err := queries.InsertCSVArtifact(ctx, InsertCSVArtifactParams{ID: ref, TenantID: tenantID, Sha256: placeholder}); err != nil {
		return idmports.CSVArtifact{}, err
	}
	digest := sha256.New()
	var size int64
	var page int32
	emit := func(payload []byte) error {
		if len(payload) > csvArtifactChunkBytes {
			return errors.New("CSV artifact page exceeds chunk size")
		}
		if err := queries.InsertCSVArtifactChunk(ctx, InsertCSVArtifactChunkParams{ArtifactID: ref, ChunkNumber: page, Payload: payload}); err != nil {
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
	if err := queries.UpdateCSVArtifactDigest(ctx, UpdateCSVArtifactDigestParams{ID: ref, Sha256: metadata.SHA256, ByteSize: metadata.ByteSize}); err != nil {
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
	if page < 0 || page > math.MaxInt32 {
		return nil, idmports.CSVArtifact{}, idmports.ErrCSVArtifactNotFound
	}
	row, err := New(s.Pool).FindCSVArtifactPage(ctx, FindCSVArtifactPageParams{
		TenantID: tenantID, ID: ref, ChunkNumber: int32(page),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, idmports.CSVArtifact{}, idmports.ErrCSVArtifactNotFound
	}
	if err != nil {
		return nil, idmports.CSVArtifact{}, err
	}
	metadata.SHA256, metadata.ByteSize = row.Sha256, row.ByteSize
	return row.Payload, metadata, nil
}

// csvChunkReadBatch は一度の問い合わせで読むチャンク数である。成果物全体をメモリへ
// 載せず、読み取りの間も接続を占有しないよう、チャンク番号を起点に少しずつ読む。
const csvChunkReadBatch = 16

type csvChunkReader struct {
	// io.Reader の Read は ctx を受け取れないので、開いた時点の ctx で読む。
	ctx     context.Context
	queries *Queries
	ref     string
	next    int32
	pending [][]byte
	done    bool
	current []byte
	offset  int
}

func (r *csvChunkReader) Read(p []byte) (int, error) {
	for r.offset >= len(r.current) {
		if len(r.pending) == 0 {
			if r.done {
				return 0, io.EOF
			}
			if err := r.fetch(); err != nil {
				return 0, err
			}
			continue
		}
		r.current, r.pending = r.pending[0], r.pending[1:]
		r.offset = 0
	}
	n := copy(p, r.current[r.offset:])
	r.offset += n
	return n, nil
}

func (r *csvChunkReader) fetch() error {
	payloads, err := r.queries.ListCSVArtifactChunkPayloadsFrom(r.ctx, ListCSVArtifactChunkPayloadsFromParams{
		ArtifactID: r.ref, FromChunkNumber: r.next, PageLimit: csvChunkReadBatch,
	})
	if err != nil {
		return err
	}
	r.pending = payloads
	r.next += int32(len(payloads)) //nolint:gosec // len is bounded by csvChunkReadBatch
	r.done = len(payloads) < csvChunkReadBatch
	return nil
}

func (r *csvChunkReader) Close() error {
	return nil
}

var _ idmports.CSVArtifactStore = (*CSVArtifactStore)(nil)
