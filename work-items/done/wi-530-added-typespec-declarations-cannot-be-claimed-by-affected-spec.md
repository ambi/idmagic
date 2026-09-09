---
depends_on: []
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-10
priority: p2
change_kind: tooling
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: 検査どうしの突き合わせ方だけを直す。標準の行も TypeSpec も製品の振る舞いも変わらないので、リリースの読み手に見えるものが無い。
  references: []
spec_impact: { kind: none, reason: "検査の内側の突き合わせ方だけを直す。標準の行も TypeSpec も製品の振る舞いも変えない。" }
initial_context:
  specification: []
  typespec: []
  source:
    - tools/check/src/documentation-impact.ts
    - tools/workspace/src/check-workspace.ts
    - tools/check/src/spec-diff.ts
    - tools/check/src/work-item-references.ts
  tests:
    - tools/check/src/documentation-impact.test.ts
    - tools/check/src/work-item-references.test.ts
    - tools/workspace/src/check-workspace.test.ts
  stop_before_reading:
    - backend
    - frontend
    - spec
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

正規化は 1 か所に置き、両方の呼び出し元がそれを通る。`documentation-impact.ts` は環境からファイルを読まない純関数として保たれているので、正規化も同じ層に置き、そこから公開する。

```ts
// tools/check/src/documentation-impact.ts
export function claimsSpecificationAddition(
  record: { affected_spec?: unknown },
  added: Iterable<string>,
): boolean
```

`added` は `addedScenarios`、`addedStandards`、`addedDeclarations` を連ねたものである。素の id はそのまま `requirement` または `symbol` と一致で照合する。`<path>:<name>` の形の要素だけ最後の `:` で割り、`path` の一致と、`symbol` の最後のドット区切りが `name` に一致することの両方を求める。`path` を条件から外すと、別コンテキストの同名宣言まで名乗れてしまう。

`ownsSpecificationAdditions` と `check-workspace.ts` の `specificationAdditionsClaimed` は、どちらもこの関数を呼ぶだけになる。前者は純関数なのでそのまま観測できる。後者は work item のファイルを読んで `changed` で絞る入出力の輪であり、照合そのものを持たなくなる。

**RED を置く境界。** 製品の要求境界を持たない道具の変更なので、`REQ-*` に対応する Acceptance 境界は無い。代わりに検査の公開境界である `verifyDocumentationImpact` を Acceptance 相当として使う。宣言を追加した記録が押し上げられ、同時に開いている無関係な記録が押し上げられないことを、1 つの `specificationAdditionsClaimed: true` の環境で対にして読む。Unit は `claimsSpecificationAddition` の照合規則そのもので、`path` が違う同名宣言を名乗れないことを含める。

`work-item-references.ts` の側は変更しない。`{ path, symbol: 'Demo.Operations.StartTask' }` が解決することは既存の検査が押さえており、案 A はその書き方をそのまま使う。

## Plan

1. `verifyDocumentationImpact` の対の観測を `documentation-impact.test.ts` へ置き、Acceptance 相当の RED を観測する。
2. `claimsSpecificationAddition` の照合規則の観測を同じファイルへ置き、Unit RED を観測する。
3. 照合を `claimsSpecificationAddition` へ 1 つだけ置き、`ownsSpecificationAdditions` と `specificationAdditionsClaimed` をそこへ通す。GREEN。
4. `mise run verify`。

## Tasks

- [x] T001 [Acceptance] `verifyDocumentationImpact` で、宣言を追加した記録と無関係な兄弟の対を観測し、RED を観測する。
  recipe: `mise run test-tools`
- [x] T002 [Unit] `claimsSpecificationAddition` の照合規則 (path の一致を含む) を観測し、Unit RED を観測する。
  recipe: `mise run test-tools`
- [x] T003 [App] 照合を 1 か所へ置き、`ownsSpecificationAdditions` と `specificationAdditionsClaimed` を通す。
  recipe: `mise run test-tools`
- [x] T004 [Verify] `mise run verify`。

## Verification

- `affected_spec` に `{ path: spec/.../models.tsp, symbol: IdMagic.Contract.X }` を書いた記録が、`X` の追加を名乗れる。
- 同じ記録が `work-item-references.ts` の解決も通る。
- 宣言を追加した記録と同時に開いている無関係な記録の `documentation_impact` が押し上げられない。
- `mise run verify`

## Risk Notes

- **正規化を緩めすぎると、別のファイルの同名宣言まで名乗れてしまう。** 照合は `path` の一致も条件にする。名前だけの一致で通すと、`models.tsp` の追加を別コンテキストの記録が名乗れる。
- **押し上げが起きなくなること自体は、押し上げの規則が壊れたことと区別が付きにくい。** 名乗り手がいない追加が今までどおり全記録へ配られることを、同じ fixture の対照として残す。
- **検査を緩めた結果、宣言の追加が誰にも記録されなくなる。** 名乗り手がいない場合の扱いは変えない。変えるのは、名乗ろうとした記録が名乗れるようにすることだけである。

## Completion

- **Completed At**: 2026-09-10
- **Summary**:
  追加された TypeSpec 宣言を、work item が `affected_spec` の `{ path, symbol }` で名乗れる
  ようになった。`mise run spec-diff` は規範差分なしを返す。検査の突き合わせ方だけを直しており、
  標準の行も TypeSpec も製品の振る舞いも変えていないので、これが期待どおりの出力である。
  照合は `claimsSpecificationAddition` 1 つになった。素の id はこれまでどおり `requirement` と
  `symbol` に完全一致で当たり、`<path>:<name>` の形の要素だけ最後の `:` で割って、`path` の一致と
  `symbol` の最後のドット区切りの一致の両方を求める。`documentation-impact.ts` の
  `ownsSpecificationAdditions` と `check-workspace.ts` の `specificationAdditionsClaimed` は、
  どちらもこの関数を呼ぶだけになり、照合の規則を自分では持たなくなった。
  **同じ問いを 2 か所で別々に実装していたことが原因だった。** 参照としては解決する `symbol` の
  書き方が、持ち主の判定では 1 件も一致しない。片方だけを読んでいる限り、どちらも正しく見える。
  名乗り手がいない追加を全記録へ配る規則は変えていない。変えたのは、名乗ろうとした記録が
  名乗れるようにすることだけである。
- **Acceptance RED Evidence**:
  - **Test**: `verifyDocumentationImpact > gives an added TypeSpec declaration to the record that declares it`
    (`tools/check/src/documentation-impact.test.ts`)
  - **Requirement**: N/A: 検査どうしの突き合わせを直す道具の変更であり、REQ 番号を持つ規範シナリオに対応しない。
  - **Observed Failure**: `expect(received).toContain(expected)` /
    `Expected to contain: "documentation_impact none is weaker than inferred release_note"` /
    `Received: []`。宣言を `{ path, symbol }` で名乗った記録が、自分の追加を名乗れずに素通りした。
  - **Detection Reason**: 観測は `verifyDocumentationImpact` という検査の公開境界にあり、
    `check-work-items` が報告するのと同じ値を読む。名乗った記録が押し上げられることと、同時に
    開いている兄弟が押し上げられないことを 1 つの環境で対にして読むので、「全記録を素通りさせる」
    実装と「全記録を押し上げる」実装のどちらとも区別できる。
- **Unit RED Evidence**:
  - **Test**: `verifyDocumentationImpact > matches an added declaration by file and declaration name together`
    (`tools/check/src/documentation-impact.test.ts`)
  - **Requirement**: N/A: 同上。照合規則そのものに対応する規範シナリオは無い。
  - **Observed Failure**: `SyntaxError: Export named 'claimsSpecificationAddition' not found in module`
    `'/Users/tn/src/idmagic/tools/check/src/documentation-impact.ts'`。照合規則がまだ 1 か所に
    存在しない状態である。
  - **Detection Reason**: 一致する場合だけでなく、`path` だけ違う同名宣言と `symbol` だけ違う
    参照の 2 つを不一致として読む。名前だけで照合する実装は前者で落ちる。素の `REQ-DEMO-001` が
    これまでどおり `requirement` に当たることも同じ検査に置いてあるので、宣言の対応を足すために
    既存の対応を壊した実装も落ちる。
- **Change-Resistance Results**:
  単体の観測に加えて、実際のリポジトリで wi-523 の状況を再現し、対で読んだ。`models.tsp` へ
  `enum PresentationTokenType` を一時的に足し、`wi-530` の `spec_impact` を
  `affected_spec: [{ path: spec/contexts/oauth2/models.tsp, symbol: IdMagic.Contract.PresentationTokenType }]`
  へ差し替えて `mise run check-work-items` を走らせた。

  | 状態 | 落ちた記録 |
  |---|---|
  | `path` が実在の宣言と一致（名乗りが成立する） | `wi-530` だけ。`wi-495` は通った |
  | `path` を `spec/contexts/apitoken/models.tsp` に取り違え（名乗りが成立しない） | `wi-495`。`wi-530` は通った |

  2 行目が修正前の症状そのものであり、`path` の一致を条件から外した実装ではこの区別が付かなくなる。
  一時的な変更はどちらも元へ戻してある。
- **Verification Results**:
  - `mise run test-tools -- check/src/documentation-impact.test.ts` - passed（11 件）
  - `mise run test-tools` - passed（35 ファイル 461 件）
  - `mise run check-work-items` - passed（528 件）
  - `mise run check-ids` - passed（528 件）
  - `mise run spec-diff` - passed（規範差分なし）
  - `mise run test-ui-e2e` - `N/A: 変更はリポジトリ検査の TypeScript だけで、ブラウザーへ到達する経路が無い。`
  - `mise run verify` - passed（終了コード 0 を直接確認）
