package handlers_http_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication"

	"github.com/ambi/idmagic/backend/idmanagement"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	"github.com/ambi/idmagic/backend/sharedsignals"
	sharedsignalsdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
	sharedsignalsmemory "github.com/ambi/idmagic/backend/sharedsignals/internal/db_memory"
	sharedsignalspush "github.com/ambi/idmagic/backend/sharedsignals/internal/push_http"
	sharedsignalssign "github.com/ambi/idmagic/backend/sharedsignals/internal/sign_jose"
	sharedsignalsusecases "github.com/ambi/idmagic/backend/sharedsignals/internal/usecases"
	"github.com/ambi/idmagic/backend/signingkeys"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"

	"github.com/labstack/echo/v5"
)

//spec:covers REQ-SHAREDSIGNALS-006: 管理 API で登録した送信 stream に失効 event を射影し、worker が実 HTTP receiver へ SET を配送して delivered にする。
//verify:operation ListSecurityEventDeliveries: 実 HTTP 配送後の管理 API は 200 JSON で同じ配送 ID、stream_id、SET jti、delivered、試行回数、時刻を返す。
func TestTransmitterStreamLifecycleAndDelivery(t *testing.T) {
	var received atomic.Int32
	var receivedBody string
	receiver := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Errorf("read receiver body: %v", err)
		}
		receivedBody = string(body)
		received.Add(1)
		response.WriteHeader(http.StatusAccepted)
	}))
	defer receiver.Close()

	now := time.Now().UTC()
	users := idmanagement.NewMemoryModule().UserRepo
	_ = users.Save(context.Background(), &userdomain.User{
		ID: "admin", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "admin", PasswordHash: "unused",
		Roles: []string{"admin"}, CreatedAt: now, UpdatedAt: now,
	})
	streams := sharedsignalsmemory.NewSsfStreamRepository()
	configs := sharedsignalsmemory.NewSsfTransmitterConfigRepository()
	deliveries := sharedsignalsmemory.NewSecurityEventDeliveryRepository()
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer:        "http://idp.test",
		AuthnResolver: authentication.DemoHeaderResolver{},
		IdManagement:  idmanagement.Module{UserRepo: users},
		SharedSignals: sharedsignals.Module{
			StreamRepo: streams, TransmitterConfigRepo: configs, ReceiverConfigRepo: sharedsignalsmemory.NewSsfReceiverConfigRepository(),
			DeliveryRepo: deliveries, ReceivedEventRepo: sharedsignalsmemory.NewReceivedSecurityEventRepository(),
			RevocationEpochRepo: sharedsignalsmemory.NewAgentRevocationEpochRepository(),
			// 受信側はループバックの試験サーバーなので、配送先の検査を除く。
			CheckDeliveryEndpoint: func(context.Context, string) error { return nil },
		},
	})
	csrf, cookie := sharedSignalsAdminCSRF(t, e)
	created := sharedSignalsAdminPost(t, e, "/api/admin/v1/shared-signals/streams/transmitter", csrf, cookie, map[string]any{
		"delivery_endpoint": receiver.URL,
		"audience":          "https://receiver.example",
		"event_types":       []string{string(sharedsignalsdomain.CaepEventTypeSessionRevoked)},
	})
	if created.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", created.Code, created.Body.String())
	}
	var stream struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &stream); err != nil || stream.ID == "" {
		t.Fatalf("stream=%+v err=%v", stream, err)
	}

	keyStore, err := signingkeys.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenantports.WithTenant(context.Background(), &tenancydomain.Tenant{ID: tenancydomain.DefaultTenantID}, "", "")
	if err := sharedsignalsusecases.ProjectAgentAccessRevoked(ctx, sharedsignalsusecases.ProjectorDeps{
		StreamRepo: streams, TransmitterConfigRepo: configs, DeliveryRepo: deliveries,
		Signer: &sharedsignalssign.Signer{KeyStore: keyStore}, Issuer: "https://idp.example/realms/default",
	}, &sharedsignalsdomain.AgentAccessRevoked{
		At: now, TenantID: tenancydomain.DefaultTenantID, AgentID: "agent-1", Reason: sharedsignalsdomain.RevocationReasonAgentKilled,
	}); err != nil {
		t.Fatal(err)
	}
	processed, err := sharedsignalsusecases.ProcessDueDeliveries(ctx, sharedsignalsusecases.DeliverDeps{
		DeliveryRepo: deliveries, TransmitterConfigRepo: configs,
		Pusher: &sharedsignalspush.HTTPSecurityEventPusher{Client: receiver.Client()},
	}, now.Add(time.Second), 10)
	if err != nil {
		t.Fatal(err)
	}
	if processed != 1 || received.Load() != 1 || receivedBody == "" {
		t.Fatalf("processed=%d received=%d body=%q", processed, received.Load(), receivedBody)
	}
	listed, err := deliveries.ListByStream(ctx, tenancydomain.DefaultTenantID, stream.ID)
	if err != nil || len(listed) != 1 || listed[0].Status != sharedsignalsdomain.SecurityEventDeliveryStatusDelivered {
		t.Fatalf("deliveries=%+v err=%v", listed, err)
	}
	req := httptest.NewRequest(http.MethodGet, "/realms/default/api/admin/v1/shared-signals/streams/"+stream.ID+"/deliveries", http.NoBody)
	req.Header.Set("X-Demo-Sub", "admin")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	var body struct {
		Deliveries []struct {
			ID           string     `json:"id"`
			TenantID     string     `json:"tenant_id"`
			StreamID     string     `json:"stream_id"`
			SetJTI       string     `json:"set_jti"`
			Status       string     `json:"status"`
			AttemptCount *int       `json:"attempt_count"`
			CreatedAt    time.Time  `json:"created_at"`
			DeliveredAt  *time.Time `json:"delivered_at"`
		} `json:"deliveries"`
	}
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &body) != nil || len(body.Deliveries) != 1 {
		t.Fatalf("delivery list status=%d body=%s", rec.Code, rec.Body.String())
	}
	got := body.Deliveries[0]
	want := listed[0]
	if got.ID != want.ID || got.TenantID != tenancydomain.DefaultTenantID || got.StreamID != stream.ID || got.SetJTI == "" || got.SetJTI != want.SetJTI || got.Status != "delivered" || got.AttemptCount == nil || *got.AttemptCount != 1 || !got.CreatedAt.Equal(want.CreatedAt) || got.DeliveredAt == nil || !got.DeliveredAt.Equal(*want.DeliveredAt) {
		t.Fatalf("HTTP delivery=%+v, want %+v", got, want)
	}
}
