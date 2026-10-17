---
status: in_progress
authors: [tn]
risk: low
created_at: 2026-09-25
priority: p3
depends_on: []
change_kind: tooling
spec_impact: { kind: none, reason: "作業項目の検査が、取り除いた規範要素をどの記録に帰属させるかだけを変え、プロダクトの仕様は変えない。" }
---

# 取り除いた規範要素による文書化水準の要求を、その要素を名指しする記録だけに課す

## 動機

`check-work-items` は、作業ツリーと `main` の仕様の差分から、各記録に必要な文書化の最低水準を導く。
追加した規範要素は、`affected_spec` で名指ししている記録だけに帰属させている（`claimsSpecificationAddition`）。
一方、取り除いた規範要素は、差分を持つとみなされたすべての記録に `removal_notice` を求める。
差分を持つとみなされるのは、進行中の記録すべてと、作業ツリーで変更された記録である。

wi-68415 で TypeSpec の操作を改名したとき、次の記録が無関係なのに `removal_notice` を求められ、コミット前の `mise run verify` が失敗した。

| 記録 | 差分を持つとみなされた理由 |
| --- | --- |
| wi-496（進行中） | 進行中の記録はすべて差分全体を持つとみなされる |
| 完了済みの 5 件 | 改名したテストのファイル名とテスト名への参照を直したため、変更された記録になった |

`main` へコミットすると差分がなくなるので検査は通る。
しかし、ブランチで作業する場合はマージまで失敗が続き、完了済みの記録の参照を直すだけの変更が文書化の要求を受ける。

## 対象範囲

- 取り除いた規範要素（シナリオ、標準行、TypeSpec の宣言）による `removal_notice` の要求を、`affected_spec` でその要素を名指ししている記録に限る。
- 追加の帰属と同じ照合規則を使う。

## 対象外

- 非推奨化と破壊的な API 変更による水準の導出。必要になったら別の項目で扱う。
- 機能レジストリの成熟度の差分による導出。

## 設計

`minimumDocumentationImpact` の除去の分岐を、追加の分岐と同じ形にする。
除去した要素は作業ツリーにないので、記録の `affected_spec` が旧名を名指ししているかで照合する。

### 帰属の判断

- `DocumentationImpactEnvironment` に `specificationRemovalsClaimed?: boolean` を足す。
  作業ツリーで変更された記録のどれかが除去した要素を名指しすれば真にする。
  追加の `specificationAdditionsClaimed` と同じ求め方である。
- 真なら、除去した要素を名指しする記録だけが `removal_notice` を負う。
  偽なら、従来どおり差分を持つ全記録が負う。
- 照合は追加と同じ `referenceNames` を使う。
  関数名 `claimsSpecificationAddition` は追加に限らないので `claimsSpecificationElement` へ改名する。

### 改名の扱い

改名する記録は、新名と旧名の両方を `affected_spec` に挙げる。
新名で追加を、旧名で除去を名指しするので、`release_note` と `removal_notice` の両方の要求を受ける。
公開後の改名は旧名の利用者に対する除去なので、この要求は正しい。

旧名は作業ツリーにないため、現在の `verifyWorkItemReferences` は参照を解決できないとして拒否する。
採用した案と退けた案を次に示す。

| 案 | 内容 | 採否と理由 |
| --- | --- | --- |
| A | `main` との差分で取り除かれた要素を名指しする参照は、解決を求めない | 採用。差分がある間だけの免除で、照合規則は帰属と共有できる |
| B | 旧名を `affected_spec` 以外の欄（例: `removed_spec`）に書かせる | 不採用。参照の形式と解決の規則が二つになる |
| C | 除去を、同じファイルへ追加を名指しする記録へ帰属させる | 不採用。ファイルごと消す除去や、改名でない除去を扱えない |

案 A では `ReferenceEnvironment` に、差分で取り除かれた要素の一覧 `removedSpecification?: readonly string[]` を足す。
差分がなくなった後（`main` へ統合した後）も進行中の記録が旧名を残していれば、解決できない参照として報告される。
完了時には記録を縮めて `affected_spec` を落とすので、通常の流れでは問題にならない。

### シグネチャ

```ts
export function namesSpecificationElement(
  reference: Record<string, unknown>,
  elements: Iterable<string>,
): boolean
export function claimsSpecificationElement(
  record: { affected_spec?: unknown },
  elements: Iterable<string>,
): boolean
```

## 計画

1. 除去の帰属を追加と同じ形に揃える単体テストを先に書き、RED を確かめる。
2. 帰属の分岐を実装する。
3. 取り除いた要素への参照を解決の対象から外す単体テストを書き、RED を確かめてから実装する。
4. `check-work-items` から差分の除去一覧を両方へ渡す。

## タスク

- [x] T001 [Acceptance] 無関係な進行中の記録に `removal_notice` を求めないことを RED として確かめる。
- [x] T002 [App] 除去の帰属を実装する。
- [x] T003 [App] 取り除いた要素を名指しする `affected_spec` を解決の対象から外す。
- [x] T004 [App] `check-work-items` で除去の帰属と参照の免除を配線する。
- [x] T005 [Docs] `affected_spec` の解決規則の記述へ、取り除いた要素の扱いを足す。
- [x] T006 [Verify] 変更を検証する。

## 検証

- `mise run test-tools`
- `mise run check-work-items`

## リスク

- **本来必要な告知を見逃す。** 除去を名指ししない記録が、除去そのものを行っている場合に告知の要求が消える。除去した要素を名指しする記録が 1 件もないときは、従来どおり差分を持つ全記録へ求める。

## 要約

`check-work-items` は、取り除いたシナリオ、標準行、TypeSpec の宣言による `removal_notice` の要求を、`affected_spec` でその要素を名指しする記録だけに課す。
変更された記録のどれも名指ししなければ、従来どおり差分を持つすべての記録に課す。
照合には追加の帰属と同じ規則を使い、関数名を `claimsSpecificationElement` へ改めた。
改名する記録は新名と旧名を両方挙げ、追加と除去の両方の要求を受ける。
旧名は作業ツリーで解決できないので、`main` との差分が取り除いた要素として示す間は、その参照の解決を求めない。
旧名を別の欄に書かせる案と、同じファイルへの追加の記録に帰属させる案は採らなかった。
初回公開前は告知の最低水準が常に `none` なので、wi-68415 で起きた失敗は公開後に再び現れるはずだったものである。
`mise run spec-diff` は規範の差分を報告しない。
