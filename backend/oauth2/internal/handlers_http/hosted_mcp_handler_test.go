package handlers_http

import (
	"bufio"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
)

//spec:covers REQ-OAUTH2-092: SSE の最初の断片を終了前に返し、切断を上流へ伝播し、本文の上限と不正な応答で転送を止める。
func TestHostedMcpStreamingAndOutboundLimits(t *testing.T) {
	t.Run("SSE flush and disconnect", func(t *testing.T) {
		canceled := make(chan struct{})
		e := echo.New()
		e.GET("/", func(c *echo.Context) error {
			reader, writer := io.Pipe()
			defer func() { _ = reader.Close() }()
			go func() {
				_, _ = io.WriteString(writer, "data: first\n\n")
				<-c.Request().Context().Done()
				_ = writer.CloseWithError(c.Request().Context().Err())
				close(canceled)
			}()
			return streamHostedMcp(c, &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: reader}, nil, 1024)
		})
		server := httptest.NewServer(e)
		defer server.Close()
		client := server.Client()
		client.Timeout = 2 * time.Second
		response, err := client.Get(server.URL)
		if err != nil {
			t.Fatal(err)
		}
		line, err := bufio.NewReader(response.Body).ReadString('\n')
		if err != nil || line != "data: first\n" {
			t.Fatalf("first event=%q, err=%v", line, err)
		}
		_ = response.Body.Close()
		select {
		case <-canceled:
		case <-time.After(2 * time.Second):
			t.Fatal("upstream did not observe client disconnection")
		}
	})
	t.Run("response limit aborts the stream", func(t *testing.T) {
		e := echo.New()
		e.GET("/", func(c *echo.Context) error {
			return streamHostedMcp(c, &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader("12345"))}, nil, 4)
		})
		server := httptest.NewServer(e)
		defer server.Close()
		response, err := server.Client().Get(server.URL)
		if err == nil {
			defer func() { _ = response.Body.Close() }()
			body, readErr := io.ReadAll(response.Body)
			if readErr == nil || len(body) > 4 {
				t.Fatalf("overflow returned success or excessive bytes: %q, err=%v", body, readErr)
			}
		}
	})
	t.Run("redirect and non-MCP response", func(t *testing.T) {
		for _, tc := range []struct {
			status  int
			content string
		}{{302, "application/json"}, {401, "application/json"}, {200, "text/html"}} {
			e := echo.New()
			e.GET("/", func(c *echo.Context) error {
				return streamHostedMcp(c, &http.Response{StatusCode: tc.status, Header: http.Header{"Content-Type": {tc.content}, "Location": {"https://other.example"}, "Set-Cookie": {"secret=s"}}, Body: io.NopCloser(strings.NewReader("secret"))}, nil, 100)
			})
			recorder := httptest.NewRecorder()
			e.ServeHTTP(recorder, httptest.NewRequest("GET", "/", http.NoBody))
			if recorder.Code != 502 || strings.Contains(recorder.Body.String(), "secret") || recorder.Header().Get("Location") != "" || recorder.Header().Get("Set-Cookie") != "" {
				t.Fatalf("unsafe response %d: %s %v", recorder.Code, recorder.Body.String(), recorder.Header())
			}
		}
	})
}

//spec:covers REQ-OAUTH2-092: 本番の生成器で private IP への TCP 接続を行わず、環境プロキシとリダイレクトを使わない。
func TestManagedMcpProductionHTTPClientRefusesPrivateNetworks(t *testing.T) {
	runtime := NewHostedMcpRuntime(nil)
	transport, ok := runtime.client.Transport.(*http.Transport)
	if !ok || transport.Proxy != nil {
		t.Fatal("gateway is not wired to a direct checked transport")
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = listener.Close() }()
	request, _ := http.NewRequest(http.MethodPost, "https://"+listener.Addr().String()+"/mcp", http.NoBody)
	response, err := runtime.client.Do(request)
	if response != nil {
		_ = response.Body.Close()
	}
	if err == nil {
		t.Fatal("private target was accepted")
	}
	tcpListener := listener.(*net.TCPListener)
	_ = tcpListener.SetDeadline(time.Now().Add(25 * time.Millisecond))
	connection, err := tcpListener.Accept()
	if connection != nil {
		_ = connection.Close()
		t.Fatal("private TCP connection reached the listener")
	}
	var timeout net.Error
	if !errors.As(err, &timeout) || !timeout.Timeout() {
		t.Fatalf("unexpected listener error: %v", err)
	}
	if err := runtime.client.CheckRedirect(request, nil); !errors.Is(err, http.ErrUseLastResponse) {
		t.Fatalf("redirect followed: %v", err)
	}
}

//spec:covers REQ-OAUTH2-092: テナントの予算は他のテナントへ影響せず、全体の上限は 503 となり、終了時に枠を返す。
func TestManagedMcpTransferBudgets(t *testing.T) {
	runtime := NewHostedMcpRuntime(nil)
	var releases []func()
	for tenant := range 8 {
		for range 8 {
			release, status := runtime.acquire(string(rune('a' + tenant)))
			if status != 0 {
				t.Fatalf("early refusal: %d", status)
			}
			releases = append(releases, release)
		}
		if tenant == 0 {
			if _, status := runtime.acquire("a"); status != 429 {
				t.Fatalf("tenant overflow=%d", status)
			}
		}
	}
	if _, status := runtime.acquire("new"); status != 503 {
		t.Fatalf("global overflow=%d", status)
	}
	for _, release := range releases {
		release()
	}
	release, status := runtime.acquire("new")
	if status != 0 {
		t.Fatalf("capacity was not returned: %d", status)
	}
	release()
}
