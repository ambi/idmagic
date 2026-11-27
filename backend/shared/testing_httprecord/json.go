package testing_httprecord

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
)

// CompareJSON は、受け取った JSON の本文が記録した本文と同じ値であることを確かめる。
//
// 欄の順序と空白は問わず、欄の過不足と値の違いを食い違いとする。CompareForm と違い記録にない欄も許さないのは、
// 製品が送る欄の誤り（欄の付け忘れ、余分な欄）を検出するためである。Replay の CheckBody に渡して使う。
func CompareJSON(recorded, got []byte) []string {
	recorded, got = bytes.TrimSpace(recorded), bytes.TrimSpace(got)
	switch {
	case len(recorded) == 0 && len(got) == 0:
		return nil
	case len(recorded) == 0:
		return []string{fmt.Sprintf("body = %s, recorded none", got)}
	case len(got) == 0:
		return []string{fmt.Sprintf("body is empty, recorded %s", recorded)}
	}
	var want, have any
	if err := json.Unmarshal(recorded, &want); err != nil {
		return []string{fmt.Sprintf("recorded body: %v", err)}
	}
	if err := json.Unmarshal(got, &have); err != nil {
		return []string{fmt.Sprintf("body is not JSON: %v", err)}
	}
	if !reflect.DeepEqual(want, have) {
		return []string{fmt.Sprintf("body = %s, recorded %s", got, recorded)}
	}
	return nil
}
