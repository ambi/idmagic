package usecases

// 監査イベントレコードから sidecar 検索属性を生成する抽出器 (wi-145 / wi-46)。
//
// event.type / outcome / actor.id / client.id / session.id / target.id / transaction.id /
// correlation.id / request.id の非 PII raw id に加え、actor.username / client.ip も ADR-104
// (ADR-046 の username/IP 条項を撤回) により平文のまま抽出する。actor.username は実アカウントが
// 確定しないイベント (AuthenticationFailed 等、payload.username を持つイベント) でのみ値を持つ。

import (
	"strconv"

	"github.com/ambi/idmagic/backend/audit/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// failureEventTypes / successAuthEventTypes は outcome 属性の分類 (認証系のみ)。
// spec の EventType 文字列と一致させ、handler の category マップとドリフトしないようにする。
var failureEventTypes = map[string]bool{
	(&spec.AuthenticationFailed{}).EventType():     true,
	(&spec.AuthenticationStepFailed{}).EventType(): true,
	(&spec.MfaChallengeFailed{}).EventType():       true,
}

var successAuthEventTypes = map[string]bool{
	(&spec.UserAuthenticated{}).EventType():           true,
	(&spec.AuthenticationStepCompleted{}).EventType(): true,
	(&spec.MfaChallengeSucceeded{}).EventType():       true,
	(&spec.MfaEnrollmentRequired{}).EventType():       true,
	(&spec.MfaEnrollmentCompleted{}).EventType():      true,
	(&spec.MfaEnrollmentBypassConsumed{}).EventType(): true,
	(&spec.SessionStarted{}).EventType():              true,
}

// ExtractSearchAttributes は監査イベントレコードから sidecar 検索属性 (attr_name -> 値) を返す。
// 値が空の属性は載せない。抽出属性が無ければ nil を返す。
func ExtractSearchAttributes(rec *ports.AuditEventRecord) map[string]string {
	if rec == nil {
		return nil
	}
	attrs := map[string]string{}
	set := func(field, value string) {
		if value != "" {
			attrs[field] = value
		}
	}

	set("event.type", rec.Type)
	set("outcome", eventOutcome(rec.Type))
	actorID := payloadString(rec.Payload, "actorUserId")
	if actorID == "" {
		actorID = payloadString(rec.Payload, "userId")
	}
	set("actor.id", actorID)
	targetID := payloadString(rec.Payload, "targetUserId")
	if targetID == "" && payloadString(rec.Payload, "actorUserId") != "" {
		targetID = payloadString(rec.Payload, "userId")
	}
	set("target.id", targetID)
	set("client.id", payloadString(rec.Payload, "clientId"))
	set("session.id", payloadString(rec.Payload, "sessionId"))
	set("transaction.id", payloadString(rec.Payload, "transactionId"))
	set("correlation.id", payloadString(rec.Payload, "correlationId"))
	set("request.id", payloadString(rec.Payload, "requestId"))
	set("actor.username", payloadString(rec.Payload, "username"))
	set("client.ip", payloadString(rec.Payload, "ip"))
	set("workflow.id", payloadString(rec.Payload, "workflowId"))
	set("workflow_run.id", payloadString(rec.Payload, "runId"))
	set("workflow_step.id", payloadNumberString(rec.Payload, "stepIndex"))

	if len(attrs) == 0 {
		return nil
	}
	return attrs
}

// eventOutcome は認証系イベントの成否を返す (success / failure)。非認証イベントは "" (未分類)。
func eventOutcome(eventType string) string {
	switch {
	case failureEventTypes[eventType]:
		return "failure"
	case successAuthEventTypes[eventType]:
		return "success"
	default:
		return ""
	}
}

func payloadString(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	if v, ok := payload[key].(string); ok {
		return v
	}
	return ""
}

// payloadNumberString reads a JSON number field (decoded as float64 by
// encoding/json into map[string]any) and renders it as an integer string,
// e.g. WorkflowStepFailed.stepIndex for the workflow_step.id search attribute.
func payloadNumberString(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	if v, ok := payload[key].(float64); ok {
		return strconv.FormatInt(int64(v), 10)
	}
	return ""
}
