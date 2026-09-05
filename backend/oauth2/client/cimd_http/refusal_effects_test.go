package cimd_http

// クライアントメタデータ取得の SSRF 防護が、拒否を返すだけでなく
// 対象 IP へ接続していないこと、文書を取り込んでいないことまで確かめる。
//
// この防護は「拒否したかどうか」を応答から読めない。取得は失敗しても
// 「未知の client_id」へ畳まれるので、応答だけを読むテストは、内部ネットワークへ
// 接続してから結果を捨てる実装と、そもそも接続しない実装を区別できない。

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	oauth2memory "github.com/ambi/idmagic/backend/oauth2/client/db_memory"
	clientdomain "github.com/ambi/idmagic/backend/oauth2/client/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// countingListener は受理した接続を数える。SSRF 防護が効いていれば 0 のままである。
type countingListener struct {
	net.Listener
	accepted atomic.Int64
}

func (l *countingListener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err == nil {
		l.accepted.Add(1)
	}
	return conn, err
}

// EX-OAUTH2-017-02: プライベート、ループバック、リンクローカル、CGNAT へ解決される
// メタデータ URL はフェイルクローズで拒否され、当該 IP へ接続もしない。
func TestClientMetadataResolutionDoesNotReachNonPublicAddresses(t *testing.T) {
	// ループバックだけは実際に待ち受けを立てられるので、「接続が試みられていない」を
	// 受理数で直接読む。他の範囲は環境に束縛されないよう IP リテラルで指定する。
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	counting := &countingListener{Listener: listener}
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"client_id":"x","client_name":"x","redirect_uris":["https://x.example/cb"]}`))
	}))
	server.Listener = counting
	server.StartTLS()
	t.Cleanup(server.Close)
	loopbackURL := "https://" + server.Listener.Addr().String() + "/client.json"
	// StartTLS 自体は接続を受理しないが、待ち受け開始までの内部通信を数えないよう
	// ここで基準値を取る。
	baseline := counting.accepted.Load()

	var emitted []spec.DomainEvent
	repository := &ClientRepositoryWithCIMD{
		// 登録簿は空にする。CIMD の解決は登録簿を外したときだけ走る。
		OAuth2ClientRepository: oauth2memory.NewClientRepository(),
		Fetcher:                NewFetcher(),
		Emit:                   func(event spec.DomainEvent) { emitted = append(emitted, event) },
	}

	targets := map[string]string{
		"loopback":   loopbackURL,
		"private":    "https://10.0.0.1/client.json",
		"link-local": "https://169.254.0.1/client.json",
		"cgnat":      "https://100.64.0.1/client.json",
	}
	for name, clientIDURL := range targets {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
			defer cancel()
			resolved, err := repository.FindByID(ctx, "default", clientIDURL)
			if err != nil {
				t.Fatalf("解決は「未知の client_id」へ畳まれるべきで、err ではない: %v", err)
			}
			if resolved != nil {
				t.Fatalf("非公開 IP のメタデータからクライアントが取り込まれた: %#v", resolved)
			}
		})
	}

	if got := counting.accepted.Load() - baseline; got != 0 {
		t.Fatalf("ループバックの待ち受けが接続を %d 件受理した。SSRF 防護は接続前に閉じなければならない", got)
	}

	// 拒否は 4 件とも監査に残る。応答が「未知の client_id」に畳まれる以上、
	// 拒否が起きたことはこの事象でしか読めない。
	rejected := 0
	for _, event := range emitted {
		switch event.(type) {
		case *clientdomain.ClientIdMetadataDocumentRejected:
			rejected++
		case *clientdomain.ClientIdMetadataDocumentResolved:
			t.Fatalf("非公開 IP のメタデータが解決済みとして記録された: %#v", event)
		}
	}
	if rejected != len(targets) {
		t.Fatalf("拒否の記録が %d 件、期待は %d 件", rejected, len(targets))
	}

	// 対照: 同じ取得処理は、公開 IP へ解決されるホストでは実際に接続を試みる。
	// これが無いと「そもそも取得していないだけ」と区別できない。
	fetcher := NewFetcher()
	_, err = fetcher.Fetch(t.Context(), "https://cimd-should-not-resolve.invalid/client.json")
	if err == nil || strings.Contains(err.Error(), "non-public") {
		t.Fatalf("前提が壊れている: 公開ホスト向けの取得が非公開判定で止まった: %v", err)
	}
}
