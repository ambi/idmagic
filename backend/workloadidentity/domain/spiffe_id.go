package domain

import (
	"errors"
	"strings"
)

const (
	spiffeScheme      = "spiffe://"
	maxSpiffeIDLength = 2048
)

// ErrInvalidSpiffeID は、値が SPIFFE ID の構文を満たさないことを表す
// (docs/modules/workloadidentity/data.md の SPIFFE ID)。
var ErrInvalidSpiffeID = errors.New("workload identity: value is not a valid SPIFFE ID")

// SpiffeID は構文を確かめた SPIFFE ID。構文を満たさない値は正規化せずに拒否するため、
// 作れた値の文字列表現は入力と同じであり、比較はバイト列の完全一致で足りる。
type SpiffeID struct {
	trustDomain string
	path        string
}

// ParseSpiffeID は raw を SPIFFE ID として読む。スキームとトラストドメインの大文字、
// ポートや利用者情報、クエリ、フラグメント、空のセグメント、`.` と `..` のセグメント、
// 末尾の `/` を拒否する。
func ParseSpiffeID(raw string) (SpiffeID, error) {
	if len(raw) > maxSpiffeIDLength || !strings.HasPrefix(raw, spiffeScheme) {
		return SpiffeID{}, ErrInvalidSpiffeID
	}
	rest := raw[len(spiffeScheme):]
	trustDomain, path := rest, ""
	if i := strings.IndexByte(rest, '/'); i >= 0 {
		trustDomain, path = rest[:i], rest[i:]
	}
	if !ValidTrustDomain(trustDomain) || !validSpiffePath(path) {
		return SpiffeID{}, ErrInvalidSpiffeID
	}
	return SpiffeID{trustDomain: trustDomain, path: path}, nil
}

func (id SpiffeID) TrustDomain() string { return id.trustDomain }

func (id SpiffeID) String() string { return spiffeScheme + id.trustDomain + id.path }

// ValidTrustDomain は raw が SPIFFE のトラストドメインの構文 (小文字の英字、数字、
// `.`、`-`、`_` からなる 1 文字以上) を満たすかを返す。
func ValidTrustDomain(raw string) bool {
	if raw == "" {
		return false
	}
	for _, c := range []byte(raw) {
		if !isLowerAlnum(c) && c != '.' && c != '-' && c != '_' {
			return false
		}
	}
	return true
}

func validSpiffePath(path string) bool {
	if path == "" {
		return true
	}
	// path は '/' で始まる。先頭の '/' を除いて分けた各セグメントを確かめれば、
	// 空のセグメント ("//" と末尾の "/") も拒否できる。
	for segment := range strings.SplitSeq(path[1:], "/") {
		if segment == "" || segment == "." || segment == ".." {
			return false
		}
		for _, c := range []byte(segment) {
			if !isSpiffePathChar(c) {
				return false
			}
		}
	}
	return true
}

func isSpiffePathChar(c byte) bool {
	return isLowerAlnum(c) || ('A' <= c && c <= 'Z') || c == '.' || c == '-' || c == '_'
}

func isLowerAlnum(c byte) bool {
	return ('a' <= c && c <= 'z') || ('0' <= c && c <= '9')
}
