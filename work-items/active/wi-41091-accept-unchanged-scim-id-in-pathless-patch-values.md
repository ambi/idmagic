---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-11-28
priority: p2
group: scim-provisioning
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/sourcing/standards.md, requirement: RFC7644-PATCH-PATH-OPTIONAL }
  - { path: spec/modules/sourcing/main.tsp, symbol: IdMagic.Contract.PatchScimGroup }
  - { path: spec/modules/sourcing/main.tsp, symbol: IdMagic.Contract.PatchScimUser }
---

# SCIM の path を省いた PATCH で、今と同じ値の id を含む value を受け付ける

## 動機

Okta の SCIM 2.0 protocol reference（Last updated on September 18, 2026。2026-11-28 に本文を読んだ）の「Update a specific group name > PATCH method request」では、Okta は Group の名前を次の形で変える。

```json
{ "op": "replace", "value": { "id": "abf4dd94-a4c0-4f67-89c9-76b03340cb9b", "displayName": "Test SCIMv2" } }
```

`value` の `id` は、PATCH の対象の Group の今の `id` と同じ値である。
IdMagic は path を省いた操作の `value` の各メンバーを path として扱い、readOnly の `id` を名指した時点で 400 の `mutability` で拒否する。
そのため Okta から Group の名前を変えると、必ず失敗する。
2026-11-28 に [[wi-21740-decide-recordings-for-requests-idmagic-receives]] で、この例を記録して再生して見つけた。

RFC 7644 §3.5.2 は、クライアントが readOnly の属性を変更することを禁じ、両立しない操作に適切な誤りを返すことを求める。
今と同じ値は変更ではないので、拒否は規範に求められてはいない。
一方、受け付けることも規範は明示していない。
したがって規範への違反ではなく、主要なクライアントとの相互運用の欠陥である。
グループの基準は p3 だが、Okta の新しい連携の既定の要求が通らないので p2 とする。

## 対象範囲

- User と Group の path を省いた `add` と `replace` で、`value` の `id` が対象の資源の今の `id` と一致するときは、その項目を変更なしとして無視し、残りの項目を適用する。
- 一致しない `id` と、`meta` と `schemas` の扱いは今のまま 400 の `mutability` で拒否する。
- `RFC7644-PATCH-PATH-OPTIONAL` の文面と、`PatchScimUser` と `PatchScimGroup` の説明に、この扱いを書く。
- [[wi-21740-decide-recordings-for-requests-idmagic-receives]] の `oktaNotReplayed` から `rename-group-patch` を消し、記録の往復を再生する。

## 対象外

- path に `id` を名指した操作。今と同じ値でも、これまでどおり拒否する。Okta の例はこの形を送らない。

## 設計

`id` の比較は大文字と小文字を区別する（RFC 7643 §3.1 の caseExact）。
比較には PATCH の対象の SCIM の id を使い、ドメインの層で `patchTargets` が値のオブジェクトを path へ展開するときに、展開の前に除く。

## 計画

1. `spec-change` で規範の文面と TypeSpec の説明を更新する。
2. 記録の再生で RED を観測し、実装を直す。

## タスク

- [ ] T001 [Spec] 規範の文面と TypeSpec の説明を更新する。
- [ ] T002 [Acceptance] `rename-group-patch` の再生で RED を確認する。
- [ ] T003 [App] 展開の前に一致する `id` を除き、一致しない `id` の拒否を検査で固定する。
- [ ] T004 [Verify] 差分から選んだ検査を実行する。

## 検証

- `mise run test-go-package -- ./backend/sourcing/internal/scim/handlers_http`、`mise run test-go-package -- ./backend/sourcing/internal/scim/domain`。

## リスク

一致の判定を誤ると、readOnly の `id` を書き換える要求を受け付ける。
一致しない `id` を拒否する検査を、受け付ける検査と対にして置く。
