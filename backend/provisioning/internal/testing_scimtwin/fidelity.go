package testing_scimtwin

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
	"strings"
)

// volatileKeys は相手が割り当てる、または相手ごとに違ってよい値を持つ属性である。
// 模倣との照合では、これらの値は型だけを比べる。
var volatileKeys = map[string]bool{
	"id": true, "value": true, "$ref": true, "location": true, "version": true,
	"created": true, "lastModified": true, "documentationUri": true, "specUri": true,
	"description": true, "maxOperations": true, "maxPayloadSize": true, "maxResults": true,
	"totalResults": true, "detail": true,
}

// comparedHeaders は記録が持つときに模倣も返すべき応答のヘッダーである。
var comparedHeaders = []string{"Content-Type", "ETag", "Location"}

// Diverge は記録の往復を順に模倣へ送り、模倣の応答が記録と食い違う点を返す。
//
// 記録の識別子は模倣が割り当てたものへ読み替える。作成の応答の id で対応を学び、
// 後の要求のパスと本文の中の同じ文字列を置き換える。
// 比べるのは状態コード、記録が持つヘッダーの有無と Content-Type、本文の構造と値である。
// 本文は記録の属性がすべて模倣の応答にあることを求め、模倣が多く返すことは許す。
func Diverge(tw *Twin, rec Recording) []string {
	server := httptest.NewServer(tw)
	defer server.Close()

	ids := map[string]string{}
	rename := func(s string) string {
		for recorded, actual := range ids {
			s = strings.ReplaceAll(s, recorded, actual)
		}
		return s
	}
	var diffs []string
	for _, ex := range rec.Exchanges {
		for _, res := range ex.Setup {
			kind := "Users"
			if res.Kind == "group" {
				kind = "Groups"
			}
			tw.Put(kind, res.ID, map[string]any{"displayName": res.DisplayName})
		}
		target := server.URL + rename(ex.Request.Path)
		if ex.Request.Query != "" {
			target += "?" + ex.Request.Query
		}
		req, err := http.NewRequestWithContext(context.Background(), ex.Request.Method, target, bytes.NewReader([]byte(rename(string(ex.Request.Body)))))
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
		for _, name := range comparedHeaders {
			want, recorded := ex.Response.Headers[name]
			if !recorded {
				continue
			}
			got := resp.Header.Get(name)
			switch {
			case got == "":
				diffs = append(diffs, fmt.Sprintf("%s: header %s is missing, recorded %q", ex.Name, name, want))
			case name == "Content-Type" && mediaType(got) != mediaType(want):
				diffs = append(diffs, fmt.Sprintf("%s: header Content-Type = %q, recorded %q", ex.Name, got, want))
			}
		}
		if len(ex.Response.Body) == 0 {
			if len(bytes.TrimSpace(body)) != 0 {
				diffs = append(diffs, fmt.Sprintf("%s: body = %s, recorded none", ex.Name, body))
			}
			continue
		}
		var want, got any
		if err := json.Unmarshal(ex.Response.Body, &want); err != nil {
			return append(diffs, fmt.Sprintf("%s: recorded body: %v", ex.Name, err))
		}
		if err := json.Unmarshal(body, &got); err != nil {
			diffs = append(diffs, fmt.Sprintf("%s: body is not JSON: %v", ex.Name, err))
			continue
		}
		diffs = append(diffs, compareJSON(ex.Name, "", want, got)...)
		learnID(ids, want, got)
	}
	return diffs
}

func mediaType(v string) string {
	mt, _, err := mime.ParseMediaType(v)
	if err != nil {
		return v
	}
	return mt
}

// learnID は作成の応答から、記録の id と模倣の id の対応を覚える。
func learnID(ids map[string]string, want, got any) {
	w, ok1 := want.(map[string]any)
	g, ok2 := got.(map[string]any)
	if !ok1 || !ok2 {
		return
	}
	wid, ok1 := w["id"].(string)
	gid, ok2 := g["id"].(string)
	if ok1 && ok2 && wid != gid {
		ids[wid] = gid
	}
}

// compareJSON は want のすべての属性が got にあり、型が一致し、揮発しない値が一致することを確かめる。
// 配列は、記録が要素を持つとき、先頭の要素どうしを比べる。
func compareJSON(exchange, path string, want, got any) []string {
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
			if volatileKeys[k] {
				if typeName(gv) != typeName(w[k]) {
					diffs = append(diffs, fmt.Sprintf("%s: %s is %s, recorded %s", exchange, child, typeName(gv), typeName(w[k])))
				}
				continue
			}
			diffs = append(diffs, compareJSON(exchange, child, w[k], gv)...)
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
			if !jsonEqual(w, g) {
				return []string{fmt.Sprintf("%s: %s = %v, recorded %v", exchange, at(), g, w)}
			}
			return nil
		}
		return compareJSON(exchange, at()+"[0]", w[0], g[0])
	default:
		if !jsonEqual(want, got) {
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

func jsonEqual(a, b any) bool {
	return reflect.DeepEqual(a, b)
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
