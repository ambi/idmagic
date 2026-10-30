package domain

import (
	"errors"
	"net/netip"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// 監査イベントの配送の値。意味は docs/modules/sharedsignals/features/transmitter.md が定める。
const (
	AuditEventWildcard         = "*"
	AuditSetEventURIPrefix     = "urn:idmagic:secevent:audit:"
	MaxAuditEventTypes         = 100
	AutoDisableFailures        = 20
	AutoDisableElapsed         = time.Hour
	DeliveryRetention          = 30 * 24 * time.Hour
	maxDeliveryEndpointLength  = 2048
	subIDFormatOpaque          = "opaque"
	auditSetActorTypeUser      = "user"
	auditSetActorTypeAgent     = "agent"
	securityEventOutcomePrefix = "SecurityEvent"
)

// AuditSetActor は監査の SET の行為者である。
type AuditSetActor struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

// AuditSetEvent は監査の記録から取り出した配送の内容である。許可リストであり、項目の追加には
// docs/modules/sharedsignals/decisions.md の判断の更新を要する。
type AuditSetEvent struct {
	EventID    string         `json:"event_id"`
	EventType  string         `json:"event_type"`
	TenantID   string         `json:"tenant_id"`
	OccurredAt time.Time      `json:"occurred_at"`
	Actor      *AuditSetActor `json:"actor,omitempty"`
	TargetID   string         `json:"target_id,omitempty"`
}

// NewAuditSetEvent は監査の本文から許可した識別子だけを取り出す。
func NewAuditSetEvent(eventID, eventType, tenantID string, occurredAt time.Time, payload map[string]any) AuditSetEvent {
	event := AuditSetEvent{EventID: eventID, EventType: eventType, TenantID: tenantID, OccurredAt: occurredAt.UTC()}
	if id, ok := payload["agentId"].(string); ok && id != "" {
		event.Actor = &AuditSetActor{Type: auditSetActorTypeAgent, ID: id}
	}
	if id, ok := payload["userId"].(string); ok && id != "" {
		event.Actor = &AuditSetActor{Type: auditSetActorTypeUser, ID: id}
	}
	if id, ok := payload["targetUserId"].(string); ok && id != "" {
		event.TargetID = id
	}
	return event
}

// SubjectID は SET の sub_id の値を返す。対象、行為者の順に選び、主体のないイベントはテナントを示す。
func (e AuditSetEvent) SubjectID() map[string]any {
	switch {
	case e.TargetID != "":
		return OpaqueSubjectID(e.TargetID)
	case e.Actor != nil:
		return OpaqueSubjectID(e.Actor.ID)
	default:
		return OpaqueSubjectID(e.TenantID)
	}
}

// OpaqueSubjectID は RFC 9493 の opaque の Subject Identifier を返す。
func OpaqueSubjectID(id string) map[string]any {
	return map[string]any{"format": subIDFormatOpaque, "id": id}
}

// CapturableAuditType は種類が捕捉の対象かを返す。配送の結果の記録が配送を生む増幅を防ぐ。
func CapturableAuditType(eventType string) bool {
	return !strings.HasPrefix(eventType, securityEventOutcomePrefix)
}

var auditEventTypePattern = regexp.MustCompile(`^[A-Z][A-Za-z0-9]{0,127}$`)

// ErrAuditEventTypesInvalid は audit_event_types の指定が不正である。
var ErrAuditEventTypesInvalid = errors.New("sharedsignals: audit event types are invalid")

// NormalizeAuditEventTypes は指定を検査し、重複を除く。空は購読しないことを表す。
func NormalizeAuditEventTypes(types []string) ([]string, error) {
	if len(types) > MaxAuditEventTypes {
		return nil, ErrAuditEventTypesInvalid
	}
	if len(types) == 1 && types[0] == AuditEventWildcard {
		return []string{AuditEventWildcard}, nil
	}
	seen := make(map[string]struct{}, len(types))
	out := make([]string, 0, len(types))
	for _, eventType := range types {
		if !auditEventTypePattern.MatchString(eventType) || !CapturableAuditType(eventType) {
			return nil, ErrAuditEventTypesInvalid
		}
		if _, dup := seen[eventType]; !dup {
			seen[eventType] = struct{}{}
			out = append(out, eventType)
		}
	}
	return out, nil
}

// SubscribesAudit はストリームがその監査イベントを受け取るかを返す。
func (s SsfStream) SubscribesAudit(eventType string) bool {
	if !CapturableAuditType(eventType) {
		return false
	}
	for _, candidate := range s.AuditEventTypes {
		if candidate == AuditEventWildcard || candidate == eventType {
			return true
		}
	}
	return false
}

// ErrDeliveryEndpointNotAllowed は配送先が要件を満たさない。
var ErrDeliveryEndpointNotAllowed = errors.New("sharedsignals: delivery endpoint is not allowed")

// ValidateDeliveryEndpoint は解決を伴わない配送先の検査である。解決先は pusher が検査する。
func ValidateDeliveryEndpoint(raw string) error {
	if raw == "" || len(raw) > maxDeliveryEndpointLength {
		return ErrDeliveryEndpointNotAllowed
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" || parsed.User != nil || parsed.Hostname() == "" {
		return ErrDeliveryEndpointNotAllowed
	}
	host := strings.ToLower(strings.TrimSuffix(parsed.Hostname(), "."))
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return ErrDeliveryEndpointNotAllowed
	}
	if addr, err := netip.ParseAddr(host); err == nil && !IsPublicAddress(addr) {
		return ErrDeliveryEndpointNotAllowed
	}
	return nil
}

var nonPublicPrefixes = func() []netip.Prefix {
	values := []string{
		"0.0.0.0/8", "10.0.0.0/8", "100.64.0.0/10", "127.0.0.0/8", "169.254.0.0/16", "172.16.0.0/12",
		"192.0.0.0/24", "192.0.2.0/24", "192.88.99.0/24", "192.168.0.0/16", "198.18.0.0/15",
		"198.51.100.0/24", "203.0.113.0/24", "224.0.0.0/4", "240.0.0.0/4",
		"::/128", "::1/128", "64:ff9b::/96", "64:ff9b:1::/48", "100::/64", "2001::/23", "2001:db8::/32",
		"2002::/16", "fc00::/7", "fe80::/10", "fec0::/10", "ff00::/8",
	}
	prefixes := make([]netip.Prefix, len(values))
	for i, value := range values {
		prefixes[i] = netip.MustParsePrefix(value)
	}
	return prefixes
}()

// IsPublicAddress はアドレスが公開の到達先かを返す。IPv4 射影は IPv4 として判定する。
func IsPublicAddress(addr netip.Addr) bool {
	addr = addr.Unmap().WithZone("")
	if !addr.IsValid() {
		return false
	}
	for _, prefix := range nonPublicPrefixes {
		if prefix.Contains(addr) {
			return false
		}
	}
	return true
}

// SsfStreamAutoDisabled は連続の失敗でストリームを自動的に無効化した事実である。
type SsfStreamAutoDisabled struct {
	At       time.Time `json:"-"`
	TenantID string    `json:"tenantId"`
	StreamID string    `json:"streamId"`
}

func (e *SsfStreamAutoDisabled) EventType() string     { return "SsfStreamAutoDisabled" }
func (e *SsfStreamAutoDisabled) OccurredAt() time.Time { return e.At }
