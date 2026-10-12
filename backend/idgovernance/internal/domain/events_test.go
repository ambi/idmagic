package domain

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
)

//spec:covers EX-AUDIT-003-01: 公開する失敗イベント自体にも属性値やメール本文を含めない。
func TestWorkflowFailureEventsOmitSensitivePayloadFields(t *testing.T) {
	now := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	for _, event := range []spec.DomainEvent{
		&LifecycleWorkflowRunPartiallyFailed{At: now, TenantID: "acme", WorkflowID: "workflow-1", RunID: "run-1", TargetUserID: "user-1"},
		&LifecycleWorkflowStepFailed{At: now, TenantID: "acme", WorkflowID: "workflow-1", RunID: "run-1", StepIndex: 2, ActionKind: "send_email", ErrorCode: "smtp_unavailable"},
	} {
		data, err := json.Marshal(event)
		if err != nil {
			t.Fatal(err)
		}
		var payload map[string]any
		if err := json.Unmarshal(data, &payload); err != nil {
			t.Fatal(err)
		}
		for _, key := range []string{"attributeValue", "attributeValues", "emailBody", "body", "content"} {
			if _, ok := payload[key]; ok {
				t.Fatalf("%s must not carry %q: %s", event.EventType(), key, data)
			}
		}
	}
}
