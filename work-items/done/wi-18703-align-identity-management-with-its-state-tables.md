---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-03
priority: p2
depends_on: [wi-96676-transcribe-implicit-specifications-of-identity-management, wi-21670-move-identity-management-to-the-feature-and-design-layout]
change_kind: bugfix
evidence_policy: risk-based-v3
documentation_impact:
  level: release_note
  reason: 管理 API の無効化と再有効化に 409 の拒否が加わり、エクスポートの保持期限の起点と成果物の削除が変わるので、API の利用者と運用者へ知らせる。
  references:
    - { kind: release_note, path: docs/releases/changes/wi-18703-align-identity-management-with-its-state-tables.md }
initial_context:
  specification:
    - docs/domain/identity-management/principals/user/README.md
    - docs/domain/identity-management/principals/user/lifecycle.md
    - docs/domain/identity-management/bulk-transfer/data-export/README.md
    - docs/domain/identity-management/bulk-transfer/csv-transfer/README.md
    - docs/design/data/lifecycle.md
  typespec: [IdMagic.Contract.UserNotPendingDeletionError]
  source:
    - backend/idmanagement/user/usecases/admin_users.go
    - backend/idmanagement/user/handlers_http/admin_user_handler.go
    - backend/idmanagement/usecases/data_export.go
    - backend/idmanagement/ports/csv_artifact.go
    - backend/idmanagement/db_postgres/csv_artifacts.go
    - backend/cmd/internal/bootstrap/retention.go
    - backend/cmd/idmagic-batch/main.go
  tests: [backend/idmanagement/user/usecases, backend/idmanagement/usecases, backend/idmanagement/db_postgres]
  stop_before_reading: [frontend]
affected_spec:
  - { path: docs/domain/identity-management/principals/user/lifecycle.md, requirement: REQ-IDMANAGEMENT-046 }
  - { path: docs/domain/identity-management/bulk-transfer/data-export/README.md, requirement: REQ-IDMANAGEMENT-079 }
  - { path: docs/domain/identity-management/bulk-transfer/csv-transfer/README.md, requirement: REQ-IDMANAGEMENT-080 }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.DisableAdminUser }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.EnableAdminUser }
  - { path: spec/contexts/identity-management/models.tsp, symbol: IdMagic.Contract.UserPendingDeletionError }
primary_use_cases:
  - id: pending-deletion-user-cannot-be-disabled
    requirement: REQ-IDMANAGEMENT-046
    observable_result: 削除予約中の User の無効化と再有効化が 409 `user_pending_deletion` で拒否され、User は `PendingDeletion` のまま残る。
    unit_test:
      path: backend/idmanagement/user/usecases/user_rules_test.go
      name: TestSetUserDisabledRefusesAPendingDeletionUser
      task: test-go-race
    e2e_test:
      path: backend/idmanagement/handlers_http/lifecycle_refusal_effects_test.go
      name: TestDisableAndEnableRefuseAPendingDeletionUser
      task: test-go-race
    unit_fault_model: 無効化が `PendingDeletion` の User を `Disabled` にし、削除予約の猶予期間の判定を迂回する。
    e2e_fault_model: ハンドラーが拒否のエラーを 409 に写さず、500 を返す。
  - id: export-expires-from-completion
    requirement: REQ-IDMANAGEMENT-079
    observable_result: 成功したエクスポートの `expires_at` は完了の時刻に 30 日を加えた時刻であり、その時刻までダウンロードできる。
    unit_test:
      path: backend/idmanagement/usecases/data_export_rules_test.go
      name: TestDataExportExpiresThirtyDaysAfterCompletion
      task: test-go-race
    e2e_test:
      path: backend/idmanagement/handlers_http/export_refusal_effects_test.go
      name: TestExportStaysDownloadableUntilThirtyDaysAfterCompletion
      task: test-go-race
    unit_fault_model: 期限を作成の時刻から数え、生成に時間のかかったエクスポートを早く期限切れにする。
    e2e_fault_model: 参照の応答が、作成の時刻から数えた `expires_at` を返す。
  - id: retention-sweep-deletes-old-csv-artifacts
    requirement: REQ-IDMANAGEMENT-080
    observable_result: 保持期限の削除のバッチが、作成から 30 日を過ぎた CSV の成果物を消し、それより新しい成果物を残す。
    unit_test:
      path: backend/idmanagement/usecases/csv_artifact_retention_test.go
      name: TestPurgeExpiredCSVArtifactsDeletesOnlyArtifactsPastRetention
      task: test-go-race
    e2e_test:
      path: backend/idmanagement/db_postgres/csv_artifacts_test.go
      name: TestCSVArtifactStoreDeletesArtifactsCreatedBeforeTheCutoff
      task: test-go-race
    unit_fault_model: 削除の境界を誤り、保持期限の内側の成果物まで消す。
    e2e_fault_model: 成果物の行を消しても分割片が残る、または保持期限の内側の行まで消す。
---

# IdManagement の実装を状態遷移表に合わせる

## 動機

IdManagement の既存コードを書き起こしたとき、状態遷移表と実装が食い違う点を見つけた。
状態遷移表は規範なので、現在の挙動を規則として書き起こさず、この work item で扱う。
製品は未リリースなので、実装を状態遷移表に合わせる。

| 状態機械 | 状態遷移表 | 実装 |
| --- | --- | --- |
| `UserLifecycle` | `PendingDeletion` から出る遷移は、`UserRestored` による `Active` と、`UserDeleted` による `Deleted` だけである | 無効化は `PendingDeletion` の User を `Disabled` にし、再有効化は `PendingDeletion` の User を猶予期間も確かめずに `Active` にする。どちらも `UserRestored` を発行しない |
| `DataExportLifecycle` | `succeeded` から `expired` への遷移のガードは `completed_at` からの経過であり、`expired` ではファイル本体を完全削除する | 期限を `created_at` に 30 日を加えた時刻で判定する。成果物を消す処理がない |

CSV の成果物を消す経路は、エクスポートだけでなくインポートのペイロードと行のエラーのページにもない（システムの[データライフサイクル設計](../../docs/design/data/lifecycle.md)が記録している）。

## 対象範囲

- `PendingDeletion` の User の無効化と再有効化を、409 と `user_pending_deletion` で拒否する。管理 API の契約に 409 を加える。
- エクスポートの保持期限を、完了の時刻から数える。`expires_at` は `succeeded` のエクスポートにだけ返す。
- 作成から 30 日を過ぎた CSV の成果物（エクスポートのファイル、インポートのペイロード、行のエラーのページ）を、Batch の `retention-sweep` で消す。

## 対象外

- 状態の追加と削除。
- `Locked`、`Staged`、`Suspended` など、状態遷移表にない `UserStatus` の値の扱い。

## 設計

### 削除予約中の User の無効化と再有効化

`SetUserDisabled` は、対象が `PendingDeletion` なら、ほかの判定（自分自身の無効化、すでにその状態）より前に `ErrUserPendingDeletion` を返す。
ハンドラーはそれを 409 と `user_pending_deletion` に写す。
削除予約を取り消す経路は、猶予期間を確かめる復元だけにする。

### エクスポートの保持期限

`exportViewFromJob` は、`succeeded` のジョブだけに `expires_at` を置き、その値を完了の時刻（ジョブの `updated_at`）に `DataExportTTL` を加えた時刻にする。
`succeeded` のジョブをその時刻以降に読むと `expired` として返し、ダウンロードできない。

### CSV の成果物の削除

```go
// ports
type CSVArtifactPurger interface {
    DeleteCSVArtifactsCreatedBefore(ctx context.Context, cutoff time.Time) (int64, error)
}

// usecases
const CSVArtifactRetention = 30 * 24 * time.Hour
func PurgeExpiredCSVArtifacts(ctx context.Context, purger ports.CSVArtifactPurger, now time.Time) (int64, error)
```

成果物は、エクスポートでは完了の時点で、インポートではプレビューと適用の時点で作る。
作成の時刻から 30 日で消すと、エクスポートのファイルは完了の時刻から 30 日の保持期限と同じ時点で消え、インポートのペイロードはプレビューのジョブの記録と同じ期間で消える。
時刻は Batch の起動時刻を引数で受け取り、ストアは境界の時刻より前に作った成果物とその分割片を消す。
Batch の `RunRetentionSweepOnce` は、Authentication の保持期限の削除に続けて、成果物の削除を呼ぶ。

### 採らない案

| 案 | 採らない理由 |
| --- | --- |
| 状態遷移表に `PendingDeletion` からの無効化と再有効化を加える | 再有効化が猶予期間の判定と `UserRestored` を迂回する経路になる |
| 期限切れのエクスポートを参照したときに成果物を消す | 誰も参照しないエクスポートの成果物が残り続ける |
| ジョブの記録を読み、参照されなくなった成果物だけを消す | ジョブの種類ごとに成果物の参照の置き場所が違い、削除の判定が成果物ストアの外へ広がる |

## 計画

1. 規則と契約を先に変える（REQ-046 の追加、REQ-079 と REQ-080 の新設、TypeSpec の 409）。
2. 削除予約中の User の拒否を実装する。
3. エクスポートの保持期限の起点を直す。
4. 成果物の削除を、ストア、ユースケース、Batch の順に実装する。
5. 設計文書（データ、リスク、システムのデータライフサイクル設計）を改める。

## タスク

- [x] T001 [Spec] 規則と TypeSpec を変える。
  Acceptance RED：`mise run check-spec` が、追加した EX-IDMANAGEMENT-046-04、079-01、080-01 をテストが引いていないこと、REQ-080 の担保手段が `backend/` に宣言されていないことを報告した。
- [x] T002 [App] 削除予約中の User の無効化と再有効化を拒否する。
  Unit RED：`TestSetUserDisabledRefusesAPendingDeletionUser` が `err=<nil>, want ErrUserPendingDeletion` で失敗することを確かめてから実装した。E2E の `TestDisableAndEnableRefuseAPendingDeletionUser` は実装の後に書いたので、ハンドラーの写像を外す誤りを注入し、`status=500` で失敗することを確かめた。
  実行：`mise run test-go-test -- ./backend/idmanagement/user/usecases <test>`、`mise run test-go-test -- ./backend/idmanagement/handlers_http <test>`、`mise run lint-go`、`mise run check-contract-drift`。
- [x] T003 [App] エクスポートの保持期限を完了の時刻から数える。
  Unit RED：`TestDataExportExpiresThirtyDaysAfterCompletion` が、`queued` のエクスポートが作成から数えた `expires_at` を返すことで失敗するのを確かめてから実装した。実装すると、作成の時刻だけを遡らせて期限切れを作っていた既存の `TestExpiredUserExportRefusesDownloadAndReturnsNoCSV` が `status="succeeded"` で失敗したので、作業台の完了の時刻も同じだけ遡らせるよう直した。E2E の `TestExportStaysDownloadableUntilThirtyDaysAfterCompletion` は実装の後に書いたので、起点を作成の時刻へ戻す誤りを注入し、`status="expired"` で失敗することを確かめた。
  実行：`mise run test-go-test -- ./backend/idmanagement/usecases <test>`、`mise run test-go-package -- ./backend/idmanagement/handlers_http`、`mise run lint-go`。
- [x] T004 [App] CSV の成果物を保持期限の削除で消す。
  Unit RED：`TestPurgeExpiredCSVArtifactsDeletesOnlyArtifactsPastRetention` が `deleted=0, want 1` で失敗し、PostgreSQL の `TestCSVArtifactStoreDeletesArtifactsCreatedBeforeTheCutoff` が同じく失敗し、Batch の配線の `TestRetentionSweepDeletesCSVArtifactsPastRetention` が「保持期限を過ぎた成果物が残った」で失敗することを、それぞれ確かめてから実装した。PostgreSQL のテストは成果物の行の作成の時刻を SQL で遡らせ、境界の前後と分割片の削除を確かめる。メモリ版は保存の時刻を差し替えて境界を確かめる。
  実行：`mise run test-go-test -- <package> <test>`（PostgreSQL はサンドボックスの外）、`mise run sqlc-generate`、`mise run lint-go`、`mise run test-go-changed`。
- [x] T005 [Docs] 設計文書を改める。
  N/A: 製品の規範 ID はない。Context の設計（アーキテクチャの実行時の流れ、データ、リスク）、データエクスポートの設計、システムのデータライフサイクル設計を、是正後の振る舞いに合わせた。`mise run check` が、作業項目の `documentation_impact` の不足と、TypeSpec のモデルが `affected_spec` にないことを RED として検出し、リリース文書を加えて GREEN にした。
- [x] T006 [Verify] 変更を検証する。
  `mise run check`、`mise run test-go-changed`、`mise run test-go-mutation -- backend/idmanagement/usecases`、`mise run verify` が通る。

## 検証

- `mise run check`
- `mise run verify`

## リスク

- 削除予約中の User の無効化を拒否すると、その操作に頼る管理者の手順が失敗する。製品は未リリースなので、移行は要らない。
- 成果物を消すと、30 日より前のプレビューを指定した適用は、ペイロードの照合で失敗する。プレビューのジョブの記録も同じ期間で消えるので、適用の可否の境界は変わらない。

## 完了

- **Completed At**: 2026-10-03
- **Summary**:
  `mise run spec-diff -- work-item/wi-21670` は、追加した規則 REQ-IDMANAGEMENT-079 と 080、変更した規則 REQ-IDMANAGEMENT-046、追加した TypeSpec の `UserPendingDeletionError`、変更した `DisableAdminUser` と `EnableAdminUser` を報告する。
  IdManagement の実装を `UserLifecycle` と `DataExportLifecycle` の状態遷移表に合わせた。
  削除予約中の User の無効化と再有効化は、409 と `user_pending_deletion` で拒否し、User を変えない。
  データエクスポートの保持期限を、作成の時刻ではなく完了の時刻から 30 日にし、`expires_at` は `succeeded` のエクスポートにだけ返す。
  Batch の `retention-sweep` が、作成から 30 日を過ぎた CSV の成果物（エクスポートのファイル、インポートのペイロード、行のエラーのページ）を分割片ごと消す。
- **Primary Use Case Evidence**:
  - id: pending-deletion-user-cannot-be-disabled
    unit_red: テスト `TestSetUserDisabledRefusesAPendingDeletionUser` が `err=<nil>, want ErrUserPendingDeletion` で失敗した。
    e2e_red: テスト `TestDisableAndEnableRefuseAPendingDeletionUser` は実装の後に書いたので RED を観測していない。代わりに下の誤りの注入で検出を確かめた。
    unit_fault_injection: 実装の `SetUserDisabled` の `PendingDeletion` の判定を消すと、単体テストが同じ失敗に戻る（実装前の RED と同じ状態）。
    e2e_fault_injection: ハンドラーの `ErrUserPendingDeletion` の写像を外すと、`TestDisableAndEnableRefuseAPendingDeletionUser` が `status=500` で失敗した。
  - id: export-expires-from-completion
    unit_red: テスト `TestDataExportExpiresThirtyDaysAfterCompletion` が、`queued` のエクスポートが作成から数えた `expires_at` を返すことで失敗した。
    e2e_red: 実装すると、作成の時刻だけを遡らせて期限切れを作っていた `TestExpiredUserExportRefusesDownloadAndReturnsNoCSV` が `status="succeeded"` で失敗した。`TestExportStaysDownloadableUntilThirtyDaysAfterCompletion` は実装の後に書いた。
    unit_fault_injection: 変異テストで、`succeeded` のときだけ `expires_at` を置く条件の否定を `TestDataExportExpiresThirtyDaysAfterCompletion` が検出した。
    e2e_fault_injection: 期限の起点を作成の時刻へ戻すと、`TestExportStaysDownloadableUntilThirtyDaysAfterCompletion` が `status="expired"` で失敗した。
  - id: retention-sweep-deletes-old-csv-artifacts
    unit_red: テスト `TestPurgeExpiredCSVArtifactsDeletesOnlyArtifactsPastRetention` が `deleted=0, want 1` で失敗した。
    e2e_red: テスト `TestCSVArtifactStoreDeletesArtifactsCreatedBeforeTheCutoff` が `deleted=0, want 1` で失敗し、Batch の配線の `TestRetentionSweepDeletesCSVArtifactsPastRetention` が「保持期限を過ぎた成果物が残った」で失敗した。
    unit_fault_injection: 変異テストで、`PurgeExpiredCSVArtifacts` の境界の計算の変異をテストが検出した。
    e2e_fault_injection: Batch の配線を外した状態（実装前）で、`TestRetentionSweepDeletesCSVArtifactsPastRetention` が失敗することを確かめた。
- **Change-Resistance Results**:
  `mise run test-go-mutation -- backend/idmanagement/usecases` の生き残りは 6 件で、どれも今回変えていない箇所（エクスポーターが配線されていない場合の分岐、`failed` のエラーコードの写し、結果のない `succeeded` のジョブ）にある。今回変えた行の変異は、すべてテストが検出した。
- **Verification Results**:
  - `mise run check` - 成功
  - `mise run test-go-changed` - 成功
  - `mise run verify` - 成功
