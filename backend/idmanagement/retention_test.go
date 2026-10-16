package idmanagement_test

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/idmanagement"
	idmmemory "github.com/ambi/idmagic/backend/idmanagement/internal/db_memory"
	idmports "github.com/ambi/idmagic/backend/idmanagement/internal/ports"
	idmusecases "github.com/ambi/idmagic/backend/idmanagement/internal/usecases"
)

// 成果物の保存時刻はメモリの保存先の時計で決まるので、具体型に触れられるこのディレクトリで確かめる。
// 保持期限の削除のバッチがこの操作を呼ぶことは bootstrap のテストが確かめる。
//
//spec:covers EX-IDMANAGEMENT-080-01: 保持期限の削除のバッチが、IdManagement の成果物ストアから保持期限を過ぎた成果物を消し、新しい成果物を残すこと。
func TestPurgeExpiredCSVArtifactsKeepsArtifactsWithinRetention(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	store := idmmemory.NewCSVArtifactStore()
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
	expired := put(now.Add(-idmusecases.CSVArtifactRetention - time.Hour))
	fresh := put(now.Add(-time.Hour))

	if _, purged, err := (idmanagement.Module{CSVArtifacts: store}).PurgeExpiredCSVArtifacts(ctx, now); err != nil || !purged {
		t.Fatalf("purged=%v err=%v", purged, err)
	}
	if _, _, err := store.OpenCSVArtifact(ctx, "tenant-a", expired); !errors.Is(err, idmports.ErrCSVArtifactNotFound) {
		t.Fatalf("保持期限を過ぎた成果物が残った: err=%v", err)
	}
	if _, _, err := store.OpenCSVArtifact(ctx, "tenant-a", fresh); err != nil {
		t.Fatalf("新しい成果物が消えた: %v", err)
	}
}
