---
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-13
priority: p2
depends_on: []
change_kind: tooling
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: "開発者向けの検証ツールと手順だけを変更し、利用者へ知らせる製品差分を生まない。"
  references: []
initial_context:
  specification: []
  typespec: []
  source:
    - WORK_ITEM_FORMAT.md
    - mise.toml
    - docs/development/testing.md
    - docs/development/specification-first-workflow.md
    - docs/development/go-mutation-testing-tool-evaluation.md
    - backend/idmanagement/group/domain/dynamic_group_rule.go
    - backend/idmanagement/group/domain/group_csv.go
  tests:
    - tools/check/src/mise-config.test.ts
    - backend/idmanagement/group/domain/dynamic_group_rule_test.go
    - backend/idmanagement/group/domain/group_csv_test.go
  stop_before_reading:
    - frontend
    - spec
    - docs/contexts
    - backend/cmd
    - backend/shared
spec_impact:
  kind: none
  reason: "変異テストの実行系と検算方法だけを変え、製品の観測可能な振る舞いも公開契約も変えない。"
---

# Go の変異テストを gomutants へ移行し、判定を検算できるようにする

## Motivation

[[wi-493-mutation-testing-tool-for-change-resistance-evidence]] は Gremlins 0.6.0 を選び、`mise run test-go-mutation -- <package-directory>` を導入した。
当時の gomutants は公開された直後で、固定して採用できる版と実績がなかった。

その後、gomutants 0.6.1 が公開され、overlay、対象 package の絞り込み、stable mutant ID、永続キャッシュを備えた実行系を IdMagic で試せる状態になった。
[Go mutation testing tool の再評価](../../docs/development/go-mutation-testing-tool-evaluation.md)では、`backend/idmanagement/group/domain` に Gremlins と同系統の 5 operator を適用し、gomutants が約 22.0 秒、Gremlins が約 30.5 秒で完了した。
同じ再評価で、gomutants は既知の境界値不足を lived と判定したが、現在の Gremlins は covered mutant 56 件をすべて killed と判定した。

Gremlins 0.6.0 は終了コード 2 だけを not viable として扱い、終了コード 1 を killed として扱う。
Go の compile error と setup failure も終了コード 1 になり得るため、今回の全 killed という結果をテストの検出結果として採用できない。
小さな独立 package では lived を正しく報告したため、Gremlins 全般が常に誤判定するという主張ではない。
IdMagic の change-resistance evidence に使う経路で、判定を個別に検算できないことが問題である。

## Scope

- `mise.toml` が固定する変異テストツールを Gremlins 0.6.0 から gomutants 0.6.1 へ変更する。
- `test-go-mutation` という task 名と、package directory を必須の第 1 引数にする呼び出し方を維持する。
- report の stable mutant ID を指定して 1 件だけ再実行する task を追加する。
- worker の既定値を 2 とし、Gremlins と比較可能な 5 operator を既定の集合にする。
- 永続キャッシュを既定では無効にし、明示した cache file に限って反復実行で利用できるようにする。
- lived と not viable を区別できる小さな canary package と検査を追加する。
- ツールを固定する検査、開発文書、task の利用方法を gomutants に合わせて更新する。

## Out of Scope

- 既定の operator を gomutants の全 28 種類へ広げること。
- 現在見つかっている survivor をこの work item ですべて killed にすること。
- mutation score の閾値と、Pull Request の必須 gate を導入すること。
- gomutants の import 先を含む cache invalidation を IdMagic 側で再実装すること。
- Ooze の未リリース次版を再評価すること。
- UI の TypeScript に変異テストを導入すること。

## Design

### 既定の実行契約

`mise run test-go-mutation -- <package-directory> [workers] [cache-file] [report-file]` を公開する。
`workers` の既定値は 2、`cache-file` の既定値は `off` とする。
同一の変更上で survivor を調べ直す場合だけ、開発者が `/tmp` などの cache file を明示する。
`report-file` の既定値は一時領域の `idmagic-mutation-report.json` とし、repository root へ生成物を残さない。

`mise run test-go-mutation-mutant -- <package-directory> <mutant-id> [report-file]` は cache を使わず、指定した stable mutant ID だけを再実行する。

既定の operator は `ARITHMETIC_BASE`、`CONDITIONALS_BOUNDARY`、`CONDITIONALS_NEGATION`、`INCREMENT_DECREMENT`、`INVERT_NEGATIVES` の 5 種類とする。
これは Gremlins の既定集合に合わせた移行境界であり、operator の違いと runner の違いを同時に持ち込まないためである。
全 operator の価値は survivor の仕分け量を確認してから別の work item で判断する。

cache を既定で無効にするのは、gomutants 0.6.1 の cache key が対象 package の import 先にある変更を完全には追跡しないためである。
cache file を明示できる余地は残し、同じ source と test を変更せずに反復する局面では約 1.7 秒まで短縮できた実測を利用する。
依存 package を変更した後は、新しい cache file を使うか cache を無効にする。

### 判定 canary

`tools/check/testdata` に、意図的にテストが殺さない境界値 mutant と、文字列演算を書き換えると compile error になる mutant を持つ小さな Go package を置く。
canary 検査は JSON report を読み、前者が `LIVED`、後者が `NOT VIABLE` であることを確認する。
同じ入力を cache ありで再実行し、再利用後も分類が変わらないことを確認する。

canary は mutation score を保証する検査ではない。
runner がテスト失敗、survivor、compile error を混同しないことを、既知の入力で検算する境界である。
通常の `check` と `verify` には入れず、ツール更新時と change-resistance evidence を作る前に手動で実行する。

### 採用しない案

Gremlins の継続利用は、IdMagic で観測した全 killed の原因を利用側から判別できないため採用しない。
worker ごとに約 1.1 GB の module を複製する実行方式も、手動実行の固定費を残す。

Ooze 0.2.0 は coverage prefilter、変更差分による絞り込み、永続結果 cache を持たず、mutant ごとに repository を materialize するため採用しない。
Mutago 2.9.5 は operator の種類と変更行への絞り込みに優れるが、永続結果 cache がなく、今回優先する短い反復では gomutants に劣るため次点に置く。

## Plan

1. canary package と判定検査を先に追加し、現在の実行経路では期待する分類を確認できない RED を記録する。
2. `mise-config.test.ts` の期待を gomutants の pin、5 operator、worker 数、cache の既定値、gate 外という契約へ更新し、RED を確認する。
3. `mise.toml` の tool pin と `test-go-mutation` task を gomutants 0.6.1 へ置き換える。
4. canary と `backend/idmanagement/group/domain` を実行し、分類と実行時間を記録する。
5. 開発文書へ task の引数、cache の制約、単一 mutant の再実行方法を反映する。
6. repository の検査を通し、Gremlins 固有の設定と説明が残っていないことを確認する。

実装内容を変える未決事項はない。

Acceptance RED は、現在の `mise run test-go-mutation -- backend/idmanagement/group/domain` が、単独では lived と確認できる境界値 mutant を含めて covered mutant 56 件をすべて killed と報告することである。
移行後は同じ境界値 mutant が lived、文字列演算の mutant が not viable と分類されなければならない。

Unit RED は、`mise-config.test.ts` に gomutants 0.6.1、既定の 5 operator、worker 数 2、cache off を要求する表明を追加し、現在の Gremlins 設定に対して失敗することである。

## Tasks

- [x] T001 [Acceptance] 既知の lived と not viable を持つ canary package を追加し、移行前の判定を記録する。
- [x] T002 [Unit] `mise-config.test.ts` を新しい task 契約へ更新し、Gremlins 設定に対する RED を確認する。
- [x] T003 [Tooling] `mise.toml` の pin と `test-go-mutation` task を gomutants 0.6.1 へ置き換える。
- [x] T004 [Tooling] canary の JSON report を検算する手動 task を追加する。
- [x] T005 [Docs] 調査文書とテスト手順へ、最終的な task 契約、cache の制約、単一 mutant の再実行方法を反映する。
- [x] T006 [Verify] canary と代表 package を実行し、狭い検査と repository 全体の検査を通す。

## Verification

- `mise run test-tools -- check/src/mise-config.test.ts`
- `mise run check-go-mutation-tool`
- `mise run test-go-mutation -- backend/idmanagement/group/domain`
- cache file を明示した同じ実行を 2 回行い、2 回目が結果を再利用して同じ分類を返すことを確認する。
- `mise run check-command-map`
- `mise run check-links`
- `mise run verify`

## Risk Notes

- gomutants 0.6.1 は公開から日が浅い 0.x 系である。
  版を固定し、既知の verdict を検算する canary を置くことで、更新による分類の変化を観測できるようにする。
- cache の既知制約を無視すると、依存 package の変更前の verdict を再利用する可能性がある。
  既定を `off` とし、cache file を明示した反復に用途を限定する。
- tool 間では mutant の分割と重複除去が異なるため、総 mutant 数と mutation score の一致を移行条件にしない。
  lived と not viable の既知の個別例、および代表 package の survivor 一覧を比較する。
- 変更は tool pin、task、検査、文書に閉じており、Gremlins の pin と task 定義へ戻せる。

## Completion

- **Completed At**: 2026-09-13
- **Summary**:
  `mise run spec-diff` は main に対する規範仕様差分がないと報告した。
  change-resistance evidence の runner を Gremlins 0.6.0 から gomutants 0.6.1 へ変更し、既定の 5 operator、worker 数 2、cache off を固定した。
  通常実行は JSON report を一時領域へ出し、明示した cache file による反復と stable mutant ID 1 件の再実行を `mise` task から利用できる。
  固定 canary は lived と not viable を初回と cache 再利用後の両方で検算する。
- **Acceptance RED Evidence**:
  - **Test**: `mise run test-go-mutation -- backend/idmanagement/group/domain`。
  - **Requirement**: N/A: 製品の観測可能な振る舞いを変えないツーリング変更である。
  - **Observed Failure**: Gremlins 0.6.0 は、単独で survivor と確認できる `dynamic_group_rule.go` の境界値 mutant を含む covered mutant 56 件をすべて killed と報告した。
  - **Detection Reason**: killed と compile error を区別できない runner は、テストが検出した故障を過大に報告する。
    移行後の同じ package は killed 38、lived 14、not viable 1、not covered 16 を報告し、既知の境界値 mutant 1 件の再実行も lived を返した。
- **Unit RED Evidence**:
  - **Test**: `mise run test-tools-file -- check/src/mise-config.test.ts`。
  - **Requirement**: N/A: tool pin と task 契約を固定する repository tooling test である。
  - **Observed Failure**: gomutants の呼び出し、0.6.1 の pin、5 operator、cache off、canary task を要求する 5 件が Gremlins 設定に対して失敗した。
    相対 directory の正規化、report の一時領域出力、単一 mutant task も各実装前に 1 件ずつ失敗した。
  - **Detection Reason**: 表明は tool 名だけでなく、誤判定と不要な repository 生成物を防ぐ実行境界、および既存の `backend/...` 呼び出しとの互換性を固定する。
- **Change-Resistance Results**:
  - JSON report の compile failure を `KILLED` に差し替えた入力を `verifyMutationCanary` が拒否することを単体テストで確認した。
  - 既知の境界値 survivor を `KILLED` に差し替えた入力も同じ検査が拒否した。
  - 実物の canary は初回と cache 再利用後の双方で lived 1、not viable 1 を返し、2 回目は 2 mutant の結果を再利用した。
- **Verification Results**:
  - `mise run test-tools-file -- check/src/mise-config.test.ts check/src/mutation-canary.test.ts` - 34 tests passed
  - `mise run check-go-mutation-tool` - lived 1、not viable 1、cached 2
  - `mise run test-go-mutation -- backend/idmanagement/group/domain` - killed 38、lived 14、not viable 1、not covered 16、14.81 秒
  - cache file を明示した代表 package の cold 実行 - 15.37 秒
  - 同じ cache file を使った warm 実行 - 53 件を再利用し、同じ分類で 1.83 秒
  - `mise run test-go-mutation-mutant` - 既知の境界値 mutant 1 件を lived として 3.1 秒で再現
  - `mise run check-command-map` - passed
  - `mise run check-links` - passed
  - `mise run verify` - passed
  - `mise run test-ui-e2e` は実行していない。
    変更は開発用 tool、repository 検査、文書、標準の package 探索から除外される canary に閉じており、ブラウザへ到達しない。
