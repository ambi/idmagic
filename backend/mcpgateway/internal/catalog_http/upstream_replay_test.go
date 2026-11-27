package catalog_http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"

	"github.com/ambi/idmagic/backend/shared/testing_mcptwin"
)

// catalogSink は IdMagic のカタログの入口の代役で、受け取ったカタログを順に覚える。
type catalogSink struct {
	mu       sync.Mutex
	catalogs []sentCatalog
}

type sentCatalog struct {
	tools    []Tool
	complete string
}

func (s *catalogSink) serve(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Error(err)
		}
		var tools []Tool
		if err := json.Unmarshal([]byte(r.PostForm.Get("tools")), &tools); err != nil {
			t.Error(err)
		}
		s.mu.Lock()
		s.catalogs = append(s.catalogs, sentCatalog{tools: tools, complete: r.PostForm.Get("complete")})
		s.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)
	return server
}

func (s *catalogSink) sent() []sentCatalog {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]sentCatalog(nil), s.catalogs...)
}

func replayPublisher(t *testing.T, upstream *httptest.Server) (Publisher, *catalogSink) {
	t.Helper()
	sink := &catalogSink{}
	idmagic := sink.serve(t)
	return Publisher{
		Upstream: upstream.URL + "/mcp", CatalogURL: idmagic.URL, ClientID: "gateway", ClientSecret: "secret",
		UpstreamClient: upstream.Client(), IdMagicClient: idmagic.Client(),
	}, sink
}

// MCP 2026-07-28 の server/tools の例のツールを、REQ-MCPGATEWAY-011 の項目に読んだ値である。
var (
	recordedWeather = Tool{Name: "get_weather", Title: "Weather Information Provider", Description: "Get current weather information for a location", Arguments: []string{"location"}}
	recordedTime    = Tool{Name: "get_current_time", Description: "Returns the current server time"}
)

//spec:covers REQ-MCPGATEWAY-011: 仕様の例の tools/list の二ページを nextCursor でたどり、名前、表示名、説明、引数のキーをカタログとして送る。
func TestPublisher_ReadsRecordedToolsList(t *testing.T) {
	rec := testing_mcptwin.MCP20260728Recording(t)
	publisher, sink := replayPublisher(t, testing_mcptwin.NewReplayServer(t, rec.Exchange(t, "tools-list-first-page"), rec.Exchange(t, "tools-list-last-page")))

	result, err := publisher.SendCatalog(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := []sentCatalog{{tools: []Tool{recordedWeather, recordedTime}, complete: "true"}}
	if got := sink.sent(); !reflect.DeepEqual(got, want) || result != (Result{Tools: 2, Complete: true}) {
		t.Errorf("SendCatalog() = %+v and sent %+v, want 2 complete tools %+v", result, got, want)
	}
}

//spec:covers REQ-MCPGATEWAY-011: 空文字列の nextCursor を終わりと読まず、cursor に空文字列を載せて次のページを取る。
func TestPublisher_FollowsEmptyNextCursor(t *testing.T) {
	rec := testing_mcptwin.MCP20260728Recording(t)
	publisher, sink := replayPublisher(t, testing_mcptwin.NewReplayServer(t, rec.Exchange(t, "tools-list-empty-cursor"), rec.Exchange(t, "tools-list-after-empty-cursor")))

	if _, err := publisher.SendCatalog(context.Background()); err != nil {
		t.Fatal(err)
	}
	want := []sentCatalog{{tools: []Tool{recordedWeather, recordedTime}, complete: "true"}}
	if got := sink.sent(); !reflect.DeepEqual(got, want) {
		t.Errorf("sent %+v, want %+v", got, want)
	}
}

// listenWith は記録の subscriptions/listen を一度だけ再生し、待ち受けの終わり方と送信の契機の数を返す。
func listenWith(t *testing.T, exchange string) (listenOutcome, int) {
	t.Helper()
	rec := testing_mcptwin.MCP20260728Recording(t)
	upstream := testing_mcptwin.NewReplayServer(t, rec.Exchange(t, exchange))
	listener := Listener{Publisher: Publisher{Upstream: upstream.URL + "/mcp"}, Client: upstream.Client()}
	triggers := 0
	outcome := listener.listen(context.Background(), func() { triggers++ })
	return outcome, triggers
}

//spec:covers REQ-MCPGATEWAY-012: 仕様の例の受信確認と tools/list_changed をそれぞれ送信の契機にし、購読への成功の応答を終了として読んで開き直す。
func TestListener_ReadsRecordedSubscription(t *testing.T) {
	outcome, triggers := listenWith(t, "subscriptions-listen")
	if triggers != 2 || !outcome.acknowledged || outcome.stop || outcome.reason.Error() != "the upstream ended the subscription" {
		t.Errorf("listen() = %+v with %d triggers, want acknowledged, not stopped, ended, and 2 triggers", outcome, triggers)
	}
}

//spec:covers REQ-MCPGATEWAY-012: 別の購読の id を持つ仕様の例の tools/list_changed は契機にしない。
func TestListener_IgnoresRecordedNotificationOfAnotherSubscription(t *testing.T) {
	outcome, triggers := listenWith(t, "subscriptions-listen-other-subscription")
	if triggers != 1 || !outcome.acknowledged || outcome.stop {
		t.Errorf("listen() = %+v with %d triggers, want acknowledged, not stopped, and only the acknowledgment as trigger", outcome, triggers)
	}
}
