---
status: completed
authors: [tn]
risk: low
created_at: 2026-07-05
---

# postgres adapter の未テスト・リポジトリに対する統合テストを追加する

## Motivation

wi-129 (cf771d17) で postgres adapter に embedded-postgres ハーネスと初期テストが導入され
カバレッジが 0% → ~31% へ改善したが、対象は以下の 9 リポジトリ／ストアに限定されていた:

- TenantRepository, UserRepository, OAuth2ClientRepository, ConsentRepository,
  RefreshTokenStore, MfaFactorRepository, PasswordHistoryRepository,
  AgentRepository, GroupRepository, ScimRepository

以下の 11 リポジトリ／ストアにはまだラウンドトリップテストが存在しない。
スキーマの回帰やカラムマッピング不一致を検出するために、残りのリポジトリにも
同じハーネスを用いた統合テストを追加し、postgres パッケージのカバレッジをさらに
引き上げる。

## Scope

テスト追加対象リポジトリ（実装ファイル → 主要メソッド）:

1. **ApplicationRepository** (`applications.go`)
   - `Save`, `FindByID`, `FindByBinding`, `ListByTenant`, `Delete`, `RemoveCategory`
2. **SignInPolicyRepository** (`applications.go`)
   - `Save`, `Get`, `ListByTenant`, `Delete`
3. **DefaultSignInPolicyRepository** (`applications.go`)
   - `Save`, `Get`
4. **ApplicationIconStore** (`applications.go`)
   - `Save`, `Find`, `DeleteByApplication`
5. **ApplicationAssignmentRepository** (`applications.go`)
   - `Save`, `ListByTenant`, `ListByApplication`, `ListBySubjects`, `Delete`, `DeleteByApplication`
6. **ApplicationOrderingRepository** (`applications.go`)
   - `Save`, `Get`
7. **ApplicationCategoryRepository** (`applications.go`)
   - `Save`, `FindByID`, `ListByTenant`, `Delete`
8. **AuditEventRepository** (`audit_events.go`)
   - `Append`, `List`（フィルタ条件付きクエリ）
9. **AuthEventBucketStore** (`auth_event_buckets.go`)
   - `Record`, `List`, `DeleteOlderThan`
10. **AuthorizationDetailTypeRepository** (`authorization_detail_types.go`)
    - `Save`, `FindByType`, `ListByTenant`, `Delete`
11. **EmailChangeTokenStore** (`email_change_token.go`)
    - `Save`, `Consume`
12. **KeyStore** (`keys.go`)
    - `NewKeyStore`, `GetActiveKey`, `GetAllKeys`, `FindByKID`, `Rotate`
13. **OutboxEventSink** (`outbox.go`)
    - `Emit`
14. **PasswordResetTokenStore** (`password_reset_token.go`)
    - `Save`, `Consume`
15. **SamlServiceProviderRepository** (`saml_service_providers.go`)
    - `Save`, `FindByEntityID`, `ListByTenant`
16. **TenantUserAttributeSchemaRepository** (`tenant_user_attribute_schema.go`)
    - `Save`, `FindByTenant`, `Delete`
17. **WsFedRelyingPartyRepository** (`wsfed_relying_parties.go`)
    - `Save`, `FindByWtrealm`, `ListByTenant`

## Out of Scope

- 既にテスト済みのリポジトリ（Tenant, User, OAuth2Client, Consent, RefreshToken, Mfa, PasswordHistory, Agent, Group, Scim）への追加テスト。
- postgres パッケージ以外のテスト追加（他パッケージは別 WI で対応済み）。
- CI 強制ルールの適用（wi-131 で対応）。

## Initial Context

- テストハーネス: `internal/shared/adapters/persistence/postgres/harness_test.go`
- テストフィクスチャ: `internal/shared/adapters/persistence/postgres/fixtures_test.go`
- 既存テスト例: `internal/shared/adapters/persistence/postgres/repositories_test.go`

## Affected Guarantees

- persistence adapter のカラムマッピング正当性
- CRUD ラウンドトリップの回帰検出

## Verification

- `just test-go`（全テストがグリーン）
- `just verify-go`

## Risk Notes

既存ハーネス（embedded-postgres）を再利用するため追加インフラ作業は不要。
テストデータのフィクスチャ追加（Application, AuditEvent 等）が必要だが、
既存パターンに従えば低リスク。KeyStore の `Rotate` はテスト順序依存性に注意。

## Completion

- **Completed At**: 2026-07-05
- **Summary**:
  Scope に挙げた 17 リポジトリ／ストア（Application 系 7 種、AuditEvent、AuthEventBucket、
  AuthorizationDetailType、EmailChangeToken、KeyStore、Outbox、PasswordResetToken、
  SamlServiceProvider、TenantUserAttributeSchema、WsFedRelyingParty）に対し、既存の
  embedded-postgres ハーネスを用いた往復（Roundtrip）統合テストを追加した。テストは以下の
  3 ファイルに整理した:
  - `applications_test.go`（Application / SignInPolicy / DefaultSignInPolicy / ApplicationIcon /
    ApplicationAssignment / ApplicationOrdering / ApplicationCategory）
  - `federation_test.go`（SamlServiceProvider / WsFedRelyingParty）
  - `stores_test.go`（AuditEvent / AuthEventBucket / AuthorizationDetailType / EmailChangeToken /
    PasswordResetToken / TenantUserAttributeSchema / Outbox / KeyStore）
  併せて共有フィクスチャに `seedApplication` を追加した。
  postgres パッケージのカバレッジは約 31% から 72.1% へ向上した。
- **Defect Found & Fixed (application_orderings の tenant_id 排除)**:
  `ApplicationOrderingRepository` の往復テストが `column "tenant_id" does not exist` を検出した。
  原因は ADR-083（globally unique client_id / user_id、composite FK 廃止）の適用漏れ:
  compaction commit `c1a80f1` は `application_orderings` の DDL のみを単純化（`user_id` 単一 PK、
  `tenant_id` 列削除）したが、SCL モデル・Go ドメイン型・memory / postgres リポジトリ・use case は
  `tenant_id` 前提のまま取り残されていた。`users.id` は global unique なので `tenant_id` は不要。
  したがって（当初誤って行った DDL への `tenant_id` 再追加は撤回し）実装側を ADR-083 に合わせて修正した:
  - SCL: `spec/contexts/application.yaml` の `ApplicationOrdering` から `tenant_id`（identity / field）を
    削除、`ApplicationReferencesStayTenantLocal` invariant を `ordering.user_id in Users` に更新。派生物を再生成。
  - Go: `spec.ApplicationOrdering.TenantID` を削除。postgres / memory リポジトリを `user_id` キーに変更
    （port の `tenantID` 引数は `consents` と同様に互換維持のため残置し SQL では不使用）。use case の
    struct literal と Get 呼び出しを修正。
  - Docs: `postgres.sql` 冒頭の tenant_id key policy コメントを ADR-083 反映へ更新。
    ADR-083 に `user_id` 従属子の `tenant_id` 排除方針（`application_orderings` 明記）を追記。
- **Verification Results**:
  - `just verify-go` — 成功（lint クリーン + race 有効テスト green）
  - `just yaml-check` — 成功
  - `go test -cover ./internal/shared/adapters/persistence/postgres/` — 成功（coverage 72.1%）
