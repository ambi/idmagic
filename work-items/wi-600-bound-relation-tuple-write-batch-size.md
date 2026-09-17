---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-18
priority: p2
depends_on: [wi-585-rewrite-api-rules-as-complete-api-guidelines]
change_kind: feature
affected_spec:
  - { path: spec/contexts/authorization/main.tsp, symbol: IdMagic.Authorization.Operations.WriteRelationTuples }
---

# リレーションタプルの一括書き込みに要素数の上限を設ける

## Motivation

[API ガイドライン](../docs/design/application/api-guidelines.md)の「配列の要素数」は、複数件の要素を配列で受け付けるリクエストボディに、TypeSpec の `@maxItems` で上限を宣言すると定める。
`WriteRelationTuples` の `writes`、`deletes`、`delete_objects` は上限を宣言しておらず、1 回のリクエストで書き込める件数を API 契約から読み取れない。
件数を制約しているのはリクエストボディの 64 KiB の上限だけであり、タプルの長さによって許容件数が変わる。
全件を 1 トランザクションで適用するため、件数が多いとロックの保持時間も伸びる。

## Scope

- `writes`、`deletes`、`delete_objects` の要素数の上限を決め、TypeSpec と Go の domain に設ける。
- 上限の超過を 422 で拒否する。
- API ガイドラインの適用状況を改める。

## Out of Scope

- 上限を超える件数を分割して適用する仕組み。

## Design

上限は、3 つの配列の合計に対して設ける。
配列ごとに設けると、合計が上限の 3 倍になり、トランザクションの大きさを制約できない。
上限値は、OpenFGA の `Write` が 1 リクエストあたり 100 件を既定の上限とすることを参考に、実測したトランザクションの所要時間から決める。
TypeSpec の `@maxItems` は配列ごとの上限しか表せないため、合計の上限は `@doc` に記載し、各配列の `@maxItems` には合計と同じ値を宣言する。

## Plan

1. 件数ごとのトランザクションの所要時間を測定し、上限値を決める。
2. Go の domain に検証を設け、TypeSpec に宣言する。

## Tasks

- [ ] T001 [Decision] 上限値を決める。
- [ ] T002 [Spec] TypeSpec に宣言する。
- [ ] T003 [App] Go の domain で検証する。
- [ ] T004 [Docs] API ガイドラインの適用状況を改める。
- [ ] T005 [Verify] 変更を検証する。

## Verification

- `mise run check-api-compat`
- `mise run verify`

## Risk Notes

上限の新設は、既存のクライアントにとって破壊的変更である。未リリースのため互換処理は設けないが、`mise run check-api-compat` の結果を確認する。
