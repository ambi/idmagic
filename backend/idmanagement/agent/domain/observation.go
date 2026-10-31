package domain

import (
	"cmp"
	"errors"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ambi/idmagic/backend/shared/spec"
)

const (
	// MaxObservationBatch は一度に取り込める観測の件数である。
	MaxObservationBatch = 100
	// MaxTenantObservations はテナントが持てる観測の件数である。
	MaxTenantObservations = 10_000
	// ObservationStaleAfter を過ぎて報告がない観測を古いものとして示す。
	ObservationStaleAfter = 30 * 24 * time.Hour
	// ObservationRetention を過ぎて報告がない未確認の観測を消す。
	ObservationRetention = 180 * 24 * time.Hour
	// ObservationFutureSkew は取り込み元の時計のずれとして許す未来の幅である。
	ObservationFutureSkew = 5 * time.Minute

	maxObservationKeyLength       = 512
	maxObservationAttributeLength = 256
	maxObservationResources       = 20
)

// ErrInvalidObservation は観測の項目が必須の値を欠くか、上限の外にある。
var ErrInvalidObservation = errors.New("agent observation is invalid")

// ObservationDecisionKind は管理者による観測の決定である (REQ-IDMANAGEMENT-098)。
type ObservationDecisionKind string

const (
	ObservationPending   ObservationDecisionKind = "pending"
	ObservationConfirmed ObservationDecisionKind = "confirmed"
	ObservationExcluded  ObservationDecisionKind = "excluded"
)

func (k ObservationDecisionKind) Valid() bool {
	switch k {
	case ObservationPending, ObservationConfirmed, ObservationExcluded:
		return true
	}
	return false
}

// ObservationState は決定と候補から導いた観測の状態である。
type ObservationState string

const (
	ObservationStateUnregistered ObservationState = "unregistered"
	ObservationStateCandidate    ObservationState = "candidate"
	ObservationStateConfirmed    ObservationState = "confirmed"
	ObservationStateExcluded     ObservationState = "excluded"
)

// MatchReason は候補の根拠である。名前と表示名は根拠にしない。
type MatchReason string

const (
	MatchReasonWorkloadBinding MatchReason = "workload_binding"
	MatchReasonClientBinding   MatchReason = "client_binding"
)

// AgentObservation は外部の主体を見たという未確認の報告であり、Agent の第二の正本ではない。
// 確認した状態は ConfirmedAgentID の有無で表す。
type AgentObservation struct {
	ID                     string
	TenantID               string
	Issuer                 string
	Subject                string
	ExternalRegistrationID *string
	DisplayName            *string
	Resources              []string
	ResponsibleCandidate   *string
	ObservedAt             time.Time
	SourceTokenID          string
	Excluded               bool
	ConfirmedAgentID       *string
	DecidedAt              *time.Time
	DecidedByUserID        *string
	CreatedAt              time.Time
}

// Normalize は前後の空白を除き、空の任意項目を欠落にそろえる。
func (o *AgentObservation) Normalize() {
	o.Issuer = strings.TrimSpace(o.Issuer)
	o.Subject = strings.TrimSpace(o.Subject)
	for _, field := range []**string{&o.ExternalRegistrationID, &o.DisplayName, &o.ResponsibleCandidate} {
		if *field == nil {
			continue
		}
		trimmed := strings.TrimSpace(**field)
		if trimmed == "" {
			*field = nil
			continue
		}
		*field = &trimmed
	}
	if o.Resources == nil {
		o.Resources = []string{}
	}
}

// Validate は取り込む項目の上限を確かめる (REQ-IDMANAGEMENT-096)。now は受信の時刻である。
func (o AgentObservation) Validate(now time.Time) error {
	if !withinLength(o.Issuer, maxObservationKeyLength) || !withinLength(o.Subject, maxObservationKeyLength) {
		return ErrInvalidObservation
	}
	for _, field := range []*string{o.ExternalRegistrationID, o.DisplayName, o.ResponsibleCandidate} {
		if field != nil && !withinLength(*field, maxObservationAttributeLength) {
			return ErrInvalidObservation
		}
	}
	if len(o.Resources) > maxObservationResources {
		return ErrInvalidObservation
	}
	for _, resource := range o.Resources {
		if !withinLength(resource, maxObservationAttributeLength) {
			return ErrInvalidObservation
		}
	}
	if o.ObservedAt.IsZero() || o.ObservedAt.After(now.Add(ObservationFutureSkew)) {
		return ErrInvalidObservation
	}
	return nil
}

func withinLength(value string, maximum int) bool {
	length := utf8.RuneCountInString(value)
	return length >= 1 && length <= maximum
}

// Decision は保存した決定を返す。
func (o AgentObservation) Decision() ObservationDecisionKind {
	switch {
	case o.ConfirmedAgentID != nil:
		return ObservationConfirmed
	case o.Excluded:
		return ObservationExcluded
	}
	return ObservationPending
}

// Stale は最終の観測時刻から ObservationStaleAfter を過ぎているかを返す。
func (o AgentObservation) Stale(now time.Time) bool {
	return now.Sub(o.ObservedAt) > ObservationStaleAfter
}

// MatchEvidence は一つの Agent を候補にする登録済みの対応である。
type MatchEvidence struct {
	AgentID string
	Reason  MatchReason
}

// ObservationCandidate は Agent ごとにまとめた候補である。
type ObservationCandidate struct {
	AgentID string
	Reasons []MatchReason
}

// ObservationMatch は観測の状態と候補である。
type ObservationMatch struct {
	State      ObservationState
	Candidates []ObservationCandidate
}

// MatchObservation は候補と状態を計算する (REQ-IDMANAGEMENT-098)。候補の数によらず、未決定の
// 観測を確定しない。候補は Agent の id の順、根拠は Workload binding、束縛の順に返す。
func MatchObservation(observation AgentObservation, evidence []MatchEvidence) ObservationMatch {
	byAgent := map[string][]MatchReason{}
	for _, item := range evidence {
		if !slices.Contains(byAgent[item.AgentID], item.Reason) {
			byAgent[item.AgentID] = append(byAgent[item.AgentID], item.Reason)
		}
	}
	candidates := make([]ObservationCandidate, 0, len(byAgent))
	for agentID, reasons := range byAgent {
		slices.SortFunc(reasons, func(a, b MatchReason) int { return cmp.Compare(matchReasonOrder(a), matchReasonOrder(b)) })
		candidates = append(candidates, ObservationCandidate{AgentID: agentID, Reasons: reasons})
	}
	slices.SortFunc(candidates, func(a, b ObservationCandidate) int { return cmp.Compare(a.AgentID, b.AgentID) })
	match := ObservationMatch{Candidates: candidates}
	switch {
	case observation.ConfirmedAgentID != nil:
		match.State = ObservationStateConfirmed
	case observation.Excluded:
		match.State = ObservationStateExcluded
	case len(candidates) > 0:
		match.State = ObservationStateCandidate
	default:
		match.State = ObservationStateUnregistered
	}
	return match
}

func matchReasonOrder(reason MatchReason) int {
	if reason == MatchReasonWorkloadBinding {
		return 0
	}
	return 1
}

// NewAgentObservationID は観測の識別子を生成する。
func NewAgentObservationID() (string, error) {
	return spec.NewUUIDv4()
}
