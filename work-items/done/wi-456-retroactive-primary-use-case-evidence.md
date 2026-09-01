---
status: completed
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-08-30
priority: p1
depends_on:
  - wi-445-main-use-case-unit-and-e2e-evidence
  - wi-448-feature-lifecycle-and-update-policy
  - wi-452-feature-maturity-documentation-gates
change_kind: tooling
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: "既存機能の証拠不足を検出する開発用ゲートであり、製品利用者へ告知する機能差分や移行操作はない。"
  references: []
initial_context:
  source:
    - docs/README.md
    - docs/development/testing.md
    - docs/development/specification-first-workflow.md
    - docs/scenarios.md
    - docs/contexts
    - spec/contexts
    - backend
    - frontend/tests/e2e
    - WORK_ITEM_FORMAT.md
    - tools/check/src/primary-use-case-evidence.ts
    - tools/workspace/src/check-workspace.ts
    - mise.toml
    - .agents/skills/implement-work-item/SKILL.md
    - .agents/skills/spec-change/SKILL.md
  tests:
    - backend
    - frontend/tests/e2e
    - tools/check/src/primary-use-case-evidence.test.ts
    - tools/workspace/src/check-workspace.test.ts
  stop_before_reading: [infra, load]
affected_spec:
  - { path: docs/scenarios.md, requirement: REQ-PLATFORM-001 }
  - { path: docs/scenarios.md, requirement: REQ-PLATFORM-002 }
  - { path: docs/scenarios.md, requirement: REQ-PLATFORM-003 }
  - { path: docs/contexts/saml/scenarios.md, requirement: REQ-SAML-004 }
  - { path: docs/contexts/saml/scenarios.md, requirement: REQ-SAML-006 }
  - { path: docs/contexts/identity-management/scenarios.md, requirement: REQ-IDMANAGEMENT-004 }
  - { path: docs/contexts/identity-management/scenarios.md, requirement: REQ-IDMANAGEMENT-009 }
  - { path: docs/contexts/identity-management/scenarios.md, requirement: REQ-IDMANAGEMENT-011 }
  - { path: docs/contexts/identity-management/scenarios.md, requirement: REQ-IDMANAGEMENT-015 }
  - { path: docs/contexts/identity-management/scenarios.md, requirement: REQ-IDMANAGEMENT-016 }
  - { path: docs/contexts/authentication/scenarios.md, requirement: REQ-AUTHENTICATION-001 }
  - { path: docs/contexts/authentication/scenarios.md, requirement: REQ-AUTHENTICATION-007 }
  - { path: docs/contexts/authentication/scenarios.md, requirement: REQ-AUTHENTICATION-011 }
  - { path: docs/contexts/authentication/scenarios.md, requirement: REQ-AUTHENTICATION-013 }
  - { path: docs/contexts/authentication/scenarios.md, requirement: REQ-AUTHENTICATION-016 }
  - { path: docs/contexts/authentication/scenarios.md, requirement: REQ-AUTHENTICATION-026 }
  - { path: docs/contexts/authentication/scenarios.md, requirement: REQ-AUTHENTICATION-030 }
  - { path: docs/contexts/sourcing/scenarios.md, requirement: REQ-SOURCING-002 }
  - { path: docs/contexts/signing-keys/scenarios.md, requirement: REQ-SIGNINGKEYS-001 }
  - { path: docs/contexts/authorization/scenarios.md, requirement: REQ-AUTHORIZATION-001 }
  - { path: docs/contexts/authorization/scenarios.md, requirement: REQ-AUTHORIZATION-003 }
  - { path: docs/contexts/jobs/scenarios.md, requirement: REQ-JOBS-002 }
  - { path: docs/contexts/jobs/scenarios.md, requirement: REQ-JOBS-012 }
  - { path: docs/contexts/seeding/scenarios.md, requirement: REQ-SEEDING-010 }
  - { path: docs/contexts/oauth2/scenarios.md, requirement: REQ-OAUTH2-005 }
  - { path: docs/contexts/oauth2/scenarios.md, requirement: REQ-OAUTH2-006 }
  - { path: docs/contexts/oauth2/scenarios.md, requirement: REQ-OAUTH2-026 }
  - { path: docs/contexts/oauth2/scenarios.md, requirement: REQ-OAUTH2-027 }
  - { path: docs/contexts/oauth2/scenarios.md, requirement: REQ-OAUTH2-035 }
  - { path: docs/contexts/oauth2/scenarios.md, requirement: REQ-OAUTH2-041 }
  - { path: docs/contexts/data-keys/scenarios.md, requirement: REQ-DATAKEYS-002 }
  - { path: docs/contexts/workloadidentity/scenarios.md, requirement: REQ-WORKLOADIDENTITY-001 }
  - { path: docs/contexts/workloadidentity/scenarios.md, requirement: REQ-WORKLOADIDENTITY-008 }
  - { path: docs/contexts/provisioning/scenarios.md, requirement: REQ-PROVISIONING-002 }
  - { path: docs/contexts/provisioning/scenarios.md, requirement: REQ-PROVISIONING-003 }
  - { path: docs/contexts/tenancy/scenarios.md, requirement: REQ-TENANCY-004 }
  - { path: docs/contexts/tenancy/scenarios.md, requirement: REQ-TENANCY-006 }
  - { path: docs/contexts/tenancy/scenarios.md, requirement: REQ-TENANCY-011 }
  - { path: docs/contexts/claim-mapping/scenarios.md, requirement: REQ-CLAIMMAPPING-001 }
  - { path: docs/contexts/audit/scenarios.md, requirement: REQ-AUDIT-001 }
  - { path: docs/contexts/api-tokens/scenarios.md, requirement: REQ-APITOKENS-003 }
  - { path: docs/contexts/system/scenarios.md, requirement: REQ-SYSTEM-002 }
  - { path: docs/contexts/system/scenarios.md, requirement: REQ-SYSTEM-010 }
  - { path: docs/contexts/ws-federation/scenarios.md, requirement: REQ-WSFEDERATION-002 }
  - { path: docs/contexts/ws-federation/scenarios.md, requirement: REQ-WSFEDERATION-004 }
  - { path: docs/contexts/sharedsignals/scenarios.md, requirement: REQ-SHAREDSIGNALS-001 }
  - { path: docs/contexts/sharedsignals/scenarios.md, requirement: REQ-SHAREDSIGNALS-006 }
  - { path: docs/contexts/application/scenarios.md, requirement: REQ-APPLICATION-007 }
  - { path: docs/contexts/application/scenarios.md, requirement: REQ-APPLICATION-012 }
  - { path: docs/contexts/identity-governance/scenarios.md, requirement: REQ-IDGOVERNANCE-001 }
  - { path: docs/contexts/identity-governance/scenarios.md, requirement: REQ-IDGOVERNANCE-003 }
primary_use_cases:
  - id: platform-principal-lifecycle
    requirement: REQ-PLATFORM-001
    observable_result: "無効化した主体の認証、トークン、連携経路が閉じる。"
    unit_test: { path: backend/idmanagement/user/usecases/admin_users_test.go, name: TestSetUserDisabledAllowsDisablingOtherAdmin, task: test-go-race }
    e2e_test: { path: backend/shared/http/server_http/routes_e2e_test.go, name: TestDisabledUserLoginAndExistingSessionAreRejected, task: test-go-race }
    unit_fault_model: "無効化しても User を active のまま保存する。"
    e2e_fault_model: "ログインまたは既存セッションの失効配線を外す。"
  - id: platform-deletion-lifecycle
    requirement: REQ-PLATFORM-002
    observable_result: "削除予約中は到達不能になり、復元後は再び到達できる。"
    unit_test: { path: backend/idmanagement/user/usecases/admin_users_test.go, name: TestSoftDeleteUserSetsPendingDeletionWithoutCascade, task: test-go-race }
    e2e_test: { path: backend/idmanagement/user/handlers_http/admin_user_handler_test.go, name: TestAdminUserAPISoftDeletesAndRestores, task: test-go-race }
    unit_fault_model: "削除予約で pending_deletion へ遷移しない。"
    e2e_fault_model: "復元 API が永続状態を active へ戻さない。"
  - id: platform-downstream-propagation
    requirement: REQ-PLATFORM-003
    observable_result: "上流の正の変更が接続済み下流の状態へ収束する。"
    unit_test: { path: backend/provisioning/usecases/capture_test.go, name: TestCaptureLifecycleEvent_UserAttributesChanged_TranslatesToUpdate, task: test-go-race }
    e2e_test: { path: backend/provisioning/e2e_capture_delivery_test.go, name: TestE2E_CreateUpdateDisableDelete_ReachesRealDownstream, task: test-go-race }
    unit_fault_model: "属性変更を update 配信へ投影しない。"
    e2e_fault_model: "捕捉した配信を実下流クライアントへ渡さない。"
  - id: saml-federation
    requirement: REQ-SAML-006
    observable_result: "登録済み SP が検証可能な SAML Response を受け取る。"
    unit_test: { path: backend/saml/domain/idp_profile_test.go, name: TestSamlServiceProviderMatchesOnlyAssignedProfile, task: test-go-race }
    e2e_test: { path: backend/saml/handlers_http/saml_handler_test.go, name: TestSamlSSO_SPInitiatedAuthenticatedIssuesPostForm, task: test-go-race }
    unit_fault_model: "SP と割り当て済み IdP プロファイルの一致判定を反転する。"
    e2e_fault_model: "SSO 入口で assertion へ署名しない。"
  - id: saml-configuration
    requirement: REQ-SAML-004
    observable_result: "管理設定が対象プロファイルのメタデータと信頼境界に反映される。"
    unit_test: { path: backend/saml/domain/idp_profile_test.go, name: TestSamlIdentityProviderProfileValidation, task: test-go-race }
    e2e_test: { path: backend/saml/handlers_http/saml_handler_test.go, name: TestAdminIDPProfileCRUDAndCanonicalEndpoints, task: test-go-race }
    unit_fault_model: "共有・専用プロファイルの不変条件を受理する。"
    e2e_fault_model: "管理 API の保存結果をメタデータ取得へ結ばない。"
  - id: identity-user-lifecycle
    requirement: REQ-IDMANAGEMENT-011
    observable_result: "対象 User が削除予約と復元の状態へ遷移する。"
    unit_test: { path: backend/idmanagement/user/usecases/admin_users_test.go, name: TestRestoreUserReturnsToActive, task: test-go-race }
    e2e_test: { path: backend/idmanagement/user/handlers_http/admin_user_handler_test.go, name: TestAdminUserAPISoftDeletesAndRestores, task: test-go-race }
    unit_fault_model: "復元時に lifecycle を active へ戻さない。"
    e2e_fault_model: "管理 API の復元操作を use case へ接続しない。"
  - id: identity-self-service
    requirement: REQ-IDMANAGEMENT-016
    observable_result: "保存後のプロフィール表示へ本人の変更が反映される。"
    unit_test: { path: backend/idmanagement/user/usecases/account_profile_test.go, name: TestUpdateUserProfileEditsNameAndEditableAttribute, task: test-go-race }
    e2e_test: { path: backend/idmanagement/user/handlers_http/account_handler_test.go, name: TestAccountProfilePatchUpdatesEditableAttribute, task: test-go-race }
    unit_fault_model: "編集可能属性を保存対象から外す。"
    e2e_fault_model: "account PATCH の actor を更新 use case へ渡さない。"
  - id: identity-bulk-transfer
    requirement: REQ-IDMANAGEMENT-004
    observable_result: "CSV の有効な行だけが計画どおりに User 状態へ反映される。"
    unit_test: { path: backend/idmanagement/user/usecases/user_import_planner_test.go, name: TestPlanUserImportCreateUpdateUnchangedAndFieldPresence, task: test-go-race }
    e2e_test: { path: backend/idmanagement/user/handlers_http/admin_user_import_e2e_test.go, name: TestAdminUserImportPrimaryUseCase_REQ_IDMANAGEMENT_004, task: test-go-race }
    unit_fault_model: "有効な create 行を unchanged と計画する。"
    e2e_fault_model: "preview 成果物を apply Job と実保存へ接続しない。"
  - id: identity-group-management
    requirement: REQ-IDMANAGEMENT-015
    observable_result: "Group 所属後の有効ロールに Group 由来権限が反映される。"
    unit_test: { path: backend/idmanagement/group/usecases/admin_groups_test.go, name: TestGroupCreateAddMemberEffectiveRoles, task: test-go-race }
    e2e_test: { path: backend/idmanagement/group/handlers_http/admin_group_handler_test.go, name: TestAdminGroupAPICreateAddMemberAndEffectiveRoles, task: test-go-race }
    unit_fault_model: "Group ロールを有効ロールへ合成しない。"
    e2e_fault_model: "メンバー追加 API を所属 Repository へ接続しない。"
  - id: identity-agent-management
    requirement: REQ-IDMANAGEMENT-009
    observable_result: "登録した Agent にクライアント資格情報の関連付けが保存される。"
    unit_test: { path: backend/idmanagement/agent/usecases/admin_agents_test.go, name: TestBindUnbindCredentialAndFindByClientID, task: test-go-race }
    e2e_test: { path: backend/idmanagement/handlers_http/extra_identity_test.go, name: TestAdminAgentLifecycle, task: test-go-race }
    unit_fault_model: "AgentCredentialBinding を保存しない。"
    e2e_fault_model: "Agent 管理 API から資格情報関連付けを外す。"
  - id: authentication-federation
    requirement: REQ-AUTHENTICATION-001
    observable_result: "検証済みの同一外部主体が同じローカル User のセッションになる。"
    unit_test: { path: backend/authentication/federation/usecases/broker_test.go, name: TestCompleteUsesExistingFederatedIdentityAndIssuesSession, task: test-go-race }
    e2e_test: { path: backend/authentication/federation/handlers_http/federated_login_e2e_test.go, name: TestFederatedLoginPrimaryUseCase_REQ_AUTHENTICATION_001, task: test-go-race }
    unit_fault_model: "既存の外部主体関連付けを無視して別 User を作る。"
    e2e_fault_model: "callback の検証済み subject を broker 完了処理へ渡さない。"
  - id: authentication-password-login
    requirement: REQ-AUTHENTICATION-007
    observable_result: "パスワード認証済みセッションで元の認可処理を継続できる。"
    unit_test: { path: backend/oauth2/authorization/usecases/complete_login_test.go, name: TestCompleteLogin, task: test-go-race }
    e2e_test: { path: backend/shared/http/server_http/routes_e2e_test.go, name: TestBrowserAuthorizationFlowUsesCookiesAndJSONAPI, task: test-go-race }
    unit_fault_model: "認証済み User を認可要求へ結び付けない。"
    e2e_fault_model: "browser login 成功後の session cookie または authorize 再開配線を外す。"
  - id: authentication-password-recovery
    requirement: REQ-AUTHENTICATION-016
    observable_result: "単回リンクの消費後に新しい資格情報で認証できる。"
    unit_test: { path: backend/authentication/password/usecases/password_reset_test.go, name: TestResetPasswordWithTokenConsumesTokenAndUpdatesPassword, task: test-go-race }
    e2e_test: { path: backend/authentication/password/handlers_http/password_reset_handler_test.go, name: TestPasswordResetHTTPFlow, task: test-go-race }
    unit_fault_model: "トークン消費後もパスワードハッシュを更新しない。"
    e2e_fault_model: "reset HTTP 入口を token store と password use case へ接続しない。"
  - id: authentication-mfa
    requirement: REQ-AUTHENTICATION-011
    observable_result: "登録済み要素による第二要素の成立がセッションへ反映される。"
    unit_test: { path: backend/authentication/totp/usecases/verify_totp_factor_test.go, name: TestVerifyTOTPFactorUpdatesLastUsedAt, task: test-go-race }
    e2e_test: { path: backend/shared/http/server_http/routes_e2e_test.go, name: TestBrowserAuthorizationFlowRequiresTOTPWhenPolicyRequiresMFA, task: test-go-race }
    unit_fault_model: "正しい TOTP を失敗として扱う。"
    e2e_fault_model: "第二要素の成立を login session の AMR へ反映しない。"
  - id: authentication-sessions
    requirement: REQ-AUTHENTICATION-013
    observable_result: "選択した自身のセッションが失効し、以後の利用を拒否される。"
    unit_test: { path: backend/authentication/session/usecases/sessions_test.go, name: TestRevokeOtherSessionsKeepsCurrent, task: test-go-race }
    e2e_test: { path: frontend/tests/e2e/ui-scenario-actions.spec.ts, name: account session list can revoke a different browser session, task: test-ui-e2e }
    unit_fault_model: "選択セッションの revoke を保存しない。"
    e2e_fault_model: "account UI の revoke 操作を session API へ送らない。"
  - id: authentication-trusted-device
    requirement: REQ-AUTHENTICATION-026
    observable_result: "同意済み端末の次回ログインだけが第二要素を省略できる。"
    unit_test: { path: backend/authentication/trusteddevice/usecases/trusted_devices_test.go, name: TestEvaluateTrustsTheIssuedCookieAndRotatesIt, task: test-go-race }
    e2e_test: { path: backend/shared/http/server_http/trusted_device_e2e_test.go, name: TestTrustedDeviceSkipsTheSecondFactorOnTheNextLogin, task: test-go-race }
    unit_fault_model: "有効な端末 cookie を信頼済みと判定しない。"
    e2e_fault_model: "第二要素成功時の cookie 発行または次回 login の読取りを外す。"
  - id: authentication-notifications
    requirement: REQ-AUTHENTICATION-030
    observable_result: "対象となる新規端末のサインインだけが本人へ通知される。"
    unit_test: { path: backend/authentication/securitynotification/usecases/dispatch_test.go, name: TestDispatchNotifiesOnlyTheFirstSignInFromEachDevice, task: test-go-race }
    e2e_test: { path: backend/cmd/internal/bootstrap/securitynotification_test.go, name: TestEmitFuncSendsSecurityNotificationsForCatalogEvents, task: test-go-race }
    unit_fault_model: "既知端末にも重複通知する。"
    e2e_fault_model: "組み立て地点の event emit から通知 dispatcher を外す。"
  - id: scim-sourcing
    requirement: REQ-SOURCING-002
    observable_result: "SCIM の外部資源操作が正規の User / Group 状態へ反映される。"
    unit_test: { path: backend/sourcing/scim/usecases/users_test.go, name: TestUpdateUserFullReplace, task: test-go-race }
    e2e_test: { path: backend/sourcing/scim/handlers_http/scim_test.go, name: TestScimInboundProvisioning, task: test-go-race }
    unit_fault_model: "SCIM 更新を User の正規状態へ投影しない。"
    e2e_fault_model: "SCIM HTTP 入口を source adapter と IdManagement へ接続しない。"
  - id: signing-key-lifecycle
    requirement: REQ-SIGNINGKEYS-001
    observable_result: "新しい鍵で署名しつつ猶予中の旧 kid も検証に利用できる。"
    unit_test: { path: backend/signingkeys/usecases/rotate_signing_key_test.go, name: TestRotateSigningKeyKeepsPreviousKidInJWKS, task: test-go-race }
    e2e_test: { path: backend/oauth2/handlers_http/admin_key_handler_test.go, name: TestAdminKeysRotateSucceedsAndEmitsEvent, task: test-go-race }
    unit_fault_model: "回転と同時に旧鍵を無効化し JWKS から外す。"
    e2e_fault_model: "管理 UI の回転操作を tenant key store へ接続しない。"
  - id: relationship-authorization
    requirement: REQ-AUTHORIZATION-003
    observable_result: "継承、Group、親子関係を含むアクセス判定結果が返る。"
    unit_test: { path: backend/authorization/domain/evaluator_test.go, name: TestCheckTraversesGroupAndParent, task: test-go-race }
    e2e_test: { path: backend/authorization/handlers_http/routes_test.go, name: TestAuthorizationAdminRoutes, task: test-go-race }
    unit_fault_model: "Group または親子辺の探索を打ち切る。"
    e2e_fault_model: "check HTTP 入口を関係 store と evaluator へ接続しない。"
  - id: authorization-model-admin
    requirement: REQ-AUTHORIZATION-001
    observable_result: "整合する認可モデルの版だけが公開され後続判定に利用される。"
    unit_test: { path: backend/authorization/usecases/check_access_test.go, name: TestPutAuthorizationModelRejectsAnInconsistentModel, task: test-go-race }
    e2e_test: { path: backend/authorization/handlers_http/routes_test.go, name: TestAuthorizationAdminRoutes, task: test-go-race }
    unit_fault_model: "整合しないモデルを published として保存する。"
    e2e_fault_model: "管理 API の model 更新を store へ接続しない。"
  - id: durable-job-execution
    requirement: REQ-JOBS-002
    observable_result: "投入した Job の効果が一度だけ実行され成功状態になる。"
    unit_test: { path: backend/jobs/domain/job_test.go, name: TestTransitionJobLifecycle_DeclaredTransitions, task: test-go-race }
    e2e_test: { path: backend/jobs/usecases/runner_test.go, name: TestRunner_SuccessPath, task: test-go-race }
    unit_fault_model: "running から succeeded への遷移を拒否する。"
    e2e_fault_model: "Runner が登録済み handler の実行結果を Complete へ渡さない。"
  - id: job-administration
    requirement: REQ-JOBS-012
    observable_result: "管理者が自テナントの Job を参照し未終端 Job を取消できる。"
    unit_test: { path: backend/jobs/usecases/admin_test.go, name: TestCancelJobForAdmin, task: test-go-race }
    e2e_test: { path: backend/jobs/handlers_http/admin_job_handler_test.go, name: TestCancelJob, task: test-go-race }
    unit_fault_model: "取消可能な Job を canceled へ遷移しない。"
    e2e_fault_model: "cancel API を管理 use case へ接続しない。"
  - id: convergent-seeding
    requirement: REQ-SEEDING-010
    observable_result: "部分失敗後を含め再実行で宣言した状態に収束する。"
    unit_test: { path: backend/seeding/usecases/plan_test.go, name: TestRunCanBeRetriedAfterApplyFailure, task: test-go-race }
    e2e_test: { path: backend/cmd/internal/bootstrap/seeding_test.go, name: TestSeedDryRunDoesNotMutateAndRepeatedApplyConverges, task: test-go-race }
    unit_fault_model: "既存一致資源を毎回更新対象にする。"
    e2e_fault_model: "bootstrap の contributor 集約から一つの適用処理を外す。"
  - id: oauth-authorization-code
    requirement: REQ-OAUTH2-005
    observable_result: "認可と同意を経たクライアントがアクセストークンと ID トークンを取得する。"
    unit_test: { path: backend/oauth2/token/usecases/exchange_code_test.go, name: TestExchangeCodeIssuesTokensByScope, task: test-go-race }
    e2e_test: { path: backend/shared/http/server_http/routes_e2e_test.go, name: TestBrowserAuthorizationFlowUsesCookiesAndJSONAPI, task: test-go-race }
    unit_fault_model: "交換済み認可コードを再利用可能にする。"
    e2e_fault_model: "token endpoint から認可コード交換を外す。"
  - id: oauth-token-lifecycle
    requirement: REQ-OAUTH2-006
    observable_result: "ローテーション後の新トークンだけが有効になる。"
    unit_test: { path: backend/oauth2/token/usecases/refresh_tokens_test.go, name: TestRefreshTokensAcceptsMatchingDPoPProof, task: test-go-race }
    e2e_test: { path: backend/shared/http/server_http/routes_e2e_test.go, name: TestTokenLifecycleRotatesRefreshTokenAndInvalidatesTheUsedOne, task: test-go-race }
    unit_fault_model: "refresh token のローテーション時に旧 token を失効しない。"
    e2e_fault_model: "token endpoint から refresh grant の配線を外す。"
  - id: oauth-machine-token
    requirement: REQ-OAUTH2-026
    observable_result: "許可された機械クライアントが対象資源用トークンを取得する。"
    unit_test: { path: backend/oauth2/domain/domain_wi129_test.go, name: TestScopeIntersection, task: test-go-race }
    e2e_test: { path: backend/shared/http/server_http/login_metrics_e2e_test.go, name: TestTokenMetricsRecordSuccessfulClientCredentialsGrant, task: test-go-race }
    unit_fault_model: "要求 scope を許可済み scope 集合で絞り込まない。"
    e2e_fault_model: "組み立て済み server から client credentials grant を外す。"
  - id: oauth-device-flow
    requirement: REQ-OAUTH2-027
    observable_result: "利用者の承認後にデバイスがトークンを一度だけ取得する。"
    unit_test: { path: backend/oauth2/device/usecases/device_flow_test.go, name: TestDeviceFlowPollingAndReplay, task: test-go-race }
    e2e_test: { path: backend/oauth2/handlers_http/device_handler_test.go, name: TestDeviceAuthorizationAPI, task: test-go-race }
    unit_fault_model: "承認済み device code を再利用可能にする。"
    e2e_fault_model: "device authorization endpoint と token endpoint の共有 store を分離する。"
  - id: oauth-human-approval
    requirement: REQ-OAUTH2-041
    observable_result: "対象本人の承認後にバックチャネルクライアントが一度だけトークンを取得する。"
    unit_test: { path: backend/oauth2/approval/usecases/approval_flow_test.go, name: TestApprovalFlowPollingDecisionAndReplay, task: test-go-race }
    e2e_test: { path: backend/oauth2/handlers_http/approval_handler_test.go, name: TestBackchannelApprovalIssuesTokenOnce, task: test-go-race }
    unit_fault_model: "承認要求の一回限りの交換制約を外す。"
    e2e_fault_model: "本人の approval session とバックチャネル要求の関連付けを外す。"
  - id: oauth-client-administration
    requirement: REQ-OAUTH2-035
    observable_result: "自テナントの OAuth client 設定が作成・更新・削除される。"
    unit_test: { path: backend/oauth2/client/usecases/admin_clients_test.go, name: TestAdminOAuth2Client, task: test-go-race }
    e2e_test: { path: backend/oauth2/handlers_http/admin_client_handler_test.go, name: TestAdminOAuth2ClientCRUD, task: test-go-race }
    unit_fault_model: "更新時に tenant 所有権を検証しない。"
    e2e_fault_model: "管理 API の更新対象 client を use case へ渡さない。"
  - id: data-key-lifecycle
    requirement: REQ-DATAKEYS-002
    observable_result: "DEK の更新後も新旧暗号文を復号でき、再暗号化へ進める。"
    unit_test: { path: backend/datakeys/usecases/lifecycle_test.go, name: TestRotateTenantDataKeyThenDecryptStillWorksForOldVersion, task: test-go-race }
    e2e_test: { path: backend/datakeys/field_cipher_test.go, name: TestFieldCipherDecryptsExistingCiphertextAfterRotation, task: test-go-race }
    unit_fault_model: "更新時に旧 DEK を retiring として保持しない。"
    e2e_fault_model: "管理 API の DEK 操作を tenant key provider へ接続しない。"
  - id: workload-credential-exchange
    requirement: REQ-WORKLOADIDENTITY-001
    observable_result: "登録済み信頼と関連付けから対象 Agent の資格情報を得る。"
    unit_test: { path: backend/workloadidentity/usecases/verify_workload_attestation_test.go, name: TestVerifyWorkloadAttestation_Success, task: test-go-race }
    e2e_test: { path: backend/oauth2/handlers_http/token_exchange_handler_test.go, name: TestTokenExchangeIssuesWorkloadCredential, task: test-go-race }
    unit_fault_model: "証明の issuer と subject を trust bundle に照合しない。"
    e2e_fault_model: "token exchange endpoint から workload attestation verifier を外す。"
  - id: workload-trust-administration
    requirement: REQ-WORKLOADIDENTITY-008
    observable_result: "信頼設定の状態変更が後続の資格情報交換可否に反映される。"
    unit_test: { path: backend/workloadidentity/usecases/admin_trust_bundles_test.go, name: TestWorkloadTrustBundleDisableEnableLifecycle, task: test-go-race }
    e2e_test: { path: backend/workloadidentity/handlers_http/routes_test.go, name: TestAdminWorkloadTrustBundleLifecycle, task: test-go-race }
    unit_fault_model: "disabled trust bundle を検証時に有効として扱う。"
    e2e_fault_model: "管理 API の状態変更を verifier が読む repository へ保存しない。"
  - id: provisioning-connection
    requirement: REQ-PROVISIONING-002
    observable_result: "下流接続のテスト結果と対応機能が管理状態へ保存される。"
    unit_test: { path: backend/provisioning/usecases/admin_test.go, name: TestRegisterConnection_SeedsDefaultAttributeMappingAndRejectsDuplicate, task: test-go-race }
    e2e_test: { path: backend/provisioning/handlers_http/admin_connection_handler_test.go, name: TestAdminProvisioningConnectionLifecycle, task: test-go-race }
    unit_fault_model: "接続登録時に既定属性対応付けを保存しない。"
    e2e_fault_model: "接続テスト結果を管理 API の応答と repository へ反映しない。"
  - id: provisioning-delivery
    requirement: REQ-PROVISIONING-003
    observable_result: "IdManagement の対象 User 変更が下流状態へ収束する。"
    unit_test: { path: backend/provisioning/usecases/capture_test.go, name: TestCaptureLifecycleEvent_UserCreated_AllUsersScope, task: test-go-race }
    e2e_test: { path: backend/provisioning/e2e_capture_delivery_test.go, name: TestE2E_CreateUpdateDisableDelete_ReachesRealDownstream, task: test-go-race }
    unit_fault_model: "UserCreated を下流 create 配送へ射影しない。"
    e2e_fault_model: "IdManagement event subscriber と delivery worker の接続を外す。"
  - id: tenant-resolution
    requirement: REQ-TENANCY-006
    observable_result: "正規ロケーションの要求だけが対象 Tenant へ到達する。"
    unit_test: { path: backend/tenancy/usecases/manage_tenants_test.go, name: TestSetEndpointStyle, task: test-go-race }
    e2e_test: { path: backend/shared/http/server_http/tenant_host_routes_test.go, name: TestTenantIsReachableOnlyAtItsCanonicalLocation, task: test-go-race }
    unit_fault_model: "endpoint style 変更時に正規 host/path 制約を保存しない。"
    e2e_fault_model: "組み立て済み router の canonical location guard を外す。"
  - id: tenant-administration
    requirement: REQ-TENANCY-011
    observable_result: "System 管理者の操作で Tenant の正規ロケーションと運用設定が更新される。"
    unit_test: { path: backend/tenancy/usecases/manage_tenants_test.go, name: TestTenantLifecycle, task: test-go-race }
    e2e_test: { path: backend/tenancy/handlers_http/admin_settings_handler_test.go, name: TestAdminSettingsPatchUpdatesAndEmitsEvent, task: test-go-race }
    unit_fault_model: "既存 realm と衝突する Tenant の作成を拒否しない。"
    e2e_fault_model: "control-plane PATCH を tenant repository へ接続しない。"
  - id: tenant-experience
    requirement: REQ-TENANCY-004
    observable_result: "対象 Tenant の UI と通知へ branding 設定が反映される。"
    unit_test: { path: backend/tenancy/usecases/manage_branding_test.go, name: TestUpdateBrandingPersistsAndClearsFields, task: test-go-race }
    e2e_test: { path: backend/tenancy/handlers_http/branding_handler_test.go, name: TestUpdateBrandingPersistsAndIsPubliclyVisible, task: test-go-race }
    unit_fault_model: "branding 更新の明示的な field clear を無視する。"
    e2e_fault_model: "管理更新と公開読取が別テナントの branding を指す。"
  - id: claim-release
    requirement: REQ-CLAIMMAPPING-001
    observable_result: "対応付けのある公開可能な属性だけが発行物に含まれる。"
    unit_test: { path: backend/claimmapping/usecases/floor_test.go, name: TestIssueClaimsWithFloor_RejectsPrivateSourceAttribute, task: test-go-race }
    e2e_test: { path: backend/oauth2/handlers_http/userinfo_handler_test.go, name: TestUserInfoAppliesClientClaimMappingPolicy, task: test-go-race }
    unit_fault_model: "private 属性の公開 floor を適用しない。"
    e2e_fault_model: "userinfo 発行経路から application claim policy を外す。"
  - id: audit-query
    requirement: REQ-AUDIT-001
    observable_result: "絞り込みに一致する自テナントの監査履歴を検索・取得できる。"
    unit_test: { path: backend/audit/usecases/audit_search_test.go, name: TestParseAuditFilterAcceptsAllowlisted, task: test-go-race }
    e2e_test: { path: backend/audit/handlers_http/admin_audit_event_handler_test.go, name: TestAdminAuditEventsExportSetsAttachment, task: test-go-race }
    unit_fault_model: "許可済み検索属性を filter parser で拒否する。"
    e2e_fault_model: "export endpoint の絞り込み条件を repository 検索へ渡さない。"
  - id: api-token-lifecycle
    requirement: REQ-APITOKENS-003
    observable_result: "選択スコープの API token が発行され、失効後は認証できない。"
    unit_test: { path: backend/apitoken/usecases/usecases_test.go, name: TestIssueListAndRevokeApiToken, task: test-go-race }
    e2e_test: { path: backend/apitoken/handlers_http/handlers_test.go, name: TestAdminApiTokenLifecycle, task: test-go-race }
    unit_fault_model: "失効時に token lifecycle record を無効化しない。"
    e2e_fault_model: "管理 API の失効が対象 token の記録へ届かない。"
  - id: runtime-health
    requirement: REQ-SYSTEM-002
    observable_result: "probe がプロセス lifecycle と依存障害を区別して返す。"
    unit_test: { path: backend/shared/http/server_http/health_handler_test.go, name: TestReadinessReportsDependencyFailure, task: test-go-race }
    e2e_test: { path: backend/shared/http/server_http/routes_e2e_test.go, name: TestHealthProbes, task: test-go-race }
    unit_fault_model: "未準備の依存先を ready と集約する。"
    e2e_fault_model: "組み立て済み health route が起動完了状態を読まない。"
  - id: localized-ui
    requirement: REQ-SYSTEM-010
    observable_result: "管理・アカウント・認証画面が選択言語で描画される。"
    unit_test: { path: frontend/src/lib/i18n/resolveLocale.test.ts, name: uses the saved locale when no supported hint is present, task: test-ui-unit }
    e2e_test: { path: frontend/tests/e2e/localized-ui.spec.ts, name: selected locale renders authentication account and admin surfaces, task: test-ui-e2e }
    unit_fault_model: "保存済み locale を解決優先順位から外す。"
    e2e_fault_model: "router root から LocaleProvider を外す。"
  - id: wsfed-passive-sign-in
    requirement: REQ-WSFEDERATION-002
    observable_result: "登録済み RP が検証可能な WS-Fed サインイン応答を受け取る。"
    unit_test: { path: backend/wsfederation/domain/wsfed_test.go, name: TestValidateSignIn_HappyPathWithWreply, task: test-go-race }
    e2e_test: { path: backend/wsfederation/handlers_http/wsfed_handler_test.go, name: TestWsFedSignIn_AuthenticatedIssuesPassiveForm, task: test-go-race }
    unit_fault_model: "検証結果の返信先へ要求された wreply を反映しない。"
    e2e_fault_model: "passive endpoint から署名済み assertion 発行を外す。"
  - id: wstrust-token-issuance
    requirement: REQ-WSFEDERATION-004
    observable_result: "妥当な WS-Trust Issue 要求に検証可能な RSTR を返す。"
    unit_test: { path: backend/wsfederation/responses_wsfederation/rstr_test.go, name: TestBuildRSTR, task: test-go-race }
    e2e_test: { path: backend/wsfederation/handlers_http/wsfed_handler_test.go, name: TestWsTrustUsernameMixed_IssuesRSTR, task: test-go-race }
    unit_fault_model: "RSTR の AppliesTo と token lifetime を要求から反映しない。"
    e2e_fault_model: "WS-Trust endpoint から資格情報検証または token signer を外す。"
  - id: shared-signal-revocation
    requirement: REQ-SHAREDSIGNALS-001
    observable_result: "失効対象 Agent の既発行 token が即時に active=false になる。"
    unit_test: { path: backend/sharedsignals/usecases/revocation_test.go, name: TestAdvanceRevocationEpoch_EmitsForEachAgent, task: test-go-race }
    e2e_test: { path: backend/idmanagement/handlers_http/extra_identity_test.go, name: TestAdminAgentKill_AdvancesRevocationEpoch, task: test-go-race }
    unit_fault_model: "失効時に Agent の revocation epoch を進めない。"
    e2e_fault_model: "shared signal の受信処理と token introspection の epoch repository を分離する。"
  - id: shared-signal-stream
    requirement: REQ-SHAREDSIGNALS-006
    observable_result: "セキュリティイベント配送が成功または明示的な終端状態になる。"
    unit_test: { path: backend/sharedsignals/usecases/admin_streams_test.go, name: TestRegisterSsfTransmitterStream_Succeeds, task: test-go-race }
    e2e_test: { path: backend/sharedsignals/handlers_http/e2e_stream_delivery_test.go, name: TestTransmitterStreamLifecycleAndDelivery, task: test-go-race }
    unit_fault_model: "送信 stream の event type 購読を保存しない。"
    e2e_fault_model: "stream 管理 API が保存した送信設定を配送側 repository へ書かない。"
  - id: application-configuration
    requirement: REQ-APPLICATION-007
    observable_result: "保存した Application が選択した一つのプロトコル設定を持つ。"
    unit_test: { path: backend/application/usecases/applications_admin_test.go, name: TestUpdateApplicationChangesAndNoop, task: test-go-race }
    e2e_test: { path: backend/application/handlers_http/application_handler_test.go, name: TestApplicationAdminCRUDAndAccountVisibility, task: test-go-race }
    unit_fault_model: "更新結果を application repository へ保存しない。"
    e2e_fault_model: "管理 API の Application 定義を use case と repository へ渡さない。"
  - id: application-access-policy
    requirement: REQ-APPLICATION-012
    observable_result: "可視性とプロトコル利用可否が割り当てと sign-in policy に従う。"
    unit_test: { path: backend/application/usecases/applications_test.go, name: TestCreateAndListMyApplicationsRespectsAssignmentAndVisibility, task: test-go-race }
    e2e_test: { path: backend/oauth2/handlers_http/authorize_handler_test.go, name: TestAuthorizeAllowsHiddenApplicationAssignment, task: test-go-race }
    unit_fault_model: "割り当てのない非公開 Application を account 一覧へ含める。"
    e2e_fault_model: "authorize 経路の割当判定が hidden 割当を未割当として扱う。"
  - id: lifecycle-workflow-administration
    requirement: REQ-IDGOVERNANCE-001
    observable_result: "有効な workflow 定義の版が保存され実行対象になる。"
    unit_test: { path: backend/idgovernance/usecases/lifecycle_workflows_test.go, name: TestLifecycleWorkflowCreateUpdateAndTransitions, task: test-go-race }
    e2e_test: { path: backend/idgovernance/handlers_http/admin_lifecycle_workflow_handler_test.go, name: TestAdminLifecycleWorkflowDryRunReflectsActualUserState, task: test-go-race }
    unit_fault_model: "enable 時に draft revision を固定しない。"
    e2e_fault_model: "dry-run が管理 API の対象 User を参照しない。"
  - id: lifecycle-workflow-execution
    requirement: REQ-IDGOVERNANCE-003
    observable_result: "User 変更から Group と Application の割り当てが目的状態へ収束する。"
    unit_test: { path: backend/idgovernance/usecases/lifecycle_workflow_dispatcher_test.go, name: TestLifecycleWorkflowRunHandlerEmitsRunStartedAndRunSucceeded, task: test-go-race }
    e2e_test: { path: backend/idgovernance/usecases/lifecycle_workflow_dispatcher_test.go, name: TestUserChangeRunsLifecycleWorkflowToDeclaredEffects, task: test-go-race }
    unit_fault_model: "成功した workflow step の checkpoint を保存しない。"
    e2e_fault_model: "dispatcher が未投入の run を Job へ投入しない。"
---

# 既存の全機能を主要ユースケース証拠へ遡及移行する

## Motivation

`wi-445` は、新たに着手する機能、欠陥修正、標準対応へ `risk-based-v3` を適用した一方、完了済みの `risk-based-v1` / `risk-based-v2` 記録と既存機能には遡及適用しなかった。この互換性は履歴を改変しないために必要だが、既存機能については、正式な入口から最終効果まで到達する E2E テストがあるか、内部判断と実配線を別々に壊したときに検出できるかを一律には説明できない。

過去の work item を新しい形式へ書き換えても、当時観測していない RED や故障注入結果を後付けで作るだけになる。必要なのは履歴の再解釈ではなく、現在の正準文書、公開入口、実装境界から主要ユースケース一覧を導出し、現在の実装とテストに対して新しい証拠を実測する移行である。`ProductFeatureRegistry` は実行時選択と更新影響を所有する別の仕組みであり、常時提供機能を列挙しないため、本項目の導出元にはしない。

## Scope

- 現行の非廃止 `REQ-*`、TypeSpec の公開入口、`docs/README.md` の Context 責務、実装の垂直機能境界から主要ユースケースを導出する。
- 同じ主体、正式入口の機能族、利用者・永続状態・外部境界から観測できる最終効果を共有するシナリオを一つの主要ユースケースへまとめる。
- 各主要ユースケースへ内部判断を検証する単体テストと、正式な入口から構成・経路制御・アダプターを通る E2E テストを関連付け、責任を満たさない不足テストを追加または深める。
- 単体側の内部判断と E2E 側の実配線または最終効果へ異なる故障を注入し、対応するテストが失敗した結果を残す。
- 作業項目の構造化計画と既存の `risk-based-v3` 検査を使って、要求参照、テスト識別子、必須 `mise` タスクからの到達性を検査する。別の機能台帳は追加しない。

## Out of Scope

- `ProductFeatureRegistry`、機能成熟度、機能フラグを主要ユースケースの導出に使うこと。
- 完了済み work item の frontmatter や完了証拠を `risk-based-v3` へ書き換えること。
- 全規範シナリオを一件ずつ主要ユースケースにすること。代替経路、拒否、不変条件は対応する中心経路のリスク証拠に残す。
- 監査で見つかった製品欠陥を本 work item の差分へ混ぜて修正すること。
- 行または分岐カバレッジの全体閾値と汎用的な変異試験基盤。

## Design

主要ユースケースの候補は、304 件の現行規範シナリオを出発点にする。候補を、同じ主体、同じ正式入口の機能族、同じ最終効果で最大限まとめ、正常に成立したときの中心経路を一件のアンカー `REQ-*` とする。入力形式、代替経路、拒否、テナント分離、再試行は、別の利用者効果を生む場合を除いて主要ユースケースを増やさず、その中心経路の単体・E2E・故障検出に含める。異なる正式入口または異なる最終効果を持つ場合は、同じ Context でも別の主要ユースケースにする。

この規則から次の 51 件を導出した。表は変更固有の監査結果であり、製品の実行時 registry ではない。正本の現在状態は各参照先が所有し、完了後の新規変更は `wi-445` の通常契約で自身の主要ユースケースを宣言する。

| Context | Primary use case | Anchor | Observable result |
| --- | --- | --- | --- |
| Whole system | 主体の全到達経路を無効化する | `REQ-PLATFORM-001` | 無効化した主体の認証・トークン・連携経路が閉じる。 |
| Whole system | 削除予約と復元を全到達経路へ反映する | `REQ-PLATFORM-002` | 猶予期間中は到達不能になり、復元すると再び到達できる。 |
| Whole system | 正の変更を接続済み下流へ配信する | `REQ-PLATFORM-003` | 上流の変更が対象の下流状態へ収束する。 |
| Saml | SAML フェデレーションでサインインする | `REQ-SAML-006` | 登録済み SP が検証可能な SAML Response を受け取る。 |
| Saml | SAML の IdP プロファイルと SP を管理する | `REQ-SAML-004` | 管理者の設定が対象プロファイルのメタデータと信頼境界に反映される。 |
| IdManagement | User の削除予約と復元を管理する | `REQ-IDMANAGEMENT-011` | 対象 User のライフサイクルが予約・復元状態へ遷移する。 |
| IdManagement | 利用者が自身のプロフィールを更新する | `REQ-IDMANAGEMENT-016` | 保存後のプロフィール表示へ変更が反映される。 |
| IdManagement | User / Group の CSV を検証して適用する | `REQ-IDMANAGEMENT-004` | 有効な行だけが計画どおりに永続状態へ反映される。 |
| IdManagement | Group とメンバーシップを管理する | `REQ-IDMANAGEMENT-015` | 所属変更後の有効ロールに Group 由来の権限が反映される。 |
| IdManagement | Agent とクライアント資格情報を管理する | `REQ-IDMANAGEMENT-009` | 登録した Agent に資格情報の関連付けが保存される。 |
| Authentication | 外部 IdP の主体をローカル User に相関する | `REQ-AUTHENTICATION-001` | 検証済みの同一主体が同じローカル User のセッションになる。 |
| Authentication | パスワードでブラウザー認証する | `REQ-AUTHENTICATION-007` | 認証済みセッションで元の認可処理を継続できる。 |
| Authentication | メールリンクでパスワードを再設定する | `REQ-AUTHENTICATION-016` | 単回リンクの消費後に新しい資格情報で認証できる。 |
| Authentication | MFA 要素を登録して認証に使う | `REQ-AUTHENTICATION-011` | 登録済み要素による第二要素の成立がセッションへ反映される。 |
| Authentication | 自身のセッションを一覧して失効する | `REQ-AUTHENTICATION-013` | 選択したセッションが失効し、以後の利用を拒否される。 |
| Authentication | 信頼済み端末で次回の第二要素を省略する | `REQ-AUTHENTICATION-026` | 同意済み端末の次回ログインだけが第二要素を省略できる。 |
| Authentication | 新規端末と資格情報変更を通知する | `REQ-AUTHENTICATION-030` | 対象となるセキュリティ事象だけが本人へ通知される。 |
| Sourcing | SCIM で User と Group を同期する | `REQ-SOURCING-002` | SCIM の外部資源操作が正規の User / Group 状態へ反映される。 |
| SigningKeys | 署名鍵を無停止でローテーションする | `REQ-SIGNINGKEYS-001` | 新しい鍵で署名しつつ猶予中の旧 `kid` も検証に利用できる。 |
| Authorization | 関係をたどってアクセス可否を決定する | `REQ-AUTHORIZATION-003` | 継承・Group・親子関係を含む判定結果が返る。 |
| Authorization | 認可モデルと関係タプルを管理する | `REQ-AUTHORIZATION-001` | 整合する版だけが公開され、後続判定に利用される。 |
| Jobs | 永続 Job を worker で完了する | `REQ-JOBS-002` | 投入した Job の効果が一度だけ実行され成功状態になる。 |
| Jobs | 管理者が Job を参照・取消する | `REQ-JOBS-012` | 自テナントの Job 状態を確認し、未終端 Job を取消できる。 |
| Seeding | Seed を再実行して目的状態へ収束する | `REQ-SEEDING-010` | 部分失敗後を含め、再実行で宣言した状態に収束する。 |
| OAuth2 | 認可と同意を経て認可コードを交換する | `REQ-OAUTH2-005` | クライアントがアクセストークンと ID トークンを取得する。 |
| OAuth2 | リフレッシュ・失効・照会を含むトークン寿命を管理する | `REQ-OAUTH2-006` | ローテーション後の新トークンだけが有効になる。 |
| OAuth2 | 機械クライアントへトークンを発行する | `REQ-OAUTH2-026` | 許可されたクライアントが対象資源用トークンを取得する。 |
| OAuth2 | デバイス認可フローを完了する | `REQ-OAUTH2-027` | 利用者の承認後にデバイスがトークンを取得する。 |
| OAuth2 | 人間の承認を経てバックチャネル発行する | `REQ-OAUTH2-041` | 対象本人の承認後に一度だけトークンを取得する。 |
| OAuth2 | OAuth クライアントを管理する | `REQ-OAUTH2-035` | 自テナントのクライアント設定が作成・更新・削除される。 |
| DataKeys | DEK をローテーションして暗号文を継続利用する | `REQ-DATAKEYS-002` | 新旧バージョンの暗号文を復号し、再暗号化へ進める。 |
| WorkloadIdentity | ワークロード証明を Agent 資格情報へ交換する | `REQ-WORKLOADIDENTITY-001` | 登録済み信頼と関連付けから対象 Agent の資格情報を得る。 |
| WorkloadIdentity | ワークロード信頼と Agent 関連付けを管理する | `REQ-WORKLOADIDENTITY-008` | 信頼設定の状態変更が後続の交換可否に反映される。 |
| Provisioning | 下流接続を登録して能力を確認する | `REQ-PROVISIONING-002` | 接続テストの結果と対応機能が管理状態へ保存される。 |
| Provisioning | IdManagement の変更を下流へ配信する | `REQ-PROVISIONING-003` | 対象 User の作成・更新・無効化・削除が下流へ収束する。 |
| Tenancy | 要求元から正規 Tenant を解決する | `REQ-TENANCY-006` | 正規ロケーションの要求だけが対象 Tenant へ到達する。 |
| Tenancy | System 管理者が Tenant を管理する | `REQ-TENANCY-011` | Tenant の正規ロケーションと運用設定が更新される。 |
| Tenancy | Tenant のブランドと通知体験を構成する | `REQ-TENANCY-004` | 対象 Tenant の UI と通知へ設定が反映される。 |
| ClaimMapping | 許可した属性だけをプロトコルクレームへ写す | `REQ-CLAIMMAPPING-001` | 対応付けのある公開可能な属性だけが発行物に含まれる。 |
| Audit | 監査イベントを検索・ページング・エクスポートする | `REQ-AUDIT-001` | 絞り込みに一致する自テナントの履歴を参照・取得できる。 |
| ApiTokens | API アクセストークンを発行・利用・失効する | `REQ-APITOKENS-003` | 選択スコープのトークンが発行され、失効後は認証できない。 |
| System | プロセスと依存先の健全性を公開する | `REQ-SYSTEM-002` | probe がライフサイクルと依存障害を区別して返す。 |
| System | 選択ロケールで全 UI を表示する | `REQ-SYSTEM-010` | 管理・アカウント・認証画面が選択言語で描画される。 |
| WsFederation | WS-Fed パッシブサインインを完了する | `REQ-WSFEDERATION-002` | 登録済み RP が検証可能なサインイン応答を受け取る。 |
| WsFederation | WS-Trust でトークンを発行する | `REQ-WSFEDERATION-004` | 妥当な Issue 要求に RSTR を返す。 |
| SharedSignals | 失効シグナルを即時のトークン無効化へ反映する | `REQ-SHAREDSIGNALS-001` | 対象 Agent の既発行トークンが `active=false` になる。 |
| SharedSignals | セキュリティイベントを配送・受信する | `REQ-SHAREDSIGNALS-006` | 配送が再試行を経て成功または明示的な終端状態になる。 |
| Application | Application と一つのプロトコルを構成する | `REQ-APPLICATION-007` | 保存した Application が選択プロトコルの連携設定を持つ。 |
| Application | 割り当てとサインインポリシーで利用可否を決める | `REQ-APPLICATION-012` | 可視性とプロトコル利用可否が割り当て・ポリシーに従う。 |
| IdGovernance | ライフサイクルワークフローを管理する | `REQ-IDGOVERNANCE-001` | 有効な定義の版が保存され実行対象になる。 |
| IdGovernance | User 変更からワークフロー効果を実行する | `REQ-IDGOVERNANCE-003` | Group と Application の割り当てが宣言した目的状態へ収束する。 |

単体テストはドメインまたはユースケースの公開境界を使い、E2E テストは `server_http.Register` を通る文脈 HTTP 経路、実プロセスを起動するブラウザー経路、または CLI の正式入口を使う。Context のハンドラーを直接呼ぶだけのテスト、ページが開くだけのスモーク、要求 ID のコメントだけは E2E 証拠に数えない。

## Plan

1. 51 件の一覧を作業項目の構造化 `primary_use_cases` へ移し、各アンカー、観測結果、単体・E2E の責任と故障モデルを宣言する。
2. 既存テストを責任に照らして監査し、正式入口と最終効果を満たすものだけを関連付ける。
3. 不足する単体テストと E2E テストを RED から追加し、標準 `mise` タスクまたは CI から到達させる。
4. 各主要ユースケースの内部判断と実配線・最終効果へ別々の故障を与え、検出結果を記録する。
5. 全証拠の差集合が空であること、標準検証、仕様差分の有無を確認して完了する。

## Tasks

- [x] T001 [Inventory] 現行 304 規範シナリオ、TypeSpec 公開入口、Context 責務、垂直機能境界を監査し、51 件の主要ユースケースとアンカー要求、観測結果を導出した。
- [x] T002 [Correction] `ProductFeatureRegistry` を対象集合に使う誤った実装、形式変更、完了記録を取り消し、常時提供機能を registry の有無から判断しない設計へ修正した。
- [x] T003 [Evidence Plan] 51 件を構造化 `primary_use_cases` と `affected_spec` にそろえ、単体・E2E テストと異なる故障モデルを宣言した。
- [x] T004 [Unit] 既存単体テストを責任に対応付け、不足していた readiness の依存障害判定を追加した。
- [x] T005 [E2E] 既存の正式入口テストを責任に対応付け、薄かった経路には最終効果まで到達する E2E を追加または拡張した。
- [x] T006 [Reachability] 全 E2E 証拠を `test-go-race`、`test-ui-unit`、`test-ui-e2e` から到達可能にし、`verify` に `test-ui-e2e` を含めた。
- [x] T007 [Fault Evidence] 51 件それぞれの内部判断と実配線・最終効果へ別々の故障を注入し、単体 51 件・E2E 51 件の計 102 件すべてで対応テストの失敗を観測した。検出できなかった故障は、テスト側の証拠不足として深掘りしてから再測定した。
- [x] T008 [Verify] `mise run spec-diff`、`test-go-race`、`test-ui-unit`、`test-ui-e2e`、`verify` を実行し、完了記録を残した。

## Current Status and Handoff

本節は T006 完了時点の引き継ぎ記録である。T007 と T008 は本項目の中で完了しており、最終状態は `Completion` が持つ。

T006 時点の記録は次のとおりだった。T001 から T006 までを完了した。`ProductFeatureRegistry` は導出にも検査適用条件にも使用していない。現行の非廃止規範シナリオ 304 件、TypeSpec の公開入口、Context 責務、実装の垂直機能境界を、actor・正式入口の系統・最終観測効果でまとめ、51 件の `primary_use_cases` と対応する `affected_spec` を確定した。51 件すべてについて単体・E2E のファイルとテスト名が実在し、両テストソースに対応する `REQ-*` が存在することを確認済みである。

不足証拠として追加または深掘りした主な経路は、CSV 利用者 import の preview/apply と worker 実行、フェデレーション callback から session 解決、Provisioning connection の登録・接続試験、WorkloadTrustBundle の登録・無効化、Agent kill 後の既発行 token 失効、SSF stream の登録・署名 SET 配送、readiness の依存障害、DataKey 回転後の旧暗号文復号、Device Flow と CIBA の承認後 token 発行、Workload Identity の HTTP Token Exchange、`/userinfo` の claim mapping、`hidden` Application 割り当てでの `/authorize` 成功、利用者変更から LifecycleWorkflow の job 実行を経た Group/Application 割り当て、認証・account・admin の日本語表示である。Workload Identity の試験で、`oauth2.Module.WorkloadVerifier` が composition root で無視されていたため、明示注入時はそれを使い、未指定時だけ実 verifier を組み立てるよう修正した。

引き継ぎ時に優先して確認する実装差分は次のとおりである。

- 検査適用条件: `tools/check/src/primary-use-case-evidence.ts`、`tools/check/src/primary-use-case-evidence.test.ts`
- 標準タスク到達性: `mise.toml`、`frontend/package.json`
- 新規 E2E: `backend/authentication/federation/handlers_http/federated_login_e2e_test.go`、`backend/idmanagement/user/handlers_http/admin_user_import_e2e_test.go`、`backend/provisioning/handlers_http/admin_connection_handler_test.go`、`backend/sharedsignals/handlers_http/e2e_stream_delivery_test.go`、`frontend/tests/e2e/localized-ui.spec.ts`
- 新規または拡張した効果検証: `backend/shared/http/server_http/health_handler_test.go`、`backend/datakeys/field_cipher_test.go`、`backend/oauth2/handlers_http/approval_handler_test.go`、`backend/oauth2/handlers_http/authorize_handler_test.go`、`backend/oauth2/handlers_http/device_handler_test.go`、`backend/oauth2/handlers_http/token_exchange_handler_test.go`、`backend/oauth2/handlers_http/userinfo_handler_test.go`、`backend/idgovernance/usecases/lifecycle_workflow_dispatcher_test.go`、`backend/idmanagement/handlers_http/extra_identity_test.go`、`backend/workloadidentity/handlers_http/routes_test.go`
- Workload verifier の composition: `backend/shared/http/server_http/routes.go`

既存証拠にはファイル先頭の「主要ユースケース追跡」コメントを追加した。これは件数合わせではなく、frontmatter の各参照を完了時検査が解決できるようにするための追跡情報である。署名鍵は UI の操作可視性から実際の管理 API 回転へ、DataKeys は health 表示から旧暗号文の実復号へ、CIBA は判断 API だけから一回限りの token 発行へ、Workload Identity は汎用 token exchange から workload verifier を通る交換へ、ClaimMapping は直接 use case から `/userinfo` へ、Application access は policy 設定表示から `hidden` 割り当てを使う `/authorize` へ、それぞれ証拠参照を差し替えた。

実行済みの検証は次のとおりである。

- `mise run check-work-items` は通過した。
- `mise run test-tools` は 366 tests、0 failures で通過した。`primary_use_cases` を宣言した tooling migration にも検査を適用する新規テストは、修正前に finding が空になる RED を観測し、`applicable()` の修正後に通過した。
- `mise run spec-diff` は `no normative specification change against main` で通過した。
- `mise run test-ui-e2e` は権限昇格して実行し、既存 24 件と新規 localized UI 1 件の計 25 件が 0 failures で通過した。途中の `Expected a Response object` は既存 happy-dom の雑音であり、各 suite の集計はすべて 0 failures だった。
- `mise run test-go-package --` で `backend/idmanagement/user/handlers_http`、`backend/authentication/federation/handlers_http`、`backend/provisioning/handlers_http`、`backend/idmanagement/handlers_http`、`backend/sharedsignals/handlers_http`、`backend/idgovernance/usecases`、`backend/oauth2/handlers_http`、`backend/datakeys`、`backend/shared/http/server_http` の対象 package を通過させた。
- 51 件の参照について、ファイル、テスト名、要件 ID の存在を `yq` と `rg` で再監査し、欠落 0 件を確認した。
- `git diff --check` は通過した。

未完了なのは T007 と T008 である。T007 では51件それぞれの `unit_fault_model` と `e2e_fault_model` に対して実際の変異または明示的故障注入を行い、該当テストが失敗した観測を残す必要がある。既存 GREEN や静的な assertion 対応だけを RED／故障注入として記録してはならず、過去に観測していない RED を後付けで作らないこと。本項目にはまだ `Completion` を追加せず、`completed` への変更、`work-items/done/` への移動、コミットも行っていない。T007 完了後に51件と一対一の `Primary Use Case Evidence` を記録し、`mise run test-go-race`、`mise run test-ui-unit`、`mise run verify` を実行する。`verify` は本変更で `test-ui-e2e` を含むため、ローカル待受が制限される環境では権限昇格して実行する。

作業ツリーには本項目と無関係な未追跡 `work-items/wi-457-generalized-mutation-testing-framework.md` と `work-items/wi-458-specification-adequacy-counterexample-evidence.md` がある。これらを編集、削除、移動、ステージング、コミットしてはならない。本項目のコミットでは `wi-456` に対応するファイルだけを明示的に選ぶこと。

## Verification

- `mise run check-spec`
- `mise run check-work-items`
- `mise run test-tools`
- `mise run test-go-race`
- `mise run test-ui-e2e`
- `mise run verify`
- 主要ユースケースから単体または E2E の証拠を一件外すと検査が失敗する。
- 内部判断の故障は単体テストが、構成・経路制御・最終効果の故障は E2E テストが検出する。
- 完了済み `risk-based-v1` / `risk-based-v2` work item と `ProductFeatureRegistry` は変更されていない。

## Risk Notes

リスクは high。最大の失敗は、一覧の件数だけを満たすために機能を粗くまとめることと、要求 ID を追記した薄いテストを E2E と誤認することである。異なる正式入口または異なる最終効果は別の主要ユースケースにし、正式入口、構成、経路制御、アダプター、最終効果を一つでも迂回するテストは未移行として扱う。

監査で製品欠陥を見つけた場合は、対応する正本、再現 RED、互換性判断を持つ個別の `change_kind: bugfix` work item へ分ける。本項目ではテストを通すために実装挙動を暗黙に変更しない。

## Completion

- **Completed At**: 2026-09-02
- **Summary**:
  `mise run spec-diff` は `no normative specification change against main` を返す。規範仕様の意味は変えていない。変わったのは証拠の側である。現行の非廃止規範シナリオ、TypeSpec の公開入口、Context 責務、実装の垂直機能境界から導いた 51 件の主要ユースケースに、内部判断を見る単体テストと正式入口から最終効果まで通す E2E テストを一対一で結び付け、両者へ別々の故障を注入して、対応するテストが実際に失敗することを 102 件すべてで観測した。故障を検出できなかった経路は、宣言した観測結果に対してテストが薄すぎたということなので、最終効果まで届くようテストを深掘りしてから再測定した。深掘りの結果として要求 ID を名指しするテストが増えたため、`tools/check/security-refusal-debt.json` の未検査一覧から 25 件を削除した (128 件から 103 件へ)。
- **Change-Resistance Results**:
  51 件の主要ユースケースそれぞれについて、単体側は内部判断へ、E2E 側は構成・経路制御・アダプター・最終効果のいずれかへ、別々の故障を製品コードへ注入した。注入は 1 件ずつ適用して対象テストだけを実行し、実行後に必ず元へ戻している。102 件すべてで対応するテストが失敗し、検出できなかった故障は残っていない。
  この方法の限界は、変異が宣言済みの故障モデルに沿って人手で作られている点にある。差分全体を機械的に変異させたわけではないので、宣言した故障モデルの範囲を超えた等価変異の有無はこの記録では保証しない。
  最初の測定で検出できなかった故障は 9 件あり、いずれもテスト側の証拠不足だった。次のテストを最終効果まで届くよう深掘りし、深掘り後に同じ故障を再注入して失敗を確認した。
  - `TestSamlSSO_SPInitiatedAuthenticatedIssuesPostForm`: SAMLResponse を復号し、公開証明書に対して assertion 署名・Destination・Audience を検証するようにした。
  - `TestWsFedSignIn_AuthenticatedIssuesPassiveForm`: wresult を取り出し、IdP 証明書に対して assertion 署名を検証するようにした。
  - `TestAdminAgentLifecycle`: bind / unbind の状態コードだけでなく、管理 API 経由の `client_ids` へ関連付けが現れ消えることを確かめるようにした。
  - `TestCompleteLogin`: 認可要求に認証済み主体・auth_time・amr・acr が結び付いたことを確かめるようにした。
  - `TestBrowserAuthorizationFlowRequiresTOTPWhenPolicyRequiresMFA`: 第二要素成立後の session AMR と、同意を経た認可コード発行まで到達するようにした。
  - `TestExchangeCodeIssuesTokensByScope` / `TestRefreshTokensAcceptsMatchingDPoPProof`: 認可コードと refresh token の再利用が拒否されることを確かめるようにした。
  - `TestAdminOAuth2Client`: 別テナントからの更新が `ErrClientNotFound` で拒否されることを確かめるようにした。
  - `TestRotateTenantDataKeyThenDecryptStillWorksForOldVersion`: 保存側から読み直した retiring 版の DEK で既存暗号文が復号できることを確かめるようにした。
  - `TestWorkloadTrustBundleDisableEnableLifecycle`: 検証側と同じ repository を共有し、無効化・再有効化が交換可否へ反映されることを確かめるようにした。
  - `TestUpdateApplicationChangesAndNoop`: 戻り値ではなく保存済み Application を読み直し、変更と protocol が残ることを確かめるようにした。
  - `TestAuthorizeAllowsHiddenApplicationAssignment`: 割当ゲートを迂回する first-party client ではなく通常の client を使い、未割当の拒否と hidden 割当での発行を対で確かめるようにした。
  - `TestLifecycleWorkflowRunHandlerEmitsRunStartedAndRunSucceeded`: 成功した step の checkpoint が保存されることを確かめるようにした。
  - `TestAdminApiTokenLifecycle`: 失効後の一覧で対象 token に `revoked_at` が立つことを確かめるようにした。
  - `account session list can revoke a different browser session`: 画面から行が消えるだけでなく、読み直したサーバー側の一覧からも消えることを確かめるようにした。
  故障注入で 2 件が nil 参照の panic として現れたため、同じ責任をより明確な表明の失敗として観測できる変異へ取り替えた (WS-Federation の passive 署名、SSF の送信設定保存)。
- **Verification Results**:
  - `mise run check-work-items` - passed
  - `mise run check-ids` - passed
  - `mise run spec-diff` - no normative specification change against main
  - `mise run lint-go` - passed (0 issues)
  - `mise run check-security-controls` - passed (183 declared refusal(s), 103 awaiting a test)
  - `mise run test-go-race` - passed
  - `mise run test-ui-unit` - passed (672 tests)
  - `mise run test-ui-e2e` - passed
  - `mise run verify-serial` - passed
  - `mise run -j 2 verify` - passed (30 タスクすべてが実行され、失敗 0)
  - `mise run verify` (既定の `-j 4`) - この作業機では毎回別のタスクが 1 件だけ落ちる (1 回目は `test-ui-unit`、2 回目は `test-go-race`)。同じタスクを単独で実行するといずれも通り、並列度を 2 に下げた `verify` と `verify-serial` も通る。資源競合による揺らぎであり、変更に起因する失敗ではない。
- **Primary Use Case Evidence**:
  - id: platform-principal-lifecycle
    unit_red: "`TestSetUserDisabledAllowsDisablingOtherAdmin` が故障注入下で失敗するのを観測した: --- FAIL: TestSetUserDisabledAllowsDisablingOtherAdmin (0.00s)"
    e2e_red: "`TestDisabledUserLoginAndExistingSessionAreRejected` が故障注入下で失敗するのを観測した: --- FAIL: TestDisabledUserLoginAndExistingSessionAreRejected (0.07s)"
    unit_fault_injection: "backend/idmanagement/user/usecases/admin_users.go へ「無効化しても User を active のまま保存する」を注入すると `TestSetUserDisabledAllowsDisablingOtherAdmin` が失敗した。"
    e2e_fault_injection: "backend/shared/http/support_http/auth.go へ「ログインまたは既存セッションの失効配線を外す」を注入すると `TestDisabledUserLoginAndExistingSessionAreRejected` が失敗した。"
  - id: platform-deletion-lifecycle
    unit_red: "`TestSoftDeleteUserSetsPendingDeletionWithoutCascade` が故障注入下で失敗するのを観測した: --- FAIL: TestSoftDeleteUserSetsPendingDeletionWithoutCascade (0.00s)"
    e2e_red: "`TestAdminUserAPISoftDeletesAndRestores` が故障注入下で失敗するのを観測した: --- FAIL: TestAdminUserAPISoftDeletesAndRestores (0.03s)"
    unit_fault_injection: "backend/idmanagement/user/usecases/admin_users.go へ「削除予約で pending_deletion へ遷移しない」を注入すると `TestSoftDeleteUserSetsPendingDeletionWithoutCascade` が失敗した。"
    e2e_fault_injection: "backend/idmanagement/user/usecases/admin_users.go へ「復元 API が永続状態を active へ戻さない」を注入すると `TestAdminUserAPISoftDeletesAndRestores` が失敗した。"
  - id: platform-downstream-propagation
    unit_red: "`TestCaptureLifecycleEvent_UserAttributesChanged_TranslatesToUpdate` が故障注入下で失敗するのを観測した: --- FAIL: TestCaptureLifecycleEvent_UserAttributesChanged_TranslatesToUpdate (0.00s)"
    e2e_red: "`TestE2E_CreateUpdateDisableDelete_ReachesRealDownstream` が故障注入下で失敗するのを観測した: --- FAIL: TestE2E_CreateUpdateDisableDelete_ReachesRealDownstream (0.04s)"
    unit_fault_injection: "backend/provisioning/usecases/capture.go へ「属性変更を update 配信へ投影しない」を注入すると `TestCaptureLifecycleEvent_UserAttributesChanged_TranslatesToUpdate` が失敗した。"
    e2e_fault_injection: "backend/provisioning/usecases/deliver.go へ「捕捉した配信を実下流クライアントへ渡さない」を注入すると `TestE2E_CreateUpdateDisableDelete_ReachesRealDownstream` が失敗した。"
  - id: saml-federation
    unit_red: "`TestSamlServiceProviderMatchesOnlyAssignedProfile` が故障注入下で失敗するのを観測した: --- FAIL: TestSamlServiceProviderMatchesOnlyAssignedProfile (0.00s)"
    e2e_red: "`TestSamlSSO_SPInitiatedAuthenticatedIssuesPostForm` が故障注入下で失敗するのを観測した: --- FAIL: TestSamlSSO_SPInitiatedAuthenticatedIssuesPostForm (0.07s)"
    unit_fault_injection: "backend/saml/domain/service_provider.go へ「SP と割り当て済み IdP プロファイルの一致判定を反転する」を注入すると `TestSamlServiceProviderMatchesOnlyAssignedProfile` が失敗した。"
    e2e_fault_injection: "backend/saml/handlers_http/sso_handler.go へ「SSO 入口で assertion へ署名しない」を注入すると `TestSamlSSO_SPInitiatedAuthenticatedIssuesPostForm` が失敗した。"
  - id: saml-configuration
    unit_red: "`TestSamlIdentityProviderProfileValidation` が故障注入下で失敗するのを観測した: --- FAIL: TestSamlIdentityProviderProfileValidation (0.00s) / --- FAIL: TestSamlIdentityProviderProfileValidation/dedicated_profile_rejects_a_second_binding (0.00s)"
    e2e_red: "`TestAdminIDPProfileCRUDAndCanonicalEndpoints` が故障注入下で失敗するのを観測した: --- FAIL: TestAdminIDPProfileCRUDAndCanonicalEndpoints (0.03s)"
    unit_fault_injection: "backend/saml/domain/idp_profile.go へ「共有・専用プロファイルの不変条件を受理する」を注入すると `TestSamlIdentityProviderProfileValidation` が失敗した。"
    e2e_fault_injection: "backend/saml/handlers_http/admin_idp_profile_handler.go へ「管理 API の保存結果をメタデータ取得へ結ばない」を注入すると `TestAdminIDPProfileCRUDAndCanonicalEndpoints` が失敗した。"
  - id: identity-user-lifecycle
    unit_red: "`TestRestoreUserReturnsToActive` が故障注入下で失敗するのを観測した: --- FAIL: TestRestoreUserReturnsToActive (0.00s)"
    e2e_red: "`TestAdminUserAPISoftDeletesAndRestores` が故障注入下で失敗するのを観測した: --- FAIL: TestAdminUserAPISoftDeletesAndRestores (0.04s)"
    unit_fault_injection: "backend/idmanagement/user/usecases/admin_users.go へ「復元時に lifecycle を active へ戻さない」を注入すると `TestRestoreUserReturnsToActive` が失敗した。"
    e2e_fault_injection: "backend/idmanagement/user/handlers_http/admin_user_handler.go へ「管理 API の復元操作を use case へ接続しない」を注入すると `TestAdminUserAPISoftDeletesAndRestores` が失敗した。"
  - id: identity-self-service
    unit_red: "`TestUpdateUserProfileEditsNameAndEditableAttribute` が故障注入下で失敗するのを観測した: --- FAIL: TestUpdateUserProfileEditsNameAndEditableAttribute (0.02s)"
    e2e_red: "`TestAccountProfilePatchUpdatesEditableAttribute` が故障注入下で失敗するのを観測した: --- FAIL: TestAccountProfilePatchUpdatesEditableAttribute (0.00s) / account_handler_test.go:154: status=404 body={'type':'urn:idmagic:error:user_not_found','title':'User not found','status':404,'detai…"
    unit_fault_injection: "backend/idmanagement/user/usecases/account_profile.go へ「編集可能属性を保存対象から外す」を注入すると `TestUpdateUserProfileEditsNameAndEditableAttribute` が失敗した。"
    e2e_fault_injection: "backend/idmanagement/user/handlers_http/account_handler.go へ「account PATCH の actor を更新 use case へ渡さない」を注入すると `TestAccountProfilePatchUpdatesEditableAttribute` が失敗した。"
  - id: identity-bulk-transfer
    unit_red: "`TestPlanUserImportCreateUpdateUnchangedAndFieldPresence` が故障注入下で失敗するのを観測した: --- FAIL: TestPlanUserImportCreateUpdateUnchangedAndFieldPresence (0.00s)"
    e2e_red: "`TestAdminUserImportPrimaryUseCase_REQ_IDMANAGEMENT_004` が故障注入下で失敗するのを観測した: --- FAIL: TestAdminUserImportPrimaryUseCase_REQ_IDMANAGEMENT_004 (0.00s)"
    unit_fault_injection: "backend/idmanagement/user/usecases/user_import_planner.go へ「有効な create 行を unchanged と計画する」を注入すると `TestPlanUserImportCreateUpdateUnchangedAndFieldPresence` が失敗した。"
    e2e_fault_injection: "backend/idmanagement/user/usecases/user_import.go へ「preview 成果物を apply Job と実保存へ接続しない」を注入すると `TestAdminUserImportPrimaryUseCase_REQ_IDMANAGEMENT_004` が失敗した。"
  - id: identity-group-management
    unit_red: "`TestGroupCreateAddMemberEffectiveRoles` が故障注入下で失敗するのを観測した: --- FAIL: TestGroupCreateAddMemberEffectiveRoles (0.00s)"
    e2e_red: "`TestAdminGroupAPICreateAddMemberAndEffectiveRoles` が故障注入下で失敗するのを観測した: --- FAIL: TestAdminGroupAPICreateAddMemberAndEffectiveRoles (0.00s)"
    unit_fault_injection: "backend/idmanagement/group/usecases/admin_groups.go へ「Group ロールを有効ロールへ合成しない」を注入すると `TestGroupCreateAddMemberEffectiveRoles` が失敗した。"
    e2e_fault_injection: "backend/idmanagement/group/handlers_http/admin_group_handler.go へ「メンバー追加 API を所属 Repository へ接続しない」を注入すると `TestAdminGroupAPICreateAddMemberAndEffectiveRoles` が失敗した。"
  - id: identity-agent-management
    unit_red: "`TestBindUnbindCredentialAndFindByClientID` が故障注入下で失敗するのを観測した: --- FAIL: TestBindUnbindCredentialAndFindByClientID (0.00s)"
    e2e_red: "`TestAdminAgentLifecycle` が故障注入下で失敗するのを観測した: --- FAIL: TestAdminAgentLifecycle (0.00s)"
    unit_fault_injection: "backend/idmanagement/agent/usecases/admin_agents.go へ「AgentCredentialBinding を保存しない」を注入すると `TestBindUnbindCredentialAndFindByClientID` が失敗した。"
    e2e_fault_injection: "backend/idmanagement/agent/handlers_http/admin_agent_handler.go へ「Agent 管理 API から資格情報関連付けを外す」を注入すると `TestAdminAgentLifecycle` が失敗した。"
  - id: authentication-federation
    unit_red: "`TestCompleteUsesExistingFederatedIdentityAndIssuesSession` が故障注入下で失敗するのを観測した: --- FAIL: TestCompleteUsesExistingFederatedIdentityAndIssuesSession (0.00s)"
    e2e_red: "`TestFederatedLoginPrimaryUseCase_REQ_AUTHENTICATION_001` が故障注入下で失敗するのを観測した: --- FAIL: TestFederatedLoginPrimaryUseCase_REQ_AUTHENTICATION_001 (0.00s) / federated_login_e2e_test.go:90: callback status=401 location='' body={'type':'urn:idmagic:error:federation_failed','title':'…"
    unit_fault_injection: "backend/authentication/federation/usecases/broker.go へ「既存の外部主体関連付けを無視して別 User を作る」を注入すると `TestCompleteUsesExistingFederatedIdentityAndIssuesSession` が失敗した。"
    e2e_fault_injection: "backend/authentication/federation/usecases/flow.go へ「callback の検証済み subject を broker 完了処理へ渡さない」を注入すると `TestFederatedLoginPrimaryUseCase_REQ_AUTHENTICATION_001` が失敗した。"
  - id: authentication-password-login
    unit_red: "`TestCompleteLogin` が故障注入下で失敗するのを観測した: --- FAIL: TestCompleteLogin (0.00s) / --- FAIL: TestCompleteLogin/Succeeds (0.00s)"
    e2e_red: "`TestBrowserAuthorizationFlowUsesCookiesAndJSONAPI` が故障注入下で失敗するのを観測した: --- FAIL: TestBrowserAuthorizationFlowUsesCookiesAndJSONAPI (0.06s) / routes_e2e_test.go:365: GET http://127.0.0.1:52420/realms/default/api/auth/transaction status=401 body={'type':'urn:idmagic:error:…"
    unit_fault_injection: "backend/oauth2/authorization/usecases/complete_login.go へ「認証済み User を認可要求へ結び付けない」を注入すると `TestCompleteLogin` が失敗した。"
    e2e_fault_injection: "backend/oauth2/handlers_http/authorize_login.go へ「browser login 成功後の session cookie または authorize 再開配線を外す」を注入すると `TestBrowserAuthorizationFlowUsesCookiesAndJSONAPI` が失敗した。"
  - id: authentication-password-recovery
    unit_red: "`TestResetPasswordWithTokenConsumesTokenAndUpdatesPassword` が故障注入下で失敗するのを観測した: --- FAIL: TestResetPasswordWithTokenConsumesTokenAndUpdatesPassword (0.10s)"
    e2e_red: "`TestPasswordResetHTTPFlow` が故障注入下で失敗するのを観測した: --- FAIL: TestPasswordResetHTTPFlow (0.04s) / password_reset_handler_test.go:48: reset status=410 body={'type':'urn:idmagic:error:invalid_reset_token','title':'Invalid reset token','status':410,'detai…"
    unit_fault_injection: "backend/authentication/password/usecases/reset_password_with_token.go へ「トークン消費後もパスワードハッシュを更新しない」を注入すると `TestResetPasswordWithTokenConsumesTokenAndUpdatesPassword` が失敗した。"
    e2e_fault_injection: "backend/authentication/password/handlers_http/password_reset_handler.go へ「reset HTTP 入口を token store と password use case へ接続しない」を注入すると `TestPasswordResetHTTPFlow` が失敗した。"
  - id: authentication-mfa
    unit_red: "`TestVerifyTOTPFactorUpdatesLastUsedAt` が故障注入下で失敗するのを観測した: --- FAIL: TestVerifyTOTPFactorUpdatesLastUsedAt (0.00s)"
    e2e_red: "`TestBrowserAuthorizationFlowRequiresTOTPWhenPolicyRequiresMFA` が故障注入下で失敗するのを観測した: --- FAIL: TestBrowserAuthorizationFlowRequiresTOTPWhenPolicyRequiresMFA (0.07s)"
    unit_fault_injection: "backend/authentication/totp/usecases/verify_totp_factor.go へ「正しい TOTP を失敗として扱う」を注入すると `TestVerifyTOTPFactorUpdatesLastUsedAt` が失敗した。"
    e2e_fault_injection: "backend/oauth2/handlers_http/authorize_second_factor.go へ「第二要素の成立を login session の AMR へ反映しない」を注入すると `TestBrowserAuthorizationFlowRequiresTOTPWhenPolicyRequiresMFA` が失敗した。"
  - id: authentication-sessions
    unit_red: "`TestRevokeOtherSessionsKeepsCurrent` が故障注入下で失敗するのを観測した: --- FAIL: TestRevokeOtherSessionsKeepsCurrent (0.00s)"
    e2e_red: "`account session list can revoke a different browser session` が故障注入下で失敗するのを観測した: error: expect(received).toBeLessThan(expected) / (fail) account session list can revoke a different browser session [2235.51ms]"
    unit_fault_injection: "backend/authentication/session/usecases/sessions.go へ「選択セッションの revoke を保存しない」を注入すると `TestRevokeOtherSessionsKeepsCurrent` が失敗した。"
    e2e_fault_injection: "frontend/src/api/account.ts へ「account UI の revoke 操作を session API へ送らない」を注入すると `account session list can revoke a different browser session` が失敗した。"
  - id: authentication-trusted-device
    unit_red: "`TestEvaluateTrustsTheIssuedCookieAndRotatesIt` が故障注入下で失敗するのを観測した: --- FAIL: TestEvaluateTrustsTheIssuedCookieAndRotatesIt (0.00s)"
    e2e_red: "`TestTrustedDeviceSkipsTheSecondFactorOnTheNextLogin` が故障注入下で失敗するのを観測した: --- FAIL: TestTrustedDeviceSkipsTheSecondFactorOnTheNextLogin (0.06s)"
    unit_fault_injection: "backend/authentication/trusteddevice/usecases/trusted_devices.go へ「有効な端末 cookie を信頼済みと判定しない」を注入すると `TestEvaluateTrustsTheIssuedCookieAndRotatesIt` が失敗した。"
    e2e_fault_injection: "backend/oauth2/handlers_http/authorize_second_factor.go へ「第二要素成功時の cookie 発行または次回 login の読取りを外す」を注入すると `TestTrustedDeviceSkipsTheSecondFactorOnTheNextLogin` が失敗した。"
  - id: authentication-notifications
    unit_red: "`TestDispatchNotifiesOnlyTheFirstSignInFromEachDevice` が故障注入下で失敗するのを観測した: --- FAIL: TestDispatchNotifiesOnlyTheFirstSignInFromEachDevice (0.00s)"
    e2e_red: "`TestEmitFuncSendsSecurityNotificationsForCatalogEvents` が故障注入下で失敗するのを観測した: --- FAIL: TestEmitFuncSendsSecurityNotificationsForCatalogEvents (0.00s)"
    unit_fault_injection: "backend/authentication/securitynotification/usecases/dispatch.go へ「既知端末にも重複通知する」を注入すると `TestDispatchNotifiesOnlyTheFirstSignInFromEachDevice` が失敗した。"
    e2e_fault_injection: "backend/cmd/internal/bootstrap/securitynotification.go へ「組み立て地点の event emit から通知 dispatcher を外す」を注入すると `TestEmitFuncSendsSecurityNotificationsForCatalogEvents` が失敗した。"
  - id: scim-sourcing
    unit_red: "`TestUpdateUserFullReplace` が故障注入下で失敗するのを観測した: --- FAIL: TestUpdateUserFullReplace (0.00s)"
    e2e_red: "`TestScimInboundProvisioning` が故障注入下で失敗するのを観測した: --- FAIL: TestScimInboundProvisioning (0.00s)"
    unit_fault_injection: "backend/sourcing/scim/usecases/users.go へ「SCIM 更新を User の正規状態へ投影しない」を注入すると `TestUpdateUserFullReplace` が失敗した。"
    e2e_fault_injection: "backend/sourcing/scim/handlers_http/handlers.go へ「SCIM HTTP 入口を source adapter と IdManagement へ接続しない」を注入すると `TestScimInboundProvisioning` が失敗した。"
  - id: signing-key-lifecycle
    unit_red: "`TestRotateSigningKeyKeepsPreviousKidInJWKS` が故障注入下で失敗するのを観測した: --- FAIL: TestRotateSigningKeyKeepsPreviousKidInJWKS (0.08s)"
    e2e_red: "`TestAdminKeysRotateSucceedsAndEmitsEvent` が故障注入下で失敗するのを観測した: --- FAIL: TestAdminKeysRotateSucceedsAndEmitsEvent (0.02s)"
    unit_fault_injection: "backend/signingkeys/usecases/rotate_signing_key.go へ「回転と同時に旧鍵を無効化し JWKS から外す」を注入すると `TestRotateSigningKeyKeepsPreviousKidInJWKS` が失敗した。"
    e2e_fault_injection: "backend/signingkeys/handlers_http/admin_key_handler.go へ「管理 UI の回転操作を tenant key store へ接続しない」を注入すると `TestAdminKeysRotateSucceedsAndEmitsEvent` が失敗した。"
  - id: relationship-authorization
    unit_red: "`TestCheckTraversesGroupAndParent` が故障注入下で失敗するのを観測した: --- FAIL: TestCheckTraversesGroupAndParent (0.00s) / --- FAIL: TestCheckTraversesGroupAndParent/nested_group_through_parent_folder (0.00s)"
    e2e_red: "`TestAuthorizationAdminRoutes` が故障注入下で失敗するのを観測した: --- FAIL: TestAuthorizationAdminRoutes (0.01s) / --- FAIL: TestAuthorizationAdminRoutes/publishing_a_model,_writing_tuples,_and_checking_access (0.00s)"
    unit_fault_injection: "backend/authorization/domain/evaluator.go へ「Group または親子辺の探索を打ち切る」を注入すると `TestCheckTraversesGroupAndParent` が失敗した。"
    e2e_fault_injection: "backend/authorization/handlers_http/routes.go へ「check HTTP 入口を関係 store と evaluator へ接続しない」を注入すると `TestAuthorizationAdminRoutes` が失敗した。"
  - id: authorization-model-admin
    unit_red: "`TestPutAuthorizationModelRejectsAnInconsistentModel` が故障注入下で失敗するのを観測した: --- FAIL: TestPutAuthorizationModelRejectsAnInconsistentModel (0.00s)"
    e2e_red: "`TestAuthorizationAdminRoutes` が故障注入下で失敗するのを観測した: --- FAIL: TestAuthorizationAdminRoutes (0.01s) / --- FAIL: TestAuthorizationAdminRoutes/publishing_a_model,_writing_tuples,_and_checking_access (0.00s)"
    unit_fault_injection: "backend/authorization/usecases/admin_model.go へ「整合しないモデルを published として保存する」を注入すると `TestPutAuthorizationModelRejectsAnInconsistentModel` が失敗した。"
    e2e_fault_injection: "backend/authorization/handlers_http/routes.go へ「管理 API の model 更新を store へ接続しない」を注入すると `TestAuthorizationAdminRoutes` が失敗した。"
  - id: durable-job-execution
    unit_red: "`TestTransitionJobLifecycle_DeclaredTransitions` が故障注入下で失敗するのを観測した: --- FAIL: TestTransitionJobLifecycle_DeclaredTransitions (0.00s) / job_test.go:34: TransitionJobLifecycle('running', 'Complete') unexpected error: jobs: no transition from 'running' on event 'Complete…"
    e2e_red: "`TestRunner_SuccessPath` が故障注入下で失敗するのを観測した: --- FAIL: TestRunner_SuccessPath (2.00s)"
    unit_fault_injection: "backend/jobs/domain/job.go へ「running から succeeded への遷移を拒否する」を注入すると `TestTransitionJobLifecycle_DeclaredTransitions` が失敗した。"
    e2e_fault_injection: "backend/jobs/usecases/runner.go へ「Runner が登録済み handler の実行結果を Complete へ渡さない」を注入すると `TestRunner_SuccessPath` が失敗した。"
  - id: job-administration
    unit_red: "`TestCancelJobForAdmin` が故障注入下で失敗するのを観測した: --- FAIL: TestCancelJobForAdmin (0.00s)"
    e2e_red: "`TestCancelJob` が故障注入下で失敗するのを観測した: --- FAIL: TestCancelJob (0.00s)"
    unit_fault_injection: "backend/jobs/usecases/admin.go へ「取消可能な Job を canceled へ遷移しない」を注入すると `TestCancelJobForAdmin` が失敗した。"
    e2e_fault_injection: "backend/jobs/handlers_http/admin_job_handler.go へ「cancel API を管理 use case へ接続しない」を注入すると `TestCancelJob` が失敗した。"
  - id: convergent-seeding
    unit_red: "`TestRunCanBeRetriedAfterApplyFailure` が故障注入下で失敗するのを観測した: --- FAIL: TestRunCanBeRetriedAfterApplyFailure (0.00s)"
    e2e_red: "`TestSeedDryRunDoesNotMutateAndRepeatedApplyConverges` が故障注入下で失敗するのを観測した: --- FAIL: TestSeedDryRunDoesNotMutateAndRepeatedApplyConverges (0.00s)"
    unit_fault_injection: "backend/seeding/usecases/plan.go へ「既存一致資源を毎回更新対象にする」を注入すると `TestRunCanBeRetriedAfterApplyFailure` が失敗した。"
    e2e_fault_injection: "backend/cmd/internal/bootstrap/seeding.go へ「bootstrap の contributor 集約から一つの適用処理を外す」を注入すると `TestSeedDryRunDoesNotMutateAndRepeatedApplyConverges` が失敗した。"
  - id: oauth-authorization-code
    unit_red: "`TestExchangeCodeIssuesTokensByScope` が故障注入下で失敗するのを観測した: --- FAIL: TestExchangeCodeIssuesTokensByScope (0.00s)"
    e2e_red: "`TestBrowserAuthorizationFlowUsesCookiesAndJSONAPI` が故障注入下で失敗するのを観測した: --- FAIL: TestBrowserAuthorizationFlowUsesCookiesAndJSONAPI (0.06s)"
    unit_fault_injection: "backend/oauth2/token/usecases/exchange_code.go へ「交換済み認可コードを再利用可能にする」を注入すると `TestExchangeCodeIssuesTokensByScope` が失敗した。"
    e2e_fault_injection: "backend/oauth2/handlers_http/token_handler.go へ「token endpoint から認可コード交換を外す」を注入すると `TestBrowserAuthorizationFlowUsesCookiesAndJSONAPI` が失敗した。"
  - id: oauth-token-lifecycle
    unit_red: "`TestRefreshTokensAcceptsMatchingDPoPProof` が故障注入下で失敗するのを観測した: --- FAIL: TestRefreshTokensAcceptsMatchingDPoPProof (0.00s)"
    e2e_red: "`TestTokenLifecycleRotatesRefreshTokenAndInvalidatesTheUsedOne` が故障注入下で失敗するのを観測した: --- FAIL: TestTokenLifecycleRotatesRefreshTokenAndInvalidatesTheUsedOne (0.09s) / routes_e2e_test.go:463: refresh status=400 body=map[error:invalid_grant error_description:The refresh token is invalid…"
    unit_fault_injection: "backend/oauth2/token/usecases/refresh_tokens.go へ「refresh token のローテーション時に旧 token を失効しない」を注入すると `TestRefreshTokensAcceptsMatchingDPoPProof` が失敗した。"
    e2e_fault_injection: "backend/oauth2/handlers_http/token_handler.go へ「token endpoint から refresh grant の配線を外す」を注入すると `TestTokenLifecycleRotatesRefreshTokenAndInvalidatesTheUsedOne` が失敗した。"
  - id: oauth-machine-token
    unit_red: "`TestScopeIntersection` が故障注入下で失敗するのを観測した: --- FAIL: TestScopeIntersection (0.00s)"
    e2e_red: "`TestTokenMetricsRecordSuccessfulClientCredentialsGrant` が故障注入下で失敗するのを観測した: --- FAIL: TestTokenMetricsRecordSuccessfulClientCredentialsGrant (0.04s) / login_metrics_e2e_test.go:255: status = 400, body = map[error:unauthorized_client error_description:public client not allowed…"
    unit_fault_injection: "backend/oauth2/authorization/domain/authorization_request.go へ「要求 scope を許可済み scope 集合で絞り込まない」を注入すると `TestScopeIntersection` が失敗した。"
    e2e_fault_injection: "backend/oauth2/handlers_http/token_handler.go へ「組み立て済み server から client credentials grant を外す」を注入すると `TestTokenMetricsRecordSuccessfulClientCredentialsGrant` が失敗した。"
  - id: oauth-device-flow
    unit_red: "`TestDeviceFlowPollingAndReplay` が故障注入下で失敗するのを観測した: --- FAIL: TestDeviceFlowPollingAndReplay (0.00s)"
    e2e_red: "`TestDeviceAuthorizationAPI` が故障注入下で失敗するのを観測した: --- FAIL: TestDeviceAuthorizationAPI (0.00s) / --- FAIL: TestDeviceAuthorizationAPI/DeviceAPI_Approve (0.00s)"
    unit_fault_injection: "backend/oauth2/device/usecases/device_flow.go へ「承認済み device code を再利用可能にする」を注入すると `TestDeviceFlowPollingAndReplay` が失敗した。"
    e2e_fault_injection: "backend/oauth2/handlers_http/token_handler.go へ「device authorization endpoint と token endpoint の共有 store を分離する」を注入すると `TestDeviceAuthorizationAPI` が失敗した。"
  - id: oauth-human-approval
    unit_red: "`TestApprovalFlowPollingDecisionAndReplay` が故障注入下で失敗するのを観測した: --- FAIL: TestApprovalFlowPollingDecisionAndReplay (0.00s)"
    e2e_red: "`TestBackchannelApprovalIssuesTokenOnce` が故障注入下で失敗するのを観測した: --- FAIL: TestBackchannelApprovalIssuesTokenOnce (0.00s) / approval_handler_test.go:144: approval status=403 body={'type':'urn:idmagic:error:access_denied','title':'Access denied','status':403,'detail…"
    unit_fault_injection: "backend/oauth2/approval/usecases/approval_flow.go へ「承認要求の一回限りの交換制約を外す」を注入すると `TestApprovalFlowPollingDecisionAndReplay` が失敗した。"
    e2e_fault_injection: "backend/oauth2/handlers_http/approval_handler.go へ「本人の approval session とバックチャネル要求の関連付けを外す」を注入すると `TestBackchannelApprovalIssuesTokenOnce` が失敗した。"
  - id: oauth-client-administration
    unit_red: "`TestAdminOAuth2Client` が故障注入下で失敗するのを観測した: --- FAIL: TestAdminOAuth2Client (0.00s) / --- FAIL: TestAdminOAuth2Client/UpdateAdminOAuth2Client (0.00s)"
    e2e_red: "`TestAdminOAuth2ClientCRUD` が故障注入下で失敗するのを観測した: --- FAIL: TestAdminOAuth2ClientCRUD (0.00s) / admin_client_handler_test.go:86: update status=404 body={'type':'urn:idmagic:error:client_not_found','title':'Client not found','status':404,'detail':'The…"
    unit_fault_injection: "backend/oauth2/client/usecases/admin_clients.go へ「更新時に tenant 所有権を検証しない」を注入すると `TestAdminOAuth2Client` が失敗した。"
    e2e_fault_injection: "backend/oauth2/handlers_http/admin_client_handler.go へ「管理 API の更新対象 client を use case へ渡さない」を注入すると `TestAdminOAuth2ClientCRUD` が失敗した。"
  - id: data-key-lifecycle
    unit_red: "`TestRotateTenantDataKeyThenDecryptStillWorksForOldVersion` が故障注入下で失敗するのを観測した: --- FAIL: TestRotateTenantDataKeyThenDecryptStillWorksForOldVersion (0.00s)"
    e2e_red: "`TestFieldCipherDecryptsExistingCiphertextAfterRotation` が故障注入下で失敗するのを観測した: --- FAIL: TestFieldCipherDecryptsExistingCiphertextAfterRotation (0.00s)"
    unit_fault_injection: "backend/datakeys/usecases/lifecycle.go へ「更新時に旧 DEK を retiring として保持しない」を注入すると `TestRotateTenantDataKeyThenDecryptStillWorksForOldVersion` が失敗した。"
    e2e_fault_injection: "backend/datakeys/field_cipher.go へ「管理 API の DEK 操作を tenant key provider へ接続しない」を注入すると `TestFieldCipherDecryptsExistingCiphertextAfterRotation` が失敗した。"
  - id: workload-credential-exchange
    unit_red: "`TestVerifyWorkloadAttestation_Success` が故障注入下で失敗するのを観測した: --- FAIL: TestVerifyWorkloadAttestation_Success (0.13s)"
    e2e_red: "`TestTokenExchangeIssuesWorkloadCredential` が故障注入下で失敗するのを観測した: --- FAIL: TestTokenExchangeIssuesWorkloadCredential (0.01s) / token_exchange_handler_test.go:117: workload exchange status=400 called=false body=map[error:invalid_request error_description:unsupported…"
    unit_fault_injection: "backend/workloadidentity/usecases/verify_workload_attestation.go へ「証明の issuer と subject を trust bundle に照合しない」を注入すると `TestVerifyWorkloadAttestation_Success` が失敗した。"
    e2e_fault_injection: "backend/shared/http/server_http/routes.go へ「token exchange endpoint から workload attestation verifier を外す」を注入すると `TestTokenExchangeIssuesWorkloadCredential` が失敗した。"
  - id: workload-trust-administration
    unit_red: "`TestWorkloadTrustBundleDisableEnableLifecycle` が故障注入下で失敗するのを観測した: --- FAIL: TestWorkloadTrustBundleDisableEnableLifecycle (0.03s)"
    e2e_red: "`TestAdminWorkloadTrustBundleLifecycle` が故障注入下で失敗するのを観測した: --- FAIL: TestAdminWorkloadTrustBundleLifecycle (0.00s)"
    unit_fault_injection: "backend/workloadidentity/usecases/verify_workload_attestation.go へ「disabled trust bundle を検証時に有効として扱う」を注入すると `TestWorkloadTrustBundleDisableEnableLifecycle` が失敗した。"
    e2e_fault_injection: "backend/workloadidentity/handlers_http/routes.go へ「管理 API の状態変更を verifier が読む repository へ保存しない」を注入すると `TestAdminWorkloadTrustBundleLifecycle` が失敗した。"
  - id: provisioning-connection
    unit_red: "`TestRegisterConnection_SeedsDefaultAttributeMappingAndRejectsDuplicate` が故障注入下で失敗するのを観測した: --- FAIL: TestRegisterConnection_SeedsDefaultAttributeMappingAndRejectsDuplicate (0.00s)"
    e2e_red: "`TestAdminProvisioningConnectionLifecycle` が故障注入下で失敗するのを観測した: --- FAIL: TestAdminProvisioningConnectionLifecycle (0.00s)"
    unit_fault_injection: "backend/provisioning/usecases/admin.go へ「接続登録時に既定属性対応付けを保存しない」を注入すると `TestRegisterConnection_SeedsDefaultAttributeMappingAndRejectsDuplicate` が失敗した。"
    e2e_fault_injection: "backend/provisioning/handlers_http/handlers.go へ「接続テスト結果を管理 API の応答と repository へ反映しない」を注入すると `TestAdminProvisioningConnectionLifecycle` が失敗した。"
  - id: provisioning-delivery
    unit_red: "`TestCaptureLifecycleEvent_UserCreated_AllUsersScope` が故障注入下で失敗するのを観測した: --- FAIL: TestCaptureLifecycleEvent_UserCreated_AllUsersScope (0.00s)"
    e2e_red: "`TestE2E_CreateUpdateDisableDelete_ReachesRealDownstream` が故障注入下で失敗するのを観測した: --- FAIL: TestE2E_CreateUpdateDisableDelete_ReachesRealDownstream (0.03s)"
    unit_fault_injection: "backend/provisioning/usecases/capture.go へ「UserCreated を下流 create 配送へ射影しない」を注入すると `TestCaptureLifecycleEvent_UserCreated_AllUsersScope` が失敗した。"
    e2e_fault_injection: "backend/provisioning/usecases/notify_adapters.go へ「IdManagement event subscriber と delivery worker の接続を外す」を注入すると `TestE2E_CreateUpdateDisableDelete_ReachesRealDownstream` が失敗した。"
  - id: tenant-resolution
    unit_red: "`TestSetEndpointStyle` が故障注入下で失敗するのを観測した: --- FAIL: TestSetEndpointStyle (0.00s)"
    e2e_red: "`TestTenantIsReachableOnlyAtItsCanonicalLocation` が故障注入下で失敗するのを観測した: --- FAIL: TestTenantIsReachableOnlyAtItsCanonicalLocation (0.00s) / --- FAIL: TestTenantIsReachableOnlyAtItsCanonicalLocation/subdomain_tenant_is_absent_on_the_path_prefix_route (0.00s)"
    unit_fault_injection: "backend/tenancy/usecases/manage_tenants.go へ「endpoint style 変更時に正規 host/path 制約を保存しない」を注入すると `TestSetEndpointStyle` が失敗した。"
    e2e_fault_injection: "backend/shared/http/support_http/tenant_middleware.go へ「組み立て済み router の canonical location guard を外す」を注入すると `TestTenantIsReachableOnlyAtItsCanonicalLocation` が失敗した。"
  - id: tenant-administration
    unit_red: "`TestTenantLifecycle` が故障注入下で失敗するのを観測した: --- FAIL: TestTenantLifecycle (0.00s)"
    e2e_red: "`TestAdminSettingsPatchUpdatesAndEmitsEvent` が故障注入下で失敗するのを観測した: --- FAIL: TestAdminSettingsPatchUpdatesAndEmitsEvent (0.00s) / admin_settings_handler_test.go:145: status=404 body={'type':'urn:idmagic:error:tenant_not_found','title':'Tenant not found','status':404,…"
    unit_fault_injection: "backend/tenancy/usecases/manage_tenants.go へ「既存 realm と衝突する Tenant の作成を拒否しない」を注入すると `TestTenantLifecycle` が失敗した。"
    e2e_fault_injection: "backend/tenancy/handlers_http/admin_settings_handler.go へ「control-plane PATCH を tenant repository へ接続しない」を注入すると `TestAdminSettingsPatchUpdatesAndEmitsEvent` が失敗した。"
  - id: tenant-experience
    unit_red: "`TestUpdateBrandingPersistsAndClearsFields` が故障注入下で失敗するのを観測した: --- FAIL: TestUpdateBrandingPersistsAndClearsFields (0.00s)"
    e2e_red: "`TestUpdateBrandingPersistsAndIsPubliclyVisible` が故障注入下で失敗するのを観測した: --- FAIL: TestUpdateBrandingPersistsAndIsPubliclyVisible (0.01s)"
    unit_fault_injection: "backend/tenancy/usecases/manage_branding.go へ「branding 更新の明示的な field clear を無視する」を注入すると `TestUpdateBrandingPersistsAndClearsFields` が失敗した。"
    e2e_fault_injection: "backend/tenancy/handlers_http/admin_branding_handler.go へ「管理更新と公開読取が別テナントの branding を指す」を注入すると `TestUpdateBrandingPersistsAndIsPubliclyVisible` が失敗した。"
  - id: claim-release
    unit_red: "`TestIssueClaimsWithFloor_RejectsPrivateSourceAttribute` が故障注入下で失敗するのを観測した: --- FAIL: TestIssueClaimsWithFloor_RejectsPrivateSourceAttribute (0.00s)"
    e2e_red: "`TestUserInfoAppliesClientClaimMappingPolicy` が故障注入下で失敗するのを観測した: --- FAIL: TestUserInfoAppliesClientClaimMappingPolicy (0.00s)"
    unit_fault_injection: "backend/claimmapping/usecases/floor.go へ「private 属性の公開 floor を適用しない」を注入すると `TestIssueClaimsWithFloor_RejectsPrivateSourceAttribute` が失敗した。"
    e2e_fault_injection: "backend/oauth2/handlers_http/userinfo_handler.go へ「userinfo 発行経路から application claim policy を外す」を注入すると `TestUserInfoAppliesClientClaimMappingPolicy` が失敗した。"
  - id: audit-query
    unit_red: "`TestParseAuditFilterAcceptsAllowlisted` が故障注入下で失敗するのを観測した: --- FAIL: TestParseAuditFilterAcceptsAllowlisted (0.00s) / audit_search_test.go:17: unexpected error: unknown search field: event.type"
    e2e_red: "`TestAdminAuditEventsExportSetsAttachment` が故障注入下で失敗するのを観測した: --- FAIL: TestAdminAuditEventsExportSetsAttachment (0.00s)"
    unit_fault_injection: "backend/audit/usecases/audit_search.go へ「許可済み検索属性を filter parser で拒否する」を注入すると `TestParseAuditFilterAcceptsAllowlisted` が失敗した。"
    e2e_fault_injection: "backend/audit/handlers_http/admin_audit_event_handler.go へ「export endpoint の絞り込み条件を repository 検索へ渡さない」を注入すると `TestAdminAuditEventsExportSetsAttachment` が失敗した。"
  - id: api-token-lifecycle
    unit_red: "`TestIssueListAndRevokeApiToken` が故障注入下で失敗するのを観測した: --- FAIL: TestIssueListAndRevokeApiToken (0.00s)"
    e2e_red: "`TestAdminApiTokenLifecycle` が故障注入下で失敗するのを観測した: --- FAIL: TestAdminApiTokenLifecycle (0.00s)"
    unit_fault_injection: "backend/apitoken/usecases/usecases.go へ「失効時に token lifecycle record を無効化しない」を注入すると `TestIssueListAndRevokeApiToken` が失敗した。"
    e2e_fault_injection: "backend/apitoken/handlers_http/routes.go へ「管理 API の失効が対象 token の記録へ届かない」を注入すると `TestAdminApiTokenLifecycle` が失敗した。"
  - id: runtime-health
    unit_red: "`TestReadinessReportsDependencyFailure` が故障注入下で失敗するのを観測した: --- FAIL: TestReadinessReportsDependencyFailure (0.00s)"
    e2e_red: "`TestHealthProbes` が故障注入下で失敗するのを観測した: --- FAIL: TestHealthProbes (0.01s)"
    unit_fault_injection: "backend/shared/http/server_http/health_handler.go へ「未準備の依存先を ready と集約する」を注入すると `TestReadinessReportsDependencyFailure` が失敗した。"
    e2e_fault_injection: "backend/shared/http/server_http/health_handler.go へ「組み立て済み health route が起動完了状態を読まない」を注入すると `TestHealthProbes` が失敗した。"
  - id: localized-ui
    unit_red: "`uses the saved locale when no supported hint is present` が故障注入下で失敗するのを観測した: error: expect(received).toBe(expected) / (fail) resolveLocale > uses the saved locale when no supported hint is present [0.15ms]"
    e2e_red: "`selected locale renders authentication account and admin surfaces` が故障注入下で失敗するのを観測した: error: timeout waiting for text: ログイン / (fail) selected locale renders authentication account and admin surfaces [10774.36ms]"
    unit_fault_injection: "frontend/src/lib/i18n/resolveLocale.ts へ「保存済み locale を解決優先順位から外す」を注入すると `uses the saved locale when no supported hint is present` が失敗した。"
    e2e_fault_injection: "frontend/src/main.tsx へ「router root から LocaleProvider を外す」を注入すると `selected locale renders authentication account and admin surfaces` が失敗した。"
  - id: wsfed-passive-sign-in
    unit_red: "`TestValidateSignIn_HappyPathWithWreply` が故障注入下で失敗するのを観測した: --- FAIL: TestValidateSignIn_HappyPathWithWreply (0.00s)"
    e2e_red: "`TestWsFedSignIn_AuthenticatedIssuesPassiveForm` が故障注入下で失敗するのを観測した: --- FAIL: TestWsFedSignIn_AuthenticatedIssuesPassiveForm (0.12s)"
    unit_fault_injection: "backend/wsfederation/domain/wsfed.go へ「検証結果の返信先へ要求された wreply を反映しない」を注入すると `TestValidateSignIn_HappyPathWithWreply` が失敗した。"
    e2e_fault_injection: "backend/wsfederation/handlers_http/wsfed_handler.go へ「passive endpoint から署名済み assertion 発行を外す」を注入すると `TestWsFedSignIn_AuthenticatedIssuesPassiveForm` が失敗した。"
  - id: wstrust-token-issuance
    unit_red: "`TestBuildRSTR` が故障注入下で失敗するのを観測した: --- FAIL: TestBuildRSTR (0.00s)"
    e2e_red: "`TestWsTrustUsernameMixed_IssuesRSTR` が故障注入下で失敗するのを観測した: --- FAIL: TestWsTrustUsernameMixed_IssuesRSTR (0.08s)"
    unit_fault_injection: "backend/wsfederation/responses_wsfederation/rstr.go へ「RSTR の AppliesTo と token lifetime を要求から反映しない」を注入すると `TestBuildRSTR` が失敗した。"
    e2e_fault_injection: "backend/wsfederation/handlers_http/wstrust_handler.go へ「WS-Trust endpoint から資格情報検証または token signer を外す」を注入すると `TestWsTrustUsernameMixed_IssuesRSTR` が失敗した。"
  - id: shared-signal-revocation
    unit_red: "`TestAdvanceRevocationEpoch_EmitsForEachAgent` が故障注入下で失敗するのを観測した: --- FAIL: TestAdvanceRevocationEpoch_EmitsForEachAgent (0.00s)"
    e2e_red: "`TestAdminAgentKill_AdvancesRevocationEpoch` が故障注入下で失敗するのを観測した: --- FAIL: TestAdminAgentKill_AdvancesRevocationEpoch (0.00s)"
    unit_fault_injection: "backend/sharedsignals/usecases/revocation.go へ「失効時に Agent の revocation epoch を進めない」を注入すると `TestAdvanceRevocationEpoch_EmitsForEachAgent` が失敗した。"
    e2e_fault_injection: "backend/shared/http/server_http/routes.go へ「shared signal の受信処理と token introspection の epoch repository を分離する」を注入すると `TestAdminAgentKill_AdvancesRevocationEpoch` が失敗した。"
  - id: shared-signal-stream
    unit_red: "`TestRegisterSsfTransmitterStream_Succeeds` が故障注入下で失敗するのを観測した: --- FAIL: TestRegisterSsfTransmitterStream_Succeeds (0.00s)"
    e2e_red: "`TestTransmitterStreamLifecycleAndDelivery` が故障注入下で失敗するのを観測した: --- FAIL: TestTransmitterStreamLifecycleAndDelivery (0.00s)"
    unit_fault_injection: "backend/sharedsignals/usecases/admin_streams.go へ「送信 stream の event type 購読を保存しない」を注入すると `TestRegisterSsfTransmitterStream_Succeeds` が失敗した。"
    e2e_fault_injection: "backend/sharedsignals/usecases/admin_streams.go へ「stream 管理 API が保存した送信設定を配送側 repository へ書かない」を注入すると `TestTransmitterStreamLifecycleAndDelivery` が失敗した。"
  - id: application-configuration
    unit_red: "`TestUpdateApplicationChangesAndNoop` が故障注入下で失敗するのを観測した: --- FAIL: TestUpdateApplicationChangesAndNoop (0.00s)"
    e2e_red: "`TestApplicationAdminCRUDAndAccountVisibility` が故障注入下で失敗するのを観測した: --- FAIL: TestApplicationAdminCRUDAndAccountVisibility (0.00s) / application_handler_test.go:166: create status=400 body={'type':'urn:idmagic:error:invalid_request','title':'Invalid request','status':…"
    unit_fault_injection: "backend/application/usecases/applications.go へ「更新結果を application repository へ保存しない」を注入すると `TestUpdateApplicationChangesAndNoop` が失敗した。"
    e2e_fault_injection: "backend/application/handlers_http/application_provisioning.go へ「管理 API の Application 定義を use case と repository へ渡さない」を注入すると `TestApplicationAdminCRUDAndAccountVisibility` が失敗した。"
  - id: application-access-policy
    unit_red: "`TestCreateAndListMyApplicationsRespectsAssignmentAndVisibility` が故障注入下で失敗するのを観測した: --- FAIL: TestCreateAndListMyApplicationsRespectsAssignmentAndVisibility (0.00s)"
    e2e_red: "`TestAuthorizeAllowsHiddenApplicationAssignment` が故障注入下で失敗するのを観測した: --- FAIL: TestAuthorizeAllowsHiddenApplicationAssignment (0.00s)"
    unit_fault_injection: "backend/application/usecases/assignments.go へ「割り当てのない非公開 Application を account 一覧へ含める」を注入すると `TestCreateAndListMyApplicationsRespectsAssignmentAndVisibility` が失敗した。"
    e2e_fault_injection: "backend/shared/http/support_http/application_gate.go へ「authorize 経路の割当判定が hidden 割当を未割当として扱う」を注入すると `TestAuthorizeAllowsHiddenApplicationAssignment` が失敗した。"
  - id: lifecycle-workflow-administration
    unit_red: "`TestLifecycleWorkflowCreateUpdateAndTransitions` が故障注入下で失敗するのを観測した: --- FAIL: TestLifecycleWorkflowCreateUpdateAndTransitions (0.00s)"
    e2e_red: "`TestAdminLifecycleWorkflowDryRunReflectsActualUserState` が故障注入下で失敗するのを観測した: --- FAIL: TestAdminLifecycleWorkflowDryRunReflectsActualUserState (0.00s) / admin_lifecycle_workflow_handler_test.go:131: dry_run status=400 body={'type':'urn:idmagic:error:invalid_request','title':'I…"
    unit_fault_injection: "backend/idgovernance/usecases/lifecycle_workflows.go へ「enable 時に draft revision を固定しない」を注入すると `TestLifecycleWorkflowCreateUpdateAndTransitions` が失敗した。"
    e2e_fault_injection: "backend/idgovernance/handlers_http/admin_lifecycle_workflow_handler.go へ「dry-run が管理 API の対象 User を参照しない」を注入すると `TestAdminLifecycleWorkflowDryRunReflectsActualUserState` が失敗した。"
  - id: lifecycle-workflow-execution
    unit_red: "`TestLifecycleWorkflowRunHandlerEmitsRunStartedAndRunSucceeded` が故障注入下で失敗するのを観測した: --- FAIL: TestLifecycleWorkflowRunHandlerEmitsRunStartedAndRunSucceeded (0.00s)"
    e2e_red: "`TestUserChangeRunsLifecycleWorkflowToDeclaredEffects` が故障注入下で失敗するのを観測した: --- FAIL: TestUserChangeRunsLifecycleWorkflowToDeclaredEffects (0.04s)"
    unit_fault_injection: "backend/idgovernance/usecases/lifecycle_workflow_dispatcher.go へ「成功した workflow step の checkpoint を保存しない」を注入すると `TestLifecycleWorkflowRunHandlerEmitsRunStartedAndRunSucceeded` が失敗した。"
    e2e_fault_injection: "backend/idgovernance/usecases/lifecycle_workflow_dispatcher.go へ「dispatcher が未投入の run を Job へ投入しない」を注入すると `TestUserChangeRunsLifecycleWorkflowToDeclaredEffects` が失敗した。"
