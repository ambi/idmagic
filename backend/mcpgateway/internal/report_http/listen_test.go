package report_http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"
	"time"
)

// listenServers は、tools/list に現在の一覧を返し、subscriptions/listen を回ごとの関数に任せる上流と、
// 報告されたツール名を順に受ける IdMagic の代役を立てる。
type listenServers struct {
	mu       sync.Mutex
	tools    []string
	attempts int
	reports  chan []string
}

func (s *listenServers) setTools(names ...string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tools = names
}

func newListenServers(t *testing.T, listen func(attempt int, w http.ResponseWriter, r *http.Request), tools ...string) (*listenServers, Reporter) {
	t.Helper()
	servers := &listenServers{tools: tools, reports: make(chan []string, 16)}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Header.Get("Mcp-Method") {
		case "tools/list":
			servers.mu.Lock()
			listed := make([]map[string]any, len(servers.tools))
			for index, name := range servers.tools {
				listed[index] = map[string]any{"name": name}
			}
			servers.mu.Unlock()
			_, _ = io.WriteString(w, page(t, listed, ""))
		case "subscriptions/listen":
			var request struct {
				Params struct {
					Notifications map[string]bool `json:"notifications"`
				} `json:"params"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil || !request.Params.Notifications["toolsListChanged"] {
				t.Errorf("listen request: %+v %v", request, err)
			}
			servers.mu.Lock()
			servers.attempts++
			attempt := servers.attempts
			servers.mu.Unlock()
			listen(attempt, w, r)
		default:
			t.Errorf("unexpected method %q", r.Header.Get("Mcp-Method"))
		}
	}))
	t.Cleanup(upstream.Close)
	idmagic := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Error(err)
		}
		var reported []Tool
		if err := json.Unmarshal([]byte(r.PostForm.Get("tools")), &reported); err != nil {
			t.Error(err)
		}
		names := []string{}
		for _, tool := range reported {
			names = append(names, tool.Name)
		}
		servers.reports <- names
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(idmagic.Close)
	return servers, Reporter{
		Upstream: upstream.URL, ReportURL: idmagic.URL, ClientID: "gateway", ClientSecret: "secret",
		UpstreamClient: upstream.Client(), IdMagicClient: idmagic.Client(),
	}
}

func writeEvent(w http.ResponseWriter, message map[string]any) {
	raw, err := json.Marshal(message)
	if err != nil {
		panic(err) // テストが組み立てる値は必ず符号化できる。
	}
	_, _ = fmt.Fprintf(w, "data: %s\n\n", raw)
	w.(http.Flusher).Flush()
}

func acknowledgment(notifications map[string]any) map[string]any {
	return map[string]any{"jsonrpc": "2.0", "method": "notifications/subscriptions/acknowledged", "params": map[string]any{
		"_meta": map[string]any{"io.modelcontextprotocol/subscriptionId": 1}, "notifications": notifications,
	}}
}

func toolsChanged(subscription int) map[string]any {
	return map[string]any{"jsonrpc": "2.0", "method": "notifications/tools/list_changed", "params": map[string]any{
		"_meta": map[string]any{"io.modelcontextprotocol/subscriptionId": subscription},
	}}
}

func openStream(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.WriteHeader(http.StatusOK)
	w.(http.Flusher).Flush()
}

// recordedWaits は待ちの長さを記録し、待たずに戻る。
type recordedWaits struct {
	mu        sync.Mutex
	durations []time.Duration
}

func (w *recordedWaits) wait(ctx context.Context, d time.Duration) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.durations = append(w.durations, d)
	return ctx.Err()
}

func (w *recordedWaits) all() []time.Duration {
	w.mu.Lock()
	defer w.mu.Unlock()
	return slices.Clone(w.durations)
}

func nextReport(t *testing.T, reports <-chan []string) []string {
	t.Helper()
	select {
	case names := <-reports:
		return names
	case <-time.After(5 * time.Second):
		t.Fatal("no report")
		return nil
	}
}

func runUntilReturn(t *testing.T, listener Listener) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		listener.Run(t.Context())
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("the listener did not stop")
	}
}

//spec:covers REQ-MCPGATEWAY-012: 受信確認の後に一覧を報告し、自分の購読の tools/list_changed を受けると変更後の一覧を間隔を空けて報告し直す。
func TestListenReportsAgainWhenToolsChange(t *testing.T) {
	notify := make(chan struct{})
	servers, reporter := newListenServers(t, func(_ int, w http.ResponseWriter, r *http.Request) {
		openStream(w)
		writeEvent(w, acknowledgment(map[string]any{"toolsListChanged": true}))
		<-notify
		writeEvent(w, toolsChanged(1))
		<-r.Context().Done()
	}, "read_document")
	paced := &recordedWaits{}
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		Listener{Reporter: reporter, Client: &http.Client{}, WaitToReport: paced.wait, WaitToReconnect: (&recordedWaits{}).wait}.Run(ctx)
		close(done)
	}()

	if names := nextReport(t, servers.reports); !slices.Equal(names, []string{"read_document"}) {
		t.Fatalf("first report=%v", names)
	}
	servers.setTools("read_document", "write_document")
	close(notify)
	if names := nextReport(t, servers.reports); !slices.Equal(names, []string{"read_document", "write_document"}) {
		t.Fatalf("report after the notification=%v", names)
	}
	cancel()
	<-done
	waits := paced.all()
	if len(waits) != 1 || waits[0] <= 0 || waits[0] > reportInterval {
		t.Fatalf("waits before the second report=%v, want one wait within %s", waits, reportInterval)
	}
}

//spec:covers REQ-MCPGATEWAY-011, REQ-MCPGATEWAY-012: 変更通知に対応しない上流では、起動時に一度だけ報告して待ち受けをやめ、開き直さない。
func TestListenReportsOnceAndStopsWhenTheUpstreamCannotNotify(t *testing.T) {
	for name, listen := range map[string]func(int, http.ResponseWriter, *http.Request){
		"acknowledgment without tools": func(_ int, w http.ResponseWriter, r *http.Request) {
			openStream(w)
			writeEvent(w, acknowledgment(map[string]any{}))
			<-r.Context().Done()
		},
		"HTTP 404": func(_ int, w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		},
		"JSON-RPC error": func(_ int, w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"Method not found"}}`)
		},
	} {
		t.Run(name, func(t *testing.T) {
			servers, reporter := newListenServers(t, listen, "read_document")
			reconnects := &recordedWaits{}
			runUntilReturn(t, Listener{Reporter: reporter, Client: &http.Client{}, WaitToReconnect: reconnects.wait, WaitToReport: (&recordedWaits{}).wait})
			if names := nextReport(t, servers.reports); !slices.Equal(names, []string{"read_document"}) {
				t.Fatalf("startup report=%v", names)
			}
			servers.mu.Lock()
			attempts := servers.attempts
			servers.mu.Unlock()
			if len(servers.reports) != 0 || len(reconnects.all()) != 0 || attempts != 1 {
				t.Fatalf("extra reports=%d reconnect waits=%v attempts=%d", len(servers.reports), reconnects.all(), attempts)
			}
		})
	}
}

//spec:covers REQ-MCPGATEWAY-012: 5xx、受信確認の期限切れ、購読の終了、切断では1秒から倍にして開き直し、受信確認を得たら1秒に戻し、受信確認ごとに報告する。
func TestListenReconnectsWithBackoff(t *testing.T) {
	servers, reporter := newListenServers(t, func(attempt int, w http.ResponseWriter, r *http.Request) {
		switch attempt {
		case 1:
			w.WriteHeader(http.StatusServiceUnavailable)
		case 2:
			openStream(w) // 受信確認を送らない
			<-r.Context().Done()
		case 3:
			openStream(w)
			writeEvent(w, acknowledgment(map[string]any{"toolsListChanged": true}))
			writeEvent(w, map[string]any{"jsonrpc": "2.0", "id": 1, "result": map[string]any{"resultType": "complete"}})
		case 4:
			openStream(w)
			writeEvent(w, acknowledgment(map[string]any{"toolsListChanged": true}))
		default:
			w.WriteHeader(http.StatusForbidden)
		}
	}, "read_document")
	reconnects := &recordedWaits{}
	runUntilReturn(t, Listener{
		Reporter: reporter, Client: &http.Client{}, AckTimeout: 50 * time.Millisecond,
		WaitToReconnect: reconnects.wait, WaitToReport: (&recordedWaits{}).wait,
	})
	want := []time.Duration{time.Second, 2 * time.Second, time.Second, time.Second}
	if got := reconnects.all(); !slices.Equal(got, want) {
		t.Fatalf("reconnect waits=%v, want %v", got, want)
	}
	// 起動時の報告と、開き直した後の受信確認による報告がある。近い契機は一回にまとまり得る。
	if count := len(servers.reports); count < 2 {
		t.Fatalf("reports=%d, want the startup report and one after reconnecting", count)
	}
}

//spec:covers REQ-MCPGATEWAY-012: 開き直す間隔は倍にして5分で止まる。
func TestNextBackoffDoublesUpToFiveMinutes(t *testing.T) {
	for current, want := range map[time.Duration]time.Duration{
		time.Second: 2 * time.Second, 2 * time.Minute: 4 * time.Minute, 4 * time.Minute: maxBackoff, maxBackoff: maxBackoff,
	} {
		if got := nextBackoff(current); got != want {
			t.Errorf("nextBackoff(%s)=%s, want %s", current, got, want)
		}
	}
	if maxBackoff != 5*time.Minute {
		t.Fatalf("maxBackoff=%s", maxBackoff)
	}
}

//spec:covers REQ-MCPGATEWAY-012: 前回の報告の開始から10秒以内の契機は10秒が経つまで待ち、待つ間の契機を一回の報告にまとめる。10秒を過ぎた契機は待たない。
func TestReportSchedulerSpacesAndCoalescesReports(t *testing.T) {
	var mu sync.Mutex
	now := time.Date(2026, 11, 4, 9, 0, 0, 0, time.UTC)
	clock := func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		return now
	}
	entered, release := make(chan time.Duration, 1), make(chan struct{})
	reported := make(chan struct{}, 8)
	scheduler := newReportScheduler(clock, func(ctx context.Context, d time.Duration) error {
		entered <- d
		<-release
		return ctx.Err()
	}, func(context.Context) { reported <- struct{}{} })
	done := make(chan struct{})
	go func() {
		scheduler.run(t.Context())
		close(done)
	}()
	waitFor := func(what string) {
		t.Helper()
		select {
		case <-reported:
		case <-time.After(5 * time.Second):
			t.Fatalf("no %s", what)
		}
	}

	scheduler.trigger()
	waitFor("first report")
	scheduler.trigger()
	if d := <-entered; d != reportInterval {
		t.Fatalf("wait=%s, want %s", d, reportInterval)
	}
	scheduler.trigger()
	scheduler.trigger()
	scheduler.trigger()
	release <- struct{}{}
	waitFor("coalesced report")

	mu.Lock()
	now = now.Add(reportInterval + time.Second)
	mu.Unlock()
	scheduler.trigger()
	waitFor("report after the interval")
	scheduler.close()
	<-done
	if len(reported) != 0 || len(entered) != 0 {
		t.Fatalf("extra reports=%d extra waits=%d", len(reported), len(entered))
	}
}

//spec:covers REQ-MCPGATEWAY-012: 受信確認と通知は要求の id を subscriptionId に持つものだけを読み、購読への応答は終了かエラーとして読む。
func TestClassifyListenMessage(t *testing.T) {
	for name, tc := range map[string]struct {
		message string
		want    listenEvent
	}{
		"acknowledged":             {`{"method":"notifications/subscriptions/acknowledged","params":{"_meta":{"io.modelcontextprotocol/subscriptionId":1},"notifications":{"toolsListChanged":true}}}`, listenAcknowledged},
		"acknowledged without":     {`{"method":"notifications/subscriptions/acknowledged","params":{"_meta":{"io.modelcontextprotocol/subscriptionId":1},"notifications":{}}}`, listenUnsupported},
		"another acknowledgment":   {`{"method":"notifications/subscriptions/acknowledged","params":{"_meta":{"io.modelcontextprotocol/subscriptionId":2},"notifications":{"toolsListChanged":true}}}`, listenIgnored},
		"tools changed":            {`{"method":"notifications/tools/list_changed","params":{"_meta":{"io.modelcontextprotocol/subscriptionId":1}}}`, listenToolsChanged},
		"another subscription":     {`{"method":"notifications/tools/list_changed","params":{"_meta":{"io.modelcontextprotocol/subscriptionId":2}}}`, listenIgnored},
		"no subscription":          {`{"method":"notifications/tools/list_changed"}`, listenIgnored},
		"other notification":       {`{"method":"notifications/resources/updated","params":{"_meta":{"io.modelcontextprotocol/subscriptionId":1}}}`, listenIgnored},
		"graceful end":             {`{"id":1,"result":{"resultType":"complete"}}`, listenEnded},
		"error":                    {`{"id":1,"error":{"code":-32601,"message":"no"}}`, listenFailed},
		"response to another call": {`{"id":2,"result":{}}`, listenIgnored},
		"not JSON":                 {`ping`, listenIgnored},
	} {
		if got := classifyListenMessage([]byte(tc.message)); got != tc.want {
			t.Errorf("%s: got %d, want %d", name, got, tc.want)
		}
	}
}
