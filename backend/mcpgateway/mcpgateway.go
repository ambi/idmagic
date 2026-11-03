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

// ToolReport は報告したツールの件数と、上流の一覧をすべて取得できたかである。
type ToolReport struct {
	Tools    int
	Complete bool
}

// ReportTools は上流のツールの一覧を IdMagic へ報告する。カタログを参照する proxy の構成でなければ何もしない。
// Main は起動後に一度呼ぶ。組み込む側は待ち受けの準備ができた後に呼ぶ。
func (s *Server) ReportTools(ctx context.Context) (ToolReport, error) {
	result, err := s.gateway.ReportTools(ctx)
	return ToolReport{Tools: result.Tools, Complete: result.Complete}, err
}

// Main は引数の設定で待ち受け、終了コードを返す。
func Main(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	return cli.Main(ctx, args, stdout, stderr)
}
