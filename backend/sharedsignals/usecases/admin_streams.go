// 管理者向け SsfStream ライフサイクル操作 (wi-58 T005)。SCL SharedSignals
// bounded context が所有する admin インターフェース群: ListSsfStreams / GetSsfStream /
// RegisterSsfTransmitterStream / RegisterSsfReceiverStream / UpdateSsfStream /
// DisableSsfStream / EnableSsfStream / DeleteSsfStream / ListSecurityEventDeliveries。
//
// すべての操作は tenancy.TenantID(ctx) のテナント境界に閉じる。
package usecases

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/ambi/idmagic/backend/shared/logging"
	"github.com/ambi/idmagic/backend/shared/spec"
	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
	ssports "github.com/ambi/idmagic/backend/sharedsignals/ports"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
	tenancyusecases "github.com/ambi/idmagic/backend/tenancy/usecases"
)

// AdminStreamDeps holds what the admin SsfStream usecases need.
type AdminStreamDeps struct {
	StreamRepo            ssports.SsfStreamRepository
	TransmitterConfigRepo ssports.SsfTransmitterConfigRepository
	ReceiverConfigRepo    ssports.SsfReceiverConfigRepository
	DeliveryRepo          ssports.SecurityEventDeliveryRepository
	// QuotaRepo enforces the tenant's `ssf_streams` Hard Quota
	// (REQ-SHAREDSIGNALS-009). nil skips enforcement, matching this codebase's
	// other optional-by-construction quota ports.
	QuotaRepo tenantports.QuotaRepository
	Emit      func(spec.DomainEvent) error
}

// checkStreamQuota reserves one `ssf_streams` slot before a stream row exists,
// and records the rejection as a QuotaExceeded audit event. Transmit and
// Receive share the one resource: direction is an attribute of the same set,
// not a separate budget.
func checkStreamQuota(ctx context.Context, deps AdminStreamDeps, tenantID string, now time.Time) error {
	if deps.QuotaRepo == nil {
		return nil
	}
	err := tenancyusecases.CheckQuotaAndIncrement(ctx, deps.QuotaRepo, tenantID, tenancydomain.ResourceSsfStreams, 1)
	var quotaErr *tenancydomain.QuotaExceededError
	if errors.As(err, &quotaErr) {
		// The audit trail must not mask the rejection it records.
		_ = emit(deps.Emit, &tenancydomain.QuotaExceeded{
			At: now, TenantID: tenantID, Resource: quotaErr.Resource, HardLimit: true,
		})
	}
	return err
}

// releaseStreamQuota gives the reserved slot back when the stream never
// materializes (a later validation or persistence failure) or is deleted.
func releaseStreamQuota(ctx context.Context, deps AdminStreamDeps, tenantID string) {
	if deps.QuotaRepo == nil {
		return
	}
	if err := tenancyusecases.DecrementQuota(ctx, deps.QuotaRepo, tenantID, tenancydomain.ResourceSsfStreams, 1); err != nil {
		logging.Error(ctx, "quota: failed to release the ssf_streams reservation", "error", err, "tenant_id", tenantID)
	}
}

func ListSsfStreams(ctx context.Context, deps AdminStreamDeps) ([]*ssdomain.SsfStream, error) {
	return deps.StreamRepo.ListAll(ctx, tenancy.TenantID(ctx))
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

// saveTransmitterStream は quota 予約後の永続化をまとめる。呼び出し側が失敗時に
// 予約を返す 1 箇所だけを持てるように切り出してある。
func saveTransmitterStream(
	ctx context.Context, deps AdminStreamDeps, tenantID string,
	in RegisterSsfTransmitterStreamInput, maxAttempts int, now time.Time,
) (*ssdomain.SsfStream, error) {
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
	return stream, nil
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
	if err := checkStreamQuota(ctx, deps, tenantID, now); err != nil {
		return nil, err
	}
	stream, err := saveTransmitterStream(ctx, deps, tenantID, in, maxAttempts, now)
	if err != nil {
		// 予約した枠は、stream が生まれなかった時点で必ず返す。
		releaseStreamQuota(ctx, deps, tenantID)
		return nil, err
	}
	id := stream.ID
	if err := emit(deps.Emit, &ssdomain.SsfStreamRegistered{At: now, TenantID: tenantID, StreamID: id, Direction: ssdomain.SsfStreamDirectionTransmit}); err != nil {
		return nil, err
	}
	return stream, nil
}

// saveReceiverStream は saveTransmitterStream の受信側の対。
func saveReceiverStream(
	ctx context.Context, deps AdminStreamDeps, tenantID string,
	in RegisterSsfReceiverStreamInput, now time.Time,
) (*ssdomain.SsfStream, error) {
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
	if err := checkStreamQuota(ctx, deps, tenantID, now); err != nil {
		return nil, err
	}
	stream, err := saveReceiverStream(ctx, deps, tenantID, in, now)
	if err != nil {
		releaseStreamQuota(ctx, deps, tenantID)
		return nil, err
	}
	id := stream.ID
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
	releaseStreamQuota(ctx, deps, tenantID)
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
