---
depends_on: []
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-06
priority: p2
change_kind: tooling
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: 検査の内側の帰属規則だけが変わる。製品の振る舞いも、work item の書き方も変わらないので、リリースの読み手に見えるものが無い。
  references: []
spec_impact: { kind: none, reason: "documentation_impact の推論がワークスペースの規範差分をどの記録へ帰属させるかという、検査の内側の規則である。標準の行も規範シナリオも製品の振る舞いも変えない。" }
initial_context:
  specification: []
  typespec: []
  source:
    - tools/check/src/documentation-impact.ts
    - tools/workspace/src/check-workspace.ts
  tests:
    - tools/check/src/documentation-impact.test.ts
  stop_before_reading:
    - backend
    - frontend
    - docs
---

# 開いたままの親 work item が、子の規範差分を自分のものとして受け取ってしまう

## Motivation

`documentation_impact` の推論は、ワークスペース全体の規範差分を 1 つ持ち、それをどの記録に帰属させるかを `ownsWorkspaceDiff` で決める。この関数は `status: in_progress` の記録を**無条件で**所有者とみなす。

その前提は、進行中の記録がちょうど 1 件のときだけ正しい。[[wi-495-burn-down-the-standards-coverage-debt]] は 9 件の子 work item の完了を待つ親で、`depends_on` がその順序を拘束している以上、子を 1 件ずつ実装する間ずっと `in_progress` のままである。親自身は受入集合の導入しかしておらず `documentation_impact: none` を宣言しているが、**子が規範シナリオを 1 つ足すたびに、その差分が親にも帰属して `none is weaker than inferred release_note` で落ちる。**

[[wi-508-amr-vocabulary-declaration-and-implementation-disagree]] が `REQ-AUTHENTICATION-036` を足したところで実際に起きた。残る 7 件の子のうち規範に触れるものすべてで再発する。

親に要らないリリースノートを持たせるのは記録を偽ることであり、`in_progress` を外すのは `depends_on` が拘束する順序を壊す。直すべきは帰属の規則である。

## Scope

- 規範差分が **追加した要素** の帰属を、それを `affected_spec` で宣言している記録に限る。
- 追加した要素をどの記録も宣言していないときは、これまでどおりワークスペース差分を所有するすべての記録へ帰属させる。
- `check-workspace` が、変更された記録の `affected_spec` を読んで宣言の有無を解決し、検査へ渡す。

## Out of Scope

- 完了済みの記録に対する帰属規則。`changedRecords` による現在の規則をそのまま残す。
- 削除・変更された要素、成熟度の遷移、破壊的 API 変更の帰属。今回落ちたのは追加の側だけであり、他は同じ規則のままにする。同種の取りこぼしが起きたときに、そのときの実例をもって直す。
- `documentation_impact` の水準そのものの推論。
- 進行中の記録を 1 件に制限する規則。長期の親が開いたままであること自体は、`depends_on` が意図している状態である。
- [[wi-495-burn-down-the-standards-coverage-debt]] の記録の書き換え。

## Design

**追加した要素の持ち主は、記録が自分で書いている。** `affected_spec` は「この変更が触れた規範要素」を直接参照する欄であり、[[wi-508-amr-vocabulary-declaration-and-implementation-disagree]] は `REQ-AUTHENTICATION-036` と `RFC8176-AMR-VOCABULARY` をそこに挙げている。[[wi-495-burn-down-the-standards-coverage-debt]] も本項目もどちらも挙げていない。帰属の材料はここにあり、状態や作業ツリーの偶然ではない。

最初は「作業ツリーが変更している進行中の記録だけが所有する」という規則を書いた。**これは足りなかった。** 本項目自身を起票した瞬間に、変更されている進行中の記録が 2 件になり、同じ形で本項目が wi-508 の追加を受け取った。「いま書かれているか」は 1 件に絞る材料にならない。同時に 2 件書くことは普通に起きるからである。

したがって規則は 2 段にする。

| 追加した要素を `affected_spec` で宣言している記録 | 追加の所有者 |
|---|---|
| 1 件以上 | 宣言している記録だけ |
| 0 件 | ワークスペース差分を所有するすべての記録 (従来どおり) |

下段を残すのは安全網である。仕様だけを変えて `affected_spec` に書かなかった場合、上段だけだとどの記録も所有せず検査が黙る。下段があれば従来どおり報告される。

`DocumentationImpactEnvironment` へ真偽値 `specificationAdditionsClaimed` を 1 つ足す。検査は記録を 1 件ずつ見るので、「他の誰かが宣言しているか」は呼び出し側が一度だけ計算して渡す。宣言の照合そのものは、記録が持つ `affected_spec` を読んで検査の側で行う。

呼び出し側が読むのは**変更された記録だけ**である。数か月前に完了した記録は当時その要素を追加したことを宣言しており、同じ id が今日また追加されたときにそれを持ち主とみなすのは、宣言ではなく再利用の偶然になる。

新しい検査の型は作らない。`changedRecords` と同じく、省略時は従来の振る舞いになる任意の入力として足す。

## Plan

1. `ownsWorkspaceDiff` の規則を 2 段にし、テストで RED を観測する。
2. `check-workspace` が進行中の記録を解決して渡す。
3. `mise run verify`。

## Tasks

- [x] T001 [Acceptance] `mise run check-work-items` が wi-495 について落ちることを観測する。
- [x] T002 [Tooling] 追加した要素の帰属を `affected_spec` の宣言で決める。
  recipe: `mise run test-tools`
- [x] T003 [Tooling] `check-workspace` が、変更された記録の `affected_spec` から宣言の有無を解決して渡す。
  recipe: `mise run check-work-items`
- [x] T004 [Verify] `mise run verify`。

## Verification

- 進行中の記録が 2 件あり、追加した要素を片方だけが `affected_spec` で宣言しているとき、追加はその片方にだけ帰属する。
- 追加した要素をどの記録も宣言していないとき、追加は従来どおりすべての所有者へ帰属する。
- `mise run check-work-items`
- `mise run verify`

## Risk Notes

- **安全網を落とす。** 宣言が 0 件のときの従来の振る舞いを残さないと、`affected_spec` に書かずに仕様を変えた変更が検査を素通りする。2 段の下段がそれであり、テストで両方を観測する。
- **`affected_spec` の読み直しが検査の費用になる。** 対象は変更された記録だけなので、通常は 1〜2 件である。全件を読み直す実装にしない。
- **「いま書かれているか」で絞りたくなる。** 最初にそう書いて足りなかった。同時に 2 件書くことは普通に起きるので、状態や作業ツリーは 1 件に絞る材料にならない。

## Completion

- **Completed At**: 2026-09-06
- **Summary**:
  `mise run spec-diff` は `REQ-AUTHENTICATION-036` の追加と `RFC8176-AMR-VOCABULARY` の変更を返すが、
  どちらも [[wi-508-amr-vocabulary-declaration-and-implementation-disagree]] のものであり、本項目は
  規範を 1 行も変えていない。差分は `documentation_impact` の推論の内側にある。規範差分が追加した
  要素は、それを `affected_spec` で宣言している記録にだけ帰属するようになった。宣言している記録が
  1 件も無いときは従来どおりすべての所有者へ帰属する。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-work-items`
  - **Requirement**: N/A: work item の検査の内側の規則であり、製品の規範要求ではない。
  - **Observed Failure**: exit 1。
    `work-items/wi-495-burn-down-the-standards-coverage-debt.md: documentation_impact none is weaker than inferred release_note`。
    wi-495 は受入集合の導入しかしておらず、落ちた原因は wi-508 が足した `REQ-AUTHENTICATION-036` である。
  - **Detection Reason**: この検査は宣言した `documentation_impact` と推論した最小値を突き合わせる。
    帰属が広すぎれば、規範に触れていない記録が `none` のまま落ちる。落ちる記録の名前が、帰属が
    どこまで広がっているかをそのまま指している。修正後の同じコマンドは
    `ok 508 file(s)` を返す。
- **Unit RED Evidence**:
  - **Test**: `tools/check/src/documentation-impact.test.ts` の
    `gives an added element to the record that declares it, not to its open siblings`
  - **Requirement**: N/A: 上と同じ。
  - **Observed Failure**: 追加を宣言していない兄弟の記録が
    `documentation_impact none is weaker than inferred release_note` を受け取る。
  - **Detection Reason**: テストは同じ環境で 2 つの記録を通し、宣言している側だけが所見を受け取る
    ことを観測する。片方だけを観測すると、全件に帰属する実装と全件に帰属しない実装のどちらも
    区別できない。安全網の側 (誰も宣言していないとき) も同じテストが観測する。
- **Change-Resistance Results**:
  規則の 2 段それぞれを崩し、対応する観測が落ちることを確かめた。

  | 注入した故障 | 結果 |
  |---|---|
  | 追加の帰属を見ない (修正前の規則にもどす) | `gives an added element to the record that declares it...` が落ちる。兄弟が所見を受け取る |
  | 誰も宣言していないときに追加を握りつぶす (安全網を落とす) | 上のテストと、既存の `attributes the workspace specification diff only to the records that change` の 2 件が落ちる |

  最初の実装 (「作業ツリーが変更している進行中の記録だけが所有する」) は、本項目自身を起票した
  時点で不足が露見した。変更されている進行中の記録が 2 件になり、本項目が wi-508 の追加を
  受け取った。**規則を弱い材料で書いたことを、実物の作業ツリーが即座に反証した**という記録として
  Design に残してある。
- **Verification Results**:
  - `mise run check-work-items` - passed (508 file(s))
  - `mise run check-ids` - passed
  - `mise run test-tools` - passed (453 tests)
  - `mise run verify` - passed
