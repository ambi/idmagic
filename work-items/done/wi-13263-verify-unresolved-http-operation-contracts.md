---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-11
priority: p1
depends_on: []
change_kind: tooling
evidence_policy: risk-based-v4
documentation_impact:
  level: none
  reason: 開発時の検証証拠と TypeSpec の不正確な宣言を直し、利用者が承認したクォータ更新の認可エラー、SAML ログアウトの署名、POST ログアウトの state の返送を既存の契約へ戻す。初回公開前の契約であり、利用者の操作変更や告知は不要である。
  references: []
initial_context:
  specification:
    - docs/development/testing.md
    - docs/development/specification-first-workflow.md
    - docs/modules/oauth2/token/README.md
    - docs/modules/jobs/admin/README.md
    - docs/modules/signing-keys/lifecycle/README.md
    - docs/modules/workloadidentity/trust-configuration/README.md
    - docs/modules/audit/event-search/README.md
    - docs/modules/authorization/model/README.md
    - docs/modules/authorization/relation-tuple/README.md
    - docs/modules/authorization/check/README.md
    - docs/modules/ws-federation/active-sts/README.md
    - docs/modules/ws-federation/passive-sign-in/README.md
    - docs/modules/ws-federation/relying-party/README.md
    - docs/modules/sharedsignals/stream/README.md
    - docs/modules/sharedsignals/receiver/README.md
    - docs/modules/sourcing/scim/README.md
    - docs/modules/tenancy/attribute-schema/README.md
    - docs/modules/tenancy/integration-endpoints/README.md
    - docs/modules/tenancy/notification-template/README.md
    - docs/modules/tenancy/settings/README.md
    - docs/modules/tenancy/lifecycle/README.md
    - docs/modules/tenancy/quota/README.md
    - docs/modules/tenancy/branding/README.md
    - docs/modules/tenancy/resolution/README.md#REQ-TENANCY-011
    - docs/modules/application/catalog/README.md
    - docs/modules/application/assignment/README.md
    - docs/modules/application/portal/README.md
    - docs/modules/application/sign-in-policy/README.md
    - docs/modules/identity-governance/lifecycle-workflow/README.md
    - docs/modules/identity-governance/workflow-run/README.md
    - docs/modules/provisioning/connection/README.md
    - docs/modules/provisioning/synchronization/README.md
    - docs/modules/provisioning/task/README.md
    - docs/modules/saml/idp-profile/README.md
    - docs/modules/saml/service-provider/README.md
    - docs/modules/saml/sso/README.md
    - docs/modules/identity-management/account/README.md
    - docs/modules/identity-management/account/acceptance.feature.md
    - docs/modules/identity-management/agent/README.md
    - docs/modules/identity-management/data-export/README.md
    - docs/modules/identity-management/user-csv/README.md
    - docs/modules/identity-management/group-csv/README.md
    - docs/modules/identity-management/user/README.md
    - docs/modules/identity-management/user/lifecycle.md
    - docs/modules/identity-management/group/README.md
    - docs/modules/identity-management/dynamic-group/README.md
    - docs/modules/authentication/README.md
    - docs/modules/authentication/sign-in/README.md
    - docs/modules/authentication/account-portal/README.md
    - docs/modules/authentication/session/README.md
    - docs/modules/authentication/trusted-device/README.md
    - docs/modules/authentication/sign-in-activity/README.md
    - docs/modules/authentication/security-notification/README.md
    - docs/modules/authentication/totp/README.md
    - docs/modules/authentication/recovery/README.md
    - docs/modules/authentication/mfa/README.md
    - docs/modules/authentication/webauthn/README.md
    - docs/modules/authentication/federation/README.md
    - docs/modules/system/operations/README.md
    - docs/modules/oauth2/consent/README.md
    - docs/modules/oauth2/client/README.md
    - docs/modules/oauth2/admin-access/README.md
    - docs/modules/oauth2/protocol-endpoints/README.md
    - docs/modules/oauth2/device/README.md
    - docs/modules/oauth2/approval/README.md
    - docs/modules/oauth2/logout/README.md
  typespec:
    - spec/modules/system/models.tsp
    - spec/modules/oauth2/main.tsp
    - spec/modules/jobs/main.tsp
    - spec/modules/jobs/models.tsp
    - spec/modules/signing-keys/main.tsp
    - spec/modules/signing-keys/models.tsp
    - spec/modules/workloadidentity/main.tsp
    - spec/modules/workloadidentity/models.tsp
    - spec/modules/audit/main.tsp
    - spec/modules/audit/models.tsp
    - spec/modules/authorization/main.tsp
    - spec/modules/authorization/models.tsp
    - spec/modules/ws-federation/main.tsp
    - spec/modules/ws-federation/models.tsp
    - spec/modules/sharedsignals/main.tsp
    - spec/modules/sharedsignals/models.tsp
    - spec/modules/sourcing/main.tsp
    - spec/modules/sourcing/models.tsp
    - spec/modules/tenancy/main.tsp
    - spec/modules/tenancy/models.tsp
    - spec/modules/application/main.tsp
    - spec/modules/application/models.tsp
    - spec/modules/identity-governance/main.tsp
    - spec/modules/identity-governance/models.tsp
    - spec/modules/provisioning/main.tsp
    - spec/modules/provisioning/models.tsp
    - spec/modules/saml/main.tsp
    - spec/modules/saml/models.tsp
    - spec/modules/identity-management/main.tsp
    - spec/modules/identity-management/models.tsp
    - spec/modules/authentication/main.tsp
    - spec/modules/authentication/models.tsp
  source:
    - mise.toml
    - tools/verification/src/evidence.ts
    - tools/verification/src/main.ts
    - tools/verification/src/contract-evidence-inventory.ts
    - tools/check/src/contract-drift.ts
    - tools/check/src/status-drift.ts
    - backend/shared/http/testing_stack/stack.go
    - backend/shared/spec/runtime_contract.go
    - backend/jobs/handlers_http/admin_job_handler.go
    - backend/signingkeys/handlers_http/admin_key_handler.go
    - backend/workloadidentity/handlers_http/routes.go
    - backend/audit/handlers_http/admin_audit_event_handler.go
    - backend/authorization/internal/handlers_http/routes.go
    - backend/wsfederation/handlers_http/admin_entra_handler.go
    - backend/wsfederation/handlers_http/admin_relying_party_handler.go
    - backend/wsfederation/handlers_http/wstrust_handler.go
    - backend/wsfederation/handlers_http/wsfed_handler.go
    - backend/sharedsignals/handlers_http/routes.go
    - backend/sourcing/scim/handlers_http/handlers.go
    - backend/tenancy/handlers_http/admin_tenant_handler.go
    - backend/tenancy/handlers_http/admin_user_attribute_schema_handler.go
    - backend/tenancy/handlers_http/admin_group_attribute_schema_handler.go
    - backend/shared/http/support_http/auth.go
    - backend/application/handlers_http/admin_application_handler.go
    - backend/application/handlers_http/admin_category_handler.go
    - backend/idgovernance/handlers_http/admin_lifecycle_workflow_handler.go
    - backend/idgovernance/usecases/lifecycle_workflows.go
    - backend/provisioning/handlers_http/handlers.go
    - backend/provisioning/handlers_http/routes.go
    - backend/provisioning/usecases/admin.go
    - backend/saml/handlers_http/routes.go
    - backend/saml/handlers_http/admin_idp_profile_handler.go
    - backend/saml/handlers_http/admin_service_provider_handler.go
    - backend/saml/handlers_http/idp_profile.go
    - backend/saml/handlers_http/slo_handler.go
    - backend/idmanagement/user/handlers_http/account_handler.go
    - backend/idmanagement/user/handlers_http/account_data_handler.go
    - backend/idmanagement/user/handlers_http/email_change_handler.go
    - backend/idmanagement/agent/handlers_http/admin_agent_handler.go
    - backend/idmanagement/handlers_http/admin_data_export_handler.go
    - backend/idmanagement/user/handlers_http/admin_user_handler.go
    - backend/idmanagement/group/handlers_http/admin_group_handler.go
    - backend/authentication/handlers_http/account_context_handler.go
    - backend/authentication/handlers_http/account_security_handler.go
    - backend/authentication/handlers_http/account_consents_handler.go
    - backend/authentication/handlers_http/account_activity_handler.go
    - backend/authentication/session/handlers_http/account_sessions_handler.go
    - backend/authentication/trusteddevice/handlers_http/account_trusted_devices_handler.go
    - backend/authentication/federation/handlers_http/routes.go
    - backend/authentication/webauthn/usecases/verify_webauthn_factor.go
    - backend/oauth2/handlers_http/authorize_transaction.go
    - backend/oauth2/handlers_http/authorize_second_factor.go
    - backend/oauth2/handlers_http/authorize_enrollment.go
    - backend/oauth2/handlers_http/authorize_resume.go
    - backend/oauth2/handlers_http/end_session_handler.go
  tests:
    - backend/tenancy/handlers_http/admin_tenant_handler_test.go
    - backend/shared/http/server_http/routes_e2e_test.go
    - backend/shared/http/server_http/authorization_request_standards_test.go
    - backend/shared/http/server_http/authentication_browser_examples_test.go
    - backend/shared/http/server_http/browser_contract_test.go
    - backend/authentication/federation/handlers_http/federated_login_e2e_test.go
    - backend/authentication/handlers_http/federation_contract_test.go
    - backend/oauth2/handlers_http/protocol_contract_test.go
    - backend/oauth2/handlers_http/browser_webauthn_contract_test.go
    - backend/oauth2/handlers_http/recovery_code_mfa_e2e_test.go
    - backend/oauth2/handlers_http/end_session_handler_test.go
    - backend/authentication/handlers_http/account_handlers_test.go
    - backend/authentication/handlers_http/refusal_effects_test.go
    - backend/authentication/session/handlers_http/admin_sessions_handler_test.go
    - tools/verification/src/evidence.test.ts
    - backend/idmanagement/user/handlers_http/account_handler_test.go
    - backend/sourcing/scim/handlers_http/resource_contract_test.go
    - backend/shared/http/server_http/metadata_standards_test.go
    - backend/shared/http/server_http/admin_scope_contract_test.go
    - backend/jobs/handlers_http/admin_job_handler_test.go
    - backend/signingkeys/handlers_http/scenario_examples_test.go
    - backend/workloadidentity/handlers_http/routes_test.go
    - backend/workloadidentity/handlers_http/contract_test.go
    - backend/audit/handlers_http/admin_audit_event_handler_test.go
    - backend/authorization/internal/handlers_http/routes_test.go
    - backend/wsfederation/handlers_http/passive_signin_examples_test.go
    - backend/wsfederation/handlers_http/wstrust_examples_test.go
    - backend/wsfederation/handlers_http/wsfed_handler_test.go
    - backend/sharedsignals/handlers_http/scenario_examples_test.go
    - backend/sharedsignals/handlers_http/routes_test.go
    - backend/sharedsignals/handlers_http/e2e_stream_delivery_test.go
    - backend/sharedsignals/handlers_http/contract_test.go
    - backend/sourcing/scim/handlers_http/standards_test.go
    - backend/sourcing/scim/handlers_http/collection_query_test.go
    - backend/sourcing/scim/handlers_http/operation_contract_test.go
    - backend/tenancy/handlers_http/admin_settings_handler_test.go
    - backend/tenancy/handlers_http/admin_user_attribute_schema_handler_test.go
    - backend/tenancy/handlers_http/admin_group_attribute_schema_handler_test.go
    - backend/tenancy/handlers_http/admin_notification_template_handler_test.go
    - backend/tenancy/handlers_http/integration_endpoints_handler_test.go
    - backend/tenancy/handlers_http/branding_handler_test.go
    - backend/tenancy/handlers_http/implicit_rules_test.go
    - backend/tenancy/handlers_http/control_plane_audit_test.go
    - backend/tenancy/handlers_http/control_plane_contract_test.go
    - backend/tenancy/handlers_http/contract_helpers_test.go
    - backend/application/handlers_http/application_handler_test.go
    - backend/application/handlers_http/catalog_examples_test.go
    - backend/application/handlers_http/extra_handlers_test.go
    - backend/application/handlers_http/not_found_statuses_test.go
    - backend/application/handlers_http/client_secret_examples_test.go
    - backend/application/handlers_http/client_secret_lifecycle_test.go
    - backend/application/handlers_http/account_application_order_test.go
    - backend/application/handlers_http/operation_contract_test.go
    - backend/application/handlers_http/account_contract_test.go
    - backend/application/handlers_http/contract_helpers_test.go
    - backend/idgovernance/handlers_http/scenario_examples_test.go
    - backend/idgovernance/handlers_http/admin_lifecycle_workflow_handler_test.go
    - backend/idgovernance/handlers_http/contract_test.go
    - backend/provisioning/handlers_http/admin_connection_handler_test.go
    - backend/provisioning/handlers_http/admin_connection_events_test.go
    - backend/provisioning/handlers_http/admin_task_list_pagination_test.go
    - backend/provisioning/handlers_http/contract_test.go
    - backend/saml/handlers_http/saml_handler_test.go
    - backend/saml/handlers_http/saml_standards_test.go
    - backend/saml/handlers_http/scenario_examples_test.go
    - backend/saml/handlers_http/refusal_effects_test.go
    - backend/saml/handlers_http/contract_test.go
    - backend/idmanagement/handlers_http/extra_identity_test.go
    - backend/idmanagement/handlers_http/scenario_examples_test.go
    - backend/idmanagement/handlers_http/refusal_effects_test.go
    - backend/idmanagement/handlers_http/account_refusal_effects_test.go
    - backend/idmanagement/handlers_http/account_contract_test.go
    - backend/idmanagement/handlers_http/agent_contract_test.go
    - backend/idmanagement/handlers_http/admin_data_export_handler_test.go
    - backend/idmanagement/handlers_http/export_examples_test.go
    - backend/idmanagement/handlers_http/export_refusal_effects_test.go
  stop_before_reading: [frontend, infra, backend/idmanagement/user/db_postgres]
affected_spec:
  - { path: docs/modules/identity-management/account/README.md, requirement: REQ-IDMANAGEMENT-053 }
  - { path: docs/modules/identity-management/account/acceptance.feature.md, requirement: EX-IDMANAGEMENT-053-05 }
  - { path: spec/modules/authentication/models.tsp, symbol: IdentityProviderConnection }
  - { path: spec/modules/authentication/models.tsp, symbol: LinkedExternalIdentity }
  - { path: spec/modules/authentication/main.tsp, symbol: ListLinkedExternalIdentities }
  - { path: spec/modules/authentication/main.tsp, symbol: ListMySignInActivity }
  - { path: spec/modules/authentication/main.tsp, symbol: GetMyNotificationPreferences }
  - { path: spec/modules/oauth2/main.tsp, symbol: ListMyConsents }
  - { path: spec/modules/authentication/main.tsp, symbol: GetAccountSecurity }
  - { path: spec/modules/authentication/main.tsp, symbol: ListMySessions }
  - { path: spec/modules/authentication/main.tsp, symbol: ListMyTrustedDevices }
  - { path: spec/modules/identity-management/models.tsp, symbol: AdminUserResponse }
  - { path: spec/modules/identity-management/models.tsp, symbol: GroupSummaryResponse }
  - { path: spec/modules/identity-management/main.tsp, symbol: GetUserExport }
  - { path: spec/modules/identity-management/main.tsp, symbol: DownloadUserExportFile }
  - { path: spec/modules/identity-management/main.tsp, symbol: CancelUserExport }
  - { path: spec/modules/identity-management/main.tsp, symbol: GetGroupExport }
  - { path: spec/modules/identity-management/main.tsp, symbol: DownloadGroupExportFile }
  - { path: spec/modules/identity-management/main.tsp, symbol: CancelGroupExport }
  - { path: spec/modules/identity-management/main.tsp, symbol: GetGroupMemberExport }
  - { path: spec/modules/identity-management/main.tsp, symbol: DownloadGroupMemberExportFile }
  - { path: spec/modules/identity-management/main.tsp, symbol: CancelGroupMemberExport }
  - { path: spec/modules/identity-management/main.tsp, symbol: DataExportNotFoundResponse }
  - { path: spec/modules/identity-management/main.tsp, symbol: DataExportNotDownloadableResponse }
  - { path: spec/modules/identity-management/main.tsp, symbol: DataExportNotCancelableResponse }
  - { path: spec/modules/identity-management/models.tsp, symbol: DataExportNotFoundError }
  - { path: spec/modules/identity-management/models.tsp, symbol: DataExportNotDownloadableError }
  - { path: spec/modules/identity-management/models.tsp, symbol: DataExportNotCancelableError }
  - { path: spec/modules/identity-management/main.tsp, symbol: GetUserProfile }
  - { path: spec/modules/identity-management/main.tsp, symbol: GetAccountSummary }
  - { path: spec/modules/identity-management/main.tsp, symbol: ExportAccountData }
  - { path: spec/modules/identity-management/main.tsp, symbol: RequestEmailChange }
  - { path: spec/modules/identity-management/main.tsp, symbol: ConfirmEmailChange }
  - { path: spec/modules/identity-management/main.tsp, symbol: EmailChangeConflictResponse }
  - { path: spec/modules/identity-management/main.tsp, symbol: InvalidEmailChangeTokenResponse }
  - { path: spec/modules/identity-management/models.tsp, symbol: InvalidEmailChangeTokenError }
  - { path: spec/modules/identity-management/models.tsp, symbol: AccountDataExport }
  - { path: spec/modules/identity-management/models.tsp, symbol: AccountDataExportConsent }
  - { path: spec/modules/provisioning/main.tsp, symbol: ProvisionOnDemand }
  - { path: spec/modules/provisioning/main.tsp, symbol: ResumeProvisioningConnection }
  - { path: spec/modules/identity-governance/main.tsp, symbol: DeleteLifecycleWorkflow }
  - { path: spec/modules/identity-governance/models.tsp, symbol: WorkflowRunResponse }
  - { path: spec/modules/identity-governance/models.tsp, symbol: WorkflowRunResponseTrigger }
  - { path: spec/modules/saml/models.tsp, symbol: SamlServiceProvider }
  - { path: spec/modules/tenancy/main.tsp, symbol: UpdateTenantQuota }
  - { path: spec/modules/workloadidentity/main.tsp, symbol: WorkloadTrustBundleNotFoundResponse }
  - { path: spec/modules/workloadidentity/main.tsp, symbol: AgentWorkloadBindingNotFoundResponse }
  - { path: spec/modules/workloadidentity/models.tsp, symbol: WorkloadTrustBundleNotFoundError }
  - { path: spec/modules/workloadidentity/models.tsp, symbol: AgentWorkloadBindingNotFoundError }
  - { path: spec/modules/ws-federation/main.tsp, symbol: WsTrustIssue }
  - { path: spec/modules/ws-federation/main.tsp, symbol: ConfigureEntraFederation }
  - { path: docs/modules/ws-federation/active-sts/README.md, requirement: REQ-WSFEDERATION-004 }
  - { path: docs/modules/ws-federation/active-sts/README.md, requirement: REQ-WSFEDERATION-005 }
  - { path: docs/modules/ws-federation/relying-party/README.md, requirement: REQ-WSFEDERATION-001 }
  - { path: spec/modules/workloadidentity/main.tsp, symbol: GetWorkloadTrustBundle }
  - { path: spec/modules/workloadidentity/main.tsp, symbol: UpdateWorkloadTrustBundle }
  - { path: spec/modules/workloadidentity/main.tsp, symbol: DisableWorkloadTrustBundle }
  - { path: spec/modules/workloadidentity/main.tsp, symbol: EnableWorkloadTrustBundle }
  - { path: spec/modules/workloadidentity/main.tsp, symbol: DeleteWorkloadTrustBundle }
  - { path: spec/modules/workloadidentity/main.tsp, symbol: RefreshWorkloadTrustBundleJWKS }
  - { path: spec/modules/workloadidentity/main.tsp, symbol: ListAgentWorkloadBindings }
  - { path: spec/modules/workloadidentity/main.tsp, symbol: CreateAgentWorkloadBinding }
  - { path: spec/modules/workloadidentity/main.tsp, symbol: DisableAgentWorkloadBinding }
  - { path: spec/modules/workloadidentity/main.tsp, symbol: EnableAgentWorkloadBinding }
  - { path: spec/modules/workloadidentity/main.tsp, symbol: DeleteAgentWorkloadBinding }
---

# 未解析の HTTP 操作を契約テストで検証する

## 動機

wi-90256 の Worker の配線変更を検証したとき、Go のテスト 6,359 件は成功したが、282 操作について同じ実行の契約検証の証拠がなく、`mise run verify` が失敗した。
調査時点では、リポジトリ内の Go テストに `//verify:operation` の宣言がなかった。
[テスト方針](../../docs/development/testing.md#検証結果の区分)は、構成入口の変更で全操作を選び、解析が不完全な操作には本文と状態コードを検証した契約テストを要求する。
そのため、構成入口を直す作業ごとに、既存の未解析の操作が全体検証を止める。

利用者は、監査の実装を止めず、この不足を別項目へ記録し、全体検証の未達を明示する扱いを選んだ。
これは当該作業の例外であり、検証ゲートの条件は弱めない。

## 対象範囲

- 静的解析で `unverified` になる操作を、本文と状態コードの理由別に棚卸しする。
- 既存テストが契約の要求を満たす場合は、その観測内容を確認して操作 ID の宣言を付ける。
- 不足する操作には、正式な HTTP 入口で要求、成功応答、拒否応答と状態コードを検証するテストを追加する。
- 同じ標準検証の実行で、それらの操作の証拠が成功と判定されることを確認する。
- 利用者が承認した WS-Trust の SOAP、Entra の作成時 201、WorkloadIdentity の既存の 404 について、製品の応答を維持して TypeSpec の不一致を直す。
- 同じ方針で、ワークフロー削除のリビジョンを含む要求本文と、実行履歴のトリガーの HTTP 射影の TypeSpec を直す。
- 利用者が承認したクォータ更新の認可エラー変換の欠陥を直し、非 System 管理者の要求を既存の 403 契約へ戻す。
- 同じ宣言修正の方針で、プロビジョニングの既存の拒否と SAML の既定の署名ポリシーの省略を TypeSpec へ反映する。
- 利用者が承認した SAML ログアウトの署名後の XML 整形を削除し、既存の署名付き LogoutResponse の契約へ戻す。

## 対象外

- クォータ更新の認可と SAML ログアウトの署名を既存の契約へ戻す修正以外の、製品コードが返す HTTP 応答と業務の振る舞いの変更。
- 未解析の操作を対象から落とすこと、失敗を無視すること、状態コードだけのテストを本文の証拠とすること。
- wi-46722 の監査の書き込み失敗の検知と運用対応。

## 設計

既存の `//verify:operation <操作 ID>: <観測する契約>` と検証ランナーを使う。
宣言は、成功と拒否の本文および状態コードを実際に検証するテストにだけ置く。
契約の期待値は TypeSpec と生成 OpenAPI から導き、本番コードの返り値を期待値として写さない。
各操作へ宣言だけを一括で付ける案は、要求された観測と実行の証拠が対応しないため採らない。
静的解析を改善する案は、対象の構文を確実に解析できる場合に限って併用し、不完全な解析を成功に読み替えない。
着手時に現在の未解析一覧を取り直し、既存テストの再利用と不足テストの追加の境界を決める。

### 着手時の棚卸しと証拠

wi-26617 の実行結果では、本文の解析は 342 操作中 146 操作、状態コードの解析は 248 操作が未解析だった。
重複を除くと 282 操作に成功した契約テストの証拠がない。
本文の主な理由は変数や補助関数を通した応答、状態コードの主な理由はユースケースのエラーを受け取る変換関数である。
エラー変換の全分岐を各操作から到達できると仮定して、静的解析の状態を verified に変える案は採らない。

既存の HTTP テストは 162 ファイルあり、テスト名や URL の一致は候補として扱う。
本文と状態コードの観測を読んで、要求、成功、拒否を担当するテストを操作ごとに解決する。
同じ操作の証拠が複数テストへ分かれる場合は、一つだけの成功で他方の未実行やスキップを隠さない。
ステータスだけを検査する既存テストには、宣言の前に必要な本文の検査を追加する。

受け入れ RED の代替は、wi-26617 の `mise run verify` で観測した 282 操作の `no successful contract test for changed unverified operation` である。
規範の製品結果を追加する作業ではないため、主要ユースケースの追加は不要である。
単体 RED と故障注入は、契約の本文または状態コードの期待値を誤ったものへ変える検査と、証拠の一部を未実行にする検査で確認する。
本文は TypeSpec と生成 OpenAPI から、業務上の値は対象の既存要件と例から導く。
テストだけの差分では全体ランナーの変更操作が空になり得るため、wi-26617 を比較基準に選び、構成入口の差分を含む全操作についても検証する。

282 操作を繰り返し探す費用を抑えるため、未解析の理由、ハンドラーと候補テストを生成する `report-contract-evidence` を加える。
候補は経路の文字列の一致を優先し、一致がないときだけハンドラーと同じ配置のテストを選ぶ。
観測の確認と実行の成功をこの索引から推定しない。

### 局所検証の記録

- 証拠判定の RED：成功側だけが passed、拒否側が skipped のとき、旧判定が欠落を報告しないことを単体テストで観測した。修正後は skipped、not-run、failed をそれぞれ報告し、両側の成功だけを受理した（12 テスト、731 表明が成功）。
- 本文検査の RED：`TestProtectedResourceMetadata_unregisteredResource_rejected` の期待を `invalid_request` に変えると、実際の `invalid_target` を検出して失敗した。REQ-OAUTH2-044 と TypeSpec に従い期待を戻し、OAuth2 HTTP パッケージの race 検査 230 テストが成功した。
- メトリクス：`TestMetricsHTTPContract` の設定済み 200 のテキスト本文と Content-Type、未配線時 503 の JSON 本文を含む 3 テストが成功した。
- Jobs：既存の一覧、参照、取り消しに Problem Details と成功本文の検査を補い、6 操作を宣言した。race 検査 21 テストが成功した。
- SigningKeys：ローテーション、無効化の公開鍵メタデータと拒否本文を確認し、参照の成功と不明な kid の拒否を追加した。3 操作を宣言し、race 検査 15 テストが成功した。
- WorkloadIdentity：信頼設定と関連付けの登録から削除までを正式な配線で検査し、12 操作の成功と拒否を宣言した。JWKS はインラインの公開鍵を使い外部通信を要しない。race 検査 17 テストが成功した。
- 棚卸しツール：経路の引数名が違う場合と経路未解決の場合を含む単体検査が成功した。候補を証拠へ昇格させない。
- Audit：6 操作の本文、状態、拒否とエクスポートの attachment を補い、race 検査 35 テストが成功した。
- Authorization：6 操作について公開モデルと読み直し、保存後のタプル、判定、列挙の本文と、全操作の非管理者の拒否本文を補った。race 検査 34 テストが成功した。
- WorkloadIdentity の 404：TypeSpec に補った 11 操作の不明な資源への 404 と Problem Details を HTTP で検査した。race 検査は 29 テストが成功した。
- WS-Federation：4 操作の SOAP、署名した RSTR のフォーム、RP の作成と置き換え、Entra の設定と拒否の本文を補った。本文デコードの import 欠落によるコンパイル失敗を修正し、race 検査 84 テストが成功した。
- 仕様修正：`mise run check-spec` は要件文の応答側に置いた条件を拒否した。条件を前置きへ移し、再実行が成功した。`check-published-api-compat` は初回公開前で公開済み契約なしと判定した。
- SharedSignals：10 操作の登録、一覧、参照、更新、状態変更、削除、配送の一覧と SET の受理を宣言した。管理 API に SET 内部のイベント URI を送った初回検査は 422 を検出した。TypeSpec の管理 API のイベント名へ修正し、race 検査 26 テストが成功した。
- Sourcing：15 操作について既存の SCIM の成功、必須属性、置換、部分更新、検索を確認し、Discovery の本文と全操作の認証拒否の本文を補った。文字列型に追加した非空の仮定を Discovery の既存値が拒否したため、TypeSpec が要求する型と存在の検査へ直した。race 検査 233 テストが成功した。
- Tenancy：24 操作の成功と拒否の本文、状態コードを補った。通常の管理者のクォータ更新だけが 500 を返す RED を観測し、利用者が承認した認可エラー変換の修正後、同じテストを含む race 検査 145 テストが成功した。拒否時の保存済み上限とイベントの不変も検査する。
- Application：26 操作の成功と認可または認証の拒否について、本文、状態、保存後の再読を宣言した。realm 名をテナント ID として期待した初回検査は失敗し、fixture が使う既定テナントの UUID へ直した。race 検査 96 テストが成功した。
- IdGovernance：10 操作について定義の状態変更、本文からのリビジョン照合、試算、実行履歴と再試行の本文を検査した。成功する再試行の初回検査は fixture の JobEnqueuer 未配線で panic となり、本番と同じ投入器を fixture に配線した。race 検査 20 テストが成功した。TypeSpec の変更後の `check-spec`、仕様サイト生成と `check-spec-impact -- main` が成功した。
- Provisioning：12 操作の成功と非管理者の拒否について、接続の設定、能力探索、範囲外の On-Demand、隔離の解除、タスクの投入と再試行、削除の本文を検査した。race 検査 11 テスト、`check-spec`、`check-spec-impact -- main` が成功した。
- SAML：12 操作の管理 API の本文と認可の拒否、GET と POST の SSO と SLO を検査した。プロファイルの再読では Repository が `updated_at` を更新するため、時刻の文字列表現の同値ではなく識別子、設定と RFC3339 の型を確認する。false の署名ポリシーの省略は TypeSpec に反映した。SLO の無効な署名は四つの HTTP 経路で再現し、署名後の整形を削除する承認済みの修正後、受信したままの XML の検証を含む race 検査 80 テストと Go 静的検査が成功した。
- SAML 修正の検出能力：自動変異 133 件中 108 件を検出し、生存 9 件、未被覆 1 件、コンパイル不能 15 件だった。生存は、既存のプロファイルの SP 数の集計、SP の削除、SLO の `sp` 別名の分岐、SSO の再開 URL の RelayState とアサーションの時刻の計算で、署名後の整形は含まない。未被覆は `sp` 別名の読み取りである。既定の operator は XML の整形呼び出しを表現しないため、旧実装の整形で四つの署名検証が失敗し、削除後は同じ観測が成功した RED/GREEN を、手動故障に相当する検出能力の証拠とする。
- IdManagement の本人操作：6 操作について、正式な配線で発行した API トークンによる概要、プロフィールの読み書き、同意を含むエクスポート、実際の確認メールのトークンを使うメール変更を検査した。認証、スコープと CSRF の拒否、競合と再利用の拒否の本文と不変も検査する。race 検査 145 テストと `check-spec` が成功した。
- IdManagement の Agent：9 操作の登録、参照、更新、資格情報の関連付けと解除、状態変更と削除の本文、空の 204 と保存後の再読を検査した。全操作の非管理者の 403 と Agent とイベントの不変も検査する。本人操作を含む同じ HTTP パッケージの race 検査 156 テストと Go 静的検査が成功した。
- IdManagement の CSV エクスポート：15 操作について、三つの対象のジョブの起票、参照、一覧、取り消し、実際の生成器による CSV のダウンロードと 30 日の保持期限を検査した。全操作の非管理者の 403、不明な ID の 404、未完了のダウンロードと終了後の取り消しの 409 も検査する。初回のグループ生成は fixture の生成器未配線で失敗し、本番と同じ生成器を fixture に配線した。再実行の race 検査 176 テストが成功した。
- IdManagement の User と Group：20 操作の成功と非管理者の拒否について、プロフィールの作成と更新、必須操作、無効化と再有効化、削除予約と復元、Group の編集とメンバーの実効ロール、動的規則の保存とプレビューと状態変更を検査した。状態を文書の状態名の大文字で期待した初回検査は失敗し、TypeSpec とドメインが定める小文字の送受信値へ直した。必須でないパスワード変更時刻の仮定も除き、race 検査 200 テストと `check-spec` が成功した。
- IdManagement の CSV インポート：6 操作の成功と非管理者の拒否を宣言した。User とメンバーシップは既存の主要ユースケースで 202 の JSON、識別子、queued と保存したジョブに一致する mode を追加で観測し、Group は実際の生成器と行の確定器でプレビューと適用を経て保存した Group を読む。Group の一覧に未宣言の `query` の絞り込みを仮定した初回検査は失敗し、全体から対象の一件を読む検査へ直した。race 検査は共通の HTTP 208、User HTTP 35、Group HTTP 25 テストが成功した。
- クォータ修正の検出能力：ミューテーションツールの canary が成功した。Tenancy HTTP の自動変異 164 件中 136 件を検出し、生存 16 件、未被覆 1 件、コンパイル不能 11 件だった。修正した認可拒否の条件を含むクォータ更新の 7 件の条件変異は全て検出した。生存は既存の変更項目のイベント射影、クォータと使用量の読み取り失敗、画像サイズの複合条件と証明書の指紋のスライスの初期容量であり、修正した認可分岐の生存はない。初期容量の変異は指紋の値を変えない等価な変異である。未被覆は既存のクォータ本文の不正な JSON のエラー連結で、今回の修正対象外である。旧実装で 500 を再現した RED は、ツールが表現しないエラー変換の欠落を検出した手動故障の証拠でもある。
- ここまでの Go 静的検査は `mise run lint-go` が 0 issues で成功した。仕様サイトの生成、実行時の契約メタデータの生成と `check-spec-impact -- main` が成功した。

- 認証のアカウント操作：15 操作の成功と認証、認可または CSRF の拒否を宣言した。アカウント情報、セキュリティ、セッション、信頼済みデバイス、同意、サインイン履歴と通知設定の本文、204 の空本文と保存後の状態を検査した。Origin がない同意の取り消しは `invalid_origin` として検証し、認証 HTTP の race 検査 99 テストと通知設定 HTTP の 6 テストが成功した。
- MFA：10 操作の成功と認可または CSRF の拒否を宣言した。TOTP の実際の生成コードによる登録と解除、リカバリーコードの生成と保存済みハッシュと失効、再認証の選択肢と WebAuthn のチャレンジとセッション更新、管理者による回避の発行と認証器のリセットを観測した。テストの import とセッションの Repository 操作名のコンパイルエラーを直し、認証 HTTP の race 検査 113 テストが成功した。

- WebAuthn と外部 IdP：9 操作について、HTTP の challenge から作った none attestation、保存された COSE 公開鍵とカウンターと解除、関連付けの subject 非公開と解除、接続の作成と更新とクレームの試行とメタデータ再取得を観測した。CSRF と未認証の拒否も検査する。OIDC の外部 HTTP だけを transport のテストダブルへ置き換え、正式なルーター、Repository とプロトコルの検証を使う。空の配列の省略と関連付けの射影を TypeSpec へ反映し、認証 HTTP の race 検査 117 テストと外部 IdP HTTP の 17 テストが成功した。

- Readiness：通常と verbose の 200 の JSON、および永続化依存の障害時の 503 の JSON を操作へ結び付けた。共有 HTTP 構成の race 検査 402 テストが成功した。

- OAuth2 の管理：12 操作の作成、参照、更新、削除、同意の参照と撤回について、本文と保存後の再読、空の 204、非管理者の 403 と不明な資源の 404 を宣言した。ConsentState の送受信値は TypeSpec が定める `granted` で確認する。OAuth2 HTTP の race 検査 234 テストが成功した。

- ブラウザーの認可：認可開始、トランザクション、ログイン、同意、TOTP、MFA 登録の開始と確定、外部認証からの再開の 8 操作を宣言した。200 の JSON と 303 の Location、CSRF の 403 と外部 return_to の 400、未認証の再開の 401 を検査する。同意の要求には email も含まれるため、初回の期待値に欠けた email を補った。共有 HTTP 構成の race 検査 410 テストが成功した。
- JWKS：宣言された唯一の 200 応答について、返された公開鍵で実際に発行したトークンの署名を検証し、秘密鍵の成分を含まない既存テストを操作へ結び付けた。宣言がない拒否ケースは追加しない。共有 HTTP 構成の同じ race 検査で成功した。
- 外部認証のブラウザー境界：開始、OIDC のコールバック、SAML のコールバックと明示的な関連付け開始の 4 操作を宣言した。プロトコルドライバーの検証後の結果だけをダブルとし、HTTP のフォームと query の転送、上流の callback URL、303、セッション Cookie と保存された federated の AMR を読む。未完のコールバックと不明な接続の 400 はドライバーへ到達しない。関連付け開始は実際の OIDC ドライバーで state、nonce、PKCE を生成し、保存された試行を本人へ結び付ける。外部認証 HTTP の race 検査 19 テストと認証 HTTP の 118 テストが成功した。
- WebAuthn と復旧コードの第二要素：3 操作について、署名した assertion の受理、チャレンジと COSE 公開鍵、カウンターと最終利用時刻、復旧コードの成功と拒否を観測した。第二要素を待つ実際のセッションが mfa へ昇格して保留が解けることを読む。WebAuthn の fixture の無効な RP ID を localhost へ直し、直接ログインの成功は next ではなく既存契約の redirect_to で検査する。三つの対象テストが個別実行で成功した。パッケージ全体の race 検査と全操作を同じ実行で集める検証は、POST ログアウトの修正確認後に行う。

- OAuth2 のプロトコル：残る 13 操作について、実際の発行トークンの検査と失効、UserInfo の GET と POST、PAR、動的登録、デバイス認可と承認、CIBA と本人による承認、ログアウトの GET と POST の成功と拒否を宣言した。POST ログアウトの state 欠落の RED を維持して承認済みの修正を入れ、特殊文字の返送、query 優先と state の省略も検査した。OAuth2 HTTP の race 検査 246 テストと Go の静的検査が成功した。最初の棚卸しにある 282 操作はすべて契約テストの宣言と局所検証が揃った。

### 契約の不一致と修正案

#### 本人の認証情報の参照の認証拒否

セキュリティの概要、セッション、信頼済みデバイス、サインイン履歴、通知設定、外部アイデンティティの関連付けと同意の参照は、既存の HTTP が未認証を 401 `authentication_required` の Problem Details で拒否する。
REQ-AUTHENTICATION-003、REQ-AUTHENTICATION-004、REQ-AUTHENTICATION-013、REQ-AUTHENTICATION-014、REQ-AUTHENTICATION-029、REQ-AUTHENTICATION-033 と REQ-OAUTH2-002 は、認証した本人に閉じた参照を要求するが、TypeSpec の七つの操作にこの 401 の宣言が欠けている。
承認済みの分類 (a) の方針に従い、既存の認証拒否を宣言し、製品コードと規範は変えない。

#### 外部 IdP の接続と関連付けの HTTP 射影

外部 IdP の接続は、空の署名証明書とメールドメインの許可リストを既存の HTTP が省略するため、TypeSpec の二つの配列を任意にする。
関連付けの参照は、REQ-AUTHENTICATION-003 が要求する外部 subject の非公開を守る既存の HTTP に合わせて、subject を除いた射影を宣言する。
承認済みの分類 (a) の方針に従い、製品コードと規範は変えない。

#### 管理ユーザーとグループの HTTP 射影

`AdminUserResponse` の `tenant_id` は TypeSpec では必須だが、既存の HTTP 射影はこの項目を含まない。
User のテナント所属はドメインモデルに保持され、経路から決まるテナント境界も維持される。
管理ユーザーとグループの `roles` は、既存の HTTP 射影が空の保存値を null として返す場合がある。
承認済みの分類 (a) の方針に従い、管理ユーザーの HTTP モデルから `tenant_id` を除き、二つの HTTP モデルの `roles` は配列または null と宣言する。
製品コード、ドメインモデルとロールの判定は変えない。

#### CSV エクスポートの既存の拒否の宣言

REQ-IDMANAGEMENT-087 と REQ-IDMANAGEMENT-088 は、ダウンロードできないエクスポートの 409 `data_export_not_downloadable` と、見つからないエクスポートの 404 `data_export_not_found` を定める。
REQ-IDMANAGEMENT-041 は、終了したエクスポートの取り消しを 409 `data_export_not_cancelable` で拒否する。
既存の HTTP もこの契約どおりだが、TypeSpec の User、Group、メンバーシップの参照、ダウンロード、取り消しの宣言に欠けている。
承認済みの分類 (a) の方針で、この九つの操作の宣言を補い、製品コードと規範は変えない。

#### 本人操作の既存の拒否の宣言

プロフィール、概要、データエクスポートの参照とメール変更の起票は、未認証のときに既存の HTTP が 401 を返すが、TypeSpec にこの宣言がない。
REQ-IDMANAGEMENT-053 のメール変更の起票には、ほかの User のメールを指定したときの 409 `email_taken` と再認証の 403 `step_up_required` がある。
REQ-IDMANAGEMENT-017 と REQ-IDMANAGEMENT-054 の確定には、競合の 409 `email_taken` と無効なトークンの 410 `invalid_email_change_token` がある。
これらも TypeSpec に欠けているため、承認済みの分類 (a) の方針で既存の拒否を補い、製品コードと規範は変えない。
データエクスポートの同意は、専用の同意一覧と異なり `client_name` を含まない。
TypeSpec が両者を同じモデルで表しているため、既存のエクスポートの HTTP 射影を別モデルにし、同意一覧の契約は維持する。

#### SAML の署名ポリシーの省略

`SamlServiceProvider` の `want_authn_requests_signed` は TypeSpec で必須だが、既存の HTTP は false のときに省略する。
省略は既定の false を意味し、既存の署名ポリシーを変えない分類 (a) として、承認済みの宣言修正の方針に従い、既定値を保持して任意のプロパティと宣言する。

#### SAML LogoutResponse の署名後の整形

`TestSamlSLOBindingHTTPContracts` は、既定と追加プロファイル、GET と POST の四つの HTTP 経路で、公開証明書による LogoutResponse の署名検証を実行する。
全て `crypto/rsa: verification error` で失敗し、受信 XML の整形用の空白だけを診断用に除くと、同じ公開証明書で全て検証が成功した。
署名鍵の不一致とテストの証明書選択は、この対照で原因から除いた。
`buildLogoutResponse` は XML に署名した後で `doc.Indent(2)` を呼び、署名された XML を変更している。
修正案は `backend/saml/handlers_http/slo_handler.go` のこの一行を削除し、署名後の XML を変更せず送信することである。
REQ-SAML-009 は、割り当てたプロファイルの鍵で署名した LogoutResponse を要求しており、無効な署名は実装の欠陥と判断した。
利用者は、この項目で署名の不具合も修正する分類 (c) を承認した。
診断用の空白除去を外して受信したままの署名検証を維持し、同じ HTTP テストが GREEN になることを確認する。
仕様影響はない。既存の署名付き LogoutResponse の契約へ戻し、状態コード、宛先、RelayState とセッションの失効を維持する。

#### プロビジョニングの拒否の宣言

REQ-PROVISIONING-012 と既存の HTTP は、適用範囲外の On-Demand 要求を 409 `provisioning_conflict` で拒否するが、TypeSpec は 422 としている。
REQ-PROVISIONING-011 と既存の HTTP は、隔離していない接続の再開を 400 `invalid_request` で拒否するが、TypeSpec にこの応答がない。
承認済みの宣言修正の方針に従い、TypeSpec をこの二つの既存の規範と HTTP へ合わせ、製品コードの応答は変えない。

#### ワークフローの HTTP 宣言と内部モデルの区別

利用者が承認した TypeSpec の不一致を解消する方針に従い、同じ分類 (a) の二点を修正する。
`DeleteLifecycleWorkflow` は `expected_revision` を query と宣言しているが、既存の HTTP は JSON 本文から読み、フロントエンドと既存テストも本文で送る。
規範はリビジョンの一致を要求し、所在を query に限定していないため、要求モデルを加えて本文と宣言する。
実行履歴の `trigger` は内部の `WorkflowRunTriggerSnapshot` を参照しているが、HTTP は `kind` と `changed_fields` だけを返す。
内部の重複排除に必要な `source_occurrence_id` の要件は保持し、HTTP の射影を別モデルへ分ける。
属性変更以外の既存の実行では `changed_fields` が null になるため、射影の型は配列と null を許す。
製品コード、重複排除、応答の本文と状態コードは変えない。

#### POST ログアウトの state の読み取り漏れ

`TestEndSessionHTTPContracts/POST` は、登録済みの戻り先と `state=contract-state` をフォームで送ると、302 の Location に state が含まれず失敗する。
同じパラメーターの GET は成功する。
REQ-OAUTH2-023 は GET と POST の両方で state を戻り先へ付けることを要求し、TypeSpec の `PostEndSession` もフォーム本文に state を宣言している。
`endSessionRedirect` が state だけを query から読み、ほかのログアウトパラメーターと異なり POST 本文を参照していないため、実装の欠陥と判断した。
修正案は、query の state が空なら `c.Request().PostFormValue("state")` を読み、既存の query 優先を維持することである。
登録済み URI の検証、状態コード、セッションの失効は維持する。
利用者は、この項目で state の読み取りも修正する分類 (c) を承認した。
仕様影響はない。GET と POST の両方で state を返す既存の規範へ実装を戻す。
再現した HTTP テストの期待値を維持し、POST 本文の読み取りと query 優先を局所検査と自動変異で検証する。

#### クォータ更新の認可エラーの変換漏れ

`TestControlPlaneTenantContractsRefuseTenantAdmin` を追加すると、通常の管理者のクォータ更新だけが 500 の `internal_server_error` となり、403 の `access_denied` を期待する検査が失敗した。
`handleUpdateTenantQuota` は `RequireControlPlaneUser` のエラーをそのまま返しており、他の制御面のハンドラーは同じエラーを `WriteAdminAccessError` で応答へ変換している。
TypeSpec の `UpdateTenantQuotaError403` と規範の System 管理者だけが上限を変えられる条件から、これは宣言の不正確さではなく実装の欠陥と判断した。
修正案は、この一つの分岐を `return d.WriteAdminAccessError(c, err)` にすることである。
利用者は、この項目で仕様どおりの 403 へ修正する分類を承認した。
保存やイベントの成功経路を変えず、認可拒否を仕様どおりにする。
製品の認可要件は変えない。System 管理者だけがクォータを変更できる既存の規範に実装を戻す。
全体検証では、既存の認証エラー変換が返す 401 と 403 の `insufficient_scope` が、この操作の TypeSpec に欠けていることも検出した。
共有の `AuthenticationRequiredResponse` と `InsufficientScopeError` を追加し、承認済みの変換が返す実際の認証拒否を宣言する。
この宣言補正は分類 (a) とし、利用者が承認した TypeSpec の不一致の解消に含める。
500 を再現したテストを変えずに GREEN を確認し、非管理者の拒否がクォータの保存とイベント発行を防ぐことを確認する。

WS-Federation の証拠を調べたところ、既存の規範文書、TypeSpec、HTTP の実装に以下の不一致を確認した。
これらを宣言だけで契約の成功へ読み替えることはしない。
当初は公開 HTTP 契約の変更を対象外にしていたため、利用者へ具体的な修正案を示し、承認を得て対象範囲へ加えた。

| 操作 | TypeSpec の現在の宣言 | 既存の実装と規範文書 | 提案する修正 |
| --- | --- | --- | --- |
| WsTrustIssue | 要求と 200 応答を JSON のモデルで表し、400 と 403 を Problem Details としている | REQ-WSFEDERATION-004 は SOAP の RSTR と application/soap+xml を要求する。HTTP は SOAP を受けて SOAP を返し、400 と 403 は text/plain を返す | SOAP の要求と成功応答を application/soap+xml の文字列として TypeSpec に宣言し、拒否の本文を text/plain に合わせる。400 と 403 の本文形式を所有する要件へ明記する |
| ConfigureEntraFederation | 成功は 200 だけ | 新しい RP の作成は 201、置き換えは 200。既存テストは作成時の 201 を検査している | 作成の 201 と置き換えの 200 を TypeSpec に宣言し、REQ-WSFEDERATION-001 に状態コードを明記する |
| GetWorkloadTrustBundle など、信頼設定と関連付けの操作 | 不明な ID に対する 404 の応答が欠けている | REQ-WORKLOADIDENTITY-008、REQ-WORKLOADIDENTITY-009 とライフサイクル表は不明または他テナントの ID を 404 で拒否する | 既存の要件が定める workload_trust_bundle_not_found と agent_workload_binding_not_found の 404 を対象操作の TypeSpec に宣言する |

提案は規範と実際の HTTP に合わせて TypeSpec の不正確な表現を直す分類 (a) であり、製品コードの応答と状態遷移を維持する。
WS-Trust の SOAP と JSON は交換可能な細部ではないため、分類 (b) として未規定のまま扱わない。
TypeSpec の JSON に実装を合わせる分類 (c) は既存の SOAP 利用者の観測を変えるため、推奨しない。
利用者は、この修正案を wi-13263 の範囲に含めることを承認した。

要件の差分は次の通りである。

- REQ-WSFEDERATION-004：WS-Trust の拒否応答（400、401、403）は `text/plain` の本文を返し、署名したトークンを含めない。
- REQ-WSFEDERATION-005：不正なエンベロープの 400 拒否は `text/plain` の本文を返す。
- REQ-WSFEDERATION-001：Entra の設定は新規 RP の作成なら 201、既存 RP の置き換えなら 200 と設定値を返す。
- REQ-WORKLOADIDENTITY-008、REQ-WORKLOADIDENTITY-009：既存の 404 の要件と状態遷移表を維持し、TypeSpec の応答だけを補う。

## 計画

1. `mise run verify` の操作別の理由と既存テストを照合する。
2. 満たしている証拠を宣言し、不足する契約テストを追加する。
3. 本文または状態コードを誤る実装をテストが検出することを確かめる。
4. 全体検証で未達を解消する。

## タスク

- [x] T001 [Inventory] 未解析の操作と既存の契約観測を照合する。
- [x] T002 [Tests] 本文と状態コードの検証と操作 ID の宣言を整備する。
- [x] T003 [Verify] 誤った本文と状態コードの検出および全体検証を確認する。

## 検証

2 回目の全体検証では、全 282 操作の契約証拠と Go の 6,351 テスト、DB の 39 パッケージ、tools の 1,020 テスト、UI の 698 テスト、ブラウザー E2E の 39 テストが成功した。
残った失敗は、メール変更の既存の step-up 拒否を TypeSpec と結ぶ仕様例の不足だけである。
REQ-IDMANAGEMENT-053 は既に 403 の拒否と確認リンクを送らないことを要求するため、分類 (a) としてその例を補い、実際の期限切れセッションの HTTP テストへ結び付ける。

初回の全体検証では Go の 6,348 テストとブラウザー E2E が成功したが、クォータ更新の上記二つの宣言不足で静的契約検査が失敗した。
また、操作参照の一部にコロンと観測内容の説明がなく、成功したテストと操作を結ぶ証拠として受理されなかった。
説明をテストの観測に合わせて補い、集約検査の判定を維持したまま再検証する。
ツールの二つのテストは並列実行で 5 秒の制限を超えたが、対象の 2 ファイルの単独実行では 20 テストすべてが成功した。
クォータの認証・scope 拒否を追加した tenancy HTTP の race 検査は 148 テストが成功し、拒否時に保存しないことも確認した。
OAuth HTTP 全体の変異調査は完了レポートが残らず終了したため、その結果は証拠に使わない。
変更前の revision を指定して変異を変更行に絞る任意引数を `test-go-mutation` に加え、既定の全パッケージ調査は維持した。
変更行の調査では state の空判定と付与判定の 2 変異を検出し、未被覆、生存、生成不能、timeout はなかった。
この結果は変更行の検出能力だけを示し、変更していない行の検出能力は主張しない。

- `mise run check-contract-drift`
- `mise run test-go-race`
- `mise run verify`
- `mise run test-tools-file -- verification/src/evidence.test.ts`：証拠の一部が未実行、スキップ、失敗だった場合の判定。
- `mise run test-go-test -- <package> <test>` と `test-go-package`：操作別の契約観測と故障注入。
- `VERIFICATION_BASE_REVISION=work-item/wi-26617 mise run test-go-race`：テストだけの差分でも、構成入口の差分で選ばれる全操作の証拠を要求する。
- `VERIFICATION_BASE_REVISION=work-item/wi-26617 mise run verify-full`：上記の Go 全操作の検証を含む標準検証とブラウザー E2E を選ぶ。公開 HTTP の宣言、証拠ランナーとブラウザーのログアウトに届く本番修正の波及を同じ差分で確認する。
- `mise run test-go-mutation -- backend/oauth2/handlers_http 2 off /private/tmp/wi-13263-oauth-changed-mutation.json b217be43b`：追加した POST の state の条件分岐を自動変異で検査する。coverage と baseline はパッケージ全体を使い、変異を作る範囲だけを変更行へ絞る。

## リスク

多数の操作を一つのテストへ列挙すると、どの応答を検証したかが曖昧になる。
各操作の成功と拒否のケースを明示し、スキップと未実行を成功の証拠にしない。

## 完了

- **完了日**: 2026-10-12
- **要約**:
  未解析の 282 操作について、正式な HTTP 経路で成功と拒否の応答を観測するテストと操作参照を整備した。
  証拠ランナーは一方の成功で他方の未実行、スキップ、失敗を隠さない。
  `spec-diff b217be43b` が示した送受信モデルと既存の拒否の宣言を実際の契約へ合わせ、WS-Federation の応答形式と状態コード、およびメール変更の既存の step-up 拒否の仕様例を補った。
  承認済みのクォータ認可エラー、SAML LogoutResponse の署名後の整形、POST ログアウトの form state の読み取りを既存要件へ戻した。
- **受け入れ RED の証拠**:
  - **テスト**: wi-26617 の `mise run verify`。
  - **要件**: N/A: 製品結果の追加ではなく、既存の HTTP 契約の検証証拠を整備する。
  - **観測した失敗**: 282 操作で同じ実行の成功した契約テストがなく、集約検証が失敗した。
  - **検出できる理由**: 静的解析の未解析とテストの実行結果を分け、操作に結び付いた実行の成功を要求する。
- **単体 RED の証拠**:
  - **テスト**: `tools/verification/src/evidence.test.ts` と `TestProtectedResourceMetadata_unregisteredResource_rejected`。
  - **要件**: REQ-OAUTH2-044
  - **観測した失敗**: 旧判定は拒否側が skipped でも欠落を報告しなかった。HTTP 本文の期待を invalid_request に変えた検査は、実際の invalid_target と異なり失敗した。
  - **検出できる理由**: 一部の成功で未検証を隠す故障と、宣言と違う拒否本文を返す故障を独立して区別する。
- **変更耐性の結果**:
  クォータ更新の認可分岐の 7 変異と、ログアウトの変更行の 2 変異を自動検査で検出した。
  SAML の旧整形は四つの実際の HTTP 応答の署名検証を失敗させ、削除後は成功した。
  クォータの旧エラー返却は 500、POST の旧 state 読み取りは返送値の欠落を再現し、修正後は同じ観測が成功した。
  自動変異の全件数、生存と未被覆の分類は局所検証欄の通りであり、変更していない行の検査不足を等価な変異や対応済みとは扱わない。
  署名後の整形とエラー変換の削除は既定の operator が表現しないため、実際に観測した旧実装の RED を手動故障の証拠とする。
- **検証結果**:
  - `VERIFICATION_BASE_REVISION=work-item/wi-26617 mise run verify` が成功した。Go の race 検査は 6,352 テストが成功し、失敗、スキップ、未実行は 0、DB の 39 パッケージは verified、282 操作すべてに同じ実行の成功した契約証拠があった。
  - 同じ差分の標準検証で tools の 1,020 テスト、UI の 698 テスト、静的検査、型検査とビルドが成功した。
  - 直前の `verify-full` でブラウザー E2E の 39 テストが成功した。その後の差分は仕様例とテストの観測説明および拒否テストの追加だけで、ブラウザーが実行する製品コードは変えていないため、この成功を使う。
  - `check-spec`、`render-docs`、`spec-diff b217be43b` と各操作の局所検査を確認した。
  - 七つの視点で差分をレビューした。観測を対象の HTTP テストへ置き、既存の fixture と契約型を使い、新しい製品依存や公開境界を追加していない。外部プロトコルの検証後の結果を使うダブルは、その HTTP 境界の検査だけを主張する。
