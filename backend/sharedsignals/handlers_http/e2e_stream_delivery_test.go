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

	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
	"github.com/ambi/idmagic/backend/idmanagement"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	"github.com/ambi/idmagic/backend/sharedsignals"
	sharedsignalsmemory "github.com/ambi/idmagic/backend/sharedsignals/db_memory"
	sharedsignalsdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
	sharedsignalspush "github.com/ambi/idmagic/backend/sharedsignals/push_http"
	sharedsignalssign "github.com/ambi/idmagic/backend/sharedsignals/sign_jose"
	sharedsignalsusecases "github.com/ambi/idmagic/backend/sharedsignals/usecases"
	signingmemory "github.com/ambi/idmagic/backend/signingkeys/keys_memory"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/labstack/echo/v5"
)

// REQ-SHAREDSIGNALS-006: 管理 API で登録した送信 stream に失効 event を射影し、worker が実 HTTP receiver へ SET を配送して delivered にする。
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
	users := usermemory.NewUserRepository()
	users.Seed(&userdomain.User{
		ID: "admin", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "admin", PasswordHash: "unused",
		Roles: []string{"admin"}, CreatedAt: now, UpdatedAt: now,
	})
	streams := sharedsignalsmemory.NewSsfStreamRepository()
	configs := sharedsignalsmemory.NewSsfTransmitterConfigRepository()
	deliveries := sharedsignalsmemory.NewSecurityEventDeliveryRepository()
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer:        "http://idp.test",
		AuthnResolver: authusecases.DemoHeaderResolver{},
		IdManagement:  idmanagement.Module{UserRepo: users},
		SharedSignals: sharedsignals.Module{
			StreamRepo: streams, TransmitterConfigRepo: configs, ReceiverConfigRepo: sharedsignalsmemory.NewSsfReceiverConfigRepository(),
			DeliveryRepo: deliveries, ReceivedEventRepo: sharedsignalsmemory.NewReceivedSecurityEventRepository(),
			RevocationEpochRepo: sharedsignalsmemory.NewAgentRevocationEpochRepository(),
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

	keyStore, err := signingmemory.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{ID: tenancydomain.DefaultTenantID}, "", "")
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
}
