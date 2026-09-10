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
  reason: "変更対象は着手時に開発者が走らせる `brief` の出力だけであり、製品の振る舞い、API、設定のいずれも変わらないため、リリースの読み手に伝える差分がない。"
  references: []
initial_context:
  specification: []
  typespec: []
  source:
    - tools/brief/src
    - tools/render-spec-docs/src/traces.ts
  tests: [tools/brief/src/brief.test.ts]
  stop_before_reading:
    - backend
    - frontend
    - docs/contexts
spec_impact:
  kind: none
  reason: "着手時の読み取りを助ける `brief` の出力に、既存の索引から導ける項目を足すだけである。仕様も生成物も製品の振る舞いも変えない。"
---

# `brief` に、参照する TypeSpec 記号の宣言と実装の充足状況を出させる

## Motivation

wi-257 の Timing Analysis は、着手時の「仕様・既存設計・境界の読み取り」に **9 分 33 秒**を使い、これを「最大の思考時間」と記録している。
その 9 分 33 秒で得られた結論は、同じ行が書いているとおり「既存 TypeSpec がすでに完全だったため、新しい規範編集を避けられた」である。

**9 分 33 秒かけて「変更不要」を確かめている。**

この判定に必要な材料は、すでにリポジトリの中で機械が持っている。
`mise run brief -- <work-item>` は wi-497 が追加したもので、`tools/render-spec-docs/src/traces.ts` の索引を使い、規範 ID から仕様本文、TypeSpec の宣言位置、その ID を名指す既存のテストと実装、先行する work item までたどる。
足りないのは、たどった先を並べた後の一言、すなわち「この記号は宣言済みか」「その宣言を名指す実装があるか」である。

wi-257 の場合、この一言があれば「TypeSpec は宣言済み、実装は不在」という形が最初に見え、読む対象は実装の欠けている側に絞られていた。

## Scope

- `brief` の出力に、参照する TypeSpec 記号ごとの宣言の有無を加える。
- 同じ記号について、その宣言を名指す実装とテストが存在するかを加える。
- 規範 ID についても同じ 2 項目を加える。
- 判定の意味を出力の中で明示し、「宣言済みで実装なし」と「宣言も実装もある」を読み分けられるようにする。

## Out of Scope

- 新しい索引の構築。`traces.ts` が持つものだけを使う。
- 実装が仕様を満たしているかの判定。存在するかどうかまでを出し、内容の一致は `check-contract-drift` と `spec-diff` が持つ。
- `initial_context` の自動生成。`brief` は下書きを出す道具であり、読む範囲を決めるのは実装者だという wi-497 の設計を変えない。
- `spec-where` の統合。入力も出力も別であるという wi-497 の判断を変えない。

## Design

`traces.ts` の索引は、識別子ごとに「その ID を名指すコードとテスト」と「その ID を名指す work item」を持つ。
`brief.ts` はすでにこれを読んでおり、`partitionSources` が実装とテストを分けている。

したがって加えるのは判定と表示だけである。

| 出す項目 | 導き方 |
| --- | --- |
| TypeSpec 記号の宣言の有無 | 記号が `spec/` の宣言位置に解決できるか |
| その記号を名指す実装の有無 | 索引の実装側が空でないか |
| その記号を名指すテストの有無 | 索引のテスト側が空でないか |

3 つの組み合わせのうち、着手時の判断を変えるのは次の 2 つである。

- 宣言あり、実装なし: 仕様編集は不要で、実装だけを足す。wi-257 がこれだった。
- 宣言なし: `spec-change` から始める。

出力は locations と宣言 1 つずつという `brief.ts` の既存方針を守る。
充足状況は記号ごとに 1 行を超えない。

**退けた案: 充足状況を新しい検査タスクとして独立させる。**
着手時にしか使わない判定のために、`check` の構成要素を増やす理由がない。
`brief` は着手時に 1 回走らせる道具なので、判定はその出力の中にあるのが自然である。

**退けた案: 実装の有無を、ID を名指す文字列ではなく呼び出しグラフから導く。**
精度は上がるが、索引の作り直しになる。
着手時に要るのは「そもそも触れられているか」であって、正確な到達可能性ではない。

## Plan

1. `brief.ts` に判定を足し、単体テストで 3 つの組み合わせを固定する。
2. `main.ts` の表示を足す。
3. 完了済みの work item に対して走らせ、wi-257 が 9 分 33 秒で得た結論と一致することを確かめる。

実装内容を変える未決事項はない。

## Tasks

- [x] T001 [App] `brief.ts` に、記号と規範 ID ごとの宣言、実装、テストの有無を返す関数を足す。Unit RED を先に確認する。実行: `mise run test-tools`。
- [x] T002 [App] `main.ts` の出力に充足状況を 1 記号 1 行で加える。
- [x] T003 [Verify] `mise run brief -- wi-257` を走らせ、TypeSpec は宣言済みで実装が不在という形が出ることを確かめる。宣言なしの例として、着手前の状態を作れる完了済み work item 1 件でも確かめる。
- [x] T004 [Verify] `mise run verify` を通す。

## Verification

- Acceptance RED: 変更前の `mise run brief -- wi-257` の出力に、参照する TypeSpec 記号が宣言済みかどうかを示す項目が無いことを観測する。
- Unit RED: `tools/brief/src/brief.test.ts` に、宣言あり実装なし、宣言あり実装あり、宣言なしの 3 つを与える表明を足し、実装前に失敗させる。
- `mise run test-tools`
- `mise run verify`

## Risk Notes

- 索引は ID を名指す文字列に基づくので、名指していない実装は「実装なし」と出る。逆に、コメントで言及しているだけのファイルは「実装あり」と出る。着手時の下書きとしては許容できるが、出力の文言で「名指している」ことを示し、到達可能性の保証と読み違えられないようにする。
- 出力が増えると、`brief` 自体を読む時間が増える。記号ごと 1 行を超えないという制約を T002 で守る。
- wi-257 の 9 分 33 秒がすべてこの判定に費やされたわけではない。境界の把握はこの変更では短くならない。効果の主張はこの work item では「判定に要する時間」に限る。

## Completion
- **Completed At**: 2026-09-11
- **Summary**:
  `mise run spec-diff` reports no normative specification change against `main`, which is the intended
  result for `spec_impact: none`. What changed is the terminal answer: every symbol and requirement a
  work item references now carries one `Coverage` line saying whether it is declared and whether anything
  outside `docs/` and `spec/` names it, with the reading that follows from those two answers — start from a
  specification change, add the missing code, or read the files that already name it. The verdict counts
  tests as naming, because `REQ-*` identifiers are named by tests rather than by implementation, and a
  verdict that ignored them would report implemented behavior as unimplemented. Symbols are asked for by
  their declared name, not their fully qualified one, because no Go or TypeScript file carries
  `IdMagic.Contract.*` — the same name `declarationOf` already resolves the declaration by.
- **Acceptance RED Evidence**:
  - **Test**: `mise run brief -- wi-257 | grep -c '^- Coverage:'` at the tool's own output boundary.
  - **Requirement**: N/A: spec_impact は none であり、この tooling 変更は製品の規範要件を持たない。
  - **Observed Failure**: `0` matches, `grep` exited `1`, before the change.
  - **Detection Reason**: The acceptance boundary for this work is the printed brief, since that is the
    only thing a reader consumes. Counting the verdict line distinguishes a build that computes coverage
    internally but never prints it from one that answers the question at the terminal. After the change the
    same command reports 15 lines for `wi-257`.
- **Unit RED Evidence**:
  - **Test**: `mise run test-tools` over the `coverageOf` and `coverageLine` blocks added to
    `tools/brief/src/brief.test.ts`.
  - **Requirement**: N/A: 同上。判定は規範要件ではなく着手時の読み取り補助である。
  - **Observed Failure**: `SyntaxError: Export named 'coverageLine' not found in module
    '.../tools/brief/src/brief.ts'` — 451 pass, 1 fail, 1 error.
  - **Detection Reason**: The assertions pin the reading clause per combination rather than the three
    booleans alone, so an implementation that reports the state but sends the reader to the wrong place
    still fails. That is not hypothetical: the first implementation said "what is missing is code" for an
    identifier only its tests named, the second RED (`Expected to contain: "read those files before
    changing either side"` / `Received: "... what is missing is code and not a specification edit"`) caught
    it, and the wording was corrected before it shipped.
- **Change-Resistance Results**:
  Not required at `risk: low`. One correction was nonetheless observed: the tests-only combination was
  wrong in the first GREEN implementation and was detected by a new assertion before the code changed.
- **Verification Results**:
  - `mise run verify` - passed
  - `mise run test-tools` - passed (468 tests)
  - `mise run brief -- wi-257` - all three combinations observed: declared with naming files, declared with
    none, and (through a scratch record naming an undeclared symbol) not declared.
