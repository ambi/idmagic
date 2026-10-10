// Package entropy は、domain の計算が暗号論的に安全な乱数を受け取るための供給元を提供する。
//
// 供給元は io.Reader から作る構築関数を公開しない。本番は crypto/rand を包む Crypto だけを使い、
// math/rand のような予測できる乱数を誤って渡せないようにする。
package entropy

import (
	"bytes"
	"crypto/rand"
	"io"
	"math/big"
)

// Source は乱数の供給元である。ゼロ値は使えないので、Crypto か Fixed で作る。
type Source struct{ r io.Reader }

// Crypto は crypto/rand を読む供給元を返す。
func Crypto() Source { return Source{r: rand.Reader} }

// Fixed は与えたバイト列を先頭から返す供給元を返す。テストで生成する値を固定するために使う。
// バイト列を使い切った後の読み取りはエラーになる。
func Fixed(b []byte) Source { return Source{r: bytes.NewReader(bytes.Clone(b))} }

// Read は b を乱数で満たす。
func (s Source) Read(b []byte) error {
	_, err := io.ReadFull(s.r, b)
	return err
}

// Int は 0 以上 n 未満の一様な乱数を返す。上限以上の値を捨てて引き直すので、剰余による偏りがない。
func (s Source) Int(n int) (int, error) {
	v, err := rand.Int(s.r, big.NewInt(int64(n)))
	if err != nil {
		return 0, err
	}
	return int(v.Int64()), nil
}
