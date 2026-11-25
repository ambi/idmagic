package testing_scimtwin

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// NewReplayServer は、記録した往復を順に一度ずつ返すサーバーを起動する。
//
// 受け取った要求は方式とパスの形、記録した要求が持つ Accept と Content-Type で記録と照合し、
// 食い違えばテストを失敗させる。識別子と本文は相手が割り当てたものや例の値なので比べない。
// クライアントが応答から取り出す値はテストが表明する。
func NewReplayServer(tb testing.TB, exchanges ...Exchange) *httptest.Server {
	tb.Helper()
	var (
		mu   sync.Mutex
		next int
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if next >= len(exchanges) {
			tb.Errorf("replay: unexpected request %s %s after all %d recorded exchanges", r.Method, r.URL.Path, len(exchanges))
			http.Error(w, "no recorded exchange left", http.StatusInternalServerError)
			return
		}
		ex := exchanges[next]
		next++
		checkRequest(tb, ex, r)
		writeRecordedResponse(w, ex.Response)
	}))
	tb.Cleanup(func() {
		server.Close()
		mu.Lock()
		defer mu.Unlock()
		if next != len(exchanges) {
			tb.Errorf("replay: %d of %d recorded exchanges were not requested", len(exchanges)-next, len(exchanges))
		}
	})
	return server
}

func checkRequest(tb testing.TB, ex Exchange, r *http.Request) {
	tb.Helper()
	if r.Method != ex.Request.Method || pathShape(r.URL.Path) != pathShape(ex.Request.Path) {
		tb.Errorf("replay %s: request %s %s, recorded %s %s", ex.Name, r.Method, r.URL.Path, ex.Request.Method, ex.Request.Path)
	}
	for _, name := range []string{"Accept", "Content-Type"} {
		want, recorded := ex.Request.Headers[name]
		if recorded && r.Header.Get(name) != want {
			tb.Errorf("replay %s: request header %s = %q, recorded %q", ex.Name, name, r.Header.Get(name), want)
		}
	}
}

func writeRecordedResponse(w http.ResponseWriter, resp RecordedResponse) {
	for name, value := range resp.Headers {
		w.Header().Set(name, value)
	}
	w.WriteHeader(resp.Status)
	if len(resp.Body) > 0 {
		_, _ = w.Write(resp.Body)
	}
}
