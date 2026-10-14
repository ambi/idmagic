---
status: completed
archived_from: 18a20e15a54838c336457fc95eaa5fcd96d7aa6e
---

# inbound SCIM collection 検索を filter と offset pagination の契約どおりに実装する

RFC 7644 §3.4.2.2 filter grammar の recursive-descent parser/evaluator (`backend/scim/domain/filter.go`) を
属性・演算子 allowlist 付きで追加し、User (`userName`, `active`, `name.*`, `emails.value`, `id`) と
Group (`displayName`, `id`) それぞれの許可範囲に閉じた。`and`/`or`/`not`/丸括弧、`eq`/`ne`/`co`/`sw`/`ew`/`pr`、
JSON 相当の文字列 escape、入力長・ネスト深さの資源上限をサポートする。`backend/scim/domain/query.go` に
RFC 7644 §3.4.2.4 の 1-origin `startIndex` 正規化と `count`(省略時既定値・0・clamp・負値エラー)を実装し、
`ListScimUsers`/`ListScimGroups` の usecase (`usecases.ListQuery`/`ListResult`)・HTTP adapter で
filter 適用後の `totalResults`/`Resources`/`itemsPerPage` を返すよう usecases.go を再実装した。
malformed/unsupported filter は `scimType: invalidFilter`、不正 pagination は `invalidValue` の
HTTP 400 SCIM Error にマップする。ServiceProviderConfig の `filter.maxResults` を実装済み上限
(`domain.MaxResults = 100`) と一致させた。application-side evaluation により memory/PostgreSQL
両 adapter で同一の検索意味を保つ (tenant-scoped repository の `FindAll`/`ListByTenant` はそのまま)。
`usecases.go` の複雑度予算超過を `usecases.go`/`users.go`/`groups.go`/`list.go` への関心事分割で解消した。

全文は `git show 18a20e15a54838c336457fc95eaa5fcd96d7aa6e:<このファイルのパス>` で読む。
