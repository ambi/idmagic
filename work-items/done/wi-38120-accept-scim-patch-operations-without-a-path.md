---
status: completed
---

# `path` を省いた SCIM の PATCH の `add` と `replace` を受け入れる

SCIM の User と Group の PATCH で、`path` を省いた `add` と `replace` を受け入れるようにした。
`value` のオブジェクトの各メンバーを、その名前を `path` にした操作へ展開し、`path` 付きの操作と同じ許可の一覧、読み取り専用の検査、値の検査へ通す。
Enterprise 拡張の URN をキーにしたオブジェクトは、URN で修飾した各属性の操作へ展開する。
`path` を省いた `remove` は `noTarget`、空でないオブジェクトでない `value` は `invalidValue` で拒否する。

`standards.md` に受け入れの標準要件 `RFC7644-PATCH-PATH-OPTIONAL` を追加し、REQ-SOURCING-004 に展開、名前の優先、拒否の五文を加えた。
デコードした JSON の `map` は通信上の順を保たないため、展開した操作を名前の順に並べる。
`displayName` と `name.formatted` は同じ正規の名前を書くので、作成と置換と同じく、null でない `name.formatted` を優先する。
`path` が空文字列の操作は省略と見なさず、従来どおり `invalidPath` で拒否する。
`path` の値フィルターは wi-248 が扱う。
