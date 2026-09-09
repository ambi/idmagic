---
depends_on: []
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-10
priority: p1
change_kind: tooling
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: 実装中の検査の走らせ方だけを変える。製品の振る舞いも公開契約も変わらないので、リリースの読み手に見えるものが無い。
  references: []
spec_impact:
  kind: none
  reason: "実装中にどの検査をいつ走らせるかという手順だけを変える。検査そのものの内容も、製品の観測可能な振る舞いも公開契約も変えない。"
initial_context:
  specification: []
  typespec: []
  source:
    - .agents/skills/implement-work-item/SKILL.md
    - docs/development/specification-first-workflow.md
    - tools/check/src/agent-guidance.ts
  tests: []
  stop_before_reading:
    - backend
    - frontend
    - spec
---

# 層ごとの検査を、その層が GREEN になった直後へ前倒しする

## Motivation

`implement-work-item` skill の第 8 段は「lint と集約ゲートは最終検証まで遅らせる」と指示している。
この指示は wi-497 より前の費用構造に基づく。
当時の `lint-go` は 83 秒だったので、実装中に何度も払う理由がなかった。

wi-497 が費用を変えた後、根拠が消えている。
2026-09-10 に測り直した。

| 状態 | 実測 | いつ起きるか |
| --- | ---: | --- |
| 冷状態 | 18.3秒 | worktree 作成直後、依存更新後 |
| 1 ファイル編集後 | 5.3秒 | 実装中の各回 |
| 温状態 (入力不変) | 1.7秒 | 続けて 2 回目 |

`check-contract-drift` と `check-api-compat` は各 1 秒、`check-work-items` は 2.4 秒である。

遅らせた代償は wi-257 の Timing Analysis に出ている。
最終集約ゲートの初回 37.90 秒が、lint 11 件、API DTO の TypeSpec 漏れ、未導入 UI 依存関係を**同時に**検出した。
そこからの修正に、lint で約 3 分、API DTO の TypeSpec 漏れで約 4 分かかっている。
合計 7 分は能動作業時間の 8% にあたり、5.3 秒の検査を数回払う額をはるかに超える。

金額の差だけが問題ではない。
最終ゲートで検出されたコードは、書いた時点から離れているので読み直しが要る。
同じ修正でも、書いた直後なら安い。

## Scope

- `implement-work-item` skill の第 8 段から、lint を最終検証まで遅らせる指示を外す。
- 触った層に対応する秒単位の検査を、その層が GREEN になった直後に走らせる指示へ置き換える。
- 冷状態の 18.3 秒を作業開始時に払う指示を、対応する箇所へ入れる。
- 検証のはしごの記述 (`docs/development/specification-first-workflow.md`) を同じ内容にそろえる。

## Out of Scope

- 検査そのものの高速化。wi-497 が測ったとおり、実行時間はすでに律速ではない。
- `lint-go` の linter の取捨。検証ゲートの意味を変える判断であり、この work item と混ぜない。
- 作業開始時の setup と狭い段の並列実行。wi-529 が持つ。
- 最終集約ゲートを走らせる回数。実装中は対象検査、最後は `test-go-changed` と `verify` を各 1 回という wi-257 の結論を変えない。

## Design

触った層と、その直後に走らせる検査を対応させる。

| 触った層 | 直後に走らせる | 実測 |
| --- | --- | ---: |
| 管理 API、DTO | `check-contract-drift`、`check-api-compat` | 約2秒 |
| Go の 1 挙動が GREEN | `lint-go` | 5.3秒 |
| work item の frontmatter、Completion | `check-work-items` | 2.4秒 |

`lint-go` はリポジトリ全体を対象にするが、キャッシュが効くので変更したパッケージぶんしか実際には走らない。
パッケージを絞る新しいタスクは要らない。

**退けた案: `lint-go-package` を足して対象を絞る。**
1 ファイル編集後の実測が 5.3 秒である以上、絞って得られる差は数秒に満たない。
一方でタスクが 1 つ増え、`lint-go` との間で対象の食い違いが生まれる。

**退けた案: 最終ゲートのままにして、失敗の読み方だけを直す。**
wi-497 が集約ゲートを「落ちたゲートを全部報告する」形にしたので、1 回の実行で 3 種類の失敗を受け取れるようになっている。
それでも wi-257 は 7 分を払った。
まとめて受け取れることと、書いた直後に直せることは別である。

第 8 段の現在の指示は、tight feedback loop を守れという趣旨自体は正しい。
変えるのは「lint と集約ゲートを遅らせる」という費用前提の部分だけで、対象テストの粒度に関する指示は残す。

## Plan

1. 実測を skill の中に書かず、判断の根拠としてこの work item に残す。skill には対応表だけを書く。
2. `implement-work-item` skill の第 8 段を書き換える。
3. `docs/development/specification-first-workflow.md` の検証のはしごを同じ内容にそろえる。
4. `mise run check-agent-guidance` と `mise run verify` を通す。

実装内容を変える未決事項はない。

## Tasks

- [x] T001 [Docs] `.agents/skills/implement-work-item/SKILL.md` の第 8 段から「Defer lint and aggregate gates to final verification」を外し、層と検査の対応表を入れた。冷状態の lint を 1 回払う指示は第 1 段の読み取りと並べた。検査: `mise run check-agent-guidance` (必須マーカーと Acceptance RED → Unit RED → GREEN → refactor の順序を保つこと)。
- [x] T002 [Docs] `docs/development/specification-first-workflow.md` 第 5 節の検証のはしごへ、作業開始時の `lint-go` を第 1 段、層ごとの検査を第 5 段として入れ、なぜ最終ゲートまで待たないかを段の後に書いた。検査: `mise run check-links`。
- [x] T003 [Verify] `mise run check-agent-guidance`、`mise run check-links`、`mise run verify` を通した。

## Verification

- Acceptance RED: 変更前の `implement-work-item` skill 第 8 段が「Defer lint and aggregate gates to final verification」と書いていること、および `lint-go` の 1 ファイル編集後の実測が 5.3 秒であることを観測する。指示が前提としている費用が現存しない。
- `mise run check-agent-guidance`
- `mise run verify`

## Risk Notes

- 実測は 1 台の機械の 1 回の観測である。ほかの機械の絶対値をこの表と比べない。ただし判断が変わるのは lint が数十秒に戻る場合だけで、その場合はキャッシュが効いていない別の問題である。
- 検査を前倒しすると、実装の途中で lint の指摘を直すことになり、その挙動がまだ GREEN でない状態で編集が混ざる。混ざるのを避けるため、走らせるのは「1 挙動が GREEN になった直後」であって「編集のたび」ではない。
- 手順を細かくすると、手順そのものを読む時間が増える。対応表を 3 行に収め、条件と対象だけを書く。
- `check-contract-drift` は作業ツリーに残っている生成物を読む。仕様を書き換えた直後に走らせると、前回の生成物に対して `ok` を出しうる。この性質は wi-497 の Risk Notes が記録しており、前倒しでも変わらない。skill には仕様生成の後に走らせる順序で書く。

## Completion

- **Completed At**: 2026-09-10
- **Summary**:
  `mise run spec-diff` は main に対して規範仕様の差分を報告しない。変わったのは実装中の手順だけである。
  `implement-work-item` skill の第 8 段は、lint と集約ゲートを最終検証まで遅らせる指示を捨て、触った層と直後に走らせる検査の対応表を持つようになった。
  冷状態の `lint-go` を 1 回払う指示は第 1 段の読み取りと並ぶ。
  `docs/development/specification-first-workflow.md` の検証のはしごは、作業開始時の `lint-go` を第 1 段、層ごとの検査を第 5 段として持ち、段の数は 6 から 8 になった。
- **Acceptance RED Evidence**:
  - **Test**: `sed -n '43,48p' .agents/skills/implement-work-item/SKILL.md` による第 8 段の観測と、1 ファイル編集後の `mise run lint-go` の実測。
  - **Requirement**: N/A: 実装中の手順だけを変える tooling 変更であり、対応する規範要求が無い。
  - **Observed Failure**: 第 8 段は「Defer lint and aggregate gates to final verification unless the work changes those gates.」と書いていた。その指示が前提とする費用を測ると、温状態 1.64 秒、`backend/saml/module.go` に 1 行足した後 5.25 秒だった。遅らせる根拠となる費用は現存しない。
  - **Detection Reason**: 観測は指示の文面と、その文面が前提とする費用の 2 つを別々に読む。文面だけを読むなら、費用が今も高い場合と区別がつかない。費用だけを測るなら、指示がすでに直っている場合と区別がつかない。両方を読んで初めて「高い費用を前提とした指示が、費用の消えた後も残っている」という不一致を指せる。
- **Unit RED Evidence**:
  - **Test**: `N/A: 文書だけの変更であり、内側の挙動を持つ単位が無い。` 代わりに走った検査は `mise run check-agent-guidance` と `mise run check-links` である。
  - **Requirement**: N/A: 上と同じ。
  - **Observed Failure**: どちらも RED にならなかった。`check-agent-guidance` が守るのは必須マーカーの存在と `Acceptance RED → Unit RED → GREEN → refactor` の出現順序であり、第 8 段の費用前提はその対象外である。この不一致を機械が捕まえられないことが、work item として記録する理由そのものである。
  - **Detection Reason**: 変更後も両検査は `ok` を返す。すなわち編集は、skill が保つべき構造をひとつも壊していない。ただし変更が正しいことの根拠ではなく、壊していないことの根拠である。正しさの根拠は上の Acceptance RED が持つ。
- **Change-Resistance Results**:
  `N/A: risk は low であり、証拠契約は change-resistance を選ばない。`
- **Verification Results**:
  - `mise run check-agent-guidance` - passed
  - `mise run check-links` - passed
  - `mise run verify` - passed (19.35s)
  - `mise run test-ui-e2e` - `N/A: 文書だけの変更であり、ブラウザへ届かない。`
