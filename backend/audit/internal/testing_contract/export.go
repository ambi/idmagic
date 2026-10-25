package testing_contract

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/audit/ports"
)

// RunExport は実ストアの確定、分離、再配送と期限の共通契約を検査する。
//
//spec:covers REQ-AUDIT-010: 全チャンクの確定前の障害で部分成果物を公開しない。
//spec:covers REQ-AUDIT-011: 成果物はテナントに閉じ、期限で容量からも除去する。
func RunExport(t *testing.T, store ports.ExportArtifactStore, tenantA, tenantB, jobID string, now time.Time) {
	t.Helper()
	ctx := context.Background()
	data := bytes.Repeat([]byte("audit-event\n"), 200000)
	failed := errors.New("build interrupted")
	for _, beforePublish := range []bool{false, true} {
		_, err := store.Write(ctx, tenantA, jobID, func(out io.Writer) (ports.ExportCompletion, error) {
			if _, err := out.Write(data); err != nil {
				return ports.ExportCompletion{}, err
			}
			if !beforePublish {
				return ports.ExportCompletion{}, failed
			}
			return ports.ExportCompletion{Events: 200000, ExpiresAt: now.Add(time.Hour), BeforePublish: func() error { return failed }}, nil
		})
		if !errors.Is(err, failed) {
			t.Fatalf("Write failure=%v", err)
		}
		reader, _, err := store.Open(ctx, tenantA, jobID)
		if reader != nil {
			_ = reader.Close()
		}
		if !errors.Is(err, ports.ErrExportArtifactNotFound) {
			t.Fatalf("partial Open=%v", err)
		}
		if size, err := store.Usage(ctx, tenantA); err != nil || size != 0 {
			t.Fatalf("partial Usage=%d %v", size, err)
		}
	}
	meta, err := store.Write(ctx, tenantA, jobID, func(out io.Writer) (ports.ExportCompletion, error) {
		_, err := out.Write(data)
		return ports.ExportCompletion{Events: 200000, ExpiresAt: now.Add(time.Hour)}, err
	})
	if err != nil {
		t.Fatal(err)
	}
	if meta.ByteSize != int64(len(data)) || meta.SHA256 == "" {
		t.Fatalf("metadata=%+v", meta)
	}
	repeated, err := store.Write(ctx, tenantA, jobID, func(io.Writer) (ports.ExportCompletion, error) {
		t.Error("replayed write")
		return ports.ExportCompletion{}, failed
	})
	if err != nil || repeated.Ref != meta.Ref {
		t.Fatalf("replay=%+v %v", repeated, err)
	}
	reader, got, err := store.Open(ctx, tenantA, jobID)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(reader)
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	if err != nil || !bytes.Equal(body, data) || got.SHA256 != meta.SHA256 {
		t.Fatalf("download bytes=%d err=%v", len(body), err)
	}
	reader, _, err = store.Open(ctx, tenantB, jobID)
	if reader != nil {
		_ = reader.Close()
	}
	if !errors.Is(err, ports.ErrExportArtifactNotFound) {
		t.Fatalf("other tenant=%v", err)
	}
	if size, err := store.Usage(ctx, tenantA); err != nil || size != meta.ByteSize {
		t.Fatalf("usage=%d %v", size, err)
	}
	if removed, err := store.Expire(ctx, meta.ExpiresAt.Add(-time.Microsecond)); err != nil || len(removed) != 0 {
		t.Fatalf("early expiry=%+v %v", removed, err)
	}
	if removed, err := store.Expire(ctx, meta.ExpiresAt); err != nil || len(removed) != 1 {
		t.Fatalf("expiry=%+v %v", removed, err)
	}
	if size, err := store.Usage(ctx, tenantA); err != nil || size != 0 {
		t.Fatalf("expired usage=%d %v", size, err)
	}
}
