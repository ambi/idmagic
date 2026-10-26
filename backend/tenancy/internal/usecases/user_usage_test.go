package usecases

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/ambi/idmagic/backend/shared/logging"
	"github.com/ambi/idmagic/backend/tenancy/domain"
	memory "github.com/ambi/idmagic/backend/tenancy/internal/db_memory"
)

//spec:covers REQ-TENANCY-049: 置き換える前の使用量と件数が異なるときだけ、テナント、置き換える前の使用量と件数を警告ログへ残す。
func TestRecountUserUsageWarnsOnlyWhenTheUsageDrifted(t *testing.T) {
	var buf bytes.Buffer
	previousLogger := logging.Default()
	logging.SetDefault(logging.New(&buf, slog.LevelDebug, "test", ""))
	t.Cleanup(func() { logging.SetDefault(previousLogger) })
	ctx := context.Background()
	repo := memory.NewQuotaRepository()
	if err := repo.CheckAndIncrement(ctx, "tenant-a", domain.ResourceUsers, 3); err != nil {
		t.Fatal(err)
	}
	count := func(n int) func(context.Context) (int, error) {
		return func(context.Context) (int, error) { return n, nil }
	}

	if err := RecountUserUsage(ctx, repo, "tenant-a", count(3)); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 0 {
		t.Fatalf("a recount without drift logged %s", buf.String())
	}
	if err := RecountUserUsage(ctx, repo, "tenant-a", count(2)); err != nil {
		t.Fatal(err)
	}
	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("log %q: %v", buf.String(), err)
	}
	if entry["level"] != "WARN" || entry["tenant_id"] != "tenant-a" || entry["previous_usage"] != float64(3) || entry["counted"] != float64(2) {
		t.Fatalf("warning = %v, want the tenant, the previous usage 3 and the count 2", entry)
	}
	usage, err := repo.GetUsage(ctx, "tenant-a")
	if err != nil || usage.Users != 2 {
		t.Fatalf("usage = %+v, %v; want users 2", usage, err)
	}
}
