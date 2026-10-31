package cli

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

func freeAddress(t *testing.T) string {
	t.Helper()
	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = listener.Close() }()
	return listener.Addr().String()
}

//spec:covers REQ-MCPGATEWAY-001: 設定の誤りは 0 以外で終了し、理由を標準エラーへ出す
func TestMainRefusesToStartOnABadConfig(t *testing.T) {
	var stderr bytes.Buffer
	if code := Main(t.Context(), nil, &bytes.Buffer{}, &stderr); code != 2 || !strings.Contains(stderr.String(), "--config") {
		t.Fatalf("no flag: code=%d stderr=%s", code, stderr.String())
	}
	stderr.Reset()
	path := writeConfig(t, validConfig+"surprise: true\n")
	if code := Main(t.Context(), []string{"--config", path}, &bytes.Buffer{}, &stderr); code != 1 || stderr.Len() == 0 {
		t.Fatalf("bad config: code=%d stderr=%s", code, stderr.String())
	}
}

// 正式な入口が待ち受け、MCP の入口と指標を別のアドレスで公開し、ctx の終了で止まる。
//
//spec:covers REQ-MCPGATEWAY-005, REQ-MCPGATEWAY-006: 待ち受けの配線、メタデータ、指標の公開
func TestMainServesAndStops(t *testing.T) {
	listen, metrics := freeAddress(t), freeAddress(t)
	content := strings.ReplaceAll(validConfig, "127.0.0.1:8080", listen) + fmt.Sprintf("metrics_listen: %s\n", metrics)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan int, 1)
	go func() {
		done <- Main(ctx, []string{"--config", writeConfig(t, content)}, &bytes.Buffer{}, &bytes.Buffer{})
	}()

	get := func(url string) int {
		for range 100 {
			request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
			if err != nil {
				t.Fatal(err)
			}
			response, err := http.DefaultClient.Do(request)
			if err == nil {
				_ = response.Body.Close()
				return response.StatusCode
			}
			time.Sleep(20 * time.Millisecond)
		}
		t.Fatalf("%s did not answer", url)
		return 0
	}
	if status := get("http://" + listen + "/.well-known/oauth-protected-resource/mcp"); status != http.StatusOK {
		t.Fatalf("metadata status = %d", status)
	}
	if status := get("http://" + metrics + "/metrics"); status != http.StatusOK {
		t.Fatalf("metrics status = %d", status)
	}
	cancel()
	select {
	case code := <-done:
		if code != 0 {
			t.Fatalf("code = %d", code)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("did not stop")
	}
}
