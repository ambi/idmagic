---
status: pending
authors: [tn]
risk: medium
reversibility: irreversible
created_at: 2026-11-28
priority: p1
group: scim-provisioning
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/sourcing/standards.md, requirement: RFC7643-CORE-RESOURCES }
  - { path: spec/modules/sourcing/main.tsp, symbol: IdMagic.Contract.GetScimSchemas }
  - { path: spec/modules/sourcing/main.tsp, symbol: IdMagic.Contract.GetScimResourceTypes }
---

# SCIM の /Schemas と /ResourceTypes の一覧を ListResponse で返す

## 動機

RFC 7644 §4 は、`/Schemas` への GET が「all supported schemas in ListResponse format」を返すことと、複数の ResourceType または Schema を返すときに ListResponse の形を使うことを SHALL で求める（2026-11-28 に本文を読んだ）。
IdMagic の `GetScimSchemas` と `GetScimResourceTypes` は、TypeSpec も実装も、資源の素の JSON 配列を返す。
2026-11-28 に [[wi-21740-decide-recordings-for-requests-idmagic-receives]] で Microsoft Learn の Entra ID の SCIM の手引きの例を再生したところ、手引きの Discover schema の応答の例は ListResponse であり、IdMagic の応答は配列だった。
手引きは、Entra ID のプロビジョニングサービスが設定の保存時に `/schemas` を要求すると述べる。
グループの基準は p3 だが、これは任意の機能ではなく MUST に反する欠陥なので p1 とする。

## 対象範囲

- `GetScimSchemas` と `GetScimResourceTypes` の成功の本文を、`schemas` に `urn:ietf:params:scim:api:messages:2.0:ListResponse`、`totalResults`、`Resources`（と RFC 7644 §3.4.2 の `startIndex`、`itemsPerPage`）を持つ ListResponse にする。
- RFC 7644 §4 のとおり、照会の引数（filter、ソート、ページング）を無視する。
- [[wi-21740-decide-recordings-for-requests-idmagic-receives]] の `entraNotReplayed` から `discover-schema` を消し、記録の往復を再生する。
- 既存の `TestScimGetSchemasReturnsRealAttributes` と Discovery の契約テストを新しい形に合わせる。

## 対象外

- 個々の Schema と ResourceType を URI で読む経路（`/Schemas/{id}`）。今は提供しておらず、別の判断である。

## 設計

公開済みの HTTP 契約の形を変える。
公開状態と互換性の扱いは着手時に [公開状態](../../docs/development/release.md#公開状態と日常開発) で確かめる。
Provisioning（送る側）が相手の `/Schemas` を読むなら、その解析も ListResponse を前提にしているかを着手時に確かめる。

## 計画

1. `spec-change` で TypeSpec の二つの操作の本文の型を ListResponse にする。
2. 受け入れの検査（記録の再生と契約テスト）で RED を観測し、実装を直す。

## タスク

- [ ] T001 [Spec] TypeSpec を更新する。
- [ ] T002 [Acceptance] `discover-schema` の再生で RED を確認する。
- [ ] T003 [App] ハンドラーを直す。
- [ ] T004 [Verify] 差分から選んだ検査を実行する。

## 検証

- `mise run test-go-package -- ./backend/sourcing/internal/scim/handlers_http`、`mise run check-api-compat`。

## リスク

配列を前提に読んでいる既存の利用者がいれば壊れる。
RFC が求める形への是正であり、主要なクライアント（Entra ID、Okta）は ListResponse を前提にする。
