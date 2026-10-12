package handlers_http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

type streamContractBody struct {
	ID         string     `json:"id"`
	TenantID   string     `json:"tenant_id"`
	Direction  string     `json:"direction"`
	EventTypes []string   `json:"event_types"`
	Status     string     `json:"status"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  *time.Time `json:"updated_at,omitempty"`
}

func assertStreamContract(t *testing.T, rec *httptest.ResponseRecorder, status int, direction, state string, eventTypes []string) streamContractBody {
	t.Helper()
	var stream streamContractBody
	if rec.Code != status || !strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("stream status=%d headers=%v body=%s, want %d JSON", rec.Code, rec.Header(), rec.Body.String(), status)
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &stream); err != nil {
		t.Fatal(err)
	}
	if stream.ID == "" || stream.TenantID != tenancydomain.DefaultTenantID || stream.Direction != direction || stream.Status != state || !slices.Equal(stream.EventTypes, eventTypes) || stream.CreatedAt.IsZero() {
		t.Fatalf("stream=%+v, want %s %s with events %v", stream, direction, state, eventTypes)
	}
	return stream
}

func assertStreamNoContent(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if rec.Code != http.StatusNoContent || rec.Body.Len() != 0 {
		t.Fatalf("status=%d body=%s, want empty 204", rec.Code, rec.Body.String())
	}
}

//spec:covers REQ-SHAREDSIGNALS-009: 登録した両方向のストリームを管理 HTTP API で読み、更新と削除を応答本文と再読で確かめる。
//spec:covers REQ-SHAREDSIGNALS-008: 無効化と再有効化は空の 204 を返し、再読した状態に反映される。
//verify:operation RegisterSsfTransmitterStream: 201 JSON は送信側の ID、テナント、イベント種別、enabled、作成時刻を含む。
//verify:operation RegisterSsfReceiverStream: 201 JSON は受信側の ID、テナント、イベント種別、enabled、作成時刻を含む。
//verify:operation ListSsfStreams: 200 JSON の streams は登録した両方向のストリームと一致し、削除後に空になる。
//verify:operation GetSsfStream: 200 JSON は登録、更新、無効化と再有効化後の状態を返す。
//verify:operation UpdateSsfStream: 200 JSON は更新したイベント種別と更新時刻を含み、再読に一致する。
//verify:operation DisableSsfStream: 空の 204 を返し、再読した状態が disabled になる。
//verify:operation EnableSsfStream: 空の 204 を返し、再読した状態が enabled に戻る。
//verify:operation DeleteSsfStream: 両方向の削除は空の 204 を返し、一覧から消える。
func TestSharedSignalsStreamHTTPContracts(t *testing.T) {
	s := newExampleStack(t)
	eventTypes := []string{"session-revoked"}
	transmitter := assertStreamContract(t, s.admin(t, http.MethodPost, "/api/admin/v1/shared-signals/streams/transmitter", map[string]any{
		"delivery_endpoint": "https://receiver.example/events", "audience": "https://receiver.example", "event_types": eventTypes,
	}), http.StatusCreated, "Transmit", "enabled", eventTypes)
	receiver := assertStreamContract(t, s.admin(t, http.MethodPost, "/api/admin/v1/shared-signals/streams/receiver", receiverRequest(newTransmitterKey(t))), http.StatusCreated, "Receive", "enabled", eventTypes)
	listed := s.admin(t, http.MethodGet, "/api/admin/v1/shared-signals/streams", nil)
	var body struct {
		Streams []streamContractBody `json:"streams"`
	}
	if listed.Code != http.StatusOK || json.Unmarshal(listed.Body.Bytes(), &body) != nil || len(body.Streams) != 2 {
		t.Fatalf("list status=%d body=%s", listed.Code, listed.Body.String())
	}
	for _, want := range []streamContractBody{transmitter, receiver} {
		if !slices.ContainsFunc(body.Streams, func(got streamContractBody) bool { return reflect.DeepEqual(got, want) }) {
			t.Fatalf("list=%+v, missing %+v", body.Streams, want)
		}
		path := "/api/admin/v1/shared-signals/streams/" + want.ID
		got := assertStreamContract(t, s.admin(t, http.MethodGet, path, nil), http.StatusOK, want.Direction, "enabled", eventTypes)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("read=%+v, want %+v", got, want)
		}
	}
	path := "/api/admin/v1/shared-signals/streams/" + transmitter.ID
	updatedTypes := []string{"credential-change"}
	updated := assertStreamContract(t, s.admin(t, http.MethodPatch, path, map[string]any{"event_types": updatedTypes}), http.StatusOK, "Transmit", "enabled", updatedTypes)
	if updated.ID != transmitter.ID || !updated.CreatedAt.Equal(transmitter.CreatedAt) || updated.UpdatedAt == nil || updated.UpdatedAt.IsZero() {
		t.Fatalf("updated=%+v, original=%+v", updated, transmitter)
	}
	read := assertStreamContract(t, s.admin(t, http.MethodGet, path, nil), http.StatusOK, "Transmit", "enabled", updatedTypes)
	if !reflect.DeepEqual(read, updated) {
		t.Fatalf("read=%+v, want %+v", read, updated)
	}
	assertStreamNoContent(t, s.admin(t, http.MethodPost, path+"/disable", nil))
	assertStreamContract(t, s.admin(t, http.MethodGet, path, nil), http.StatusOK, "Transmit", "disabled", updatedTypes)
	assertStreamNoContent(t, s.admin(t, http.MethodPost, path+"/enable", nil))
	assertStreamContract(t, s.admin(t, http.MethodGet, path, nil), http.StatusOK, "Transmit", "enabled", updatedTypes)
	for _, id := range []string{transmitter.ID, receiver.ID} {
		assertStreamNoContent(t, s.admin(t, http.MethodDelete, "/api/admin/v1/shared-signals/streams/"+id, nil))
	}
	listed = s.admin(t, http.MethodGet, "/api/admin/v1/shared-signals/streams", nil)
	if listed.Code != http.StatusOK || json.Unmarshal(listed.Body.Bytes(), &body) != nil || body.Streams == nil || len(body.Streams) != 0 {
		t.Fatalf("after deletion status=%d body=%s", listed.Code, listed.Body.String())
	}
}

//spec:covers REQ-SHAREDSIGNALS-011: 管理者でない利用者の一覧、取得、配送の一覧を Problem Details で拒否する。
//verify:operation ListSsfStreams: 非管理者には 403 access_denied の Problem Details を返す。
//verify:operation GetSsfStream: 非管理者には 403 access_denied の Problem Details を返す。
//verify:operation ListSecurityEventDeliveries: 非管理者には 403 access_denied の Problem Details を返す。
func TestSharedSignalsReadsRefuseNonAdmin(t *testing.T) {
	s := newExampleStack(t)
	id := s.registerTransmitter(t, "https://receiver.example/events")
	for _, path := range []string{"/api/admin/v1/shared-signals/streams", "/api/admin/v1/shared-signals/streams/" + id, "/api/admin/v1/shared-signals/streams/" + id + "/deliveries"} {
		assertAccessDenied(t, s.request(t, exampleNonAdmin, http.MethodGet, path, nil))
	}
}

//spec:covers REQ-SHAREDSIGNALS-010: 署名した SET の受理は空の 202 を返し、失効エポックと受理イベントを確定する。
//verify:operation ReceiveSecurityEvent: 登録した受信側への妥当な署名付き SET は空の 202 と no-store を返し、対象の失効エポックを進める。
func TestSharedSignalsReceiveHTTPContract(t *testing.T) {
	s := newExampleStack(t)
	s.seedAgent(t, tenancydomain.DefaultTenantID, exampleAgentID)
	key := newTransmitterKey(t)
	id := s.registerReceiver(t, key)
	rec := s.postSET(t, id, key.sign(t, "contract-set", issSub(exampleAgentID)))
	if rec.Code != http.StatusAccepted || rec.Body.Len() != 0 || rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("status=%d headers=%v body=%s", rec.Code, rec.Header(), rec.Body.String())
	}
	if s.epoch(t, tenancydomain.DefaultTenantID, exampleAgentID) == nil || s.events.count("SecurityEventReceived") != 1 {
		t.Fatal("accepted SET did not advance epoch and emit SecurityEventReceived")
	}
}
