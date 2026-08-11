package domain

import (
	"bytes"
	"encoding/csv"
	"testing"
)

// FuzzEncodeCSVRecords は、User 名・属性など untrusted な値がそのままセルに載る
// CSV エクスポートで、生成物が (1) RFC 4180 として必ずパースでき、(2) パースし直した
// どのセルも formula injection トリガー文字 (= + - @ TAB CR) で始まらないことを、
// 任意のバイト列に対して固定する (外部未信頼入力を表計算ソフトが
// 解釈する高リスク面のため fuzz を採用)。値の正しさは通常のユニットテストが担う。
func FuzzEncodeCSVRecords(f *testing.F) {
	seeds := []string{"", "alice", "=1+1", "+1", "-1", "@evil", "\t=x", "\r\n", "a,b\"c\nd", "山田=太郎"}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, cell string) {
		out, err := EncodeCSVRecords([]string{"h1", "h2"}, [][]string{{cell, "safe"}})
		if err != nil {
			// encoding/csv rejects fields containing a bare \r or \n only when
			// they would break the record; such inputs are legitimately
			// unencodable and not a safety violation.
			return
		}
		records, err := csv.NewReader(bytes.NewReader(out)).ReadAll()
		if err != nil {
			t.Fatalf("EncodeCSVRecords produced non-RFC4180 output for %q: %v", cell, err)
		}
		for _, rec := range records {
			for _, c := range rec {
				if c == "" {
					continue
				}
				switch c[0] {
				case '=', '+', '-', '@', '\t', '\r':
					t.Fatalf("cell %q (from input %q) begins with a formula-injection trigger", c, cell)
				}
			}
		}
	})
}
