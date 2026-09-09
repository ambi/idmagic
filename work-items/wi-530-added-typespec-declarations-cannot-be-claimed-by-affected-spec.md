---
depends_on: []
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-10
priority: p2
change_kind: tooling
spec_impact: { kind: none, reason: "検査の内側の突き合わせ方だけを直す。標準の行も TypeSpec も製品の振る舞いも変えない。" }
---

# 追加した TypeSpec 宣言を `affected_spec` で名乗れないので、無関係な記録へ配られる

## Motivation

[[wi-523-introspection-token-type-is-not-the-rfc6749-token-type]] が TypeSpec に `enum` を 1 つ足そうとして見つけた。

`tools/check/src/documentation-impact.ts` の `ownsSpecificationAdditions` は、追加された規範要素の持ち主を「`affected_spec` でその要素を名乗った記録」と定める。名乗り手がいなければ、追加は開いているすべての記録へ配られる。これは意図された設計であり、doc コメントがそう書いている。

ところが TypeSpec の宣言については、**名乗れる文字列が存在しない。**

`tools/check/src/spec-diff.ts:157` は宣言を `<path>:<name>` の形で数える。`spec/contexts/oauth2/models.tsp:PresentationTokenType` のような文字列である。`ownsSpecificationAdditions` と `tools/workspace/src/check-workspace.ts:201` の `specificationAdditionsClaimed` は、`affected_spec` の `requirement` と `symbol` の値をこの文字列と完全一致で突き合わせる。

一方 `tools/check/src/work-item-references.ts:68` は、`affected_spec` の `symbol` を最後のドット区切りが宣言名であるものとして解決し、`\b(?:alias|enum|model|op|scalar|union)\s+<name>\b` が対象ファイルに現れることを求める。`spec/contexts/oauth2/models.tsp:PresentationTokenType` の最後のドット区切りは `tsp:PresentationTokenType` なので、この検査は通らない。

**2 つの検査を同時に満たす `symbol` の値が無い。** 結果として、TypeSpec の宣言を追加した work item は必ず「名乗り手のいない追加」を作り、そのとき開いている全記録の `documentation_impact` が `release_note` 以上へ押し上げられる。wi-523 の作業中には、無関係な [[wi-495-burn-down-the-standards-coverage-debt]] が `documentation_impact none is weaker than inferred release_note` で落ちた。

`requirement` の側は同じ問題を持たない。`addedScenarios` と `addedStandards` は `REQ-<CONTEXT>-NNN` や `RFC7662-INTROSPECT` のような素の id で数えられ、`affected_spec` の `requirement` がその形で書かれるからである。食い違っているのは宣言だけである。

## Scope

- 追加された TypeSpec 宣言を、`affected_spec` の `{ path, symbol }` で名乗れるようにする。
- 名乗れることを、`ownsSpecificationAdditions` と `specificationAdditionsClaimed` の両方で観測するテストを置く。
- 宣言を追加した記録が、同時に開いている無関係な記録の `documentation_impact` を押し上げないことを観測する。

## Out of Scope

- `documentation_impact` の推論そのもの。追加が持ち主へ届いたあと、どの水準を最小とするかの規則は変えない。
- `addedScenarios` と `addedStandards` の突き合わせ。素の id で一致しており、問題は無い。
- 過去の完了済み記録の書き換え。履歴として読む。

## Design

突き合わせの相手を揃える方法は 2 つある。

**案 A: 突き合わせ側で正規化する。** `ownsSpecificationAdditions` と `specificationAdditionsClaimed` が、追加された宣言 `<path>:<name>` を `path` と `name` に割り、記録の `{ path, symbol }` と「`path` が一致し、`symbol` の最後のドット区切りが `name` に一致する」で照合する。記録の書き方は今のまま (`IdMagic.Contract.PresentationTokenType`) でよく、`work-item-references.ts` の解決規則ともそのまま揃う。

**案 B: `affected_spec` の書き方を変える。** `symbol` に `<path>:<name>` を書くことにし、`work-item-references.ts` の解決規則をそちらへ合わせる。既存の記録の `symbol` をすべて書き換えることになる。

案 A を採る。記録の書き方は 500 件以上に及ぶうえ、`IdMagic.Contract.X` という TypeSpec の名前空間の表記は読み手にとって `path:X` より正確である。直すべきは、内部表現をそのまま突き合わせに使っている検査の側である。

正規化は 1 か所に置き、両方の呼び出し元がそれを通る。`documentation-impact.ts` は環境からファイルを読まない純関数として保たれているので、正規化も同じ層に置く。

## Plan

1. 宣言を追加した記録が自分の追加を名乗れることを観測する検査を、`documentation-impact.test.ts` と `check-workspace.test.ts` へ置く。RED を観測する。
2. 照合の正規化を 1 か所へ置き、両方の呼び出し元をそこへ通す。GREEN。
3. 無関係な記録が押し上げられないことを、記録 2 件の fixture で観測する。
4. `mise run verify`。

## Tasks

- [ ] T001 [Acceptance] 宣言の追加を `{ path, symbol }` で名乗れることを観測する検査を置き、RED を観測する。
  recipe: `mise run test-tools`
- [ ] T002 [App] 照合の正規化を置き、両方の呼び出し元を通す。
  recipe: `mise run test-tools`
- [ ] T003 [App] 無関係な記録が押し上げられないことを、記録 2 件の fixture で観測する。
  recipe: `mise run test-tools`
- [ ] T004 [Verify] `mise run verify`。

## Verification

- `affected_spec` に `{ path: spec/.../models.tsp, symbol: IdMagic.Contract.X }` を書いた記録が、`X` の追加を名乗れる。
- 同じ記録が `work-item-references.ts` の解決も通る。
- 宣言を追加した記録と同時に開いている無関係な記録の `documentation_impact` が押し上げられない。
- `mise run verify`

## Risk Notes

- **正規化を緩めすぎると、別のファイルの同名宣言まで名乗れてしまう。** 照合は `path` の一致も条件にする。名前だけの一致で通すと、`models.tsp` の追加を別コンテキストの記録が名乗れる。
- **押し上げが起きなくなること自体は、押し上げの規則が壊れたことと区別が付きにくい。** 名乗り手がいない追加が今までどおり全記録へ配られることを、同じ fixture の対照として残す。
- **検査を緩めた結果、宣言の追加が誰にも記録されなくなる。** 名乗り手がいない場合の扱いは変えない。変えるのは、名乗ろうとした記録が名乗れるようにすることだけである。
