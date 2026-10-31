// Command idmagic-mcp-gateway は、既存の MCP サーバーの手前で、IdMagic の評価が許可した要求だけを転送する。
// IdMagic の bootstrap を使わず、設定ファイルだけを読む。
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/ambi/idmagic/backend/mcpgateway"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := mcpgateway.Main(ctx, os.Args[1:], os.Stdout, os.Stderr)
	stop()
	os.Exit(code)
}
