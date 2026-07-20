package db_valkey

import (
	"context"
	"strings"
	"time"

	"github.com/ambi/idmagic/backend/shared/resilience"

	goredis "github.com/redis/go-redis/v9"
)

// ValkeyConfig は Valkey 接続とレジリエンスの設定を集約する。
type ValkeyConfig struct {
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	QueryTimeout time.Duration // Valkey 操作時のコンテキストタイムアウト
}

func Open(ctx context.Context, rawURL string, cfg ValkeyConfig, cb *resilience.CircuitBreaker) (*goredis.Client, error) {
	if after, ok := strings.CutPrefix(rawURL, "valkey://"); ok {
		rawURL = "redis://" + after
	}
	options, err := goredis.ParseURL(rawURL)
	if err != nil {
		return nil, err
	}

	options.DialTimeout = cfg.DialTimeout
	options.ReadTimeout = cfg.ReadTimeout
	options.WriteTimeout = cfg.WriteTimeout

	client := goredis.NewClient(options)

	// サーキットブレイカーとタイムアウトをフック
	if cb != nil {
		client.AddHook(&resilienceHook{
			cb:      cb,
			timeout: cfg.QueryTimeout,
		})
	}

	// Exponential Backoff を用いた初期接続（Ping）のリトライ
	err = resilience.RetryWithBackoff(ctx, func() error {
		pingCtx := ctx
		var cancel context.CancelFunc
		if cfg.DialTimeout > 0 {
			pingCtx, cancel = context.WithTimeout(ctx, cfg.DialTimeout)
			defer cancel()
		}
		return client.Ping(pingCtx).Err()
	})
	if err != nil {
		_ = client.Close()
		return nil, err
	}
	return client, nil
}

type resilienceHook struct {
	cb      *resilience.CircuitBreaker
	timeout time.Duration
}

func (h *resilienceHook) DialHook(next goredis.DialHook) goredis.DialHook {
	return next
}

func (h *resilienceHook) ProcessHook(next goredis.ProcessHook) goredis.ProcessHook {
	return func(ctx context.Context, cmd goredis.Cmder) error {
		qctx := ctx
		var cancel context.CancelFunc
		if h.timeout > 0 {
			qctx, cancel = context.WithTimeout(ctx, h.timeout)
			defer cancel()
		}

		if h.cb != nil {
			return h.cb.Execute(func() error { //nolint:contextcheck // CB state machine does not rely on request context
				return next(qctx, cmd)
			})
		}
		return next(qctx, cmd)
	}
}

func (h *resilienceHook) ProcessPipelineHook(next goredis.ProcessPipelineHook) goredis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []goredis.Cmder) error {
		qctx := ctx
		var cancel context.CancelFunc
		if h.timeout > 0 {
			qctx, cancel = context.WithTimeout(ctx, h.timeout)
			defer cancel()
		}

		if h.cb != nil {
			return h.cb.Execute(func() error { //nolint:contextcheck // CB state machine does not rely on request context
				return next(qctx, cmds)
			})
		}
		return next(qctx, cmds)
	}
}
