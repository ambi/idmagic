package usecases_test

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	idmmemory "github.com/ambi/idmagic/backend/idmanagement/internal/db_memory"
	idmports "github.com/ambi/idmagic/backend/idmanagement/internal/ports"
	idmusecases "github.com/ambi/idmagic/backend/idmanagement/internal/usecases"
)

//spec:covers REQ-IDMANAGEMENT-080: 実行の期限は元の入力からの残り時間になり、期限に達すると実行コンテキストを終了する。
func TestCSVImportContextExpiresAtTheInputDeadline(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Microsecond)
	store := idmmemory.NewCSVArtifactStore()
	store.Now = func() time.Time { return now.Add(-30 * 24 * time.Hour).Add(20 * time.Millisecond) }
	artifact, err := store.PutCSVArtifact(context.Background(), "tenant", func(w io.Writer) error { _, err := io.WriteString(w, "alice@example.com"); return err })
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel, err := idmusecases.CSVImportContext(context.Background(), store, "tenant", artifact.Ref, now)
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()
	select {
	case <-ctx.Done():
		if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
			t.Fatalf("err=%v", ctx.Err())
		}
	case <-time.After(time.Second):
		t.Fatal("入力の期限で実行を終了しなかった")
	}
}

type slowCSVArtifactStore struct{ *idmmemory.CSVArtifactStore }

func (s slowCSVArtifactStore) OpenCSVArtifact(ctx context.Context, tenantID, ref string) (io.ReadCloser, idmports.CSVArtifact, error) {
	time.Sleep(40 * time.Millisecond)
	return s.CSVArtifactStore.OpenCSVArtifact(ctx, tenantID, ref)
}

//spec:covers REQ-IDMANAGEMENT-080: メタデータの取得が入力の残り時間を超えた後に、実行の猶予を追加しない。
func TestCSVImportContextIncludesMetadataLookupInItsDeadline(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Microsecond)
	store := idmmemory.NewCSVArtifactStore()
	store.Now = func() time.Time { return now.Add(-30 * 24 * time.Hour).Add(20 * time.Millisecond) }
	artifact, err := store.PutCSVArtifact(context.Background(), "tenant", func(w io.Writer) error { _, err := io.WriteString(w, "alice@example.com"); return err })
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel, err := idmusecases.CSVImportContext(context.Background(), slowCSVArtifactStore{store}, "tenant", artifact.Ref, now)
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()
	if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
		t.Fatalf("入力の期限を延長した: %v", ctx.Err())
	}
}
