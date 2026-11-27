package breaches_hibp

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/shared/testing_httprecord"

	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func sha1PrefixSuffix(candidate string) (prefix, suffix string) {
	return hibpRangePrefixSuffix(candidate)
}

// hibpRangeRecording は、api.pwnedpasswords.com から実際に取った Range API の応答の記録を返す。
func hibpRangeRecording(t *testing.T) testing_httprecord.Recording {
	t.Helper()
	data, err := os.ReadFile("testdata/hibp-range-api.json")
	if err != nil {
		t.Fatal(err)
	}
	rec, err := testing_httprecord.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	return rec
}

// replayedHibpChecker は、記録した往復を返す再生サーバーへ問い合わせる checker を返す。
// 再生サーバーは要求のパスと Add-Padding を記録と照合する。
func replayedHibpChecker(t *testing.T) *HibpBreachedPasswordChecker {
	t.Helper()
	server := testing_httprecord.Replay{Headers: []string{"Add-Padding"}}.Serve(t, hibpRangeRecording(t).Exchange(t, "range-password"))
	return NewHibpBreachedPasswordChecker("test",
		WithHibpEndpoint(server.URL+"/range/"), WithHibpHTTPClient(server.Client()))
}

// Range API の文書は、応答の各行を「SHA-1 の接尾辞:件数」とする。
// password の SHA-1 は 5BAA61E4C9B93F3F0682250B6CF8331B7EE68FD8 で、記録の応答は接尾辞 1E4C9B…FD8 に正の件数を返している。
func TestHibpIsBreachedReportsRecordedMatch(t *testing.T) {
	if !replayedHibpChecker(t).IsBreached(context.Background(), "password") {
		t.Fatal("IsBreached(password) = false, want true for the recorded suffix with a positive count")
	}
}

// 接頭辞が同じでも、応答に接尾辞がなければ漏洩していない。
// idm-miss-5170674 の SHA-1 は 5BAA6FF4A2359C9225C7416A735098455A83B56F で、記録の応答はその接尾辞を持たない。
func TestHibpIsBreachedMissesSuffixAbsentFromRecordedRange(t *testing.T) {
	if replayedHibpChecker(t).IsBreached(context.Background(), "idm-miss-5170674") {
		t.Fatal("IsBreached(idm-miss-5170674) = true, want false for a suffix the range does not list")
	}
}

// 実物の応答は行を CRLF で区切り、最後の行に改行を付けない。
// Add-Padding の文書は、詰め物の行の件数が 0 で、捨ててよいとする。
// 記録の応答は 2085 行で、そのうち 107 行が件数 0 の詰め物なので、漏洩した接尾辞は 1978 件である。
// 先頭の行は 003CD2…8237:2、改行のない最後の行は詰め物の 3204EF…AC4D:0 である。
func TestParseHibpRangeReadsRecordedLinesAndDropsPadding(t *testing.T) {
	body := testing_httprecord.BodyBytes(hibpRangeRecording(t).Exchange(t, "range-password").Response.Body)
	suffixes := parseHibpRange(string(body))
	if len(suffixes) != 1978 {
		t.Errorf("len(parseHibpRange(recorded)) = %d, want 1978", len(suffixes))
	}
	if _, ok := suffixes["003CD215739D7C1B2218670D26F81408237"]; !ok {
		t.Error("first recorded line 003CD2…8237:2 is missing")
	}
	for _, padding := range []string{"FC56D89F9F597261B39EC900AF4A6F3FCFB", "3204EF55DB0EFEBE15CFA43B75DB2D9AC4D"} {
		if _, ok := suffixes[padding]; ok {
			t.Errorf("recorded padding line %s:0 is reported as breached", padding)
		}
	}
}

// Range API の利用者は User-Agent を送る必要がある。記録の要求は curl の User-Agent なので、製品の値は手書きの応答で確かめる。
func TestHibpSendsUserAgent(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("User-Agent")
	}))
	defer srv.Close()

	NewHibpBreachedPasswordChecker("1.2.3",
		WithHibpEndpoint(srv.URL+"/"), WithHibpHTTPClient(srv.Client())).IsBreached(context.Background(), "password")
	if got != "idmagic/1.2.3" {
		t.Fatalf("User-Agent = %q, want idmagic/1.2.3", got)
	}
}

func TestHibpFailOpenOnServerError(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	checker := NewHibpBreachedPasswordChecker("test",
		WithHibpEndpoint(srv.URL+"/"), WithHibpHTTPClient(srv.Client()),
		WithHibpMeter(provider.Meter("test")))
	if checker.IsBreached(context.Background(), "password") {
		t.Fatal("expected fail-open (false) on 5xx")
	}
	if got := failureCount(t, reader, "server_error"); got != 1 {
		t.Fatalf("server_error failure metric=%d, want 1", got)
	}
}

func TestHibpFailOpenOnTimeout(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(200 * time.Millisecond)
		fmt.Fprint(w, "00000000000000000000000000000000003:1\r\n")
	}))
	defer srv.Close()

	checker := NewHibpBreachedPasswordChecker("test",
		WithHibpEndpoint(srv.URL+"/"),
		WithHibpHTTPClient(&http.Client{Timeout: 30 * time.Millisecond}),
		WithHibpMeter(provider.Meter("test")))
	if checker.IsBreached(context.Background(), "password") {
		t.Fatal("expected fail-open (false) on timeout")
	}
	if got := failureCount(t, reader, "timeout"); got != 1 {
		t.Fatalf("timeout failure metric=%d, want 1", got)
	}
}

func TestHibpFailOpenOnConnectionError(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	// port 1 は接続拒否される。
	checker := NewHibpBreachedPasswordChecker("test",
		WithHibpEndpoint("http://127.0.0.1:1/"),
		WithHibpHTTPClient(&http.Client{Timeout: 500 * time.Millisecond}),
		WithHibpMeter(provider.Meter("test")))
	if checker.IsBreached(context.Background(), "password") {
		t.Fatal("expected fail-open (false) on connection error")
	}
	if got := failureCount(t, reader, "connection_error"); got != 1 {
		t.Fatalf("connection_error failure metric=%d, want 1", got)
	}
}

func TestHibpCachesPrefixLookups(t *testing.T) {
	const candidate = "password"
	_, suffix := sha1PrefixSuffix(candidate)
	var requests atomic.Int64

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		fmt.Fprintf(w, "%s:42\r\n", suffix)
	}))
	defer srv.Close()

	checker := NewHibpBreachedPasswordChecker("test",
		WithHibpEndpoint(srv.URL+"/"), WithHibpHTTPClient(srv.Client()))
	for range 3 {
		if !checker.IsBreached(context.Background(), candidate) {
			t.Fatal("expected breached=true")
		}
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("HTTP requests=%d, want 1 (subsequent lookups must hit cache)", got)
	}
}

func TestPrefixCacheEvictsByTTL(t *testing.T) {
	now := time.Unix(0, 0)
	cache := newPrefixCache(time.Minute, 8, func() time.Time { return now })
	cache.put("ABCDE", map[string]struct{}{"X": {}})

	if _, ok := cache.get("ABCDE"); !ok {
		t.Fatal("entry should be present before TTL")
	}
	now = now.Add(2 * time.Minute)
	if _, ok := cache.get("ABCDE"); ok {
		t.Fatal("entry should be evicted after TTL")
	}
}

func failureCount(t *testing.T, reader sdkmetric.Reader, reason string) int64 {
	t.Helper()
	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatalf("collect metrics: %v", err)
	}
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != hibpFailuresMetricName {
				continue
			}
			sum, ok := m.Data.(metricdata.Sum[int64])
			if !ok {
				continue
			}
			for _, dp := range sum.DataPoints {
				if v, ok := dp.Attributes.Value(attribute.Key("reason")); ok && v.AsString() == reason {
					return dp.Value
				}
			}
		}
	}
	return 0
}
