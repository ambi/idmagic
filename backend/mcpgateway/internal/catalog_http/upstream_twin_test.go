package catalog_http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/shared/testing_mcptwin"
)

func twinPublisher(t *testing.T, tw *testing_mcptwin.Twin) (Publisher, *catalogSink) {
	t.Helper()
	upstream := httptest.NewServer(tw)
	t.Cleanup(upstream.Close)
	return replayPublisher(t, upstream)
}

func twinWithTools(names ...string) *testing_mcptwin.Twin {
	tw := testing_mcptwin.NewTwin()
	for _, name := range names {
		tw.AddTool(map[string]any{"name": name, "inputSchema": map[string]any{"type": "object"}})
	}
	return tw
}

func names(tools []Tool) []string {
	out := make([]string, 0, len(tools))
	for _, tool := range tools {
		out = append(out, tool.Name)
	}
	return out
}

//spec:covers REQ-MCPGATEWAY-011: 模倣の上流の一ページずつの一覧を、空文字列を含む不透明なカーソルでたどる。
func TestPublisher_FollowsTwinPagination(t *testing.T) {
	tw := twinWithTools("a", "b", "c")
	tw.PageSize, tw.Cursors = 1, []string{"", "opaque=="}
	publisher, sink := twinPublisher(t, tw)

	if _, err := publisher.SendCatalog(context.Background()); err != nil {
		t.Fatal(err)
	}
	sent := sink.sent()
	if len(sent) != 1 || !slices.Equal(names(sent[0].tools), []string{"a", "b", "c"}) || sent[0].complete != "true" {
		t.Errorf("sent %+v, want a, b, c complete", sent)
	}
	if got := tw.Requests(); len(got) != 3 {
		t.Errorf("requests = %q, want three tools/list pages", got)
	}
}

//spec:covers REQ-MCPGATEWAY-011: 二ページ目の 5xx と不正な本文では取得できた分を不完全として送り、最初のページの失敗ではカタログを送らない。
func TestPublisher_HandlesTwinFaults(t *testing.T) {
	for _, tc := range []struct {
		name   string
		faults []testing_mcptwin.Fault
		want   []string
		err    string
	}{
		{"二ページ目が 503", []testing_mcptwin.Fault{{}, {Status: 503}}, []string{"a"}, ""},
		{"二ページ目が不正な本文", []testing_mcptwin.Fault{{}, {Status: 200, ContentType: "application/json", Body: "{"}}, []string{"a"}, ""},
		{"一ページ目が 500", []testing_mcptwin.Fault{{Status: 500}}, nil, "HTTP 500"},
		{"一ページ目が不正な本文", []testing_mcptwin.Fault{{Status: 200, ContentType: "application/json", Body: "<html>"}}, nil, "not JSON-RPC"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tw := twinWithTools("a", "b")
			tw.PageSize = 1
			tw.InjectFaults(tc.faults...)
			publisher, sink := twinPublisher(t, tw)
			_, err := publisher.SendCatalog(context.Background())
			if tc.err != "" {
				if err == nil || !strings.Contains(err.Error(), tc.err) || len(sink.sent()) != 0 {
					t.Errorf("SendCatalog() error = %v and sent %+v, want %q and no catalog", err, sink.sent(), tc.err)
				}
				return
			}
			sent := sink.sent()
			if err != nil || len(sent) != 1 || !slices.Equal(names(sent[0].tools), tc.want) || sent[0].complete != "false" {
				t.Errorf("SendCatalog() error = %v and sent %+v, want %v incomplete", err, sent, tc.want)
			}
		})
	}
}

//spec:covers REQ-MCPGATEWAY-011: 上流が送る版を受け付けないか、トークンを求めるなら、最初のページの失敗としてカタログを送らない。
func TestPublisher_SendsNoCatalogWhenTheTwinRefuses(t *testing.T) {
	unsupported := twinWithTools("a")
	unsupported.Supported = []string{"2025-11-25"}
	guarded := twinWithTools("a")
	guarded.Tokens = map[string][]string{"t": nil}
	for name, tw := range map[string]*testing_mcptwin.Twin{"HTTP 400": unsupported, "HTTP 401": guarded} {
		publisher, sink := twinPublisher(t, tw)
		if _, err := publisher.SendCatalog(context.Background()); err == nil || !strings.Contains(err.Error(), name) || len(sink.sent()) != 0 {
			t.Errorf("SendCatalog() error = %v and sent %+v, want %s and no catalog", err, sink.sent(), name)
		}
	}
}

func nextSent(t *testing.T, sink *catalogSink, count int) sentCatalog {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if sent := sink.sent(); len(sent) >= count {
			return sent[count-1]
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("catalog %d was not sent", count)
	return sentCatalog{}
}

//spec:covers REQ-MCPGATEWAY-012: 模倣の上流でツールを足すと、変更通知を受けて足した後の一覧を送り直し、購読が終わると開き直す。
func TestListener_ResendsWhenTheTwinChangesTools(t *testing.T) {
	tw := twinWithTools("a")
	publisher, sink := twinPublisher(t, tw)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	reconnected := make(chan struct{}, 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		Listener{
			Publisher: publisher, Client: &http.Client{},
			WaitToSend: func(context.Context, time.Duration) error { return nil },
			WaitToReconnect: func(context.Context, time.Duration) error {
				reconnected <- struct{}{}
				return context.Canceled
			},
		}.Run(ctx)
	}()

	if got := nextSent(t, sink, 1); !slices.Equal(names(got.tools), []string{"a"}) {
		t.Fatalf("first catalog = %+v, want a", got)
	}
	tw.AddTool(map[string]any{"name": "b"})
	if got := nextSent(t, sink, 2); !slices.Equal(names(got.tools), []string{"a", "b"}) {
		t.Fatalf("catalog after the change = %+v, want a, b", got)
	}
	tw.EndSubscriptions()
	select {
	case <-reconnected:
	case <-time.After(5 * time.Second):
		t.Fatal("the listener did not reconnect after the subscription ended")
	}
	<-done
}
