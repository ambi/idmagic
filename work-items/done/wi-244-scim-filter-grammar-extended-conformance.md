---
status: completed
archived_from: 18a20e15a54838c336457fc95eaa5fcd96d7aa6e
---

# SCIM inbound filter grammar の対応範囲を広げ、adoption の隙間を縮める

`spec/contexts/scim.yaml` に `RFC7644-FILTERING` requirement (`adoption: partial`) を追加し、
`ListScimUsers`/`ListScimGroups` の description を dateTime 比較・URN プレフィックス対応まで
更新した。`backend/scim/domain/filter.go` に `AttrDateTime` kind と `dateTimeAttr` を追加し、
`meta.created`/`meta.lastModified` への `gt`/`ge`/`lt`/`le`/`eq`/`ne` を RFC3339 実時刻
(`time.Parse` した `time.Time` の `Before`/`After`/`Equal`)で比較するようにした(文字列辞書順では
ないため、offset 表記が異なる同一時刻も正しく等価判定される)。`stripSchemaURNPrefix` を追加し、
`urn:ietf:params:scim:schemas:core:2.0:User:`/`...:Group:` プレフィックス付き属性名を、剥がした
後に既存の allowlist で解決するようにした(新しい検証経路を作らず、prefix なしと同じ allowlist を
通す)。未知の URN プレフィックスは `invalidFilter` にする。`userFilterAttrs`/`groupFilterAttrs`
(`backend/scim/usecases/users.go`/`groups.go`)に `meta.created`/`meta.lastmodified` を追加した。
範囲選択の理由は [[ADR-123-scim-filter-datetime-and-urn-prefix-scope]] に記録した。

全文は `git show 18a20e15a54838c336457fc95eaa5fcd96d7aa6e:<このファイルのパス>` で読む。
