---
depends_on: []
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-12
priority: p1
change_kind: tooling
spec_impact: { kind: none, reason: "被覆負債の例外管理と検査の実装を変更するだけであり、規範シナリオ、標準行、TypeSpec、製品の振る舞いは変更しない。検査が検出した未被覆 ID は、対応するテストまたは別の work item で扱う。" }
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: 開発者向けの検査手順を tools/check/README.md で更新し、利用者向けの製品変更はない。
  references: []
initial_context:
  specification: [docs/development/specification-first-workflow.md, WORK_ITEM_FORMAT.md]
  typespec: []
  source: [tools/check/src/check-documents.ts, tools/check/src/normative-coverage.ts, tools/check/src/registry.ts, tools/check/src/runner.ts, mise.toml, .github/workflows/idmagic-ci.yaml, tools/check/README.md]
  tests: [tools/check/src/normative-coverage.test.ts, tools/check/src/repository-checks.acceptance.test.ts]
  stop_before_reading: [backend, frontend, spec/idmagic.openapi.baseline.json]
---

# 被覆負債の baseline を廃し、Git 差分で台帳の追加を拒否する

## Motivation

`example-coverage-debt-baseline.json` と `standards-coverage-debt-baseline.json` は、導入時の ID 集合を固定し、debt 台帳へ初期集合に無い ID を追加することを拒否している。

しかし、被覆済みになって debt から消した ID は baseline に残り続けるため、後の変更でその ID を debt に戻せる。

この仕組みは「初期集合の外からの流入」は防ぐが、「解消済みの負債の再流入」は防がない。

被覆負債は一度解消した ID を再許可する理由を持たない。

基準リビジョンの debt 台帳と現在の台帳を比較すれば、baseline を保守せずに、初めて現れた ID と再流入した ID を同じく拒否できる。

## Scope

- `example-coverage-debt.json` と `standards-coverage-debt.json` の ID 集合が、Git の基準リビジョンにある各台帳の ID 集合の部分集合であることを検査する。
- 基準リビジョンに無い ID の追加を、初めての追加と再流入の区別なく拒否する。
- 各 entry の `reason` の更新は許可し、ID の追加だけを ratchet 違反とする。
- 2 つの coverage debt baseline ファイルと、`checkNormativeCoverage` の `debtBaseline` 入出力および読取処理を削除する。
- Git の基準リビジョンを明示的に受け取る `mise` タスクを追加し、通常の検証と CI が Pull Request の基準コミットまたは main への push の直前コミットに対して実行するようにする。
- `tools/check/README.md`、未完了の [[wi-495-burn-down-the-standards-coverage-debt]]、[[wi-496-burn-down-the-example-coverage-debt]]、[[wi-414-boundary-fitness-functions]] を新しい ratchet の責務に合わせる。

## Out of Scope

- `spec/idmagic.openapi.baseline.json` と API 互換性検査の変更。
- debt 台帳に残る具体例または標準行の消化。
- 完了済み work item の履歴記述の書き換え。
- Git の基準リビジョン自体を改変できる権限を持つ者に対するアクセス制御。

## Design

選ぶ設計は、基準リビジョンから読んだ debt の ID 集合を `before`、現在の debt の ID 集合を `current` として、`current - before` が空であることを検査する Git ratchet である。

`current - before` に含まれる ID は、仕様を新規に debt へ入れた場合と、過去に解消した ID を戻した場合のどちらでも失敗する。

台帳の entry が持つ `reason` は、実装と規範の不一致を発見したときに書き換える必要があるため、比較対象を ID に限る。

`checkNormativeCoverage` は現在のツリーだけを読み、各宣言 ID がテストまたは理由付き debt によって説明されることを引き続き検査する。

Git を読む ratchet はこの純粋な文書検査へ混ぜず、基準リビジョンを引数として受け取る別の検査にする。

ローカルでは既定の `main` を基準に実行する。

Pull Request の CI はイベントが示す base SHA を基準に実行し、main への push は push 前の SHA を基準に実行する。

このため CI の checkout は必要な基準コミットを必ず取得する。

baseline を固定の初期母数として残す案は採らない。

初期の件数は Git 履歴から復元できる一方、実行時の許可集合として残すと、解消済み ID を再許可する穴を作るためである。

baseline を debt と同時に削る案も採らない。

許可集合を別に保つ限り、その集合への再追加を別途防がなければならず、Git ratchet より状態が一つ多い。

## Plan

1. `RevisionReader.read(ref, path)` を Git 基準リビジョンから JSON を得る effect boundary とし、`addedDebtIds(before, current)` を ID 集合だけを比較する純粋関数にする。
2. ID の追加を失敗させ、削除と `reason` の変更を許す単体テストを RED で追加する。
3. 2 つの coverage debt baseline を削除し、文書被覆検査から baseline の型、読取、比較を外す。
4. Git ratchet を registry と `mise` の入口へ接続し、ローカルと CI が適切な基準 SHA を渡すようにする。
5. 正本文書と未完了 work item の説明を更新し、baseline を前提とする将来の設計を残さない。

## Tasks

- [x] T001 [Design] Git 基準リビジョンを読む effect boundary と、2 台帳に共通の ID 差分の型を定める。
- [x] T002 [Unit] 基準に無い ID の追加と解消済み ID の再追加を拒否し、ID の削除と `reason` の更新を許す RED を `tools/check` の単体テストへ追加する。
- [x] T003 [Tooling] coverage debt baseline ファイル、`debtBaseline`、`readDebtBaseline`、対応する既存テストを削除し、Git ratchet を registry と `mise` タスクへ接続する。
- [x] T004 [Acceptance] 基準コミットの台帳に無い ID を現在の 2 台帳のそれぞれへ追加した fixture または一時 Git repository を用意し、新検査が両方を失敗させることを確認する。
- [x] T005 [CI] Pull Request と main への push が適切な基準 SHA を取得して ratchet を実行するよう CI を更新する。
- [x] T006 [Docs] `tools/check/README.md` と未完了の関連 work item を、baseline ではなく Git ratchet が負債の追加を拒否する説明へ更新する。
- [x] T007 [Verify] `mise run check-work-items`、新しい ratchet、`mise run check-spec`、`mise run test-tools`、`mise run verify` を実行する。

## Verification

- 基準リビジョンに無い ID を `example-coverage-debt.json` へ追加すると、新しい ratchet が ID を報告して失敗する。
- 一度削除した ID を基準リビジョンより後で `example-coverage-debt.json` へ戻すと、新しい ratchet が失敗する。
- 同じ性質が `standards-coverage-debt.json` にも成り立つ。
- 既存 ID の `reason` だけを変更しても、新しい ratchet は成功する。
- coverage debt の baseline ファイルが存在せず、`checkNormativeCoverage` が baseline を受け取らない。
- `mise run check-spec` が、現在の debt とテスト名指しの整合を引き続き検査する。
- `mise run check-work-items`、`mise run test-tools`、`mise run verify` が成功する。

## Risk Notes

- **基準リビジョンを取得できない。** 基準 SHA を明示的に受け取り、CI の checkout 深さを必要な履歴まで広げる。ローカルで `main` が無い場合は、曖昧に成功させず基準を指定するよう失敗させる。
- **同じ変更で基準と現在を取り違える。** 基準から読む JSON と作業ツリーから読む JSON を別の effect boundary とし、一時 Git repository を使う受入検査で区別する。
- **理由を書き換えられなくなる。** 比較対象を `id` だけに限定する単体テストを置く。
- **OpenAPI baseline まで削除する。** 対象パスを `tools/check/*-coverage-debt-baseline.json` に限定し、API 互換性検査を Out of Scope に明記する。

## Completion

- **Completed At**: 2026-09-12
- **Summary**:
  `mise run spec-diff -- main` は規範仕様の変更なしと報告した。
  被覆負債の許可集合を baseline JSON から Git 基準 revision へ置き換えた。
- **Acceptance RED Evidence**:
  - **Test**: `tools/check/src/repository-checks.acceptance.test.ts` の `coverage debt ratchet > rejects additions to either ledger beyond the named Git revision` を `mise run test-tools -- repository-checks.acceptance.test.ts` で実行した。
  - **Requirement**: N/A: repository tooling has no normative product requirement.
  - **Observed Failure**: registry に `coverage-debt-ratchet` が無く、runner が unknown check selector として拒否した。
  - **Detection Reason**: 一時 Git repository の基準コミットと現在の二つの台帳を分け、両方へ追加した ID が利用者向けの runner 境界で検出されることを確認する。
- **Unit RED Evidence**:
  - **Test**: `tools/check/src/coverage-debt-ratchet.test.ts` の `addedDebtIds` を `mise run test-tools -- coverage-debt-ratchet.test.ts` で実行した。
  - **Requirement**: N/A: repository tooling has no normative product requirement.
  - **Observed Failure**: `coverage-debt-ratchet.ts` が存在せず、テストが module not found で失敗した。
  - **Detection Reason**: 基準に無い ID の追加を返し、既存 ID の理由更新と ID の削除を返さない純粋比較を区別する。
- **Change-Resistance Results**:
  一時 Git repository の現在台帳へ `EX-DEMO-002-01` と `RFC-DEMO-002` を追加する変異を投入した。
  `repository-checks.acceptance.test.ts` は両 ID を Git 基準に無いものとして検出した。
- **Verification Results**:
  - `mise run check-work-items` - passed
  - `mise run check-coverage-debt-ratchet -- HEAD` - passed
  - `mise run check-spec` - passed
  - `mise run test-tools` - passed
  - `mise run verify` - passed
