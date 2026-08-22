package usecases

// 監査イベントレコードから sidecar 検索属性を生成する抽出器 (wi-145 / wi-46 / wi-377)。
//
// event.type / outcome / actor.id / client.id / session.id / target.id / transaction.id /
// correlation.id / request.id の非 PII raw id に加え、actor.username / client.ip も
// (username/IP 条項を撤回) により平文のまま抽出する。actor.username は実アカウントが
// 確定しないイベント (AuthenticationFailed 等、payload.username を持つイベント) でのみ値を持つ。
//
// 委譲の軸 (actor.type / agent.id / delegation.*) は、エージェントが利用者を代行した操作を
// 利用者本人の操作と区別して引くためのもの (REQ-AUDIT-005 / REQ-AUDIT-006)。値はイベントの
// ペイロードから読むだけで、監査側で導出し直さない。とりわけ delegation.mode は
// REQ-OAUTH2-049 の導出関数が出した値をそのまま使う。二つ目の導出を置けば、
// イントロスペクションの応答と監査の記録が食い違いうる。

import (
	"strconv"

	"github.com/ambi/idmagic/backend/audit/ports"
)

// failureEventTypes / successAuthEventTypes は outcome 属性の分類 (認証系のみ)。
// spec の EventType 文字列と一致させ、handler の category マップとドリフトしないようにする。
var failureEventTypes = map[string]bool{
	"AuthenticationFailed":     true,
	"AuthenticationStepFailed": true,
	"MfaChallengeFailed":       true,
}

var successAuthEventTypes = map[string]bool{
	"UserAuthenticated":           true,
	"AuthenticationStepCompleted": true,
	"MfaChallengeSucceeded":       true,
	"MfaEnrollmentRequired":       true,
	"MfaEnrollmentCompleted":      true,
	"MfaEnrollmentBypassConsumed": true,
	"SessionStarted":              true,
}

// agentActorEventTypes は payload の agentId が「行為者」を指すイベント (wi-377)。
//
// agentId を持つイベントの多くでは、Agent は行為者ではなく操作の対象である。管理者が
// Agent を登録・無効化する AgentRegistered / AgentDisabled や、利用者が承認を与える
// BackchannelAuthApproved がそれにあたる。行為者はいずれも人間である。種別を agentId の
// 有無から推測すると、これらが Agent の操作として記録される。
//
// そのため、Agent が自ら振る舞う経路を列挙する。列挙に無いイベントの agentId は対象を
// 指すものとして扱い、agent.id 軸にだけ載せる。agent.id は「この Agent に関するイベント」、
// actor.type は「その Agent が行為者だったか」を表す、別の問いに答える軸である。
var agentActorEventTypes = map[string]bool{
	"TokenExchanged":           true,
	"WorkloadTokenExchanged":   true,
	"AgentApprovalRequired":    true,
	"BackchannelAuthRequested": true,
}

// ExtractSearchAttributes は監査イベントレコードから sidecar 検索属性 (attr_name -> 値の並び) を
// 返す。値が空の属性は載せない。抽出属性が無ければ nil を返す。多値になるのは委譲チェーンの
// 参加者 (delegation.actor) だけで、他の軸は値を 1 つしか持たない。
func ExtractSearchAttributes(rec *ports.AuditEventRecord) map[string][]string {
	if rec == nil {
		return nil
	}
	attrs := map[string][]string{}
	set := func(field, value string) {
		if value != "" {
			attrs[field] = []string{value}
		}
	}
	setAll := func(field string, values []string) {
		if len(values) > 0 {
			attrs[field] = values
		}
	}

	set("event.type", rec.Type)
	set("outcome", eventOutcome(rec.Type))
	agentID := payloadString(rec.Payload, "agentId")
	agentActor := agentID != "" && agentActorEventTypes[rec.Type]
	actorID := payloadString(rec.Payload, "actorUserId")
	if actorID == "" {
		if agentActor {
			// Agent が行為者のイベントで userId へ読み替えない。読み替えると、Agent が
			// 利用者の代わりに行った操作が、利用者本人の操作と検索上まったく区別できなくなる。
			// 行為者を名指せる値は Agent 自身の識別子である。
			actorID = agentID
		} else {
			actorID = payloadString(rec.Payload, "userId")
		}
	}
	set("actor.id", actorID)
	set("agent.id", agentID)
	if actorType := actorType(agentActor, actorID); actorType != "" {
		set("actor.type", actorType)
	}
	targetID := payloadString(rec.Payload, "targetUserId")
	if targetID == "" && (payloadString(rec.Payload, "actorUserId") != "" || agentActor) {
		// Agent が行為者なら、payload の userId は代行された側であって行為者ではない。
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
	setAll("delegation.actor", payloadStrings(rec.Payload, "actorChain"))
	set("delegation.depth", delegationDepth(rec.Payload))
	set("delegation.mode", payloadString(rec.Payload, "delegationMode"))

	if len(attrs) == 0 {
		return nil
	}
	return attrs
}

// actorType は行為者の種別を返す。行為者が定まらないイベント (対象も操作者も持たない
// 設定変更など) では "" を返し、軸そのものを載せない。値を補って埋めると、種別を持たない
// イベントが種別で絞り込んだ結果に混ざる。
func actorType(agentActor bool, actorID string) string {
	switch {
	case agentActor:
		return ports.ActorTypeAgent
	case actorID != "":
		return ports.ActorTypeUser
	default:
		return ""
	}
}

// delegationDepth は委譲の深さを返す。トークン交換は delegationDepth、認可判定は
// actorChainDepth という名前で同じ意味の値を持つので、どちらからも同じ軸へ載せる。
func delegationDepth(payload map[string]any) string {
	if depth := payloadNumberString(payload, "delegationDepth"); depth != "" {
		return depth
	}
	return payloadNumberString(payload, "actorChainDepth")
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

// payloadStrings は JSON の文字列配列 (encoding/json が []any へ復号したもの) を読む。
// 空文字と文字列でない要素は落とす。
func payloadStrings(payload map[string]any, key string) []string {
	if payload == nil {
		return nil
	}
	raw, ok := payload[key].([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		if s, ok := item.(string); ok && s != "" {
			out = append(out, s)
		}
	}
	return out
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
