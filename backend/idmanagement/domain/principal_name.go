package domain

import (
	"strings"

	"golang.org/x/text/cases"
)

// NameKey は User のユーザー名、Group と Agent の名前を比べるための比較キーを返す。
// 前後の空白を除いて Unicode の case folding をかけるので、大文字と小文字だけが異なる名前は
// 同じキーになる。アプリケーションの判定とデータベースの一意索引がこのキーだけを使うので、
// 比較を変えるときはここだけを変える。strings.EqualFold、strings.ToLower、PostgreSQL の
// lower() は、`ß` と `SS` のような文字やデータベースのロケールで結果が食い違う。
func NameKey(name string) string {
	// cases.Caser は状態を持ち、ゴルーチンの間で共有できないので呼び出しごとに作る。
	return cases.Fold().String(strings.TrimSpace(name))
}

// EmailKey は User のメールアドレスを比べるための比較キーを返す。名前と同じ比較をする。
func EmailKey(email string) string {
	return NameKey(email)
}
