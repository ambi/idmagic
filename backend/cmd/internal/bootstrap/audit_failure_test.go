package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	auditmemory "github.com/ambi/idmagic/backend/audit/db_memory"
	auditports "github.com/ambi/idmagic/backend/audit/ports"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	"github.com/ambi/idmagic/backend/shared/logging"
	metrics "github.com/ambi/idmagic/backend/shared/observability/metrics_prometheus"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

//spec:covers REQ-AUDIT-008: 変換と追記の失敗を実際の /metrics と本文を含まないログで観測でき、配信点は正常に戻る。
func TestAuditWriteFailureIsObservable(t *testing.T) {
	for _, stage := range []string{"conversion", "append"} {
		t.Run(stage, func(t *testing.T) {
			deps, _ := newSecurityNotificationDeps(t)
			m, err := metrics.NewMetrics("audit-test", "test")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = m.Shutdown(t.Context()) })
			deps.AuditFailures = m
			deps.OAuth2.EventSink = auditTestSink{}
			repo := &auditAppendFailure{AuditEventRepository: auditmemory.NewAuditEventStore(100)}
			deps.Audit.AuditEventRepo = repo
			var logs bytes.Buffer
			emit := deps.NewEmitFunc(logging.New(&logs, slog.LevelDebug, "audit-test", "test"))
			assertAuditFailureCount(t, m, "conversion", 0)
			assertAuditFailureCount(t, m, "append", 0)
			now := time.Date(2026, 10, 11, 1, 2, 3, 0, time.UTC)
			var event spec.DomainEvent = &idmdomain.UserUpdated{
				At: now, TenantID: "tenant-a", ActorUserID: "admin", TargetUserID: "alice", ChangedFields: []string{"private-field"},
			}
			if stage == "conversion" {
				event = &unserializableAuditEvent{At: now, Secret: "private-field", Unsupported: make(chan int)}
			} else {
				repo.err = errors.New("audit repository unavailable")
			}
			emit(event)
			assertAuditFailureCount(t, m, stage, 1)
			other := "conversion"
			if stage == other {
				other = "append"
			}
			assertAuditFailureCount(t, m, other, 0)
			var entry map[string]any
			if err := json.Unmarshal(bytes.TrimSpace(logs.Bytes()), &entry); err != nil {
				t.Fatalf("failure log = %s: %v", logs.String(), err)
			}
			if entry["audit_stage"] != stage || entry["event_type"] != event.EventType() || entry["occurred_at"] != now.Format(time.RFC3339Nano) {
				t.Fatalf("failure metadata = %#v", entry)
			}
			if strings.Contains(logs.String(), "private-field") || entry["payload"] != nil {
				t.Fatalf("event body leaked into failure log: %s", logs.String())
			}
			if stage == "append" {
				if repo.record == nil || entry["audit_event_id"] != repo.record.ID || entry["tenant_id"] != "tenant-a" || repo.tenantID != "tenant-a" {
					t.Fatalf("append correlation = %#v, record=%+v, tenant=%s", entry, repo.record, repo.tenantID)
				}
			} else if repo.attempts != 0 {
				t.Fatalf("conversion failure attempted append: %d", repo.attempts)
			}
			// 原因が直った後の正常な発生を、既存の経路で記録できる。
			repo.err = nil
			deps.OAuth2.EventSink = auditTestSink{err: errors.New("console unavailable")}
			emit(&idmdomain.UserUpdated{At: now, TenantID: "tenant-a", TargetUserID: "alice"})
			assertAuditFailureCount(t, m, stage, 1)
			assertAuditFailureCount(t, m, other, 0)
			rows, err := repo.List(t.Context(), auditports.AuditEventQuery{TenantID: "tenant-a"})
			if err != nil || len(rows) != 1 {
				t.Fatalf("later delivery = %v, %v", rows, err)
			}
		})
	}
}

func assertAuditFailureCount(t *testing.T, m *metrics.Metrics, stage string, count int) {
	t.Helper()
	rec := httptest.NewRecorder()
	m.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", http.NoBody))
	if rec.Code != http.StatusOK {
		t.Fatalf("/metrics status=%d", rec.Code)
	}
	var observed []string
	for line := range strings.SplitSeq(rec.Body.String(), "\n") {
		if strings.HasPrefix(line, "audit_record_failures_total{") {
			for _, forbidden := range []string{"tenant_id=", "user_id=", "audit_event_id=", "event_type="} {
				if strings.Contains(line, forbidden) {
					t.Fatalf("high-cardinality audit label: %s", line)
				}
			}
			observed = append(observed, line)
			if strings.Contains(line, fmt.Sprintf("stage=%q", stage)) && strings.HasSuffix(line, fmt.Sprintf("} %d", count)) {
				return
			}
		}
	}
	t.Fatalf("/metrics audit failures: %v; want stage=%s count=%d", observed, stage, count)
}

type auditAppendFailure struct {
	auditports.AuditEventRepository
	err      error
	record   *auditports.AuditEventRecord
	tenantID string
	attempts int
}

func (r *auditAppendFailure) Append(ctx context.Context, rec *auditports.AuditEventRecord) error {
	r.record = rec
	r.tenantID = tenantports.TenantID(ctx)
	r.attempts++
	if r.err != nil {
		return r.err
	}
	return r.AuditEventRepository.Append(ctx, rec)
}

type auditTestSink struct{ err error }

func (s auditTestSink) Emit(context.Context, spec.DomainEvent) error { return s.err }

type unserializableAuditEvent struct {
	At          time.Time `json:"occurredAt"`
	Secret      string    `json:"secret"`
	Unsupported chan int  `json:"unsupported"`
}

func (*unserializableAuditEvent) EventType() string       { return "AuditConversionTest" }
func (e *unserializableAuditEvent) OccurredAt() time.Time { return e.At }
