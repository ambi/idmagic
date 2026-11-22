// Package discovery_http は、管理者の操作で公開された MCP サーバーへ接続する外向きの HTTP を持つ
// (REQ-OAUTH2-095)。宛先の制限は safehttp に任せ、リダイレクトを追わず、本文の量を限る。
package discovery_http

import (
	"io"
	"net/http"
	"time"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/shared/security/safehttp"
)

// RequestTimeout は一回の要求の期限である。取得全体の期限は呼び出し元の context が決める。
const RequestTimeout = 10 * time.Second

// ErrResponseTooLarge は本文が呼び出し元の上限を超えたことを表す。接続の失敗とは区別する。
var ErrResponseTooLarge = oauthports.ErrMcpResponseTooLarge

var _ oauthports.McpServerFetcher = (*Fetcher)(nil)

// Fetcher は一つの要求を送り、状態コードと上限までの本文を返す。
type Fetcher struct {
	client *http.Client
}

// NewFetcher は client を複製し、リダイレクトを追わず、要求ごとの期限を付ける。nil なら本番の
// safehttp のクライアントを使う。テストだけが接続先を差し替える。
func NewFetcher(client *http.Client) *Fetcher {
	if client == nil {
		client = safehttp.NewClient(safehttp.Config{
			DialTimeout: 5 * time.Second, TLSTimeout: 5 * time.Second, RequestTimeout: RequestTimeout,
			ValidateURL: oauthdomain.ValidateMcpUpstreamURL,
		})
	}
	cloned := *client
	cloned.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	cloned.Timeout = RequestTimeout
	return &Fetcher{client: &cloned}
}

// Do は要求を送り、limit バイトを超える本文を ErrResponseTooLarge にする。接続の失敗はそのまま返す。
func (f *Fetcher) Do(request *http.Request, limit int64) (int, []byte, error) {
	response, err := f.client.Do(request)
	if err != nil {
		return 0, nil, err
	}
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil {
		return 0, nil, err
	}
	if int64(len(body)) > limit {
		return 0, nil, ErrResponseTooLarge
	}
	return response.StatusCode, body, nil
}
