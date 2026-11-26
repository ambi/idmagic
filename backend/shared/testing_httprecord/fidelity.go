package testing_httprecord

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
)

// Fidelity は、模倣の応答を記録とどう比べるかである。
type Fidelity struct {
	// VolatileKeys は、相手が割り当てる、または相手ごとに違ってよい値を持つ本文の属性である。値は型だけを比べる。
	VolatileKeys map[string]bool
	// Headers は、記録が持つときに模倣も返すべき応答のヘッダーである。
	// Content-Type はメディア型を比べ、それ以外は有無だけを比べる。
	Headers []string
	// ExactHeaders は、記録が持つときに模倣も同じ値を返すべき応答のヘッダーである。
	ExactHeaders []string
	// Setup は、往復を送る前に、記録の前から相手が持っていた状態を模倣へ置く。nil なら何もしない。
	Setup func(ex Exchange)
	// Rewrite は、送る前に要求のパスと本文を書き換える。記録の識別子を模倣が割り当てたものへ読み替えるのに使う。
	Rewrite func(s string) string
	// Learn は、本文のある応答を受けるたびに、Decode で読んだ記録の本文と模倣の本文を受け取る。読み替えの対応を覚えるのに使う。
	Learn func(want, got any)
	// Decode は、応答の Content-Type と本文から、比べる値を作る。記録の本文と模倣の本文の両方に使う。
	// nil なら本文を JSON として読む。SSE のように JSON でない本文を、メッセージの並びとして比べるのに使う。
	Decode func(contentType string, body []byte) (any, error)
}

func (f Fidelity) decode(contentType string, body []byte) (any, error) {
	if f.Decode != nil {
		return f.Decode(contentType, body)
	}
	var value any
	err := json.Unmarshal(body, &value)
	return value, err
}

// Diverge は記録の往復を順に模倣へ送り、模倣の応答が記録と食い違う点を返す。
//
// 比べるのは状態コード、Headers と ExactHeaders のヘッダー、本文の構造と値である。
// 本文は記録の属性がすべて模倣の応答にあることを求め、模倣が多く返すことは許す。
func (f Fidelity) Diverge(twin http.Handler, rec Recording) []string {
	server := httptest.NewServer(twin)
	defer server.Close()

	rewrite := f.Rewrite
	if rewrite == nil {
		rewrite = func(s string) string { return s }
	}
	var diffs []string
	for _, ex := range rec.Exchanges {
		if f.Setup != nil {
			f.Setup(ex)
		}
		target := server.URL + rewrite(ex.Request.Path)
		if ex.Request.Query != "" {
			target += "?" + ex.Request.Query
		}
		reqBody := rewrite(string(BodyBytes(ex.Request.Body)))
		req, err := http.NewRequestWithContext(context.Background(), ex.Request.Method, target, bytes.NewReader([]byte(reqBody)))
		if err != nil {
			return append(diffs, fmt.Sprintf("%s: build request: %v", ex.Name, err))
		}
		for name, value := range ex.Request.Headers {
			req.Header.Set(name, value)
		}
		resp, err := server.Client().Do(req)
		if err != nil {
			return append(diffs, fmt.Sprintf("%s: send request: %v", ex.Name, err))
		}
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		if resp.StatusCode != ex.Response.Status {
			diffs = append(diffs, fmt.Sprintf("%s: status = %d, recorded %d", ex.Name, resp.StatusCode, ex.Response.Status))
		}
		diffs = append(diffs, f.compareHeaders(ex, resp.Header)...)
		recorded := BodyBytes(ex.Response.Body)
		if len(recorded) == 0 {
			if len(bytes.TrimSpace(body)) != 0 {
				diffs = append(diffs, fmt.Sprintf("%s: body = %s, recorded none", ex.Name, body))
			}
			continue
		}
		want, err := f.decode(ex.Response.Headers["Content-Type"], recorded)
		if err != nil {
			return append(diffs, fmt.Sprintf("%s: recorded body: %v", ex.Name, err))
		}
		got, err := f.decode(resp.Header.Get("Content-Type"), body)
		if err != nil {
			diffs = append(diffs, fmt.Sprintf("%s: body cannot be decoded: %v", ex.Name, err))
			continue
		}
		diffs = append(diffs, f.compareJSON(ex.Name, "", want, got)...)
		if f.Learn != nil {
			f.Learn(want, got)
		}
	}
	return diffs
}

func (f Fidelity) compareHeaders(ex Exchange, header http.Header) []string {
	var diffs []string
	for _, name := range f.Headers {
		want, recorded := ex.Response.Headers[name]
		if !recorded {
			continue
		}
		got := header.Get(name)
		switch {
		case got == "":
			diffs = append(diffs, fmt.Sprintf("%s: header %s is missing, recorded %q", ex.Name, name, want))
		case name == "Content-Type" && MediaType(got) != MediaType(want):
			diffs = append(diffs, fmt.Sprintf("%s: header Content-Type = %q, recorded %q", ex.Name, got, want))
		}
	}
	for _, name := range f.ExactHeaders {
		want, recorded := ex.Response.Headers[name]
		if recorded && header.Get(name) != want {
			diffs = append(diffs, fmt.Sprintf("%s: header %s = %q, recorded %q", ex.Name, name, header.Get(name), want))
		}
	}
	return diffs
}

// MediaType はメディア型の引数を除いた部分を返す。解釈できなければ値をそのまま返す。
func MediaType(v string) string {
	mt, _, err := mime.ParseMediaType(v)
	if err != nil {
		return v
	}
	return mt
}

// compareJSON は want のすべての属性が got にあり、型が一致し、揮発しない値が一致することを確かめる。
// 配列は、記録が要素を持つとき、先頭の要素どうしを比べる。
func (f Fidelity) compareJSON(exchange, path string, want, got any) []string {
	at := func() string {
		if path == "" {
			return "body"
		}
		return path
	}
	switch w := want.(type) {
	case map[string]any:
		g, ok := got.(map[string]any)
		if !ok {
			return []string{fmt.Sprintf("%s: %s is %T, recorded object", exchange, at(), got)}
		}
		keys := make([]string, 0, len(w))
		for k := range w {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var diffs []string
		for _, k := range keys {
			gv, present := g[k]
			if !present {
				diffs = append(diffs, fmt.Sprintf("%s: %s.%s is missing", exchange, at(), k))
				continue
			}
			child := at() + "." + k
			if f.VolatileKeys[k] {
				if typeName(gv) != typeName(w[k]) {
					diffs = append(diffs, fmt.Sprintf("%s: %s is %s, recorded %s", exchange, child, typeName(gv), typeName(w[k])))
				}
				continue
			}
			diffs = append(diffs, f.compareJSON(exchange, child, w[k], gv)...)
		}
		return diffs
	case []any:
		g, ok := got.([]any)
		if !ok {
			return []string{fmt.Sprintf("%s: %s is %T, recorded array", exchange, at(), got)}
		}
		if len(w) == 0 {
			return nil
		}
		if len(g) == 0 {
			return []string{fmt.Sprintf("%s: %s is empty, recorded %d element(s)", exchange, at(), len(w))}
		}
		if isScalarArray(w) {
			if !reflect.DeepEqual(w, g) {
				return []string{fmt.Sprintf("%s: %s = %v, recorded %v", exchange, at(), g, w)}
			}
			return nil
		}
		return f.compareJSON(exchange, at()+"[0]", w[0], g[0])
	default:
		if !reflect.DeepEqual(want, got) {
			return []string{fmt.Sprintf("%s: %s = %v, recorded %v", exchange, at(), got, want)}
		}
		return nil
	}
}

func isScalarArray(values []any) bool {
	for _, v := range values {
		switch v.(type) {
		case map[string]any, []any:
			return false
		}
	}
	return true
}

func typeName(v any) string {
	switch v.(type) {
	case map[string]any:
		return "object"
	case []any:
		return "array"
	case string:
		return "string"
	case float64:
		return "number"
	case bool:
		return "boolean"
	case nil:
		return "null"
	default:
		return fmt.Sprintf("%T", v)
	}
}
