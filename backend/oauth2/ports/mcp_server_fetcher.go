package ports

import (
	"errors"
	"net/http"
)

// ErrMcpResponseTooLarge は MCP サーバーの応答の本文が上限を超えたことを表す。接続の失敗と区別する。
var ErrMcpResponseTooLarge = errors.New("MCP server response exceeds the limit")

// McpServerFetcher は公開された MCP サーバーへ一つの要求を送る (REQ-OAUTH2-095)。宛先の制限と
// リダイレクトを追わないことは実装が持つ。limit を超える本文は ErrMcpResponseTooLarge を返す。
type McpServerFetcher interface {
	Do(request *http.Request, limit int64) (status int, body []byte, err error)
}
