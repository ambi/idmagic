---
status: completed
archived_from: 45110d5c42349972a617e4ddedfa62ed5fa5cb0e
---

# domain のパッケージが現在時刻、乱数、ネットワークを直接使わないようにする

`mise run spec-diff` は規範の差分なしを報告する。
`domain` の関数が `time.Now` と `crypto/rand` を直接使わず、現在時刻と乱数を引数で受け取る形にした。
乱数は共有ライブラリ `backend/shared/security/entropy` の `Source` で受け取る。本番は `crypto/rand` を包む `entropy.Crypto()` だけを使い、`io.Reader` から作る構築関数を置かないので `math/rand` は渡せない。
`domain-effect` の検査は、構文解析だけを行う `net/url`、`net/mail`、`net/netip` を作用とみなさないようにし、`backend.md` の規則を合わせた。
境界の負債の台帳から `domain-effect` の 20 件を消し、台帳は 157 件から 137 件になった。
生成する値の形式、HTTP の応答、発行するイベント、永続状態は変えていない。

全文は `git show 45110d5c42349972a617e4ddedfa62ed5fa5cb0e:<このファイルのパス>` で読む。
