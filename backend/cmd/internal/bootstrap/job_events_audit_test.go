package bootstrap

import (
	"context"
	"testing"
	"time"

	auditports "github.com/ambi/idmagic/backend/audit/ports"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	"github.com/ambi/idmagic/backend/shared/logging"
	"github.com/ambi/idmagic/backend/shared/spec"
)

//spec:covers REQ-AUDIT-001: 正式な配信点を通った Jobs の六種類のイベントは、呼び出し元のテナントの監査の検索に jobId とともに現れ、別のテナントの検索には現れない。
func TestJobEventsReachTheTenantScopedAuditSearch(t *testing.T) {
	ctx := context.Background()
	deps, err := assembleMemory(SharedConfig{})
	if err != nil {
		t.Fatal(err)
	}
	emit := deps.NewEmitFunc(logging.Default())
	at := time.Date(2026, 11, 23, 0, 0, 0, 0, time.UTC)
	events := []spec.DomainEvent{
		&jobsdomain.JobEnqueued{At: at, JobID: "job-1", TenantID: "tenant-a", Kind: jobsdomain.KindNoopEcho},
		&jobsdomain.JobStarted{At: at, JobID: "job-1", TenantID: "tenant-a", WorkerID: "worker-1", Attempt: 1},
		&jobsdomain.JobSucceeded{At: at, JobID: "job-1", TenantID: "tenant-a"},
		&jobsdomain.JobFailed{At: at, JobID: "job-1", TenantID: "tenant-a", Attempt: 1, Error: "boom"},
		&jobsdomain.JobRetried{At: at, JobID: "job-1", TenantID: "tenant-a", Attempt: 2, NextRunAt: at},
		&jobsdomain.JobCanceled{At: at, JobID: "job-1", TenantID: "tenant-a"},
	}
	for _, event := range events {
		emit(event)
	}

	for _, event := range events {
		records, err := deps.Audit.AuditEventRepo.List(ctx, auditports.AuditEventQuery{TenantID: "tenant-a", Type: event.EventType()})
		if err != nil || len(records) != 1 || records[0].TenantID != "tenant-a" || records[0].Payload["jobId"] != "job-1" {
			t.Errorf("%s in tenant-a = %+v err=%v, want one record with jobId job-1", event.EventType(), records, err)
		}
	}
	if records, err := deps.Audit.AuditEventRepo.List(ctx, auditports.AuditEventQuery{TenantID: "tenant-b"}); err != nil || len(records) != 0 {
		t.Fatalf("tenant-b = %+v err=%v, want no Jobs records", records, err)
	}
}
