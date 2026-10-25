// Package testing_contract defines the shared identity-management persistence contract.
package testing_contract

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/ambi/idmagic/backend/idmanagement/internal/ports"
)

type Fixture struct {
	Artifacts ports.CSVArtifactStore
	TenantA   string
	TenantB   string
}

type NewFixture func(t *testing.T) Fixture

func Run(t *testing.T, newFixture NewFixture) {
	t.Helper()
	f := newFixture(t)
	ctx := context.Background()
	payload := bytes.Repeat([]byte("id,email\n1,alice@example.com\n"), 4_000)
	metadata, err := f.Artifacts.PutCSVArtifact(ctx, f.TenantA, func(output io.Writer) error {
		_, err := output.Write(payload)
		return err
	})
	if err != nil {
		t.Fatalf("PutCSVArtifact: %v", err)
	}
	if metadata.Ref == "" || metadata.TenantID != f.TenantA || metadata.ByteSize != int64(len(payload)) || metadata.SHA256 == "" {
		t.Fatalf("metadata = %+v", metadata)
	}
	reader, opened, err := f.Artifacts.OpenCSVArtifact(ctx, f.TenantA, metadata.Ref)
	if err != nil {
		t.Fatalf("OpenCSVArtifact: %v", err)
	}
	got, readErr := io.ReadAll(reader)
	closeErr := reader.Close()
	if readErr != nil || closeErr != nil || !bytes.Equal(got, payload) || opened != metadata {
		t.Fatalf("opened bytes=%d metadata=%+v readErr=%v closeErr=%v", len(got), opened, readErr, closeErr)
	}
	if reader, _, err := f.Artifacts.OpenCSVArtifact(ctx, f.TenantB, metadata.Ref); reader != nil || !errors.Is(err, ports.ErrCSVArtifactNotFound) {
		t.Fatalf("other tenant OpenCSVArtifact = (%v, %v)", reader, err)
	}

	pages := [][]byte{[]byte(`[{"row":1}]`), []byte(`[{"row":2}]`)}
	pageMetadata, err := f.Artifacts.PutCSVArtifactPages(ctx, f.TenantA, func(emit func([]byte) error) error {
		for _, page := range pages {
			if err := emit(page); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("PutCSVArtifactPages: %v", err)
	}
	for index, want := range pages {
		page, readMetadata, err := f.Artifacts.ReadCSVArtifactPage(ctx, f.TenantA, pageMetadata.Ref, index)
		if err != nil || !bytes.Equal(page, want) || readMetadata != pageMetadata {
			t.Fatalf("ReadCSVArtifactPage(%d) = (%q, %+v, %v)", index, page, readMetadata, err)
		}
	}
	if page, _, err := f.Artifacts.ReadCSVArtifactPage(ctx, f.TenantA, pageMetadata.Ref, -1); page != nil || !errors.Is(err, ports.ErrCSVArtifactNotFound) {
		t.Fatalf("negative ReadCSVArtifactPage = (%q, %v)", page, err)
	}
	usageReader, ok := f.Artifacts.(ports.CSVArtifactUsageReader)
	if !ok {
		t.Fatal("artifact store cannot measure its usage")
	}
	if _, err := f.Artifacts.PutCSVArtifact(ctx, f.TenantB, func(w io.Writer) error { _, err := w.Write([]byte("other")); return err }); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Artifacts.PutCSVArtifact(ctx, f.TenantA, func(w io.Writer) error { _, _ = w.Write([]byte("partial")); return errors.New("failed write") }); err == nil {
		t.Fatal("failed write succeeded")
	}
	for tenantID, want := range map[string]int64{f.TenantA: metadata.ByteSize + pageMetadata.ByteSize, f.TenantB: 5} {
		got, err := usageReader.CSVArtifactUsage(ctx, tenantID)
		if err != nil || got != want {
			t.Fatalf("usage of %s=%d err=%v, want %d", tenantID, got, err, want)
		}
	}
}
