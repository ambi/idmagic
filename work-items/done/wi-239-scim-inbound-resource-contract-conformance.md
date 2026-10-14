---
status: completed
archived_from: 18a20e15a54838c336457fc95eaa5fcd96d7aa6e
---

# inbound SCIM resource と mutation の公開契約を RFC 7643/7644 に整合させる

`spec/contexts/scim.yaml` の `RFC7643-CORE-RESOURCES`/`RFC7644-PATCH` を `adoption: partial` に変更し、
対応する User/Group 属性 subset を `reason` に明記した。`backend/scim/domain/mutation.go` に
`ParseUserWrite`/`ParseUserPatchOps`/`ParseGroupWrite`/`ParseGroupPatchOps`(RFC7644-PATCH の
attribute allowlist・readOnly 検証・op 検証)を、`discovery.go` に `UserCoreSchema`/`GroupCoreSchema`
(静的 schema メタデータ)を実装した。`backend/scim/usecases/users.go`/`groups.go` を
validate-first(検証を全て終えてから永続化)+ 補償クリーンアップ(ADR-122)で書き換え、
server-assigned `id`(client 指定は無視)、PUT 完全置換(省略属性は既定値にリセット)、
PATCH の path/op/value 検証、userName/displayName の 409 uniqueness、member 解決失敗の
400 invalidValue、`meta.location` を含む完全な `meta` を実装した。`/Schemas` は実際の
User/Group 属性メタデータを返すようにした。ADR-122 で「cross-context の真の DB transaction は
作らず、validate-first + 補償クリーンアップとする」ことを決定として記録した。

全文は `git show 18a20e15a54838c336457fc95eaa5fcd96d7aa6e:<このファイルのパス>` で読む。
