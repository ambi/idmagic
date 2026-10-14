---
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-04
priority: p3
depends_on: []
change_kind: maintenance
evidence_policy: risk-based-v4
documentation_impact:
  level: none
  reason: "未公開であり、三つの値はどの操作も保存しないので、既存データと実利用の契約に影響しない。"
  references: []
affected_spec:
  - { path: spec/modules/identity-governance/models.tsp, symbol: IdMagic.Contract.UserStatus }
initial_context:
  specification:
    - docs/development/specification-first-workflow.md
    - docs/formats/work-item-format.md
  typespec: [IdMagic.Contract.UserStatus]
  source:
    - backend/idmanagement/domain/enums.go
    - backend/idmanagement/usecases/data_export.go
    - backend/idgovernance/internal/domain/lifecycle_workflows.go
    - frontend/src/features/admin-lifecycle-workflows/WorkflowDefinitionForm.tsx
  tests:
    - backend/idmanagement/agent/domain/agents_test.go
    - backend/idmanagement/usecases/data_export_rules_test.go
  stop_before_reading: [spec/modules/identity-management/main.tsp]
---

# どの遷移も到達しない UserStatus の値を消す

## 動機

`UserStatus` は、UserLifecycle の状態の表にない `locked`、`staged`、`suspended` を宣言している。
本番コードはどの値へも遷移しない。それでも IdentityGovernance の公開契約 `IdMagic.Contract.UserStatus` は三つの値を載せ、ライフサイクルワークフローのトリガーとデータエクスポートの絞り込みと管理画面の選択肢が受け付ける。
状態遷移表（マトリクス形式）から予測するモデルベースのテストは表の状態だけを試すので、これらの値は仕様にもテストにも現れないまま `Valid()` を通る。
未記載の振る舞いの分類では (c) 実装を直す、に当たる。

## 対象範囲

- `backend/idmanagement/domain/enums.go` から三つの値を消し、`Valid()` を状態の表の四つに限る。
- 三つの値を使うテストを、表の状態を使う形に直す。
- `IdMagic.Contract.UserStatus` と、管理画面のワークフロー定義フォームの選択肢と翻訳から三つの値を消す。
- データエクスポートの `status` の絞り込みは `Valid()` を使うので、三つの値を拒否するようになる。

## 対象外

- 状態の表へ新しい状態を加えること。
  必要になったら、仕様の変更として別に起票する。

## 検証

- `mise run test-go-changed`
- `mise run verify`

## リスク

- 保存済みの User が三つの値のどれかを持っていると、読み込みで拒否される。
  製品は未リリースであり、本番コードはこれらの値を書かないので、移行は要らない。

## 完了
- **完了日**: 2026-10-15
- **要約**:
  `mise run spec-diff` は `spec/modules/identity-governance/models.tsp:UserStatus` の変更だけを示す。
  `UserStatus` から `locked`、`staged`、`suspended` を消し、`Valid()` を状態の表の四つに限った。
  起票時の前提と違い、三つの値は公開契約 `IdMagic.Contract.UserStatus` に載っていたので、契約、ワークフロー定義フォームの選択肢、翻訳からも消した。
  利用者の確認を得て、契約から消す範囲で進めた。
  データエクスポートの `status` の絞り込みと、ライフサイクルワークフローのトリガーの `from_status` と `to_status` は、三つの値を拒否するようになった。
- **受け入れ RED の証拠**:
  - **テスト**: `backend/idmanagement/usecases/data_export_rules_test.go` の `TestStartDataExportRejectsDuplicateColumnsAndUnknownFiltersWithoutAJob`
  - **要件**: N/A: 消した値を名指す規範要件はない。契約の変更は `IdMagic.Contract.UserStatus` で追う。
  - **観測した失敗**: 実装前は `状態の表にない状態: err=<nil>, want ErrInvalidExportFilter` で失敗した。
  - **検出できる理由**: `suspended` を有効な値として残す実装では、絞り込みが受理されて `err` が nil になる。
- **単体 RED の証拠**:
  - **テスト**: `backend/idmanagement/agent/domain/agents_test.go` の `TestEnumValid`
  - **要件**: N/A: 消した値を名指す規範要件はない。
  - **観測した失敗**: 実装前は `domain.UserStatus(locked).Valid() = true, want false` と、`staged`、`suspended` の同じ形の失敗が出た。
  - **検出できる理由**: 三つの値のどれかを `Valid()` に残すと、その値の行が失敗する。
- **変更耐性の結果**:
  RED が故障注入を兼ねる。`Valid()` が三つの値を受理する実装前の状態で、上の 2 つのテストが失敗した。変更は値の削除だけで、ほかに分岐を加えていない。
- **検証結果**:
  - `mise run test-go-package` で `./backend/idmanagement/...` と `./backend/idgovernance/...` が成功した。
  - `mise run lint-go`、`mise run check-spec`、`mise run typecheck-ui`、`mise run format-check-ui` が成功した。
  - `mise run test-ui-unit-file -- src/features/admin-lifecycle-workflows/WorkflowDefinitionForm.test.tsx`: 6 件成功。
  - 最新の main へ rebase した後に `mise run check` と `mise run verify` が成功した。
