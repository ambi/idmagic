// Package mcpgateway は、既存の MCP サーバーの手前で要求を認可する実行単位 idmagic-mcp-gateway の入口である。
// サーバーのモジュールを import しない。
package mcpgateway

import (
	"context"
	"io"
	"net/http"

	"github.com/ambi/idmagic/backend/mcpgateway/internal/cli"
)

// Server は設定から組み立てたゲートウェイである。
type Server struct{ gateway *cli.Gateway }

// Load は設定ファイルを読み、監査ログを audit へ書くゲートウェイを組み立てる。
func Load(path string, audit io.Writer) (*Server, error) {
	gateway, err := cli.Load(path, audit)
	if err != nil {
		return nil, err
	}
	return &Server{gateway: gateway}, nil
}

// Handler は MCP クライアントを受ける HTTP ハンドラーを返す。
func (s *Server) Handler() http.Handler { return s.gateway.Handler }

// Main は引数の設定で待ち受け、終了コードを返す。
func Main(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	return cli.Main(ctx, args, stdout, stderr)
}
