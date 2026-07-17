package bootstrap

import (
	"testing"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"

	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"

	"github.com/ambi/idmagic/backend/shared/spec"
)

// wi-35: emit 時点で event に載せた tenantId が、監査レコードの TenantID に
// 流れ込むこと。これがテナント所属 admin の監査ビュー絞り込み
// (auditEventMatches) に効く。
func TestNewAuditEventRecordExtractsTenantID(t *testing.T) {
	now := time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC)
	rec, err := NewAuditEventRecord(&authdomain.UserAuthenticated{
		At: now, TenantID: "acme", UserID: "user_alice", AMR: []string{"pwd"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if rec.Type != "UserAuthenticated" {
		t.Fatalf("type = %q, want UserAuthenticated", rec.Type)
	}
	if rec.TenantID != "acme" {
		t.Fatalf("tenant_id = %q, want acme (emit-time tenantId must reach the audit record)", rec.TenantID)
	}
	if got, _ := rec.Payload["tenantId"].(string); got != "acme" {
		t.Fatalf("payload tenantId = %q, want acme", got)
	}
}

// wi-36: oauth2 / token / consent / client 系の event も emit 時の tenantId が
// 監査レコードに流れ込む。代表として ClientRegistered と AccessTokenIssued。
func TestNewAuditEventRecordExtractsTenantIDForOAuth2Events(t *testing.T) {
	now := time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC)
	for _, ev := range []spec.DomainEvent{
		&oauthdomain.ClientRegistered{At: now, TenantID: "acme", ClientID: "demo-client"},
		&oauthdomain.AccessTokenIssued{At: now, TenantID: "acme", JTI: "jti", ClientID: "demo-client", UserID: "user_alice"},
		&oauthdomain.ConsentGrantedEvent{At: now, TenantID: "acme", UserID: "user_alice", ClientID: "demo-client"},
	} {
		rec, err := NewAuditEventRecord(ev)
		if err != nil {
			t.Fatalf("%s: %v", ev.EventType(), err)
		}
		if rec.TenantID != "acme" {
			t.Fatalf("%s: tenant_id = %q, want acme", ev.EventType(), rec.TenantID)
		}
	}
}

// wi-221: LifecycleWorkflow* / WorkflowRun* / WorkflowStepFailed の workflowId /
// runId / stepIndex が workflow.id / workflow_run.id / workflow_step.id の
// sidecar 検索属性として抽出され、admin が filter で監査ログを検索できること。
func TestNewAuditEventRecordExtractsLifecycleWorkflowSearchAttributes(t *testing.T) {
	now := time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC)
	rec, err := NewAuditEventRecord(&idmdomain.LifecycleWorkflowCreated{At: now, TenantID: "acme", ActorUserID: "admin-1", WorkflowID: "workflow-1"})
	if err != nil {
		t.Fatal(err)
	}
	if rec.SearchAttributes["workflow.id"] != "workflow-1" {
		t.Fatalf("workflow.id = %q, want workflow-1", rec.SearchAttributes["workflow.id"])
	}

	rec, err = NewAuditEventRecord(&idmdomain.LifecycleWorkflowRunSucceeded{At: now, TenantID: "acme", WorkflowID: "workflow-1", RunID: "run-1", TargetUserID: "user-1"})
	if err != nil {
		t.Fatal(err)
	}
	if rec.SearchAttributes["workflow.id"] != "workflow-1" || rec.SearchAttributes["workflow_run.id"] != "run-1" {
		t.Fatalf("search attrs = %#v, want workflow.id=workflow-1 workflow_run.id=run-1", rec.SearchAttributes)
	}

	rec, err = NewAuditEventRecord(&idmdomain.LifecycleWorkflowStepFailed{At: now, TenantID: "acme", WorkflowID: "workflow-1", RunID: "run-1", StepIndex: 2, ActionKind: "disable_user", ErrorCode: "resource_not_found"})
	if err != nil {
		t.Fatal(err)
	}
	if rec.SearchAttributes["workflow_run.id"] != "run-1" || rec.SearchAttributes["workflow_step.id"] != "2" {
		t.Fatalf("search attrs = %#v, want workflow_run.id=run-1 workflow_step.id=2", rec.SearchAttributes)
	}
}

// tenantId を持たない event は従来どおり空テナントで記録される (回帰の境界)。
// wi-36: SigningKeyRotated は per-tenant 鍵が無いため意図的に tenant_id 空。
func TestNewAuditEventRecordWithoutTenantIDStaysEmpty(t *testing.T) {
	now := time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC)
	for _, ev := range []spec.DomainEvent{
		&spec.EmailSent{At: now, ToHash: "deadbeef", Purpose: "password_reset", Delivered: true},
		&signingdomain.SigningKeyRotated{At: now, NewKID: "kid-2", PreviousKID: "kid-1"},
	} {
		rec, err := NewAuditEventRecord(ev)
		if err != nil {
			t.Fatalf("%s: %v", ev.EventType(), err)
		}
		if rec.TenantID != "" {
			t.Fatalf("%s: tenant_id = %q, want empty", ev.EventType(), rec.TenantID)
		}
	}
}
