---
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-15
priority: p1
depends_on: []
change_kind: tooling
evidence_policy: risk-based-v4
documentation_impact:
  level: none
  reason: "リポジトリ内の検査の判定だけを変え、製品の公開契約、既存データ、成熟度は変えない。"
  references: []
spec_impact:
  kind: none
  reason: "仕様影響の検査が作業記録を読む方法だけを変える。製品の HTTP 応答、永続状態、イベント、TypeSpec と規範要件は維持する。"
initial_context:
  specification: [docs/formats/work-item-format.md]
  typespec: []
  source:
    - tools/check/src/check-spec-impact.ts
    - tools/check/src/spec-impact.ts
    - tools/check/src/archived-work-item.ts
  tests:
    - tools/check/src/spec-impact.test.ts
    - tools/check/src/spec-impact.acceptance.test.ts
  stop_before_reading: [backend, frontend, spec]
---

# 縮めた記録と削除した宣言を、仕様影響の検査で正しく帰属させる

## 動機

2026-10-15、main のチェックアウトで `mise run check-repository` が、進行中の wi-98584 について「spec_impact is none なのに作業範囲が規範仕様を変える」と報告した。
報告された `REQ-TENANCY-012`、`REQ-TENANCY-046`、`UpdateTenantQuota`、`UserStatus` は、別の作業項目が `affected_spec` で引き受けた変更である。
[完了記録の縮約](wi-29555-collapse-completed-work-item-records-into-summaries.md)で記録が `affected_spec` を持たなくなり、仕様影響の検査が引き受け手を見つけられなくなった。
同じ理由で、`UserStatus` は範囲全体の申告漏れとしても報告された。

`GroupMembershipPushed` の削除も wi-98584 のものとして報告された。
削除した宣言は今のファイルで解決できないので、どの作業項目も `affected_spec` で名指せない。

main のチェックアウトでは比較基準が上流の `origin/main` になり、未 push のコミットが範囲に入るので、この問題が表に出た。

## 対象範囲

- 仕様影響の検査は、縮めた記録の宣言を `archived_from` のコミットにある全文から読む。
- 作業範囲の削除した規範要素は、同じ範囲で仕様を変える別の作業項目があれば、その作業のものとみなす。状態遷移の扱いと揃える。

## 対象外

- 長く進行中の作業項目の作業範囲に、ほかの作業のコミットが入る設計そのものの見直し。
- 削除した宣言を `affected_spec` で名指せるようにすること。

## 設計

縮めた記録は、`git show <archived_from>:<path>` の全文を同じ解析器で読む。
`archived_from` を取得できない環境では、これまでどおり宣言のない記録として扱う。

削除した規範要素は、状態遷移と同じく名指せない変更として扱う。
同じ範囲に仕様を変える別の作業項目があるときだけ帰属を外すので、単独の作業範囲で宣言を消した場合の検出は保つ。

採らない案は、縮めた記録に `affected_spec` を残すことである。
縮約で減らした量が戻り、参照を解決し続ける費用も戻る。

## タスク

- [x] T001 [Tooling] 縮めた記録の宣言を全文から読む。受け入れ RED を先に確かめる。
- [x] T002 [Tooling] 削除した規範要素を、別の作業項目がある範囲では帰属させない。単体 RED を先に確かめる。
- [x] T003 [Verify] main のチェックアウトで `check-repository` が通ることを確かめる。

## 検証

- `mise run test-tools-file -- check/src/spec-impact.test.ts check/src/spec-impact.acceptance.test.ts`
- `mise run check-repository`（main のチェックアウトと worktree の両方）

## リスク

削除した規範要素を、無関係な作業項目の存在によって見逃す。
状態遷移で既に受け入れている限界と同じであり、単独の範囲では引き続き検出する。

## 完了
- **完了日**: 2026-10-15
- **要約**:
  `mise run spec-diff` に規範の差分はない。
  仕様影響の検査は、要約に縮めた記録の宣言を `archived_from` のコミットの全文から読む。
  作業範囲の削除した規範要素は、同じ範囲で仕様を変える別の作業項目があれば、状態遷移と同じく、その作業のものとみなす。
  main のチェックアウトと同じ比較基準（`origin/main`）で `check` を実行し、wi-98584 への誤った帰属と `UserStatus` の申告漏れの報告が消えたことを確かめた。
- **受け入れ RED の証拠**:
  - **テスト**: `tools/check/src/spec-impact.acceptance.test.ts` の「要約に縮めた完了記録の宣言を、archived_from のコミットの全文から読む」
  - **要件**: N/A: リポジトリ内の検査の変更で、製品要件はない。
  - **観測した失敗**: 実装前は `fail  spec/modules/demo/main.tsp:Task: the specification changes here, but no work item changed in this range lists it in affected_spec as impact: modifies` で失敗した。main で観測した報告と同じ形である。
  - **検出できる理由**: 縮めた記録を宣言のない記録として読む実装では、全文が引き受けた変更が申告漏れとして必ず報告される。
- **単体 RED の証拠**:
  - **テスト**: `tools/check/src/spec-impact.test.ts` の「削除した宣言は、同じ範囲に仕様を変える別の work item があれば none の矛盾にしない」
  - **要件**: N/A: リポジトリ内の検査の変更で、製品要件はない。
  - **観測した失敗**: 実装前は、別の作業項目がある範囲でも削除した宣言が none の矛盾として報告され、空の所見と一致しなかった。
  - **検出できる理由**: 同じテストは、別の作業項目がない範囲では削除した宣言を報告することも表明するので、帰属を常に外す実装と区別できる。
- **変更耐性の結果**:
  RED が故障注入を兼ねる。変更は 2 か所の条件の追加だけで、それぞれを外すと対応するテストが失敗する。
- **検証結果**:
  - `mise run test-tools-file -- check/src/spec-impact.test.ts check/src/spec-impact.acceptance.test.ts`: 89 件成功。
  - `mise run lint-tools`、`mise run typecheck-tools`: 成功。
  - `mise run check -- origin/main`: 成功。
