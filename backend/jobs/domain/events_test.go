package domain

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
)

// 監査はイベントの送信表現を保存し、テナントを `tenantId` から決める。項目名が TypeSpec の
// JobEnqueued などと異なると、Jobs の記録はテナントに限った監査の検索から黙って抜ける。
func TestJobEvents_MarshalWithTheContractFieldNames(t *testing.T) {
	at := time.Date(2026, 11, 23, 1, 2, 3, 0, time.UTC)
	next := at.Add(time.Minute)
	envelope := func(typ string, fields map[string]any) map[string]any {
		fields["type"] = typ
		fields["occurredAt"] = "2026-11-23T01:02:03Z"
		fields["jobId"] = "job-1"
		fields["tenantId"] = "tenant-a"
		return fields
	}
	cases := []struct {
		event spec.DomainEvent
		want  map[string]any
	}{
		{
			&JobEnqueued{At: at, JobID: "job-1", TenantID: "tenant-a", Kind: KindNoopEcho},
			envelope("JobEnqueued", map[string]any{"kind": string(KindNoopEcho)}),
		},
		{
			&JobStarted{At: at, JobID: "job-1", TenantID: "tenant-a", WorkerID: "worker-1", Attempt: 1},
			envelope("JobStarted", map[string]any{"workerId": "worker-1", "attempt": float64(1)}),
		},
		{
			&JobSucceeded{At: at, JobID: "job-1", TenantID: "tenant-a"},
			envelope("JobSucceeded", map[string]any{}),
		},
		{
			&JobFailed{At: at, JobID: "job-1", TenantID: "tenant-a", Attempt: 2, Terminal: true, Error: "boom"},
			envelope("JobFailed", map[string]any{"attempt": float64(2), "terminal": true, "error": "boom"}),
		},
		{
			&JobRetried{At: at, JobID: "job-1", TenantID: "tenant-a", Attempt: 2, NextRunAt: next},
			envelope("JobRetried", map[string]any{"attempt": float64(2), "nextRunAt": "2026-11-23T01:03:03Z"}),
		},
		{
			&JobCanceled{At: at, JobID: "job-1", TenantID: "tenant-a"},
			envelope("JobCanceled", map[string]any{}),
		},
	}
	for _, c := range cases {
		t.Run(c.event.EventType(), func(t *testing.T) {
			data, err := spec.MarshalDomainEvent(c.event)
			if err != nil {
				t.Fatal(err)
			}
			var got map[string]any
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("wire = %v, want %v", got, c.want)
			}
		})
	}
}
