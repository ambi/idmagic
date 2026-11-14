---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-11-06
priority: p2
group: spec-conformance
depends_on: []
change_kind: refactor
spec_impact: { kind: none, reason: "全経路が既に反応器を通しているので、観測できる振る舞いは変わらない。組み立ての形だけを変える。" }
---

# User と Agent を止める操作が、失効の反応器を必須の依存として受け取る

## 動機

[wi-10874](../done/wi-10874-inventory-promises-kept-by-every-entry-path.md) の棚卸しで、User と Agent の停止に伴う失効エポックの前進と SET の投影が、呼び出し元の渡す `Emit` に依存していることが分かった。

| 経路 | 渡す `Emit` |
| --- | --- |
| 管理 API の User と Agent | `deps_http.Deps.ReactiveEmit` |
| ワークフロー、SCIM、Directory、保持期限バッチ | `bootstrap.Dependencies.ReactiveEmit` |
| HTTP サーバーの Authentication と OAuth2 のイベント | `server_http.userRevocationEmit` |
| 結合テストの組み立て | `testing_stack` が個別に反応器を呼ぶ |

現在は全経路が反応器を通すので、違反はない。
しかし `AdminUserDeps.Emit` と `AdminAgentDeps.Emit` は任意の `func(spec.DomainEvent) error` であり、`LegacyEmit` を渡しても型は通る。
[wi-85215](../done/wi-85215-propagate-agent-deletion-as-access-revocation.md) の削除の取りこぼしと同じく、新しい経路が監査だけの `Emit` を選べば、失効は黙って抜ける。
再発の余地である。

## 対象範囲

- User を止める操作と Agent を止める操作の依存に、失効の反応のポートを、監査の `Emit` とは別の必須の依存として持たせる。
- `ReactiveEmit` の三つの組み立てを、一つの組み立てに寄せる。

## 対象外

- 反応の内容と失敗の扱い（fail-closed）の変更。
- 失効エポックの保存と前進。既に SharedSignals の `internal` にあり、外から呼べない。

## 設計

### 所有の形

候補 A を採る。
IdManagement の停止の操作が、反応のポートを型として要求し、nil の組み立てを構築の時点で拒否する。
テストの組み立ては、何もしない反応器を明示して渡す。
SharedSignals は IdManagement に依存するので、ポートは IdManagement の公開の `ports` に置き、SharedSignals の反応器が構造的に満たす現在の向きを保つ。

境界の判断の記録は、着手時に[境界を選ぶ判断手順](../../docs/design/application/design-guidelines.md)で書く。

## タスク

- [ ] T001 [Refactor] 停止の操作の依存に反応のポートを足し、各組み立てを直す。
- [ ] T002 [Refactor] `ReactiveEmit` の重複を一つにする。

## 検証

- `mise run test-go-package -- ./backend/idmanagement/...`
- `mise run test-go-package -- ./backend/sharedsignals/...`
- `mise run check-boundaries`

## リスク

- テストの組み立てで、反応器を渡していない箇所が多い。何もしない反応器を渡す修正の範囲を着手時に数える。
