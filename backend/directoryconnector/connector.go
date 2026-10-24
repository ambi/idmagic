// Package directoryconnector は、利用者の閉域で AD と OpenLDAP を読み、Sourcing の Connector API へ
// 差分を送る実行単位 idmagic-directory-connector の入口である。サーバーのモジュールを import しない。
package directoryconnector

import (
	"context"
	"io"

	"github.com/ambi/idmagic/backend/directoryconnector/internal/cli"
)

// Main はサブコマンドを実行し、終了コードを返す。
func Main(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	return cli.Main(ctx, args, stdin, stdout, stderr)
}
