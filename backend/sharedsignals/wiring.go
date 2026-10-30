package sharedsignals

import (
	"context"
	"time"

	agentports "github.com/ambi/idmagic/backend/idmanagement/agent/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/security/tokens_jose"
	"github.com/ambi/idmagic/backend/shared/spec"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	"github.com/ambi/idmagic/backend/sharedsignals/domain"
	"github.com/ambi/idmagic/backend/sharedsignals/internal/db_memory"
	"github.com/ambi/idmagic/backend/sharedsignals/internal/db_postgres"
	"github.com/ambi/idmagic/backend/sharedsignals/internal/handlers_http"
	"github.com/ambi/idmagic/backend/sharedsignals/internal/push_http"
	"github.com/ambi/idmagic/backend/sharedsignals/internal/sign_jose"
	"github.com/ambi/idmagic/backend/sharedsignals/internal/usecases"
	"github.com/ambi/idmagic/backend/sharedsignals/internal/verify_jose"
	"github.com/ambi/idmagic/backend/sharedsignals/ports"
	signingports "github.com/ambi/idmagic/backend/signingkeys/ports"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
	"github.com/jackc/pgx/v5"

	"github.com/labstack/echo/v5"
)

func NewAgentRevocationEpochRepository() ports.AgentRevocationEpochRepository {
	return db_memory.NewAgentRevocationEpochRepository()
}

// NewUserRevocationEpochRepository はメモリの User の失効エポックを返す。
func NewUserRevocationEpochRepository() ports.UserRevocationEpochRepository {
	return db_memory.NewUserRevocationEpochRepository()
}

func NewSsfStreamRepository() ports.SsfStreamRepository { return db_memory.NewSsfStreamRepository() }

func NewSsfTransmitterConfigRepository() ports.SsfTransmitterConfigRepository {
	return db_memory.NewSsfTransmitterConfigRepository()
}

func NewSsfReceiverConfigRepository() ports.SsfReceiverConfigRepository {
	return db_memory.NewSsfReceiverConfigRepository()
}

func NewSecurityEventDeliveryRepository() ports.SecurityEventDeliveryRepository {
	return db_memory.NewSecurityEventDeliveryRepository()
}

func NewReceivedSecurityEventRepository() ports.ReceivedSecurityEventRepository {
	return db_memory.NewReceivedSecurityEventRepository()
}

func NewMemoryModule() Module {
	return Module{
		RevocationEpochRepo: NewAgentRevocationEpochRepository(), UserRevocationEpochRepo: NewUserRevocationEpochRepository(), StreamRepo: NewSsfStreamRepository(),
		TransmitterConfigRepo: NewSsfTransmitterConfigRepository(), ReceiverConfigRepo: NewSsfReceiverConfigRepository(),
		DeliveryRepo: NewSecurityEventDeliveryRepository(), ReceivedEventRepo: NewReceivedSecurityEventRepository(),
	}
}

func NewPostgresModule(pool sharedpg.DB) Module {
	return Module{
		RevocationEpochRepo:     &db_postgres.AgentRevocationEpochRepository{Pool: pool},
		UserRevocationEpochRepo: &db_postgres.UserRevocationEpochRepository{Pool: pool},
		StreamRepo:              &db_postgres.SsfStreamRepository{Pool: pool},
		TransmitterConfigRepo:   &db_postgres.SsfTransmitterConfigRepository{Pool: pool},
		ReceiverConfigRepo:      &db_postgres.SsfReceiverConfigRepository{Pool: pool},
		DeliveryRepo:            &db_postgres.SecurityEventDeliveryRepository{Pool: pool},
		ReceivedEventRepo:       &db_postgres.ReceivedSecurityEventRepository{Pool: pool},
	}
}

func (m Module) AgentRevocationReactor(agents agentports.AgentRepository, keys signingports.KeyStore, issuer string, emit func(spec.DomainEvent)) *usecases.AgentRevocationReactor {
	return usecases.NewAgentRevocationReactor(usecases.RevocationReactorDeps{
		EpochRepo: m.RevocationEpochRepo, UserEpochRepo: m.UserRevocationEpochRepo, AgentRepo: agents,
		Projector: usecases.ProjectorDeps{
			StreamRepo: m.StreamRepo, TransmitterConfigRepo: m.TransmitterConfigRepo,
			DeliveryRepo: m.DeliveryRepo, Signer: &sign_jose.Signer{KeyStore: keys}, Issuer: issuer,
		}, Emit: emit,
	})
}

type RouteDeps struct {
	support.Deps
	*support.Authenticator
	AgentRepo   agentports.AgentRepository
	QuotaRepo   tenantports.QuotaRepository
	JWKResolver *tokens_jose.JWKResolver
	// Users と EndUserSessions は、受理した SET の主体を User として反映する。
	Users           userports.UserRepository
	EndUserSessions func(ctx context.Context, userID string) error
	Emit            func(spec.DomainEvent) error
}

func (m Module) RegisterRoutes(g *echo.Group, d RouteDeps) {
	handlers_http.RegisterRoutes(g, handlers_http.Deps{
		Deps: d.Deps, Authenticator: d.Authenticator,
		StreamRepo: m.StreamRepo, TransmitterConfigRepo: m.TransmitterConfigRepo,
		ReceiverConfigRepo: m.ReceiverConfigRepo, DeliveryRepo: m.DeliveryRepo,
		ReceivedEventRepo: m.ReceivedEventRepo, EpochRepo: m.RevocationEpochRepo, UserEpochRepo: m.UserRevocationEpochRepo,
		Users: d.Users, EndUserSessions: d.EndUserSessions,
		AgentRepo: d.AgentRepo, Verifier: &verify_jose.Verifier{JWKResolver: d.JWKResolver},
		QuotaRepo: d.QuotaRepo, CheckDeliveryEndpoint: m.CheckDeliveryEndpoint, Emit: d.Emit,
	})
}

// WithDeliveryEndpointCheck は登録時の配送先の検査に、接続時と同じ解決先の検査を使う。
func (m Module) WithDeliveryEndpointCheck() Module {
	m.CheckDeliveryEndpoint = push_http.NewHTTPSecurityEventPusher().CheckDestination
	return m
}

// DeliveryProcessor は worker の配送のループの一周を返す。keys と issuer は監査の SET を送信時に署名する。
func (m Module) DeliveryProcessor(keys signingports.KeyStore, issuer string, emit func(spec.DomainEvent) error) func(context.Context, time.Time, int) (int, error) {
	d := m.deliverDeps(push_http.NewHTTPSecurityEventPusher(), keys, issuer, emit)
	return func(ctx context.Context, now time.Time, limit int) (int, error) {
		return usecases.ProcessDueDeliveries(ctx, d, now, limit)
	}
}

func (m Module) deliverDeps(pusher ports.SecurityEventPusher, keys signingports.KeyStore, issuer string, emit func(spec.DomainEvent) error) usecases.DeliverDeps {
	return usecases.DeliverDeps{
		DeliveryRepo: m.DeliveryRepo, TransmitterConfigRepo: m.TransmitterConfigRepo, StreamRepo: m.StreamRepo,
		Pusher: pusher, Signer: &sign_jose.Signer{KeyStore: keys}, Issuer: issuer, Emit: emit,
	}
}

// DeliveryProcessorWithPusher は送信先を差し替えた配送のループの一周を返す。公開アドレスへの
// 接続の制限を受けない受信側で、組み立てを確かめるテストのためのものである。
func (m Module) DeliveryProcessorWithPusher(pusher ports.SecurityEventPusher, keys signingports.KeyStore, issuer string, emit func(spec.DomainEvent) error) func(context.Context, time.Time, int) (int, error) {
	d := m.deliverDeps(pusher, keys, issuer, emit)
	return func(ctx context.Context, now time.Time, limit int) (int, error) {
		return usecases.ProcessDueDeliveries(ctx, d, now, limit)
	}
}

// AuditCapture は監査の記録から配送の内容を作る入力である。
type AuditCapture struct {
	EventID, EventType, TenantID string
	OccurredAt                   time.Time
	Payload                      map[string]any
}

func (c AuditCapture) event() domain.AuditSetEvent {
	return domain.NewAuditSetEvent(c.EventID, c.EventType, c.TenantID, c.OccurredAt, c.Payload)
}

// CaptureAuditEventInTx は監査の記録と同じ Tx で配送を作る（REQ-SHAREDSIGNALS-015）。
func CaptureAuditEventInTx(ctx context.Context, tx pgx.Tx, capture AuditCapture, now time.Time) error {
	return db_postgres.CaptureAuditEventInTx(ctx, tx, capture.event(), now)
}

// CaptureAuditEvent はメモリの保存先の経路で、監査の記録の追記と同時に配送を作る。
func (m Module) CaptureAuditEvent(ctx context.Context, capture AuditCapture, now time.Time) error {
	if !domain.CapturableAuditType(capture.EventType) || capture.TenantID == "" {
		return nil
	}
	return usecases.CaptureAuditEvent(ctx, m.StreamRepo, m.DeliveryRepo, capture.event(), now)
}

// PurgeExpiredDeliveries は保持期間を過ぎた終端の配送を消す。
func (m Module) PurgeExpiredDeliveries(ctx context.Context, now time.Time) (int64, error) {
	return m.DeliveryRepo.PurgeSettled(ctx, now.Add(-domain.DeliveryRetention))
}
