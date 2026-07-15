---
depends_on: [wi-153-identity-lifecycle-workflows, wi-126-async-job-runner]
status: completed
authors: [tn]
risk: high
created_at: 2026-07-16
---

# User mutation から lifecycle workflow run を耐久的に capture・dispatch する

## Motivation

Workflow definition だけでは User の lifecycle event を実行へ変換できない。User 保存と同じ整合性境界で run を確定し、enqueue 障害後にも worker が回収できる必要がある。

## Scope

- `spec/contexts/identity-management.yaml` の `models.WorkflowRun` / `models.WorkflowStep` と `scenarios.User変更とWorkflowRunのcaptureは同じ整合性境界で確定しenqueue障害から回復する`
- `backend/identitymanagement/` に WorkflowRun/WorkflowStep の memory・PostgreSQL repository と一意制約を追加する。
- User create/update/status mutation から enabled revision を評価し、source occurrence dedup 付きで run を作成する。
- `lifecycle_workflow_run` JobKind、dispatcher、handler 登録、未 enqueue run の周期回収を追加する。

## Out of Scope

- action の副作用実装、管理 HTTP/UI、監査 read model。

## Plan

- run/step checkpoint を durable model として先に追加する。
- trigger capture と dispatcher を、失敗後の再走査で収束するよう実装する。
- action executor は port として注入し、次 WI で実装する。

## Tasks

- [x] T001 [Persistence] run/step repository と tenant-safe な dedup を実装する。
- [x] T002 [Trigger] User mutation と atomic run capture を統合する。
- [x] T003 [Jobs] dispatcher、handler、periodic recovery を実装する。
- [x] T004 [Verify] 重複配送、enqueue 障害回復、tenant isolation を検証する。

## Verification

- `just test-go`
- `just verify-go`

## Risk Notes

User 保存と run 作成を別 transaction にすると event が失われる。PostgreSQL では transaction adapter を明示し、memory でも同じ観測可能な dedup をテストする。

## Completion

- **Completed At**: 2026-07-16
- **Summary**:
  WorkflowRun / WorkflowStep の memory・PostgreSQL 永続化、occurrence 単位の tenant-safe dedup、User mutation と同一 transaction の capture を実装した。worker は未 enqueue run を周期回収し、dedup key 付き `lifecycle_workflow_run` Job へ安全に handoff する。
- **Affected Guarantees State**:
  enabled workflow に一致する User の create・属性変更・status 変更は、User 保存と同時に queued run / pending step を確定する。enqueue が一時失敗しても run は残り、後続 dispatcher が tenant 境界を保って回収する。action 実行は wi-218 の責務として未変更。
- **Verification Results**:
  - `just scl-render` — passed
  - `just test-go` — passed
  - `just verify` — passed
- **Evidence**:
  - 実行日: 2026-07-16
  - 実行環境: ローカル開発環境 (macOS)
  - 実行主体: Codex
  - 対象ソース版: main（コミット前作業ツリー）
  - 保存先: 外部成果物なし。上記検証結果を本記録に要約。
