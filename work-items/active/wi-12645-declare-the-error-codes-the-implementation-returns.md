---
status: in_progress
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-04
priority: p2
depends_on: [wi-35451-rewrite-remaining-context-specifications-in-the-requirement-format]
change_kind: bugfix
affected_spec:
  - { path: spec/modules/workloadidentity/models.tsp, symbol: IdMagic.Contract.AgentWorkloadBindingPatternConflictError }
  - { path: spec/modules/workloadidentity/main.tsp, symbol: IdMagic.WorkloadIdentity.Operations.CreateAgentWorkloadBinding }
  - { path: spec/modules/identity-management/models.tsp, symbol: IdMagic.Contract.JobsUnavailableError }
  - { path: spec/modules/application/models.tsp, symbol: IdMagic.Contract.InvalidIconError }
  - { path: spec/modules/application/models.tsp, symbol: IdMagic.Contract.InvalidSignInPolicyError }
  - { path: spec/modules/application/main.tsp, symbol: IdMagic.Application.Operations.UploadApplicationIcon }
  - { path: spec/modules/application/main.tsp, symbol: IdMagic.Application.Operations.UpdateAppSignInPolicy }
  - { path: spec/modules/application/main.tsp, symbol: IdMagic.Application.Operations.UpdateTenantDefaultSignInPolicy }
  - { path: docs/modules/identity-management/features/data-export.md, requirement: REQ-IDMANAGEMENT-039 }
primary_use_cases:
  - id: export-without-job-service
    requirement: REQ-IDMANAGEMENT-039
    observable_result: ジョブ保存先のないエクスポート開始は 503 jobs_unavailable を返し、ジョブとイベントを残さない。
    boundary: acceptance
    test: { path: backend/idmanagement/internal/handlers_http/admin_data_export_handler_test.go, name: TestDataExportHTTPRejectsAnUnavailableJobService, task: test-go-race }
    fault_model: ジョブ保存先の未構成を 500 または別のエラーコードにする。
---

# 実装が返すエラーコードを、返す Context の TypeSpec で宣言する

## 動機

wi-35451 の書き直しで、要件に書いたエラーコードのうち、TypeSpec が宣言していないものと、返さない Context が宣言しているものが見つかった。
宣言のないエラーコードは、公開の契約（OpenAPI）に現れず、クライアントは依存してよいかを判断できない。

| エラーコード | 返す Context | 食い違い |
| --- | --- | --- |
| `workload_trust_bundle_not_found`、`agent_workload_binding_not_found`、`agent_workload_binding_pattern_conflict` | WorkloadIdentity | 宣言がない |
| `invalid_icon`、`invalid_sign_in_policy` | Application | 宣言がない |
| `jobs_unavailable` | IdManagement（データのエクスポート） | Jobs の TypeSpec が宣言している |

## 対象範囲

- 上の表のエラーコードを、返す Context の TypeSpec の操作のエラーとして宣言する。
- `jobs_unavailable` の宣言を IdManagement のデータのエクスポートの操作へ移し、語彙の検査の許容リストから外す。

## 対象外

- エラーコードの名前や HTTP の状態の変更。

## 設計

既存のエラーモデルの宣言の形に合わせ、操作の `@error` の和に加える。
OpenAPI の差分は追加だけになることを `mise run check-api-compat` で確かめる。

### 着手時の照合と仕様差分

WorkloadIdentity の二つの not-found エラーはすでに宣言され、操作にも結ばれていたので維持する。
残る pattern conflict の409と、アイコンとサインインポリシーの400のエラーを、既存の拒否と同じ条件で宣言する。
JobsUnavailableError は名前と応答の和を保ったまま IdManagement のモデルへ移す。
取得元の移動だけで公開の型名を変えない。

REQ-IDMANAGEMENT-039 に「ジョブの保存先が構成されていない場合、IdManagement は、エクスポートの開始を503と jobs_unavailable で拒否し、ジョブを作らず、イベントを発行しない。」を追加する。
これは既存の nil 検査を維持すべき約束 (a) として記録するものであり、ネットワーク障害全般を503へ変える実装は加えない。
入力、認可、上限、状態更新、再試行、通知の既存規則は変えず、新たな作用を導入しない。

## タスク

- [x] T001 [Spec] TypeSpec の宣言を加え、移す。
- [x] T002 [Verify] 契約、状態コード、語彙、HTTP の拒否を検証し、互換性の指摘を main と比較する。

## 検証

- `mise run check-unspecified-vocabulary`：jobs_unavailable の所有元と要件を検査する。
- `mise run check-spec`、`mise run render-docs`、`mise run generate-contract`：TypeSpec と派生ビューを同期する。
- `mise run check-contract-drift`、`mise run check-status-drift`、`mise run check-api-compat`：既存のエラー応答と追加宣言を照合する。
- 対象の `mise run test-go-package`：IdManagement、Application、WorkloadIdentity の既存 HTTP 契約と新しい未構成時の拒否を確認する。
- `mise run check`：仕様影響と参照を確認する。

## リスク

- 宣言を移すと、生成したクライアントの型名が変わる場合がある。生成物の差分を確かめる。

## 要約

既存の応答に合わせ、invalid_icon、invalid_sign_in_policy、agent_workload_binding_pattern_conflict を TypeSpec のエラーの和へ加えた。
WorkloadIdentity の二つの not-found の宣言はすでに対応済みだった。
JobsUnavailableError の名前を保って IdManagement へ移し、REQ-IDMANAGEMENT-039 に未構成時の503と無作用を記録して語彙の負債から外した。
本番コードの振る舞いは変えず、三種類のエクスポート開始の Problem Details と、ジョブとイベントを残さないことを固定した。
保存済みの公開前ベースラインとの互換性検査は123件を指摘するが、main と今回の診断は完全に同一であり、新しい互換性違反はない。
公開状態に従う必須の互換性検査は成功し、ベースラインは更新していない。
