package domain

import (
	"slices"
	"time"

	z "github.com/Oudwins/zog"

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
}

var ssfStreamSchema = z.Struct(z.Shape{
	"ID":       z.String().Min(1).Required(),
	"TenantID": z.String().Min(1).Required(),
	"Direction": z.StringLike[SsfStreamDirection]().TestFunc(
		func(value *SsfStreamDirection, _ z.Ctx) bool { return value.Valid() },
		z.Message("ssf stream direction is not in enum"),
	).Required(),
	"Status": z.StringLike[SsfStreamStatus]().TestFunc(
		func(value *SsfStreamStatus, _ z.Ctx) bool { return value.Valid() },
		z.Message("ssf stream status is not in enum"),
	).Required(),
	"CreatedAt": z.Time().Required(),
})

// Validate は構造的妥当性 (spec/contexts/sharedsignals.yaml SsfStream) を検証する:
// event_types は非空で、各要素が CaepEventType として有効でなければならない。
func (s SsfStream) Validate() error {
	if err := spec.Validate(ssfStreamSchema, &s); err != nil {
		return err
	}
	if len(s.EventTypes) == 0 {
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
