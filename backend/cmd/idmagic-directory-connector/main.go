// Command idmagic-directory-connector は、利用者の閉域で AD と OpenLDAP を読み、IdMagic の Connector API へ
// 差分を送る。IdMagic の bootstrap を使わず、設定ファイルだけを読む。
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/ambi/idmagic/backend/directoryconnector"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := directoryconnector.Main(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	stop()
	os.Exit(code)
}
