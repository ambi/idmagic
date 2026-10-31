package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/ambi/idmagic/backend/mcpgateway/internal/evaluation_http"
	"github.com/ambi/idmagic/backend/mcpgateway/internal/exchange_http"
	"github.com/ambi/idmagic/backend/mcpgateway/internal/proxy_http"
)

const shutdownTimeout = 10 * time.Second

// Gateway は組み立てたハンドラーである。
type Gateway struct {
	Config  Config
	Handler http.Handler
	Metrics http.Handler
}

// Load は設定から組み立てる。HTTP クライアントはプロキシの環境変数を読まず、評価のリダイレクトを追わない。
func Load(path string, audit io.Writer) (*Gateway, error) {
	config, err := LoadConfig(path)
	if err != nil {
		return nil, err
	}
	evaluationTransport := &http.Transport{TLSClientConfig: config.TLS, ForceAttemptHTTP2: true}
	upstreamTransport := &http.Transport{ForceAttemptHTTP2: true}
	metrics := proxy_http.NewMetrics()
	backchannel := &http.Client{
		Transport: evaluationTransport, Timeout: config.Timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	var downstream proxy_http.Exchanger
	if config.DownstreamResource != "" {
		downstream = exchange_http.Client{
			TokenURL: config.RealmURL + "/token", ClientID: config.ClientID, ClientSecret: config.ClientSecret,
			Resource: config.DownstreamResource, Key: config.DownstreamKey, HTTPClient: backchannel,
		}
	}
	handler, err := proxy_http.NewHandler(proxy_http.Options{
		Resource: config.Resource, Upstream: config.Upstream, EnvoyExtAuthz: config.EnvoyExtAuthz, RealmURL: config.RealmURL,
		AllowedOrigins: config.AllowedOrigins, MaxBodyBytes: config.MaxBodyBytes, Mapping: config.Mapping,
		Evaluator: evaluation_http.Client{
			EvaluateURL: config.RealmURL + "/resource-access/evaluate",
			ClientID:    config.ClientID, ClientSecret: config.ClientSecret,
			HTTPClient: backchannel,
		},
		DownstreamResource: config.DownstreamResource, Downstream: downstream,
		Audit:     slog.New(slog.NewJSONHandler(audit, nil)),
		Metrics:   metrics,
		Transport: upstreamTransport,
	})
	if err != nil {
		return nil, err
	}
	return &Gateway{Config: config, Handler: handler, Metrics: metrics.Handler()}, nil
}

// Main は引数の設定で待ち受け、ctx の終了で止まる。
func Main(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("idmagic-mcp-gateway", flag.ContinueOnError)
	flags.SetOutput(stderr)
	configPath := flags.String("config", "", "path to the YAML configuration file")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *configPath == "" {
		_, _ = fmt.Fprintln(stderr, "--config is required")
		return 2
	}
	gateway, err := Load(*configPath, stdout)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	// 応答には長く開く SSE があるので、書き込みの時間は制限しない。
	servers := []*http.Server{{
		Addr: gateway.Config.Listen, Handler: gateway.Handler,
		ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 30 * time.Second, IdleTimeout: 2 * time.Minute,
	}}
	if gateway.Config.MetricsListen != "" {
		mux := http.NewServeMux()
		mux.Handle("GET /metrics", gateway.Metrics)
		servers = append(servers, &http.Server{Addr: gateway.Config.MetricsListen, Handler: mux, ReadHeaderTimeout: 10 * time.Second})
	}
	failures := make(chan error, len(servers))
	for _, server := range servers {
		go func() { failures <- server.ListenAndServe() }()
	}
	code := 0
	select {
	case err := <-failures:
		if !errors.Is(err, http.ErrServerClosed) {
			_, _ = fmt.Fprintln(stderr, err)
			code = 1
		}
	case <-ctx.Done():
	}
	shutdown, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
	defer cancel()
	for _, server := range servers {
		_ = server.Shutdown(shutdown)
	}
	return code
}
