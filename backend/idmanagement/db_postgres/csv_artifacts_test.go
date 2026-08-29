package db_postgres

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

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
	page, opened, err := store.ReadCSVArtifactPage(context.Background(), tenant.ID, metadata.Ref, 1)
	if err != nil || string(page) != `[{"row":202,"code":"source_managed"}]` || opened.SHA256 != metadata.SHA256 {
		t.Fatalf("page=%s metadata=%+v err=%v", page, opened, err)
	}
}
