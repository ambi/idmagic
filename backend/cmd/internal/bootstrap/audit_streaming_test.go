package bootstrap

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/audit"
	auditports "github.com/ambi/idmagic/backend/audit/ports"
	"github.com/ambi/idmagic/backend/shared/logging"
	"github.com/ambi/idmagic/backend/shared/spec"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	pgfixtures "github.com/ambi/idmagic/backend/shared/testing/fixtures_postgres"
	"github.com/ambi/idmagic/backend/sharedsignals"
	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
	"github.com/ambi/idmagic/backend/signingkeys"
)

type capturingPusher struct{ sets []string }

func (p *capturingPusher) Push(_ context.Context, _, _, compactSET string) error {
	p.sets = append(p.sets, compactSET)
	return nil
}

type streamedEvent struct {
	At       time.Time `json:"-"`
	TenantID string    `json:"tenantId"`
	UserID   string    `json:"userId"`
	Email    string    `json:"email"`
}

func (*streamedEvent) EventType() string       { return "UserCreated" }
func (e *streamedEvent) OccurredAt() time.Time { return e.At }

//spec:covers REQ-SHAREDSIGNALS-015: 正式な組み立てと配信点を通したイベントが配送になり、ループが署名付きの SET で送る。
func TestEmittedEventReachesSsfStream(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	const tenantID = "11111111-1111-4111-8111-111111111111"
	deps, err := assembleMemory(SharedConfig{})
	if err != nil {
		t.Fatal(err)
	}
	module := deps.SharedSignals
	if err := module.StreamRepo.Save(ctx, &ssdomain.SsfStream{ID: "s1", TenantID: tenantID, Direction: ssdomain.SsfStreamDirectionTransmit, AuditEventTypes: []string{"*"}, Status: ssdomain.SsfStreamStatusEnabled, CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := module.TransmitterConfigRepo.Save(ctx, tenantID, &ssdomain.SsfTransmitterConfig{StreamID: "s1", DeliveryEndpoint: "https://receiver.example", Audience: "aud", MaxDeliveryAttempts: 8}); err != nil {
		t.Fatal(err)
	}
	keys, err := signingkeys.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	deps.NewEmitFunc(logging.Default())(&streamedEvent{At: now, TenantID: tenantID, UserID: "u1", Email: "a@example.com"})

	pusher := &capturingPusher{}
	process := module.DeliveryProcessorWithPusher(pusher, keys, "https://idp.example", func(spec.DomainEvent) error { return nil })
	if _, err := process(ctx, now.Add(time.Second), 100); err != nil {
		t.Fatal(err)
	}
	if len(pusher.sets) != 1 {
		t.Fatalf("sets = %d", len(pusher.sets))
	}
	parts := strings.Split(pusher.sets[0], ".")
	if len(parts) != 3 {
		t.Fatalf("set = %q", pusher.sets[0])
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatal(err)
	}
	var claims struct {
		Events map[string]map[string]any `json:"events"`
		SubID  map[string]any            `json:"sub_id"`
	}
	if err := json.Unmarshal(raw, &claims); err != nil {
		t.Fatal(err)
	}
	if _, ok := claims.Events["urn:idmagic:secevent:audit:UserCreated"]; !ok || claims.SubID["id"] != "u1" {
		t.Fatalf("claims = %s", raw)
	}
	if strings.Contains(string(raw), "a@example.com") {
		t.Fatalf("claims contain personal data: %s", raw)
	}
}

//spec:covers REQ-SHAREDSIGNALS-015: PostgreSQL の監査の追記から実際の捕捉までがつながり、購読するストリームの配送が残る。
func TestPostgresAppendCapturesIntoStream(t *testing.T) {
	db := pgtest.Require(t)
	ctx := context.Background()
	now := pgtest.Now()
	tenant := pgfixtures.SeedTenant(t, db)
	module := sharedsignals.NewPostgresModule(db)
	streamID, err := ssdomain.NewSsfStreamID()
	if err != nil {
		t.Fatal(err)
	}
	if err := module.StreamRepo.Save(ctx, &ssdomain.SsfStream{ID: streamID, TenantID: tenant.ID, Direction: ssdomain.SsfStreamDirectionTransmit, AuditEventTypes: []string{"UserCreated"}, Status: ssdomain.SsfStreamStatusEnabled, CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	repo := audit.NewPostgresModule(db).WithCapture(captureAuditRecordInTx, nil).AuditEventRepo
	id, err := spec.NewUUIDv4()
	if err != nil {
		t.Fatal(err)
	}
	const userID = "22222222-2222-4222-8222-222222222222"
	if err := repo.Append(ctx, &auditports.AuditEventRecord{ID: id, TenantID: tenant.ID, Type: "UserCreated", OccurredAt: now, Payload: map[string]any{"userId": userID, "email": "a@example.com"}}); err != nil {
		t.Fatal(err)
	}
	got, err := module.DeliveryRepo.ListByStream(ctx, tenant.ID, streamID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].AuditEvent == nil || got[0].AuditEvent.EventID != id || got[0].AuditEvent.Actor == nil || got[0].AuditEvent.Actor.ID != userID {
		t.Fatalf("deliveries = %+v", got)
	}
}
