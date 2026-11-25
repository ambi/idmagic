package testing_httprecord

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// Replay は、再生サーバーが受け取った要求を記録とどう照合するかである。
type Replay struct {
	// PathShape は、照合の前にパスから相手が割り当てた識別子などを除く。nil なら パスをそのまま比べる。
	PathShape func(path string) string
	// Headers は、記録した要求が持つときに値まで一致を求めるヘッダーである。nil なら Accept と Content-Type である。
	Headers []string
	// CheckBody は、記録した要求の本文と受け取った本文の食い違いを返す。nil なら本文を比べない。
	CheckBody func(recorded, got []byte) []string
}

// Serve は、記録した往復を順に一度ずつ返すサーバーを起動する。
//
// 受け取った要求は方式、パスの形、Headers のヘッダー、CheckBody で記録と照合し、
// 食い違えばテストを失敗させる。クライアントが応答から取り出す値はテストが表明する。
func (c Replay) Serve(tb testing.TB, exchanges ...Exchange) *httptest.Server {
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
		c.check(tb, ex, r)
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

func (c Replay) check(tb testing.TB, ex Exchange, r *http.Request) {
	tb.Helper()
	shape := c.PathShape
	if shape == nil {
		shape = func(path string) string { return path }
	}
	if r.Method != ex.Request.Method || shape(r.URL.Path) != shape(ex.Request.Path) {
		tb.Errorf("replay %s: request %s %s, recorded %s %s", ex.Name, r.Method, r.URL.Path, ex.Request.Method, ex.Request.Path)
	}
	headers := c.Headers
	if headers == nil {
		headers = []string{"Accept", "Content-Type"}
	}
	for _, name := range headers {
		want, recorded := ex.Request.Headers[name]
		if recorded && r.Header.Get(name) != want {
			tb.Errorf("replay %s: request header %s = %q, recorded %q", ex.Name, name, r.Header.Get(name), want)
		}
	}
	if c.CheckBody == nil {
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		tb.Errorf("replay %s: read request body: %v", ex.Name, err)
		return
	}
	for _, diff := range c.CheckBody(BodyBytes(ex.Request.Body), body) {
		tb.Errorf("replay %s: request body: %s", ex.Name, diff)
	}
}

func writeRecordedResponse(w http.ResponseWriter, resp RecordedResponse) {
	for name, value := range resp.Headers {
		w.Header().Set(name, value)
	}
	w.WriteHeader(resp.Status)
	if body := BodyBytes(resp.Body); len(body) > 0 {
		_, _ = w.Write(body)
	}
}
