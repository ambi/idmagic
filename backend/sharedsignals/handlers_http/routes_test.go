package handlers_http_test

// SharedSignals は 1 パッケージに 2 種類のエラー契約が同居する唯一のコンテキストである。
// 管理 API (/api/admin/v1/shared-signals/**) は汎用 API なので RFC 9457 Problem Details
// を返し、inbound SET receiver (POST /ssf/streams/:id/events) は RFC 8935 §2.3 が
// 固定形式を MUST で定めるため Problem Details にしない。両者を同じテストで固定する。

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
	"github.com/ambi/idmagic/backend/idmanagement"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/sharedsignals"
	sharedsignalsmemory "github.com/ambi/idmagic/backend/sharedsignals/db_memory"

	"github.com/labstack/echo/v5"
)

func newSharedSignalsHandler(t *testing.T) *echo.Echo {
	t.Helper()
	userRepo := usermemory.NewUserRepository()
	now := time.Now().UTC()
	userRepo.Seed(&userdomain.User{
		ID: "admin", PreferredUsername: "admin", PasswordHash: "unused",
		Roles: []string{"admin"}, CreatedAt: now, UpdatedAt: now,
	})
	// 管理者でない主体でも同じ経路を通せるようにしておく (REQ-SHAREDSIGNALS-011)。
	userRepo.Seed(&userdomain.User{
		ID: "alice", PreferredUsername: "alice", PasswordHash: "unused",
		CreatedAt: now, UpdatedAt: now,
	})
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer:        "http://idp.test",
		AuthnResolver: authusecases.DemoHeaderResolver{},
		IdManagement:  idmanagement.Module{UserRepo: userRepo},
		SharedSignals: sharedsignals.Module{
			StreamRepo:            sharedsignalsmemory.NewSsfStreamRepository(),
			TransmitterConfigRepo: sharedsignalsmemory.NewSsfTransmitterConfigRepository(),
			ReceiverConfigRepo:    sharedsignalsmemory.NewSsfReceiverConfigRepository(),
			DeliveryRepo:          sharedsignalsmemory.NewSecurityEventDeliveryRepository(),
			ReceivedEventRepo:     sharedsignalsmemory.NewReceivedSecurityEventRepository(),
			RevocationEpochRepo:   sharedsignalsmemory.NewAgentRevocationEpochRepository(),
		},
	})
	return e
}

func sharedSignalsAdminCSRF(t *testing.T, e *echo.Echo) (string, *http.Cookie) {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/realms/default/api/auth/account", http.NoBody)
	request.Header.Set("X-Demo-Sub", "admin")
	response := httptest.NewRecorder()
	e.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("account status=%d body=%s", response.Code, response.Body.String())
	}
	var body struct {
		CSRFToken string `json:"csrf_token"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	cookies := response.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("csrf cookie missing")
	}
	return body.CSRFToken, cookies[0]
}

func sharedSignalsAdminPost(t *testing.T, e *echo.Echo, path, csrf string, cookie *http.Cookie, body any) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/realms/default"+path, bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "http://idp.test")
	request.Header.Set("X-Csrf-Token", csrf)
	request.Header.Set("X-Demo-Sub", "admin")
	request.AddCookie(cookie)
	response := httptest.NewRecorder()
	e.ServeHTTP(response, request)
	return response
}

// REQ-SHAREDSIGNALS-011: SsfStream の登録は管理者に限られる。管理者なら受理される
// 本文をそのまま送って拒否させ、ストリームが 1 本も増えていないことを読み直す。
func TestRegisterTransmitterStreamRejectsNonAdmin(t *testing.T) {
	e := newSharedSignalsHandler(t)
	csrf, cookie := sharedSignalsAdminCSRF(t, e)

	request := httptest.NewRequest(http.MethodPost, "/realms/default/api/admin/v1/shared-signals/streams/transmitter",
		bytes.NewReader([]byte(`{"delivery_endpoint":"https://receiver.example/events","audience":"https://receiver.example","event_types":["https://schemas.openid.net/secevent/caep/event-type/session-revoked"]}`)))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "http://idp.test")
	request.Header.Set("X-Csrf-Token", csrf)
	request.Header.Set("X-Demo-Sub", "alice")
	request.AddCookie(cookie)
	response := httptest.NewRecorder()
	e.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s, want 403", response.Code, response.Body.String())
	}

	listed := httptest.NewRequest(http.MethodGet, "/realms/default/api/admin/v1/shared-signals/streams", http.NoBody)
	listed.Header.Set("X-Demo-Sub", "admin")
	listing := httptest.NewRecorder()
	e.ServeHTTP(listing, listed)
	if listing.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", listing.Code, listing.Body.String())
	}
	var view struct {
		Streams []struct {
			ID string `json:"id"`
		} `json:"streams"`
	}
	if err := json.Unmarshal(listing.Body.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	if len(view.Streams) != 0 {
		t.Fatalf("streams = %#v, want the refused registration to have left none behind", view.Streams)
	}
}

// 管理 API 側: 業務規則違反は 422 の Problem Details。
func TestRegisterTransmitterStreamRejectsEmptyEventTypes(t *testing.T) {
	e := newSharedSignalsHandler(t)
	csrf, cookie := sharedSignalsAdminCSRF(t, e)

	rec := sharedSignalsAdminPost(t, e, "/api/admin/v1/shared-signals/streams/transmitter", csrf, cookie, map[string]any{
		"delivery_endpoint": "https://receiver.example/events",
		"audience":          "https://receiver.example",
		"event_types":       []string{},
	})

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if contentType := rec.Header().Get("Content-Type"); contentType != support.ProblemContentType {
		t.Fatalf("Content-Type=%q, want %q", contentType, support.ProblemContentType)
	}
	var problem support.Problem
	if err := json.Unmarshal(rec.Body.Bytes(), &problem); err != nil {
		t.Fatalf("unmarshal body: %v (body=%s)", err, rec.Body.String())
	}
	if problem.Type != "urn:idmagic:error:ssf_stream_event_types_required" {
		t.Errorf("type=%q, want urn:idmagic:error:ssf_stream_event_types_required", problem.Type)
	}
}

// inbound SET receiver 側: RFC 8935 §2.3 の固定形式を守るため Problem Details に
// しない。stream が存在しない場合も汎用 API の envelope へ寄せてはならない。
func TestReceiveSecurityEventDoesNotUseProblemDetails(t *testing.T) {
	e := newSharedSignalsHandler(t)

	request := httptest.NewRequest(http.MethodPost, "/realms/default/ssf/streams/no-such-stream/events",
		strings.NewReader("not-a-security-event-token"))
	request.Header.Set("Content-Type", "application/secevent+jwt")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, request)

	if rec.Code == http.StatusOK || rec.Code == http.StatusAccepted {
		t.Fatalf("status=%d, want a rejection; body=%s", rec.Code, rec.Body.String())
	}
	if contentType := rec.Header().Get("Content-Type"); strings.HasPrefix(contentType, support.ProblemContentType) {
		t.Fatalf("Content-Type=%q must not be Problem Details (RFC 8935 §2.3)", contentType)
	}
	if strings.Contains(rec.Body.String(), "urn:idmagic:error:") {
		t.Fatalf("body=%s must not carry a Problem Details type URN", rec.Body.String())
	}
}

// RFC 8935 §2.3 は拒否の応答本体を `err` と `description` で定める。この接点を
// Problem Details の外に置いている理由がその標準なので、標準の形そのものになって
// いなければ理由が成り立たない。
func TestReceiveSecurityEventUsesRFC8935ErrorShape(t *testing.T) {
	e := newSharedSignalsHandler(t)

	// RFC 8935 §2.3 の形。`err` は登録済みのエラーコード、`description` は人間向けの説明。
	type rfc8935Error struct {
		Err         string `json:"err"`
		Description string `json:"description"`
		// 旧い形が残っていないことを見るための欄。RFC はこの 2 つを定めていない。
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	decode := func(t *testing.T, rec *httptest.ResponseRecorder) rfc8935Error {
		t.Helper()
		var body rfc8935Error
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode receiver error: %v (body=%s)", err, rec.Body.String())
		}
		if body.Err == "" || body.Description == "" {
			t.Fatalf("body=%s must carry err and description (RFC 8935 §2.3)", rec.Body.String())
		}
		if body.Error != "" || body.Message != "" {
			t.Fatalf("body=%s must not keep the old error/message fields", rec.Body.String())
		}
		return body
	}

	t.Run("拒否された SET は 400 で security_event_rejected を返す", func(t *testing.T) {
		// ReceiveSecurityEvent は stream が引けない場合も ErrSecurityEventRejected を
		// 返すので、この接点から 404 は出ない。stream の存在を呼び出し側へ漏らさない
		// fail-closed の設計であり、本文の形が変わっても保たれることを固定する。
		request := httptest.NewRequest(http.MethodPost, "/realms/default/ssf/streams/no-such-stream/events",
			strings.NewReader("not-a-security-event-token"))
		request.Header.Set("Content-Type", "application/secevent+jwt")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, request)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d body=%s, want 400", rec.Code, rec.Body.String())
		}
		if body := decode(t, rec); body.Err != "security_event_rejected" {
			t.Fatalf("err=%q, want security_event_rejected", body.Err)
		}
	})

	t.Run("大きすぎる SET は 413 で security_event_token_too_large を返す", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "/realms/default/ssf/streams/no-such-stream/events",
			strings.NewReader(strings.Repeat("a", 1<<20)))
		request.Header.Set("Content-Type", "application/secevent+jwt")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, request)

		if rec.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("status=%d body=%s, want 413", rec.Code, rec.Body.String())
		}
		if body := decode(t, rec); body.Err != "security_event_token_too_large" {
			t.Fatalf("err=%q, want security_event_token_too_large", body.Err)
		}
	})
}
