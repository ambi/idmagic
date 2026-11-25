// Package testing_httprecord は、IdMagic が要求を送る外部の相手との往復の記録を、テストの相手として使う部品を提供する。
//
// 部品は三つある。出どころを必ず持つ記録の形式、記録した往復を順に返す再生サーバー、
// 状態を持つ模倣が記録と同じ応答を返すかを調べる照合である。
// 相手の種類（SCIM、OAuth 2.0 のトークンエンドポイントなど）に固有の模倣と記録の中身は、相手ごとのテスト支援が持つ。
// どの部品もテストの中だけで使い、製品の経路から import しない。
package testing_httprecord

import (
	"encoding/json"
	"fmt"
	"testing"
)

// Recording は一つの相手との往復の列と、その出どころである。
// 記録が古くなったときに相手が変わったのか記録が古いのかを区別できるよう、取得日、相手の版、取得方法を必ず持つ。
type Recording struct {
	Counterpart        string     `json:"counterpart"`
	CounterpartVersion string     `json:"counterpart_version"`
	RetrievedAt        string     `json:"retrieved_at"`
	Method             string     `json:"method"`
	Exchanges          []Exchange `json:"exchanges"`
}

// Exchange は一つの要求と、相手が返した応答である。
//
// Setup は記録の前から相手が持っていた状態であり、形は相手の種類ごとのテスト支援が決める。
// 記録は相手の状態を作る往復を含まないことがあるので、模倣との照合の前に同じ状態を置くために使う。
type Exchange struct {
	Name     string           `json:"name"`
	Source   string           `json:"source"`
	Setup    json.RawMessage  `json:"setup,omitempty"`
	Request  RecordedRequest  `json:"request"`
	Response RecordedResponse `json:"response"`
}

// RecordedRequest は相手が受け取った要求である。
// Body は JSON の値をそのまま持つ。JSON でない本文（フォームなど）は JSON の文字列として持つ。
type RecordedRequest struct {
	Method  string            `json:"method"`
	Path    string            `json:"path"`
	Query   string            `json:"query,omitempty"`
	Headers map[string]string `json:"headers"`
	Body    json.RawMessage   `json:"body,omitempty"`
}

// RecordedResponse は相手が返した応答である。Body の表し方は RecordedRequest と同じである。
type RecordedResponse struct {
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers"`
	Body    json.RawMessage   `json:"body,omitempty"`
}

// BodyBytes は記録した本文を、送受信されるバイト列へ戻す。
// JSON の文字列はその中身を、それ以外の JSON の値はそのままのバイト列を返す。
func BodyBytes(body json.RawMessage) []byte {
	if len(body) > 0 && body[0] == '"' {
		var text string
		if err := json.Unmarshal(body, &text); err == nil {
			return []byte(text)
		}
	}
	return body
}

// Decode は記録を読み、出どころの四つの欄が揃っていることを確かめる。
func Decode(data []byte) (Recording, error) {
	var rec Recording
	if err := json.Unmarshal(data, &rec); err != nil {
		return Recording{}, err
	}
	if rec.Counterpart == "" || rec.CounterpartVersion == "" || rec.RetrievedAt == "" || rec.Method == "" {
		return Recording{}, fmt.Errorf("recording must state its counterpart, version, retrieval date and method")
	}
	return rec, nil
}

// Exchange は名前で往復を返す。
func (r Recording) Exchange(tb testing.TB, name string) Exchange {
	tb.Helper()
	for _, ex := range r.Exchanges {
		if ex.Name == name {
			return ex
		}
	}
	tb.Fatalf("recording has no exchange %q", name)
	return Exchange{}
}

// Select は名前を挙げた往復だけを、挙げた順に持つ記録を返す。
// 一つの記録のうち、模倣の一つの設定が再現する往復だけを照合するときに使う。
func (r Recording) Select(tb testing.TB, names ...string) Recording {
	tb.Helper()
	selected := r
	selected.Exchanges = make([]Exchange, 0, len(names))
	for _, name := range names {
		selected.Exchanges = append(selected.Exchanges, r.Exchange(tb, name))
	}
	return selected
}
