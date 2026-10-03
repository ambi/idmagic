package bootstrap

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/idmanagement"
	idmmemory "github.com/ambi/idmagic/backend/idmanagement/db_memory"
	idmports "github.com/ambi/idmagic/backend/idmanagement/ports"
	idmusecases "github.com/ambi/idmagic/backend/idmanagement/usecases"
)

//spec:covers EX-IDMANAGEMENT-080-01: 保持期限の削除のバッチが、IdManagement の成果物ストアから保持期限を過ぎた成果物を消し、新しい成果物を残すこと。
func TestRetentionSweepDeletesCSVArtifactsPastRetention(t *testing.T) {
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

	deps := &Dependencies{IdManagement: idmanagement.Module{CSVArtifacts: store}}
	if err := RunRetentionSweepOnce(ctx, deps, now); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.OpenCSVArtifact(ctx, "tenant-a", expired); !errors.Is(err, idmports.ErrCSVArtifactNotFound) {
		t.Fatalf("保持期限を過ぎた成果物が残った: err=%v", err)
	}
	if _, _, err := store.OpenCSVArtifact(ctx, "tenant-a", fresh); err != nil {
		t.Fatalf("新しい成果物が消えた: %v", err)
	}
}
