---
status: completed
authors: [tn]
risk: medium
created_at: 2026-07-03
depends_on: [wi-24-hibp-breached-password-checker]
change_kind: feature
initial_context:
  specification:
    - spec/contexts/authentication/SPECIFICATION.md#REQ-AUTHENTICATION-010
    - spec/contexts/authentication/SPECIFICATION.md#REQ-AUTHENTICATION-016
    - spec/contexts/authentication/SPECIFICATION.md#REQ-AUTHENTICATION-022
    - spec/contexts/tenancy/SPECIFICATION.md#REQ-TENANCY-019
  typespec:
    - IdMagic.Tenancy.PasswordPolicyOverride
    - IdMagic.Tenancy.PasswordPolicyDefaults
    - IdMagic.Tenancy.AdminSettingsResponse
    - IdMagic.Tenancy.Operations.UpdateAdminSettings
  source:
    - backend/authentication/password/domain/password_policy_resolver.go
    - backend/authentication/password/usecases/password_policy.go
    - backend/authentication/password/usecases/change_password.go
    - backend/authentication/password/usecases/reset_password_with_token.go
    - backend/authentication/password/handlers_http/password_reset_handler.go
    - backend/authentication/password/handlers_http/change_password_handler.go
    - backend/oauth2/handlers_http/authorize_login.go
    - backend/idmanagement/user/usecases/admin_users.go
    - backend/tenancy/domain/tenancy.go
    - backend/tenancy/usecases/manage_tenants.go
    - backend/tenancy/db_postgres/tenants.go
    - backend/tenancy/db_postgres/tenants.sql
    - infra/schema/postgres.sql
    - frontend/src/features/admin-settings/PasswordPolicyTab.tsx
    - frontend/src/features/admin-settings/AdminSettingsShared.tsx
  tests:
    - backend/authentication/password/domain/password_policy_resolver_test.go
    - backend/authentication/password/usecases/password_policy_test.go
    - backend/authentication/password/usecases/change_password_test.go
    - backend/tenancy/usecases/manage_tenants_test.go
    - backend/tenancy/db_postgres/tenants_test.go
    - frontend/src/features/admin-settings/AdminSettingsPage.test.tsx
  stop_before_reading:
    - backend/authentication/federation
    - backend/saml
    - backend/wsfederation
affected_spec:
  - { path: spec/contexts/tenancy/models.tsp, symbol: PasswordPolicyOverride }
  - { path: spec/contexts/tenancy/models.tsp, symbol: PasswordPolicyDefaults }
  - { path: spec/contexts/tenancy/scenarios.md, requirement: REQ-TENANCY-019 }
  - { path: spec/contexts/authentication/scenarios.md, requirement: REQ-AUTHENTICATION-024 }
---

# テナントのパスワードポリシーを永続化し、任意の有効期限で次回ログイン時の変更を強制する

## Motivation

`PasswordPolicyOverride` (min_length / max_length / history_depth) はモデル・use case・admin
UI まで揃っているが、実際には二つの穴がある。

1. **上書きが PostgreSQL に保存されていない。** `backend/tenancy/db_postgres/tenants.go` の
   `SaveTenant` は `password_policy_override` を書かず、`tenantFromRow` も読まない。`tenants`
   テーブルに列が無い。db_memory では動くため、開発中は動作して見えるが、PostgreSQL 配備では
   管理者が保存したポリシーが即座に消える。「設定可能なパスワードポリシー」が本番で成立して
   いない。
2. **有効期限 (max age) が無い。** NIST SP 800-63B-4 は定期変更の強制を非推奨とするので既定
   off で正しいが、規制要件でローテーションを求められるテナントに逃げ道が無い。
   `password_changed_at` と `RequiredActionUpdatePassword`、ログイン後の gate
   (`recordLoginAndRequiredAction`) は既にあるので、欠けているのは判定だけである。

加えて、ポリシー適用経路に取りこぼしがある。admin の `CreateUser` はテナント解決を通さない
`ValidatePassword` (global default) で検証しており、テナントが min_length を上げても管理者
発行パスワードには効かない。change-password は breach チェックを実行しておらず、同じ検証を
通すはずの reset-password とだけ挙動が違う。

## Scope

- **spec**:
  - `PasswordPolicyOverride` / `PasswordPolicyDefaults` に `max_age_days` を追加する。既定は
    0 (無効) で、テナントが明示的に opt-in する。
  - Authentication の Design (Password lifecycle) に、有効期限の評価基準時刻・除外対象・
    ポリシー変更時の猶予を記す。
  - 新規 `REQ-AUTHENTICATION-024` (有効期限超過ユーザーは次回ログインで変更を強制される) と、
    `REQ-TENANCY-019` の ALT (system ceiling を超える max_age_days は拒否 / 保存が永続する)。
- **domain**: `PasswordPolicySnapshot.MaxAgeDays` と、`password_changed_at` / ポリシー更新時刻 /
  現在時刻から期限超過を判定する純関数。
- **usecases**: 期限超過ユーザーに `update_password` required action を付与する use case。
  admin `CreateUser` をテナント解決済み snapshot へ移す。change-password に breach チェックを
  追加して reset-password と同じ検証段にする。
- **persistence**: `tenants` に `password_policy_override` (JSONB) と
  `password_policy_updated_at` を追加し、sqlc を再生成する。
- **http/ui**: ログイン完了時に期限を評価して既存の change-password gate に載せる。
  admin settings に max_age_days の編集と表示を追加する。
- **documentation**: README の Key Capabilities に、既定ポリシーとテナント上書き・有効期限を
  1 項目として追記する。

## Out of Scope

- セルフサービス登録経路へのポリシー適用 ([[wi-87-self-service-user-registration]] が未実装のため、
  対象の経路が存在しない)。
- テナント単位の breach チェック on/off と HIBP モード選択。breach adapter は起動時の
  `BREACHED_PASSWORD_CHECKER` で選ぶ配備単位の設定のままにする。
- 文字種混在などの composition rule (NIST が明示的に非推奨とするため導入しない)。
- 管理 UI でのポリシー変更影響人数プレビュー。
- リアルタイム strength meter と外部辞書サービス連携。
- パスワードなし強制 / passwordless-only ポリシー。

## Design

**永続化。** `password_policy_override` は JSONB 1 列で持つ。フィールドは 4 つだが、上書き有無を
NULL で表す必要があり、今後もフィールドが増える見込みのある「テナント設定の袋」であって、単体で
検索・集計する対象ではない。列を 4 本足すと、増えるたびにマイグレーションと sqlc 再生成が要る。
`tenant_quotas` のような別テーブルは、上書きが tenant 集約に埋め込まれた値オブジェクトである
以上、読み書きのたびに join を増やすだけになる。

**有効期限の基準時刻。** 期限は `max(user.password_changed_at, tenant.password_policy_updated_at)`
から測る。`password_changed_at` だけを見ると、管理者が max_age を有効にした瞬間に、それより前
から変えていない全ユーザーが一斉に期限切れになる。ポリシー更新時刻を下限に入れると、ポリシー
変更後は誰もが必ず max_age 日分の猶予を得るので、別建ての grace 設定を持たずに同じ結果になる。
`password_changed_at` が nil のユーザー (一度も変更記録が無い) も同じ規則で扱える。

**除外。** `PasswordHash` が空のユーザー (federated / passwordless) は評価対象外とする。強制した
ところで変更できるパスワードが無く、change-password 画面で行き止まりになる。

**強制の形。** 期限超過は credential を無効化せず、ログイン成立後に `update_password` required
action を付与するだけにする。既存の `recordLoginAndRequiredAction` が change-password 画面へ
gate し、change-password / reset-password の成功が action を自動解除する経路も既にある。
認証そのものを失敗させると、ユーザーはリセットメール経路へ回るしかなくなり、期限切れが
アカウントロックと区別できなくなる。

**system ceiling。** `max_age_days` はテナントが自由に短くできると DoS 相当の UX 破壊になるので、
下限 30 日・上限 3650 日で gate する。既存の `PolicyFloor` (min/max length・history depth) と
同じ `enforcePolicyFloor` で拒否し、`ErrPolicyOverrideWeaker` を返す。

## Plan

1. spec: TypeSpec に `max_age_days`、Authentication Design に期限の規則、REQ-AUTHENTICATION-024
   と REQ-TENANCY-019 の ALT。`just check-spec` → `just spec-render`。
2. domain: `PasswordPolicySnapshot.MaxAgeDays` と `PasswordExpiredAt` 判定 (RED 先行)。
3. usecases: 期限評価 use case、admin `CreateUser` のテナント解決、change-password の breach
   チェック、`enforcePolicyFloor` の max age ceiling。
4. persistence: `infra/schema/postgres.sql` に 2 列、`tenants.sql` を更新して `just
   sqlc-generate`、`tenants.go` の read/write。
5. http/ui: ログイン gate への組み込み、admin settings の max age フィールドと i18n。
6. verify: `just test-go` / `just verify`。

## Tasks

- [x] T001 [Spec] `max_age_days` を TypeSpec に追加し、Authentication の Password lifecycle
      Design と REQ-AUTHENTICATION-024 / REQ-TENANCY-019 ALT を書いて `just check-spec` を通す。
- [x] T002 [Domain] `PasswordPolicySnapshot.MaxAgeDays` と期限判定の純関数を実装する
      (RED: `TestPasswordExpired` / `TestResolvePasswordPolicy`
      @ `password_policy_resolver_test.go`, REQ-AUTHENTICATION-024)。
- [x] T003 [Usecases] 期限評価 use case、admin `CreateUser` のテナント解決、change-password の
      breach チェック、max age の system bounds を実装する
      (RED: `TestEnforcePasswordExpiry` / `TestResolveTenantPolicy` @ `password_expiry_test.go`、
      `TestChangePasswordRejectsBreachedPassword` @ `change_password_test.go`、
      `TestUpdateRejectsWeakerPolicyOverride` / `TestUpdateRecordsPasswordPolicyUpdatedAt`
      @ `manage_tenants_test.go`, REQ-AUTHENTICATION-010, REQ-AUTHENTICATION-024, REQ-TENANCY-019)。
- [x] T004 [Persistence] `tenants` に `password_policy_override` (JSONB) と
      `password_policy_updated_at` を追加し、sqlc 再生成と repository の read/write を実装する
      (RED: `TestTenantRepositoryPersistsPasswordPolicyOverride` @ `tenants_test.go`,
      REQ-TENANCY-019)。
- [x] T005 [Adapters/UI] ログイン完了時の期限評価と admin settings の max age 編集を追加する
      (RED: `TestLoginWithExpiredPasswordIsGatedToChangePassword` @ `password_expiry_e2e_test.go`、
      `saves the password expiry as part of the policy override` @ `AdminSettingsPage.test.tsx`,
      REQ-AUTHENTICATION-024)。
- [x] T006 [Verify] 境界 (system bounds / 猶予 / passwordless 除外 / 全変更経路) を検証し
      `just verify`。

## Verification

- `just test-go`
- `just verify`
- 手動: admin settings で max_age_days を設定 → 保存後にプロセスを再起動しても値が残る →
  期限超過ユーザーのログイン後に change-password へ誘導される。

## Risk Notes

`password_policy_override` の永続化は、これまで保存されていなかった値が初めて効き始めることを
意味する。db_memory では既に効いていたため挙動差が出るのは PostgreSQL 配備だけだが、既に UI から
保存を試みたテナントがあれば、その値は失われている (再入力が必要)。

有効期限は既定 off なので、opt-in しないテナントの挙動は変わらない。opt-in したテナントでも、
ポリシー更新時刻を基準に入れるため一斉ロックアウトは起きない。最大のリスクは基準時刻の計算を
誤って全員を即時期限切れにすることなので、猶予と clock 境界にテストを置く。

## Completion

- **Completed At**: 2026-08-13
- **Summary**:
  テナントのパスワードポリシー上書きが PostgreSQL に永続化されるようになり、`tenants` に
  `password_policy_override` (JSONB) と `password_policy_updated_at` が加わった。これまで
  `SaveTenant` は上書きを書いておらず、PostgreSQL 配備では管理者が保存した値が保存時点で
  失われていた (db_memory でのみ機能していた)。
  併せて、テナントが opt-in できる有効期限 `max_age_days` (既定 0 = 無効、system bounds
  30〜3650 日) を追加した。期限は `max(password_changed_at, password_policy_updated_at)` から
  測るため、有効化した瞬間に既存ユーザーが一斉に期限切れになることはない。期限超過は認証を
  失敗させず、ログイン成立後に `update_password` required action を付与して既存の
  change-password gate に載せる。password credential を持たないユーザーは対象外。
  ポリシー適用の取りこぼしも塞いだ: 管理者による `CreateUser` は global default ではなく
  テナント解決済みポリシーで検証するようになり、change-password は reset-password と同じ
  breach チェックを通すようになった。`PasswordPolicySnapshot` は domain 側の単一定義へ
  集約し (usecases 側は型エイリアス)、層をまたぐ詰め替えを無くした。
  正規シナリオの差分は `REQ-AUTHENTICATION-024` の追加と `REQ-TENANCY-019` の変更
  (弱い上書き / system bounds の ALT と、永続性の THEN)。
- **Verification Results**:
  - `just verify` - passed (test-go / test-ui-unit / lint-go / lint-ui / build-ui / check /
    check-api-compat / format-check-ui / test-tools / typecheck-tools)
  - `just check-schema` - passed (schema convergence)
  - `just spec-diff` - added: REQ-AUTHENTICATION-024 / changed: REQ-TENANCY-019
