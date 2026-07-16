---
status: pending
authors: ["tn"]
risk: high
created_at: 2026-07-16
depends_on: [wi-229-scl-3-1-assurance]
---

# SCL・Architecture・実装・テスト・証跡を結ぶ追跡グラフを構築する

## Motivation
SCL、Go/TypeScript 実装、テスト、WI completion は個別には存在するが、相互対応を検証する仕組みがない。実装済みだが仕様がない endpoint、仕様はあるが実装・テストがない要素、古い revision に対する検証結果を現在も保証済みとして扱う問題を自動検出できる必要がある。

## Scope
- `verification/manifest.yaml` とその schema を追加する。
- context-qualified な SCL assurance obligation、Architecture module、実行可能 check、`just` recipe、evidence kind、対象 revision/artifact の対応を表す。
- `ra` CLI に workspace traceability/coverage 検査を追加する。
- `backend/shared/spec/assurance_manifest.go` の手書き台帳を manifest 由来 binding または manifest 直接検査へ置換する。
- Work Item schema に `change_kind`、`initial_context`、`affected_guarantees`、`spec_impact` の条件付き必須規則を追加する。
- 既存 pending WI を strict validation 導入前に移行する。

## Out of Scope
- 個々の不足テストの実装。
- Architecture の実 import 検査。
- 外部 SaaS への evidence upload。
- SCL 内へのリポジトリパス埋め込み。

## Plan
- SCL は proof obligation、verification manifest はリポジトリ固有 binding、CI artifact は実行結果という三層に分離する。
- report は `implemented_without_spec`、`specified_without_realization`、`missing_evidence`、`stale_evidence` を区別する。
- 新規 drift は error とし、既存 debt は owner・理由・期限を持つ baseline に限定する。
- feature/bugfix/operations WI は保証義務参照を必須にし、仕様非影響の変更は `spec_impact: none` と理由を必須にする。

## Tasks
- [ ] T001 [Schema] verification manifest と evidence reference の schema を定義する。
- [ ] T002 [RA] traceability graph の loader、参照解決、分類 report を実装する。
- [ ] T003 [WorkItem] WI schema と validator に仕様影響・保証義務の条件を追加する。
- [ ] T004 [Migration] 既存 pending WI と恒久 verification binding を移行する。
- [ ] T005 [Go] 手書き AssuranceManifest を廃止または派生化する。
- [ ] T006 [CI] `just verify` に traceability 検査を追加する。
- [ ] T007 [Verify] 欠落・stale・未知参照の negative test を含め全検証を通す。

## Verification
- `just test-tools`
- `just typecheck-tools`
- `just yaml-check`
- `just verify`

## Risk Notes
検査を一度に strict 化すると既存70件の pending WI と既存 debt で開発が停止する。report-only migration、期限付き baseline、strict gate の順で導入し、空配列や無期限例外による形式的回避を禁止する。
