package domain

import (
	"slices"
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
)

// SsfStream は tenant が登録する SSF stream の共通メタデータ。direction ごとに
// SsfTransmitterConfig または SsfReceiverConfig が 1 対 1 で付随する。
type SsfStream struct {
	ID         string
	TenantID   string
	Direction  SsfStreamDirection
	EventTypes []CaepEventType
	Status     SsfStreamStatus
	CreatedAt  time.Time
	UpdatedAt  *time.Time
	// AuditEventTypes は送信側のストリームが配送する監査イベントの種類である。空は購読しない。
	AuditEventTypes []string
	// ConsecutiveFailures と FirstFailureAt は自動無効化の判定に使う。配送の結果だけが更新する。
	ConsecutiveFailures int
	FirstFailureAt      *time.Time
}

// Validate は構造的妥当性 (docs/modules/sharedsignals/ SsfStream) を検証する:
// event_types と audit_event_types の少なくとも一方は非空で、event_types の各要素は
// CaepEventType として有効でなければならない。
func (s SsfStream) Validate() error {
	if err := spec.All(
		spec.Required("ID", s.ID),
		spec.Required("TenantID", s.TenantID),
		spec.Required("Direction", s.Direction, spec.Must(SsfStreamDirection.Valid, "ssf stream direction is not in enum")),
		spec.Required("Status", s.Status, spec.Must(SsfStreamStatus.Valid, "ssf stream status is not in enum")),
		spec.Required("CreatedAt", s.CreatedAt),
	); err != nil {
		return err
	}
	if len(s.EventTypes) == 0 && len(s.AuditEventTypes) == 0 {
		return errEmptyEventTypes
	}
	for _, t := range s.EventTypes {
		if !t.Valid() {
			return errInvalidEventType
		}
	}
	return nil
}

// IsEnabled は SsfStream が配送/受理に使える状態かを返す (SsfStreamLifecycle)。
func (s SsfStream) IsEnabled() bool {
	return s.Status == SsfStreamStatusEnabled
}

// Subscribes は指定した CAEP イベント種別を本 stream が対象としているかを返す。
func (s SsfStream) Subscribes(t CaepEventType) bool {
	return slices.Contains(s.EventTypes, t)
}

// NewSsfStreamID は不変の SsfStream 識別子 (UUID v4) を生成する。
func NewSsfStreamID() (string, error) {
	return spec.NewUUIDv4()
}
