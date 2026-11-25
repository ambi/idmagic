package testing_scimtwin

import (
	"net/http/httptest"
	"testing"

	"github.com/ambi/idmagic/backend/shared/testing_httprecord"
)

// NewReplayServer は、記録した往復を順に一度ずつ返すサーバーを起動する。
//
// 受け取った要求は方式とパスの形、記録した要求が持つ Accept と Content-Type で記録と照合し、
// 食い違えばテストを失敗させる。識別子と本文は相手が割り当てたものや例の値なので比べない。
// クライアントが応答から取り出す値はテストが表明する。
func NewReplayServer(tb testing.TB, exchanges ...Exchange) *httptest.Server {
	tb.Helper()
	return testing_httprecord.Replay{PathShape: pathShape}.Serve(tb, exchanges...)
}
