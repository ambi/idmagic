// Package testing_mcptwin は、上流の MCP サーバーの代わりにテストが使う相手を提供する。
//
// 相手は二つある。MCP の仕様の版ごとの例と RFC 9728 の例を転記した記録をそのまま返す再生と、
// ツールの一覧、ページング、変更通知、版の照合、トークンの検査、Protected Resource Metadata を扱う状態を持つ模倣である。
// 記録は模倣の忠実さの基準でもあり、模倣が記録と同じ応答を返すことをこのパッケージのテストが確かめる。
// 記録の形式、再生、照合は testing_httprecord の部品を使う。
// どちらもテストの中だけで使い、製品の経路から import しない。
package testing_mcptwin

import (
	"bufio"
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/ambi/idmagic/backend/shared/testing_httprecord"
)

var (
	//go:embed testdata/mcp-2026-07-28.json
	mcp20260728Recording []byte
	//go:embed testdata/rfc9728.json
	rfc9728Recording []byte
)

// RecordedProtocolVersion は MCP の記録の版である。版を上げるときは別の記録を足し、古い記録を対応する間は残す。
const RecordedProtocolVersion = "2026-07-28"

// Recording は一つの相手との往復の列と、その出どころである。
type Recording = testing_httprecord.Recording

// Exchange は一つの要求と、相手が返した応答である。
type Exchange = testing_httprecord.Exchange

// MCP20260728Recording は MCP 2026-07-28 の仕様の例を転記した記録を返す。
//
// 仕様の Tools の節の変更通知の例は subscriptionId が "listen-1" だが、Subscriptions の節は値を
// subscriptions/listen の要求の id と定める。subscriptions-listen の往復は要求の id の 1 を使い、
// subscriptions-listen-other-subscription の往復が "listen-1" の例をそのまま持つ。
func MCP20260728Recording(tb testing.TB) Recording {
	tb.Helper()
	return decode(tb, mcp20260728Recording)
}

// RFC9728Recording は RFC 9728 §3.1 と §3.2 の例を転記した記録を返す。
func RFC9728Recording(tb testing.TB) Recording {
	tb.Helper()
	return decode(tb, rfc9728Recording)
}

func decode(tb testing.TB, data []byte) Recording {
	tb.Helper()
	rec, err := testing_httprecord.Decode(data)
	if err != nil {
		tb.Fatalf("decode recording: %v", err)
	}
	return rec
}

// NewReplayServer は、記録した往復を順に一度ずつ返すサーバーを起動する。
//
// 受け取った要求は、方式、PRM のパス、記録した要求が持つ Content-Type、Accept、MCP-Protocol-Version、
// Mcp-Method、本文の値で記録と照合し、食い違えばテストを失敗させる。MCP の入口のパスは登録や構成の URL が決め、
// プロトコルが決めないので比べない。本文は記録の値がすべてあることを求め、
// 仕様が SHOULD とする _meta の clientInfo だけは欠けてよい。
func NewReplayServer(tb testing.TB, exchanges ...Exchange) *httptest.Server {
	tb.Helper()
	return testing_httprecord.Replay{
		PathShape: endpointShape,
		Headers:   []string{"Content-Type", "Accept", "MCP-Protocol-Version", "Mcp-Method"},
		CheckBody: CompareRequestBody,
	}.Serve(tb, exchanges...)
}

// NewResponseReplayServer は、要求を方式だけで照合して記録の応答を返すサーバーを起動する。
// 記録の要求がクライアントの送らない値（未対応の版など）を持つ往復の応答を、クライアントに読ませるときに使う。
func NewResponseReplayServer(tb testing.TB, exchanges ...Exchange) *httptest.Server {
	tb.Helper()
	return testing_httprecord.Replay{PathShape: func(string) string { return "" }, Headers: []string{}}.Serve(tb, exchanges...)
}

func endpointShape(path string) string {
	if strings.HasPrefix(path, "/.well-known/") {
		return path
	}
	return "MCP endpoint"
}

// optionalRequestKeys は、記録の要求が持っていてもクライアントが送らなくてよい本文の属性である。
var optionalRequestKeys = map[string]bool{"io.modelcontextprotocol/clientInfo": true}

// CompareRequestBody は、記録した JSON の要求の値がすべて受け取った本文にあり、等しいことを確かめる。
// 記録の本文がなければ比べない。受け取った本文が多く持つことは許す。
func CompareRequestBody(recorded, got []byte) []string {
	if len(bytes.TrimSpace(recorded)) == 0 {
		return nil
	}
	var want, have any
	if err := json.Unmarshal(recorded, &want); err != nil {
		return []string{fmt.Sprintf("recorded body is not JSON: %v", err)}
	}
	if err := json.Unmarshal(got, &have); err != nil {
		return []string{fmt.Sprintf("body is not JSON: %v", err)}
	}
	return containsJSON("body", want, have)
}

func containsJSON(path string, want, got any) []string {
	w, ok := want.(map[string]any)
	if !ok {
		if !reflect.DeepEqual(want, got) {
			return []string{fmt.Sprintf("%s = %v, recorded %v", path, got, want)}
		}
		return nil
	}
	g, ok := got.(map[string]any)
	if !ok {
		return []string{fmt.Sprintf("%s is %T, recorded object", path, got)}
	}
	keys := make([]string, 0, len(w))
	for k := range w {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var diffs []string
	for _, k := range keys {
		gv, present := g[k]
		switch {
		case !present && optionalRequestKeys[k]:
		case !present:
			diffs = append(diffs, fmt.Sprintf("%s.%s is missing", path, k))
		default:
			diffs = append(diffs, containsJSON(path+"."+k, w[k], gv)...)
		}
	}
	return diffs
}

// Diverge は記録の往復を順に模倣へ送り、模倣の応答が記録と食い違う点を返す。
//
// 往復の setup は、送る前に模倣の設定を既定へ戻してから置く。setup のない往復は前の往復の状態を引き継ぐ。
// 状態コード、Content-Type のメディア型、WWW-Authenticate の有無、本文の構造と値を比べる。
// SSE の本文は data のメッセージの並びとして比べる。
func Diverge(tw *Twin, rec Recording) []string {
	var setupErrs []string
	diffs := testing_httprecord.Fidelity{
		Headers: []string{"Content-Type", "WWW-Authenticate"},
		Setup: func(ex Exchange) {
			if len(ex.Setup) == 0 {
				return
			}
			if err := tw.apply(ex.Setup); err != nil {
				setupErrs = append(setupErrs, fmt.Sprintf("%s: setup: %v", ex.Name, err))
			}
		},
		Decode: decodeBody,
	}.Diverge(tw, rec)
	return append(setupErrs, diffs...)
}

// decodeBody は SSE の本文を、data のメッセージを順に message1、message2 と名付けたオブジェクトとして読む。
// それ以外の本文は JSON として読む。
func decodeBody(contentType string, body []byte) (any, error) {
	if testing_httprecord.MediaType(contentType) != "text/event-stream" {
		var value any
		err := json.Unmarshal(body, &value)
		return value, err
	}
	messages, err := SSEMessages(body)
	if err != nil {
		return nil, err
	}
	stream := make(map[string]any, len(messages))
	for i, message := range messages {
		var value any
		if err := json.Unmarshal(message, &value); err != nil {
			return nil, fmt.Errorf("message %d: %w", i+1, err)
		}
		stream[fmt.Sprintf("message%d", i+1)] = value
	}
	return stream, nil
}

// SSEMessages は SSE の本文から、空行で区切った各イベントの data 行をつないだメッセージを順に返す。
// data 以外の欄とコメント行は読み飛ばす。
func SSEMessages(body []byte) ([][]byte, error) {
	var (
		messages [][]byte
		data     []string
	)
	flush := func() {
		if len(data) > 0 {
			messages = append(messages, []byte(strings.Join(data, "\n")))
			data = nil
		}
	}
	lines := bufio.NewScanner(bytes.NewReader(body))
	for lines.Scan() {
		line := strings.TrimSuffix(lines.Text(), "\r")
		if line == "" {
			flush()
			continue
		}
		if value, ok := strings.CutPrefix(line, "data:"); ok {
			data = append(data, strings.TrimPrefix(value, " "))
		}
	}
	flush()
	return messages, lines.Err()
}
