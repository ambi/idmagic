package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	auditports "github.com/ambi/idmagic/backend/audit/ports"
	"github.com/ambi/idmagic/backend/cmd/internal/bootstrap"
	jobdomain "github.com/ambi/idmagic/backend/jobs/domain"
	"github.com/ambi/idmagic/backend/shared/security/envelope_crypto"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

type stubFieldMigrator struct{ pending int }

func (stubFieldMigrator) ReencryptBatch(context.Context, string, int, int) (int, error) {
	return 0, nil
}

func (m stubFieldMigrator) PendingCount(context.Context, string, int) (int, error) {
	return m.pending, nil
}

// wrapFailingCrypto は一つのテナントのラップだけを失敗させる。ほかのテナントの処理が続くことを観測する。
type wrapFailingCrypto struct {
	envelope_crypto.EnvelopeCrypto
	tenantID string
}

func (c wrapFailingCrypto) Wrap(ctx context.Context, tenantID string, dek []byte) ([]byte, string, error) {
	if tenantID == c.tenantID {
		return nil, "", errors.New("provider unavailable")
	}
	return c.EnvelopeCrypto.Wrap(ctx, tenantID, dek)
}

// dataKeyLifecycleWorld は本番と同じ組み立てに、周期を過ぎた due、周期前の fresh、DEK のない empty を置く。
type dataKeyLifecycleWorld struct {
	deps *bootstrap.Dependencies
}

func newDataKeyLifecycleWorld(t *testing.T, failingTenant string) (*dataKeyLifecycleWorld, launch) {
	t.Helper()
	w := &dataKeyLifecycleWorld{}
	in := launch{
		loader: bootstrap.NewConfigLoader(func(string) string { return "" }),
		assemble: func(ctx context.Context, cfg bootstrap.SharedConfig) (*bootstrap.Dependencies, error) {
			deps, err := bootstrap.Assemble(ctx, cfg)
			if err != nil {
				return nil, err
			}
			w.deps = deps
			now := time.Now().UTC()
			for _, id := range []string{"due", "due-too", "fresh", "empty"} {
				tenant := &tenancydomain.Tenant{ID: id, Realm: id, DisplayName: id, Status: tenancydomain.TenantStatusActive, CreatedAt: now, UpdatedAt: now}
				if err := deps.Tenancy.TenantRepo.Save(ctx, tenant); err != nil {
					t.Fatal(err)
				}
			}
			aged := now.Add(-100 * 24 * time.Hour)
			for id, at := range map[string]time.Time{"due": aged, "due-too": aged, "fresh": now} {
				if _, err := deps.DataKeys.BootstrapTenantDataKey(ctx, id, at); err != nil {
					t.Fatal(err)
				}
			}
			deps.DataKeys.Migrators.Register("fixture", stubFieldMigrator{})
			if failingTenant != "" {
				deps.DataKeys.Crypto = wrapFailingCrypto{EnvelopeCrypto: deps.DataKeys.Crypto, tenantID: failingTenant}
			}
			return deps, nil
		},
	}
	return w, in
}

func (w *dataKeyLifecycleWorld) activeVersion(t *testing.T, tenantID string) int {
	t.Helper()
	key, err := w.deps.DataKeys.Repository.FindActive(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("%s: %v", tenantID, err)
	}
	return key.Version
}

func (w *dataKeyLifecycleWorld) rotatedAudits(t *testing.T, tenantID string) int {
	t.Helper()
	records, err := w.deps.Audit.AuditEventRepo.List(context.Background(), auditports.AuditEventQuery{TenantID: tenantID, Type: "DataEncryptionKeyRotated"})
	if err != nil {
		t.Fatal(err)
	}
	return len(records)
}

func (w *dataKeyLifecycleWorld) reencryptionJobs(t *testing.T, tenantID string) int {
	t.Helper()
	jobs, err := w.deps.Jobs.Repo.ListByTenantAndKinds(context.Background(), tenantID, []jobdomain.JobKind{jobdomain.KindDataKeyReencryption}, 10)
	if err != nil {
		t.Fatal(err)
	}
	return len(jobs)
}

//spec:covers REQ-DATAKEYS-007, REQ-DATAKEYS-002: 周期を過ぎたテナントだけバージョンが増え、再暗号化のジョブと監査が残り、DEK のないテナントには作らず、再実行では増えない。
func TestDataKeyLifecycleRotatesOnlyDueTenants(t *testing.T) {
	w, in := newDataKeyLifecycleWorld(t, "")
	if err := run(t.Context(), []string{"data-key-lifecycle", "--cadence-days=90"}, in); err != nil {
		t.Fatal(err)
	}

	if got := w.activeVersion(t, "due"); got != 2 {
		t.Fatalf("due active version = %d, want 2", got)
	}
	if got := w.activeVersion(t, "fresh"); got != 1 {
		t.Fatalf("fresh active version = %d, want 1", got)
	}
	if _, err := w.deps.DataKeys.Repository.FindActive(t.Context(), "empty"); err == nil {
		t.Fatal("a DEK was created for the tenant without one")
	}
	if got := w.reencryptionJobs(t, "due"); got != 1 {
		t.Fatalf("due reencryption jobs = %d, want 1", got)
	}
	if got := w.reencryptionJobs(t, "fresh"); got != 0 {
		t.Fatalf("fresh reencryption jobs = %d, want 0", got)
	}
	if got := w.rotatedAudits(t, "due"); got != 1 {
		t.Fatalf("due rotated audits = %d, want 1", got)
	}
	if got := w.rotatedAudits(t, "fresh"); got != 0 {
		t.Fatalf("fresh rotated audits = %d, want 0", got)
	}

	if err := runDataKeyLifecycle(t.Context(), w.deps, dataKeyLifecycleConfig{cadence: 90 * 24 * time.Hour}, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if got := w.activeVersion(t, "due"); got != 2 {
		t.Fatalf("rerun rotated again: active version = %d", got)
	}
}

//spec:covers REQ-DATAKEYS-007: 一つのテナントの失敗でも残りのテナントを処理し、バッチは失敗で終わる。
func TestDataKeyLifecycleContinuesPastFailingTenant(t *testing.T) {
	w, in := newDataKeyLifecycleWorld(t, "due")
	err := run(t.Context(), []string{"data-key-lifecycle"}, in)
	if err == nil || !strings.Contains(err.Error(), "due") {
		t.Fatalf("error = %v, want one naming the failing tenant", err)
	}
	if got := w.activeVersion(t, "due"); got != 1 {
		t.Fatalf("failing tenant active version = %d, want 1", got)
	}
	if got := w.activeVersion(t, "due-too"); got != 2 {
		t.Fatalf("other tenant active version = %d, want 2", got)
	}
}

//spec:covers REQ-DATAKEYS-007: 周期に正の整数の日数以外を指定すると、依存を組み立てる前に終了する。正しい指定では組み立てが呼ばれることを対照にする。
func TestDataKeyLifecycleRefusesInvalidCadenceBeforeAssembling(t *testing.T) {
	for _, flags := range [][]string{{"--cadence-days=0"}, {"--cadence-days=-1"}, {"--cadence-days=x"}, {"extra"}} {
		in, calls := countedLaunch(nil)
		if err := run(t.Context(), append([]string{"data-key-lifecycle"}, flags...), in); err == nil || errors.Is(err, errAssembled) {
			t.Errorf("%v: error = %v, want a configuration error", flags, err)
		}
		if *calls != 0 {
			t.Errorf("%v: assembled %d time(s)", flags, *calls)
		}
	}
	in, calls := countedLaunch(nil)
	if err := run(t.Context(), []string{"data-key-lifecycle", "--cadence-days=1"}, in); !errors.Is(err, errAssembled) || *calls != 1 {
		t.Fatalf("valid config: error = %v, calls = %d", err, *calls)
	}
}
