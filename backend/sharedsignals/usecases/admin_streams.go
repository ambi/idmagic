// 管理者向け SsfStream ライフサイクル操作 (ADR-057, wi-58 T005)。SCL SharedSignals
// bounded context が所有する admin インターフェース群: ListSsfStreams / GetSsfStream /
// RegisterSsfTransmitterStream / RegisterSsfReceiverStream / UpdateSsfStream /
// DisableSsfStream / EnableSsfStream / DeleteSsfStream / ListSecurityEventDeliveries。
//
// すべての操作は tenancy.TenantID(ctx) のテナント境界に閉じる。
package usecases

import (
	"context"
	"strings"
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
	ssports "github.com/ambi/idmagic/backend/sharedsignals/ports"
	"github.com/ambi/idmagic/backend/tenancy"
)

// AdminStreamDeps holds what the admin SsfStream usecases need.
type AdminStreamDeps struct {
	StreamRepo            ssports.SsfStreamRepository
	TransmitterConfigRepo ssports.SsfTransmitterConfigRepository
	ReceiverConfigRepo    ssports.SsfReceiverConfigRepository
	DeliveryRepo          ssports.SecurityEventDeliveryRepository
	Emit                  func(spec.DomainEvent) error
}

func ListSsfStreams(ctx context.Context, deps AdminStreamDeps) ([]*ssdomain.SsfStream, error) {
	return deps.StreamRepo.ListByTenant(ctx, tenancy.TenantID(ctx))
}

// GetSsfStream は別テナントの stream を未存在として扱う。
func GetSsfStream(ctx context.Context, deps AdminStreamDeps, id string) (*ssdomain.SsfStream, error) {
	stream, err := deps.StreamRepo.FindByID(ctx, tenancy.TenantID(ctx), id)
	if err != nil {
		return nil, err
	}
	if stream == nil {
		return nil, ErrStreamNotFound
	}
	return stream, nil
}

func validateEventTypes(types []ssdomain.CaepEventType) error {
	if len(types) == 0 {
		return ErrEventTypesEmpty
	}
	for _, t := range types {
		if !t.Valid() {
			return ErrEventTypeInvalid
		}
	}
	return nil
}

// RegisterSsfTransmitterStreamInput mirrors SsfTransmitterStreamRegisterRequest.
type RegisterSsfTransmitterStreamInput struct {
	EventTypes            []ssdomain.CaepEventType
	DeliveryEndpoint      string
	Audience              string
	DeliveryAuthorization *string
	MaxDeliveryAttempts   *int
}

func RegisterSsfTransmitterStream(ctx context.Context, deps AdminStreamDeps, in RegisterSsfTransmitterStreamInput, now time.Time) (*ssdomain.SsfStream, error) {
	tenantID := tenancy.TenantID(ctx)
	if err := validateEventTypes(in.EventTypes); err != nil {
		return nil, err
	}
	if in.DeliveryEndpoint == "" || !strings.HasPrefix(in.DeliveryEndpoint, "https://") {
		return nil, ErrDeliveryEndpointInvalid
	}
	if in.Audience == "" {
		return nil, ErrAudienceRequired
	}
	maxAttempts := ssdomain.DefaultMaxDeliveryAttempts
	if in.MaxDeliveryAttempts != nil && *in.MaxDeliveryAttempts > 0 {
		maxAttempts = *in.MaxDeliveryAttempts
	}
	id, err := ssdomain.NewSsfStreamID()
	if err != nil {
		return nil, err
	}
	stream := &ssdomain.SsfStream{
		ID: id, TenantID: tenantID, Direction: ssdomain.SsfStreamDirectionTransmit,
		EventTypes: in.EventTypes, Status: ssdomain.SsfStreamStatusEnabled, CreatedAt: now,
	}
	if err := stream.Validate(); err != nil {
		return nil, err
	}
	config := &ssdomain.SsfTransmitterConfig{
		StreamID: id, DeliveryEndpoint: in.DeliveryEndpoint, Audience: in.Audience,
		DeliveryAuthorization: in.DeliveryAuthorization, MaxDeliveryAttempts: maxAttempts,
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}
	if err := deps.StreamRepo.Save(ctx, stream); err != nil {
		return nil, err
	}
	if err := deps.TransmitterConfigRepo.Save(ctx, tenantID, config); err != nil {
		return nil, err
	}
	if err := emit(deps.Emit, &ssdomain.SsfStreamRegistered{At: now, TenantID: tenantID, StreamID: id, Direction: ssdomain.SsfStreamDirectionTransmit}); err != nil {
		return nil, err
	}
	return stream, nil
}

// RegisterSsfReceiverStreamInput mirrors SsfReceiverStreamRegisterRequest.
type RegisterSsfReceiverStreamInput struct {
	EventTypes        []ssdomain.CaepEventType
	TrustedIssuer     string
	JWKSURI           *string
	JWKS              map[string]any
	AcceptedAudiences []string
}

func RegisterSsfReceiverStream(ctx context.Context, deps AdminStreamDeps, in RegisterSsfReceiverStreamInput, now time.Time) (*ssdomain.SsfStream, error) {
	tenantID := tenancy.TenantID(ctx)
	if err := validateEventTypes(in.EventTypes); err != nil {
		return nil, err
	}
	if in.TrustedIssuer == "" || !strings.HasPrefix(in.TrustedIssuer, "https://") {
		return nil, ErrTrustedIssuerInvalid
	}
	if in.JWKSURI == nil && in.JWKS == nil {
		return nil, ErrJWKSSourceRequired
	}
	if len(in.AcceptedAudiences) == 0 {
		return nil, ErrAcceptedAudiencesEmpty
	}
	id, err := ssdomain.NewSsfStreamID()
	if err != nil {
		return nil, err
	}
	stream := &ssdomain.SsfStream{
		ID: id, TenantID: tenantID, Direction: ssdomain.SsfStreamDirectionReceive,
		EventTypes: in.EventTypes, Status: ssdomain.SsfStreamStatusEnabled, CreatedAt: now,
	}
	if err := stream.Validate(); err != nil {
		return nil, err
	}
	config := &ssdomain.SsfReceiverConfig{
		StreamID: id, TrustedIssuer: in.TrustedIssuer, JWKSURI: in.JWKSURI, JWKS: in.JWKS,
		AcceptedAudiences: in.AcceptedAudiences,
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}
	if err := deps.StreamRepo.Save(ctx, stream); err != nil {
		return nil, err
	}
	if err := deps.ReceiverConfigRepo.Save(ctx, tenantID, config); err != nil {
		return nil, err
	}
	if err := emit(deps.Emit, &ssdomain.SsfStreamRegistered{At: now, TenantID: tenantID, StreamID: id, Direction: ssdomain.SsfStreamDirectionReceive}); err != nil {
		return nil, err
	}
	return stream, nil
}

// UpdateSsfStreamInput mirrors SsfStreamUpdateRequest: event_types のみ更新可能、
// 省略 (nil) は現状維持。
type UpdateSsfStreamInput struct {
	EventTypes *[]ssdomain.CaepEventType
}

func UpdateSsfStream(ctx context.Context, deps AdminStreamDeps, id string, in UpdateSsfStreamInput, now time.Time) (*ssdomain.SsfStream, error) {
	tenantID := tenancy.TenantID(ctx)
	stream, err := deps.StreamRepo.FindByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if stream == nil {
		return nil, ErrStreamNotFound
	}
	if in.EventTypes == nil {
		return stream, nil
	}
	if err := validateEventTypes(*in.EventTypes); err != nil {
		return nil, err
	}
	updated := *stream
	updated.EventTypes = *in.EventTypes
	updated.UpdatedAt = &now
	if err := updated.Validate(); err != nil {
		return nil, err
	}
	if err := deps.StreamRepo.Save(ctx, &updated); err != nil {
		return nil, err
	}
	if err := emit(deps.Emit, &ssdomain.SsfStreamUpdated{At: now, TenantID: tenantID, StreamID: id}); err != nil {
		return nil, err
	}
	return &updated, nil
}

func DisableSsfStream(ctx context.Context, deps AdminStreamDeps, id string, now time.Time) (*ssdomain.SsfStream, error) {
	return setStreamStatus(ctx, deps, id, ssdomain.SsfStreamStatusDisabled, now)
}

func EnableSsfStream(ctx context.Context, deps AdminStreamDeps, id string, now time.Time) (*ssdomain.SsfStream, error) {
	return setStreamStatus(ctx, deps, id, ssdomain.SsfStreamStatusEnabled, now)
}

func setStreamStatus(ctx context.Context, deps AdminStreamDeps, id string, status ssdomain.SsfStreamStatus, now time.Time) (*ssdomain.SsfStream, error) {
	tenantID := tenancy.TenantID(ctx)
	stream, err := deps.StreamRepo.FindByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if stream == nil {
		return nil, ErrStreamNotFound
	}
	if stream.Status == status {
		return stream, nil
	}
	updated := *stream
	updated.Status = status
	updated.UpdatedAt = &now
	if err := updated.Validate(); err != nil {
		return nil, err
	}
	if err := deps.StreamRepo.Save(ctx, &updated); err != nil {
		return nil, err
	}
	var event spec.DomainEvent
	if status == ssdomain.SsfStreamStatusDisabled {
		event = &ssdomain.SsfStreamDisabled{At: now, TenantID: tenantID, StreamID: id}
	} else {
		event = &ssdomain.SsfStreamEnabled{At: now, TenantID: tenantID, StreamID: id}
	}
	if err := emit(deps.Emit, event); err != nil {
		return nil, err
	}
	return &updated, nil
}

// DeleteSsfStream は SsfStream を削除し、付随する Transmitter/ReceiverConfig を
// 先に cascade 削除する (DB の ON DELETE CASCADE と併せた二重の保証)。
func DeleteSsfStream(ctx context.Context, deps AdminStreamDeps, id string, now time.Time) error {
	tenantID := tenancy.TenantID(ctx)
	stream, err := deps.StreamRepo.FindByID(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if stream == nil {
		return ErrStreamNotFound
	}
	if stream.Direction == ssdomain.SsfStreamDirectionTransmit {
		if err := deps.TransmitterConfigRepo.Delete(ctx, tenantID, id); err != nil {
			return err
		}
	} else {
		if err := deps.ReceiverConfigRepo.Delete(ctx, tenantID, id); err != nil {
			return err
		}
	}
	if err := deps.StreamRepo.Delete(ctx, tenantID, id); err != nil {
		return err
	}
	return emit(deps.Emit, &ssdomain.SsfStreamDeleted{At: now, TenantID: tenantID, StreamID: id})
}

// ListSecurityEventDeliveries は direction=Transmit の stream の配送状況
// (pending/delivered/failed/dead_letter) を返す。delivery health 確認用 (T004 の
// projector/delivery worker が生成・更新する)。
func ListSecurityEventDeliveries(ctx context.Context, deps AdminStreamDeps, streamID string) ([]*ssdomain.SecurityEventDelivery, error) {
	tenantID := tenancy.TenantID(ctx)
	stream, err := deps.StreamRepo.FindByID(ctx, tenantID, streamID)
	if err != nil {
		return nil, err
	}
	if stream == nil {
		return nil, ErrStreamNotFound
	}
	return deps.DeliveryRepo.ListByStream(ctx, tenantID, streamID)
}
