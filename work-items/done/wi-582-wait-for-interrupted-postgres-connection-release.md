---
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-16
priority: p1
depends_on: []
change_kind: maintenance
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: 利用者向けの振る舞いは変わらず、既存の接続解放テストを依存ライブラリの非同期破棄契約へ合わせるためである。
  references: []
initial_context:
  specification: [docs/domain/system/scenarios.feature.md#REQ-SYSTEM-012]
  typespec: []
  source: [backend/shared/storage/db_postgres/base.go]
  tests: [backend/shared/storage/db_postgres/base_test.go]
  stop_before_reading: [backend/shared/http, frontend]
spec_impact:
  kind: none
  reason: EX-SYSTEM-012-02 が定める接続解放の最終状態は変えず、依存ライブラリが非同期で破棄する接続をテストが待つようにするだけである。
---

# 中断された PostgreSQL 接続の解放完了をテストで待つ

## Motivation

期限超過した `QueryRow().Scan()` の接続は pgx から puddle の非同期破棄へ渡される。
テストが `Scan()` の直後にプール統計を読むと、破棄処理が完了する前の接続をリークと誤判定する。

## Scope

- 中断された単一行クエリの接続数が期限内に基準値へ戻ることを検査する。
- 待機に上限を設け、接続リーク時には失敗させる。

## Out of Scope

- `ResilientDB` の公開インターフェースとタイムアウト動作を変更しない。
- pgx または puddle の依存版を変更しない。

## Design

テストは接続数を短い間隔で再評価し、固定した上限までに取得数が基準値へ戻ることを要求する。
一度だけ固定時間を sleep する案は、必要以上にテストを遅くし、遅い環境で同じ競合を残すため採用しない。

## Plan

既存の失敗を再現し、接続解放の待機ヘルパーをテスト内へ追加して、対象テストとパッケージ全体を検証する。

## Tasks

- [x] T001 [Readiness] 規範シナリオ、ラッパー、pgx の接続解放経路、対象テストを確認する。
- [x] T002 [RED] `Scan()` 直後のプール統計が破棄中の接続を示すことを確認する。
- [x] T003 [Test] 上限付きで接続解放の最終状態を待つ。
- [x] T004 [Verify] 対象パッケージと全体を検証する。

## Verification

- `mise run test-go-test -- ./backend/shared/storage/db_postgres TestResilientDBQueryRowReportsDeadlineExceeded`
- `mise run test-go-package -- ./backend/shared/storage/db_postgres`
- `mise run verify`

## Risk Notes

待機上限が長すぎると実際のリークの検出が遅れ、短すぎると負荷の高い環境で誤失敗する。
通常は条件成立時に即座に戻り、上限到達時だけ失敗する検査とする。

## Completion

- **Completed At**: 2026-09-16
- **Summary**:
  `mise run spec-diff` は main に対する規範仕様差分がないと報告した。
  期限切れ接続の非同期破棄が完了するまで、対象テストが1秒の上限付きでプール統計を再評価するようになった。
- **Acceptance RED Evidence**:
  - **Test**: `TestResilientDBQueryRowReportsDeadlineExceeded`。
  - **Requirement**: REQ-SYSTEM-012
  - **Observed Failure**: `QueryRow().Scan()` は `context deadline exceeded` を返したが、直後の `AcquiredConns()` は破棄中の接続1件を示した。
  - **Detection Reason**: 期限超過後に接続数が基準値へ戻る最終状態を直接検査し、接続リークを検出する。
- **Unit RED Evidence**:
  - **Test**: N/A: 製品コードを変更せず、実 PostgreSQL と接続プールを使う既存の受け入れ境界だけを修正した。
  - **Requirement**: N/A: テストの観測タイミングを依存ライブラリの非同期破棄契約へ合わせる保守変更である。
  - **Observed Failure**: N/A: より狭い単体境界では puddle の非同期破棄を再現できない。
  - **Detection Reason**: 実プールの統計を使う対象テストが、接続解放の最終状態を観測する最小の適切な境界である。
- **Change-Resistance Results**:
  - 待機しない旧表明を代表故障とし、単独実行で取得済み接続1件を検出して RED になることを確認した。
  - 実際に接続が残り続ける場合は1秒後にも取得数が基準値へ戻らず失敗するため、リークの検出力を維持する。
  - 製品コードを変更していないため mutation test は実行していない。
- **Verification Results**:
  - `mise run test-go-test -- ./backend/shared/storage/db_postgres TestResilientDBQueryRowReportsDeadlineExceeded` - passed
  - `mise run test-go-package -- ./backend/shared/storage/db_postgres` - passed
  - `mise run lint-go` - 0 issues
  - `mise run test-go-changed` - passed
  - `mise run verify` - passed
