---
depends_on: []
status: completed
authors: [tn]
risk: low
created_at: 2026-08-28
priority: p1
change_kind: tooling
evidence_policy: risk-based-v2
initial_context:
  specification: [WORK_ITEM_FORMAT.md, DEVELOPMENT.md]
  source: [tools/check/schemas/work-item.schema.json, tools/check/src/main.ts]
  tests: [tools/check/src/lib.test.ts, tools/check/src/work-item-markdown.test.ts]
  stop_before_reading: [backend, frontend, spec]
spec_impact: { kind: none, reason: "方法文書とその検査の変更であり、製品の振る舞いと外部契約を変えない。" }
---

# 実装前の承認を証拠契約から外し、リスクに応じた証拠だけを残す

## Motivation

`DEVELOPMENT.md` の証拠契約は、`medium` 以上のリスクを持つ work item に、実装開始前の人間の承認を要求している。`WORK_ITEM_FORMAT.md` はその記録場所として `approval: {by, at, scope, baseline}` を定め、schema は risk が `medium` 以上で `status` が `in_progress` になった時点でこのフィールドを必須にしている。

**この承認は、実際には既に与えられている。** work item は人間が依頼して初めて存在する。何を作るかは Motivation・Scope・Out of Scope に書かれ、依頼した人間がそれを読んでいる。`approval.scope` に「実装してよい振る舞いと設計の境界」を改めて書き写しても、既に Scope が述べたことの重複でしかない。その状態で承認の記録を要求すれば、記入は儀式になる。**無思考で承認欄を埋める習慣がひとつ増えるだけで、判断はひとつも増えない。**

リスクの高い変更に注意が必要であることは変わらない。だがその注意は、承認欄の有無とは無関係に働く。認証、認可、テナント境界、暗号、プロトコル互換性、永続データの移行に触れるとき、人間が慎重になるのは、承認フィールドが必須だからではなく、壊れたときの結果がわかっているからである。逆に言えば、承認欄がなければ注意しない書き手は、承認欄があっても注意しない。**強制力を持つのは、承認の記録ではなく、独立検証と変化耐性の証拠 — つまり「間違った実装が実際に検出されること」を示す側である。**

`Post-Approval Changes` も同じ理由で残らない。この欄が求めていたのは `mise run spec-diff <baseline>` の結果、すなわち実装中に規範そのものが動いていないかの観測である。しかし `DEVELOPMENT.md` は既に、完了時の Summary を `spec-diff` から導けと定めている。Summary が「その変更が持ち込んだ意味の差分」を観測に基づいて述べる以上、同じ観測をもう一度別の欄に書く理由はない。承認という基準点が消えれば、比較対象は分岐元 (`spec-diff` の既定である `main`) になり、それは変更全体の規範差分そのものである。

## Scope

- `WORK_ITEM_FORMAT.md` から `approval` ブロックと `Post-Approval Changes` の記述を削除する。
- `DEVELOPMENT.md` の §3 loop 表、§4 証拠契約 (risk 表と本文)、§5 検証の梯子、§7、§9 から承認を削除する。
- `tools/check/schemas/work-item.schema.json` から `approval` の定義、`medium` 以上で `approval` を必須にする分岐、`completion.post_approval_changes` とその必須指定を削除する。
- `tools/check/src/main.ts` の `post-approval changes` ラベルの解析を削除する。
- `.agents/skills/implement-work-item/SKILL.md` と `.agents/skills/new-work-item/SKILL.md` から承認の手順を削除する。
- `mise run spec-diff <baseline>` の指示を、基準点を取らない `mise run spec-diff` に置き換える。

## Out of Scope

- `medium` 以上の独立検証、変化耐性の証拠、Acceptance RED / Unit RED の契約。これらは残す。リスク表そのものも残す。
- 「実装を通すために scenario を緩めてはならない」という規則。承認とは独立に成り立つので、承認への言及だけを外して残す。
- `work-items/done/` の 5 件 (wi-105, wi-410, wi-411, wi-412, wi-424) が持つ `approval` frontmatter と `Post-Approval Changes` 行。これらは当時の契約に基づく履歴の記録であり、書き換えない。
- `spec/idmagic.openapi.baseline.json` とリリース互換性の baseline。名前が同じだけで別の概念である。
- push・merge・本番操作・外部システムへの書き込みの可否。証拠契約はもともとこれらの許可を与えていない。

## Design

**承認の削除は、証拠契約の弱体化ではなく、位置の訂正である。** 現在の契約は「実装前に人間が承認する」と「完了前に独立検証と変化耐性を示す」の二段構えだが、前者は依頼の時点で満たされており、後者だけが観測を要求している。前者を外しても、後者は無傷で残る。

`risk` 表の `Before implementation` 列は、`medium` 以上で承認だけを述べていた。承認を外すと、`low` の行が述べる内容 (`initial_context` の確定、作るものを変えうる疑問の解決、Acceptance RED と Unit RED の指名) が全リスクに共通する要求として残る。`medium` はそれを引き継ぎ、`high` / `critical` は加えてセキュリティ・互換性・移行・ロールバックの前提を明示する。**列は空にならない。実装前に固定すべきものは、承認ではなく証拠の指名だった。**

schema の変更は 3 箇所である。`allOf` から `approval` を必須にする分岐を削除し、`properties.approval` の定義を削除し、`medium` 以上の完了記録に対する `completion.required` から `post_approval_changes` を外して定義も削除する。root と `completion` はいずれも `additionalProperties: false` を指定していないため、**定義を消しても `done/` の 5 件は落ちない。** 履歴として残る余分なキーは検証されないだけである。

`main.ts` の Markdown 解析も同様で、未知のラベルは無視される (該当する `else if` に入らないだけで、エラーにはならない)。したがって `post-approval changes` の分岐を削除すれば、`done/` の既存記録はその行を持ったまま通る。

検討して採らなかった案:

- **`approval` を任意フィールドとして残す。** 残せば「書いてもよい」ものになり、書くべきかの判断を各 work item に押し付ける。契約は必須か不在かのどちらかであるべきで、任意の証拠欄は最も価値が低い。
- **`Post-Approval Changes` を `Normative Diff` などに改名して残す。** 承認への依存は消えるが、Summary が同じ観測を述べる以上、二重記録になる。名前を変えた儀式は儀式である。
- **リスク表ごと削除する。** 独立検証と変化耐性は、依頼だけでは代替できない観測を要求している。ここは残す価値がある。

## Plan

1. 方法文書 (`WORK_ITEM_FORMAT.md`、`DEVELOPMENT.md`) を先に直す。schema は同文書からの派生物なので、順序は文書が先である。
2. 検査の RED を確認してから schema と `main.ts` を直す。
3. skill を追随させる。
4. `mise run verify` で `done/` を含む全 work item が通ることを確認する。

自己言及について。本 work item は承認契約を削除する変更であり、それ自体を `medium` と分類すれば、削除しようとしている承認欄の記入を要求されることになる。実際のリスクは製品の振る舞い・外部契約・永続データのいずれにも及ばず、影響はリポジトリ自身の検査に閉じているので `low` とする。

## Tasks

- [x] T001 [Spec] `WORK_ITEM_FORMAT.md` と `DEVELOPMENT.md` から承認を削除し、`medium` 以上の `Before implementation` を証拠の指名として書き直す。§4 の見出しは `Evidence contract` になった。
- [x] T002 [Acceptance] risk `medium`・`status: in_progress`・`approval` 無しの work item を `check-work-items` が受理することを RED で確認した (`lib.test.ts` の `starts medium-risk work without an approval record`)。
- [x] T003 [App] schema から `approval` の必須分岐・定義・`post_approval_changes` を削除して GREEN にした。
- [x] T004 [App] `main.ts` の `post-approval changes` ラベル解析を削除し、Markdown 解析のテストの fixture から承認記録を外した。
- [x] T005 [Skill] `implement-work-item` と `new-work-item` から承認の手順を削除し、`spec-diff` の呼び出しを基準点なしにした。
- [x] T006 [Verify] `mise run verify` を通し、`work-items/done/` の 5 件が承認記録を残したまま 431 件すべて OK になることを確認した。

## Verification

- `mise run check-work-items`
- `mise run check-ids`
- `mise run verify`

## Risk Notes

- **承認の削除が、独立検証の削除と読まれる危険。** §4 の見出しが「Evidence contract and approval」であるため、承認を消すと節全体が緩んだように読まれうる。見出しを証拠契約だけの名前に変え、独立検証と変化耐性の要求は文言を維持することで防ぐ。
- **`done/` の履歴が壊れる危険。** schema の緩和方向の変更であり、root と `completion` が `additionalProperties: false` を持たないことを確認済み。T006 で実際に検証を通して確かめる。
- **`baseline` という語の取り違え。** リリース互換性の `spec/idmagic.openapi.baseline.json` は別概念であり、`DEVELOPMENT.md` §2 の該当行と `spec-render` skill には触れない。

## Completion

- **Completed At**: 2026-08-28
- **Summary**:
  `mise run spec-diff` は `no normative specification change against main` を返す。規範仕様は動いていない。変わったのは証拠契約であり、`medium` 以上の work item が実装開始前に `approval: {by, at, scope, baseline}` を記録する義務が消え、完了記録の `Post-Approval Changes` も消えた。残ったのは、リスク表そのもの、`medium` 以上の独立検証と変化耐性の証拠、Acceptance RED / Unit RED である。`DEVELOPMENT.md` §4 の見出しは `Evidence contract and approval` から `Evidence contract` になり、`medium` の `Before implementation` は「`low` の要求を適用する」— すなわち `initial_context` の確定、作るものを変えうる疑問の解決、両 RED の指名 — になった。完了時の Summary は `mise run spec-diff` から読み取ることが明示され、`Post-Approval Changes` が担っていた観測はそこに吸収された。
- **Acceptance RED Evidence**:
  - **Test**: `tools/check/src/lib.test.ts` の `validateAgainstSchema — work-item > starts medium-risk work without an approval record`
  - **Requirement**: N/A: リポジトリの検査ツールであり、対応する規範的な製品要件を持たない。
  - **Observed Failure**: `schema: / must have required property 'approval' (missing: approval)` と `schema: / must match "then" schema` の 2 件。`193 pass 1 fail`。
  - **Detection Reason**: この表明は、ゲートが `approval` を実際に要求しているかどうかだけで通過と失敗が分かれる。schema の記述を読むのではなく、承認記録を持たない `medium`・`in_progress` の記録を検証器に通した結果を見ている。承認の必須指定を消し忘れた実装、あるいは別の分岐に書き写した実装は、この表明を通せない。
- **Unit RED Evidence**:
  - **Test**: `tools/check/src/work-item-markdown.test.ts` の 2 件 (`parses structured RED and stronger completion evidence`、`parses separate Acceptance RED and Unit RED evidence`) を、変更前の schema (`git show HEAD:tools/check/schemas/work-item.schema.json`) に対して実行した。
  - **Requirement**: N/A: Markdown の完了記録の解析であり、対応する規範的な製品要件を持たない。
  - **Observed Failure**: 両方とも `expect(exitCode, stderr).toBe(0)` が `Received: 1` で失敗。`0 pass 2 fail`。
  - **Detection Reason**: fixture から承認 frontmatter と `Post-Approval Changes` 行を取り除いた完了記録が、CLI の終了コードで受理されるかを見ている。schema 側だけを緩めて Markdown 解析側に `post_approval_changes` の生成が残る実装、あるいはその逆は、この 2 件で分かれる。
- **Verification Results**:
  - `mise run verify` - passed (431 work-item file(s) OK、tools 194 pass / 0 fail、frontend 667 pass / 0 fail)
  - `mise run spec-diff` - `no normative specification change against main`
