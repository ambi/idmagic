---
status: accepted
authors: [tn]
created_at: 2026-07-18
---

# ADR-122: SCIM mutation の atomicity は validate-first + 補償クリーンアップとし、cross-context DB transaction は作らない

## コンテキスト

wi-239 (inbound SCIM resource/mutation contract conformance) の `## Plan` は「validate →
aggregate 変換 → persistence → response の順にし、途中失敗で mapping や membership だけが
残らないことを transaction 境界で保証する」と書いていた。しかし実装前調査の結果、このリポジトリには
bounded context をまたぐ DB transaction の仕組みが存在しない。

- 既存の transaction パターン(`backend/audit/adapters/persistence/postgres/audit_events.go`、
  `backend/idgovernance/adapters/persistence/postgres/lifecycle_workflow_runs.go`)は、すべて
  単一 context 内で `pgx.Tx` を begin/commit する形で完結している。
- SCIM の mutation は `scim.ScimRepository`(`scim_user_refs`/`scim_group_refs` テーブル)と
  `idmanagement.UserRepository`/`GroupRepository`(実データと membership)という**2つの bounded
  context の repository**にまたがって書き込む。
- `idmports.UserRepository`/`GroupRepository` には tx-aware なメソッドが一切ない。これを追加すると
  インターフェース変更が認証・管理画面 API・lifecycle workflow など SCIM 以外の全呼び出し元に波及する。

wi-239 をそのまま「transaction 境界で保証する」の文字通りに実装しようとすると、SCIM の resource
contract 修正という当初のスコープを超えて、bounded context 横断の transaction 基盤という新しい
横断的アーキテクチャ変更が必要になる([ADR-121](decisions/ADR-121-scope-narrowing-disclosure-obligation.md)
が定める「work item の Motivation/Scope が示唆する範囲より狭い/広い範囲になる」設計判断に該当し、
ADR で記録すべき対象)。

## 決定

**cross-context の真の DB transaction は作らない。** 代わりに次の2段構えで failure モードを縮小する。

1. **validate-first**: persistence を一切呼び出す前に、body の構文・必須属性・型・PATCH の
   `op`/`path`/`value`・member 解決可能性をすべて検証し終える。これにより「validate 失敗で
   部分的に書き込まれる」経路をほぼゼロにする(多段階書き込みが始まった後に検証エラーで失敗する
   ケースを構造的に排除する)。
2. **補償クリーンアップ (best-effort compensation)**: validate-first を通過した後の persistence
   ステップ自体が失敗した場合(DB接続断など、通常は稀な運用障害)は、既に成功した先行ステップを
   打ち消す補償コードを実行してから 500 系の ScimProtocolError を返す(例: `GroupRepo.AddMember`
   成功後に後続の `ScimRepo.SaveUserRef` 相当の呼び出しが失敗したら `RemoveMember` で戻す)。
   補償自体が失敗した場合はログに残し、そのまま 500 を返す(二重障害は本 ADR の対象外とする)。

`spec/contexts/scim.yaml` の `RFC7644-RESOURCE-OPERATIONS`/`RFC7644-PATCH` はこの atomicity
方針を前提とする。real な DB transaction ではないため、稀な運用障害時に mapping/membership の
不整合が残る可能性はゼロではない。

## 却下した代替案

- **`idmports.UserRepository`/`GroupRepository` に tx-aware なメソッドを追加し、真の cross-context
  transaction を実現する**: 技術的には同一 Postgres pool を共有しているため可能だが、SCIM 以外の
  全呼び出し元に影響するインターフェース変更になり、wi-239 のスコープ(SCIM resource contract)を
  大きく超える。真に必要になった時点で独立した ADR/work item として再検討する。
- **saga / outbox パターンによる結果整合性**: 非同期処理基盤(`backend/jobs`)を要し、SCIM の
  同期的な request/response モデルと相性が悪い。今回の failure モードの発生頻度(validate-first
  後の persistence 障害は稀な運用障害のみ)に対して過剰。
- **何もしない(現状維持)**: 現状は検証と書き込みが混在しており、途中失敗で mapping だけが残る
  ケースが構造的に排除されていない。wi-239 の Motivation が指摘する「静かな失敗」をそのまま残すため
  却下。

## 影響

- `backend/scim/usecases/users.go`/`groups.go` の CreateUser/UpdateUser/PatchUser/CreateGroup/
  UpdateGroup/PatchGroup は、validate → persist の順序を明確に分離し、persist 中の失敗に対する
  補償コードを持つ。
- `idmports.UserRepository`/`GroupRepository` のインターフェースは変更しない。
- 真の cross-context transaction が必要になった場合(atomicity 要件が厳格化された場合)は、
  この ADR を supersede する新しい ADR で再設計する。
