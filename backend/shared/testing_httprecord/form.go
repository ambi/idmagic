package testing_httprecord

import (
	"fmt"
	"net/url"
	"slices"
)

// CompareForm は、記録したフォームの欄がすべて、同じ値で受け取ったフォームにあることを確かめる。
//
// 記録にない欄（PKCE の code_verifier など）は許す。Replay の CheckBody に渡して、
// application/x-www-form-urlencoded の要求を記録と照合するのに使う。
func CompareForm(recorded, got []byte) []string {
	want, err := url.ParseQuery(string(recorded))
	if err != nil {
		return []string{fmt.Sprintf("recorded form: %v", err)}
	}
	have, err := url.ParseQuery(string(got))
	if err != nil {
		return []string{fmt.Sprintf("form is not urlencoded: %v", err)}
	}
	keys := make([]string, 0, len(want))
	for key := range want {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	var diffs []string
	for _, key := range keys {
		if !slices.Equal(have[key], want[key]) {
			diffs = append(diffs, fmt.Sprintf("%s = %q, recorded %q", key, have[key], want[key]))
		}
	}
	return diffs
}
