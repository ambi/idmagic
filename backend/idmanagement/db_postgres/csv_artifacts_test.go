package db_postgres

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	idmports "github.com/ambi/idmagic/backend/idmanagement/ports"
	pgfixtures "github.com/ambi/idmagic/backend/shared/storage/fixtures_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
)

func TestCSVArtifactStoreStreamsChunksAndIsolatesTenant(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	other := pgfixtures.SeedTenant(t, db)
	store := &CSVArtifactStore{Pool: db}
	want := bytes.Repeat([]byte("row,with,content\n"), 10_000)
	metadata, err := store.PutCSVArtifact(context.Background(), tenant.ID, func(output io.Writer) error {
		_, err := io.Copy(output, bytes.NewReader(want))
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if metadata.ByteSize != int64(len(want)) || metadata.SHA256 == "" {
		t.Fatalf("metadata=%+v", metadata)
	}
	reader, opened, err := store.OpenCSVArtifact(context.Background(), tenant.ID, metadata.Ref)
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(reader)
	_ = reader.Close()
	if err != nil || !bytes.Equal(got, want) || opened != metadata {
		t.Fatalf("bytes=%d metadata=%+v err=%v", len(got), opened, err)
	}
	if _, _, err := store.OpenCSVArtifact(context.Background(), other.ID, metadata.Ref); !errors.Is(err, idmports.ErrCSVArtifactNotFound) {
		t.Fatalf("cross-tenant err=%v", err)
	}
}

// 読み取りはチャンクを csvChunkReadBatch 個ずつ問い合わせる。一度の問い合わせで
// 足りる大きさでは、続きを読む経路と、ちょうど読み切った後の空の問い合わせを通らない。
func TestCSVArtifactStoreReadsAcrossChunkBatches(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	store := &CSVArtifactStore{Pool: db}
	batchBytes := csvArtifactChunkBytes * csvChunkReadBatch
	for name, size := range map[string]int{
		"exact multiple of a batch": 2 * batchBytes,
		"one byte past two batches": 2*batchBytes + 1,
	} {
		t.Run(name, func(t *testing.T) {
			want := make([]byte, size)
			for i := range want {
				want[i] = byte(i % 251)
			}
			metadata, err := store.PutCSVArtifact(context.Background(), tenant.ID, func(output io.Writer) error {
				_, err := output.Write(want)
				return err
			})
			if err != nil {
				t.Fatal(err)
			}
			reader, _, err := store.OpenCSVArtifact(context.Background(), tenant.ID, metadata.Ref)
			if err != nil {
				t.Fatal(err)
			}
			got, err := io.ReadAll(reader)
			_ = reader.Close()
			if err != nil || !bytes.Equal(got, want) {
				t.Fatalf("read %d bytes, want %d; err=%v", len(got), len(want), err)
			}
		})
	}
}

func TestCSVArtifactStorePersistsResultPagesInExistingChunkTable(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	store := &CSVArtifactStore{Pool: db}
	metadata, err := store.PutCSVArtifactPages(context.Background(), tenant.ID, func(emit func([]byte) error) error {
		for _, page := range [][]byte{[]byte(`[{"row":2,"code":"invalid_email"}]`), []byte(`[{"row":202,"code":"source_managed"}]`)} {
			if err := emit(page); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	first, _, err := store.ReadCSVArtifactPage(context.Background(), tenant.ID, metadata.Ref, 0)
	if err != nil || string(first) != `[{"row":2,"code":"invalid_email"}]` {
		t.Fatalf("first page=%s err=%v", first, err)
	}
	page, opened, err := store.ReadCSVArtifactPage(context.Background(), tenant.ID, metadata.Ref, 1)
	if err != nil || string(page) != `[{"row":202,"code":"source_managed"}]` || opened.SHA256 != metadata.SHA256 {
		t.Fatalf("page=%s metadata=%+v err=%v", page, opened, err)
	}
	if _, _, err := store.ReadCSVArtifactPage(context.Background(), tenant.ID, metadata.Ref, -1); !errors.Is(err, idmports.ErrCSVArtifactNotFound) {
		t.Fatalf("negative page err=%v, want ErrCSVArtifactNotFound", err)
	}
}

// REQ-IDMANAGEMENT-080 の主要な使い方を、PostgreSQL の成果物ストアで固定する。
//
//spec:covers EX-IDMANAGEMENT-080-01: 境界の時刻より前に作った成果物を分割片ごと消し、境界の時刻ちょうどとそれより後の成果物を残すこと。
func TestCSVArtifactStoreDeletesArtifactsCreatedBeforeTheCutoff(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	store := &CSVArtifactStore{Pool: db}
	ctx := context.Background()
	cutoff := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	put := func(createdAt time.Time) string {
		t.Helper()
		metadata, err := store.PutCSVArtifact(ctx, tenant.ID, func(output io.Writer) error {
			_, err := io.WriteString(output, "preferred_username\nalice\n")
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(ctx, `UPDATE csv_artifacts SET created_at = $2 WHERE id = $1`, metadata.Ref, createdAt); err != nil {
			t.Fatal(err)
		}
		return metadata.Ref
	}
	expired := put(cutoff.Add(-time.Second))
	boundary := put(cutoff)
	fresh := put(cutoff.Add(time.Hour))

	deleted, err := store.DeleteCSVArtifactsCreatedBefore(ctx, cutoff)
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 1 {
		t.Fatalf("deleted=%d, want 1", deleted)
	}
	if _, _, err := store.OpenCSVArtifact(ctx, tenant.ID, expired); !errors.Is(err, idmports.ErrCSVArtifactNotFound) {
		t.Fatalf("期限を過ぎた成果物を開けた: err=%v", err)
	}
	var chunks int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM csv_artifact_chunks WHERE artifact_id = $1`, expired).Scan(&chunks); err != nil {
		t.Fatal(err)
	}
	if chunks != 0 {
		t.Fatalf("消した成果物の分割片が %d 件残った", chunks)
	}
	for _, ref := range []string{boundary, fresh} {
		reader, _, err := store.OpenCSVArtifact(ctx, tenant.ID, ref)
		if err != nil {
			t.Fatalf("成果物 %s が消えた: %v", ref, err)
		}
		_ = reader.Close()
	}
}
