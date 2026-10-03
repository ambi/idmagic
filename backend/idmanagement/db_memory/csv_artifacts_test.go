package db_memory

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	idmports "github.com/ambi/idmagic/backend/idmanagement/ports"
)

//spec:covers EX-IDMANAGEMENT-080-01: メモリの成果物ストアが、境界の時刻より前に保存した成果物だけを消すこと。
func TestCSVArtifactStoreDeletesArtifactsSavedBeforeTheCutoff(t *testing.T) {
	cutoff := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	store := NewCSVArtifactStore()
	ctx := context.Background()
	put := func(savedAt time.Time) string {
		t.Helper()
		store.Now = func() time.Time { return savedAt }
		metadata, err := store.PutCSVArtifact(ctx, "tenant-a", func(output io.Writer) error {
			_, err := io.WriteString(output, "preferred_username\nalice\n")
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		return metadata.Ref
	}
	expired := put(cutoff.Add(-time.Second))
	boundary := put(cutoff)

	deleted, err := store.DeleteCSVArtifactsCreatedBefore(ctx, cutoff)
	if err != nil || deleted != 1 {
		t.Fatalf("deleted=%d err=%v, want 1", deleted, err)
	}
	if _, _, err := store.OpenCSVArtifact(ctx, "tenant-a", expired); !errors.Is(err, idmports.ErrCSVArtifactNotFound) {
		t.Fatalf("期限を過ぎた成果物を開けた: err=%v", err)
	}
	if _, _, err := store.OpenCSVArtifact(ctx, "tenant-a", boundary); err != nil {
		t.Fatalf("境界の時刻の成果物が消えた: %v", err)
	}
}
