---
status: completed
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-09-03
change_kind: bugfix
priority: p1
depends_on: []
evidence_policy: risk-based-v3
documentation_impact:
  level: upgrade_note
  reason: "制御面テナント外に保存済みの `system_admin` は、この変更後にテナント横断の参照能力を失う。運用者は失われる経路と回復手順を知る必要がある。"
  references:
    - { kind: release_note, path: docs/releases/changes/wi-460.md }
    - { kind: upgrade_note, path: docs/releases/upgrades/wi-460.md }
initial_context:
  specification:
    - docs/contexts/signing-keys/scenarios.md#REQ-SIGNINGKEYS-008
    - docs/contexts/signing-keys/scenarios.md#REQ-SIGNINGKEYS-009
    - docs/contexts/data-keys/scenarios.md#REQ-DATAKEYS-006
    - docs/contexts/jobs/scenarios.md#REQ-JOBS-012
    - docs/contexts/jobs/scenarios.md#REQ-JOBS-013
    - docs/authorization.md
    - docs/contexts/signing-keys/decisions.md
    - docs/contexts/data-keys/decisions.md
    - docs/contexts/identity-management/glossary.md
  typespec:
    - IdMagic.SigningKeys.Operations.ListTenantKeyHealth
    - IdMagic.DataKeys.Operations.ListTenantDataKeyHealth
    - IdMagic.Jobs.Operations.ListJobs
    - IdMagic.Jobs.Operations.GetJob
    - IdMagic.Jobs.Operations.CancelJob
  source:
    - backend/shared/http/support_http/auth.go
    - backend/shared/http/server_http/routes.go
    - backend/signingkeys/handlers_http/admin_key_handler.go
    - backend/datakeys/handlers_http/admin_data_key_handler.go
    - backend/jobs/handlers_http/admin_job_handler.go
    - backend/tenancy/handlers_http/admin_tenant_handler.go
    - backend/audit/handlers_http/admin_audit_event_handler.go
    - backend/authentication/handlers_http/account_context_handler.go
    - frontend/src/routes/-guards.ts
  tests:
    - backend/shared/http/support_http/auth_admin_test.go
    - backend/shared/http/support_http/auth_control_plane_test.go
    - backend/shared/http/server_http/tenant_routes_test.go
    - backend/shared/http/server_http/control_plane_boundary_test.go
    - backend/oauth2/handlers_http/admin_key_handler_test.go
    - backend/oauth2/handlers_http/admin_data_key_handler_test.go
    - backend/jobs/handlers_http/admin_job_handler_test.go
    - backend/audit/handlers_http/admin_audit_event_handler_test.go
    - frontend/src/routes/-guards.test.ts
  stop_before_reading: [infra, load, docs/runbooks]
primary_use_cases:
  - id: control-plane-signing-key-health
    requirement: REQ-SIGNINGKEYS-009
    observable_result: 制御面テナント外の `system_admin` は署名鍵ヘルスを取得できず、応答に他テナントの識別子も鍵運用状態も現れない。
    unit_test:
      path: backend/shared/http/support_http/auth_control_plane_test.go
      name: TestRequireControlPlaneUser
      task: test-go-race
    e2e_test:
      path: backend/shared/http/server_http/control_plane_boundary_test.go
      name: TestControlPlaneSigningKeyHealthRejectsSystemAdminOutsideControlPlaneTenant
      task: test-go-race
    unit_fault_model: 共通判定が所属テナントを見ず、有効ロールに `system_admin` があるだけで制御面主体と認める。
    e2e_fault_model: 署名鍵ヘルスの経路が共通判定を通さず、ロールだけを見る旧来の判定へ戻る。
  - id: control-plane-data-key-health
    requirement: REQ-DATAKEYS-006
    observable_result: 制御面テナント外の `system_admin` は DEK ヘルスを取得できず、横断の健全性収集も実行されない。
    unit_test:
      path: backend/shared/http/support_http/auth_control_plane_test.go
      name: TestRequireControlPlaneUserRejectsRequestOutsideControlPlaneTenant
      task: test-go-race
    e2e_test:
      path: backend/shared/http/server_http/control_plane_boundary_test.go
      name: TestControlPlaneDataKeyHealthRejectsSystemAdminOutsideControlPlaneTenant
      task: test-go-race
    unit_fault_model: 共通判定が要求先テナントを見ず、所属テナントだけで制御面主体と認める。
    e2e_fault_model: DEK ヘルスの経路が拒否の前に横断の健全性収集を実行し、拒否応答の裏で他テナントを読み出す。
  - id: control-plane-job-oversight
    requirement: REQ-JOBS-012
    observable_result: 制御面主体は `admin` ロールを併せ持たなくてもテナント横断のジョブ一覧、詳細参照、取消しに到達できる。
    unit_test:
      path: backend/shared/http/support_http/auth_control_plane_test.go
      name: TestIsControlPlaneActor
      task: test-go-race
    e2e_test:
      path: backend/shared/http/server_http/control_plane_boundary_test.go
      name: TestControlPlaneJobOversightNeedsNoAdminRole
      task: test-go-race
    unit_fault_model: 解決済み actor に対する制御面判定が、`system_admin` 以外の管理ロールでも真を返す。
    e2e_fault_model: ジョブ管理経路が制御面主体にも `admin` ロールを要求し、横断へ到達できない。
affected_spec:
  - { path: docs/contexts/signing-keys/scenarios.md, requirement: REQ-SIGNINGKEYS-008 }
  - { path: docs/contexts/signing-keys/scenarios.md, requirement: REQ-SIGNINGKEYS-009 }
  - { path: docs/contexts/data-keys/scenarios.md, requirement: REQ-DATAKEYS-006 }
  - { path: docs/contexts/jobs/scenarios.md, requirement: REQ-JOBS-012 }
  - { path: docs/contexts/jobs/scenarios.md, requirement: REQ-JOBS-013 }
  - { path: spec/contexts/signing-keys/main.tsp, symbol: IdMagic.SigningKeys.Operations.ListTenantKeyHealth }
  - { path: spec/contexts/data-keys/main.tsp, symbol: IdMagic.DataKeys.Operations.ListTenantDataKeyHealth }
  - { path: spec/contexts/jobs/main.tsp, symbol: IdMagic.Jobs.Operations.ListJobs }
  - { path: spec/contexts/jobs/main.tsp, symbol: IdMagic.Jobs.Operations.GetJob }
  - { path: spec/contexts/jobs/main.tsp, symbol: IdMagic.Jobs.Operations.CancelJob }
---

# テナント横断ヘルスの認可を制御面主体の共通判定へ統一する

## Motivation

`docs/authorization.md` は、テナントを越える操作に `system_admin` の有効ロールと `default`（制御面）テナントへの所属を合わせて要求している。

しかし `ListTenantKeyHealth` と `ListTenantDataKeyHealth` は、認証済み User のロールに `system_admin` が含まれることしか確認していない。

そのため、`default` 以外のテナントに属する User が直接または Group 経由で `system_admin` を得ると、そのテナントの管理 API 経路から全テナントの識別子と鍵運用状態を取得できる。

`ListTenantKeyHealth` は `tenant_id`、提供元、`active_kid`、鍵数、到達性を返し、`ListTenantDataKeyHealth` は `tenant_id`、有効版、状態、提供元、到達性を返すため、この欠陥は `docs/authorization.md` の「要求者が権限を持たないテナントの識別子、名前、件数を応答に含めない」という規則に反する。

`REQ-SIGNINGKEYS-009` と `REQ-DATAKEYS-006` も拒否条件を `system_admin` の有無だけで表しており、実装だけでなくシナリオも製品全体のテナント境界より弱い。

同じ制御面主体の判定は Tenancy、Audit、Jobs、SigningKeys、DataKeys の5か所に別々の形で存在し、認証保留、有効状態、所属テナント、有効ロールの扱いが一致していない。

Jobs は横断範囲を作る前に `RequireAdmin` を通すため、TypeSpec と `REQ-JOBS-012` が要求する `system_admin` に加えて `admin` も持たなければ横断一覧、詳細参照、取消しへ到達できない。

## Scope

- `REQ-SIGNINGKEYS-008`、`REQ-SIGNINGKEYS-009`、`REQ-DATAKEYS-006` に、制御面主体の許可条件と `default` 以外のテナントに属する `system_admin` の拒否を記述する。
- `ListTenantKeyHealth` と `ListTenantDataKeyHealth` の TypeSpec 文書コメントを同じ条件へ揃える。
- 認証済みで保留状態ではなく、有効であり、要求先と所属先がともに `default` テナントで、有効ロールに `system_admin` を含む User を返す共通の `RequireControlPlaneUser` を `support_http.Authenticator` に追加する。
- Tenancy の制御面 CRUD、Audit の横断検索、Jobs の横断一覧、詳細参照、取消し、SigningKeys と DataKeys の横断ヘルスを共通判定へ移し、Jobs の余分な `admin` 条件を除く。
- `docs/contexts/signing-keys/decisions.md`、`docs/contexts/data-keys/decisions.md`、`docs/contexts/identity-management/glossary.md` に残る「`system_admin` だけでよい」という説明を、制御面テナント所属と有効ロールを含む定義へ修正する。
- フロントエンドの `requireSystemAccount` も、アカウント文脈の有効ロールと realm の両方を確認する。

## Out of Scope

- `system_admin` を予約ロールにして書き込みを制限すること。
  [[wi-463-reserve-system-admin-role-to-control-plane]] が扱う。
- 監査、ジョブなどのテナント横断画面をシステムコンソールへ移すこと。
  [[wi-462-control-plane-console-single-entry]] が扱う。
- 制御面操作に到達できる資格情報の種類を変更すること。
  [[wi-461-control-plane-credential-boundary]] が扱う。
- 認可を HTTP ハンドラーからユースケース層へ移す一般化。
  [[wi-393-guard-rules-reach-the-usecase-layer]] の検討対象であり、本作業項目では既存の強制境界を統一する。
- 制御面操作への再認証またはステップアップ。

## Design

主要なデータ型は、所属テナント、有効状態、有効ロールを保持する `userdomain.User` と、テナント解決済みの要求先を表す `requestTenantID string` である。

純粋な判定は `IsControlPlaneActor(actor *userdomain.User, requestTenantID string) bool` とし、認証済み User の取得と Group 由来ロールの合成は `(*Authenticator).RequireControlPlaneUser(c *echo.Context) (*userdomain.User, error)` が `AuthnResolver`、`UserRepository`、`GroupRepository` の各ポートを通じて行う。

Jobs の範囲変換は `scopeFor(actor *userdomain.User, requestTenantID string) jobusecases.TenantScope` とし、時刻、永続化、イベント発行は既存のユースケース境界へ残す。

### 制御面主体の条件

`RequireControlPlaneUser` は次の条件をすべて満たした User だけを返す。

| 条件 | 理由 |
| --- | --- |
| 認証コンテキストが存在し、`AuthenticationPending` ではない | 認証途中の主体を管理操作へ進めないため |
| User が存在し、`IsActive()` である | 無効化済みの主体を拒否するため |
| 解決済みの要求先テナントが `default` である | 別テナントの経路を制御面の入口として扱わないため |
| User の所属テナントが `default` である | ロール名だけでテナント境界を越えないため |
| `EffectiveRoles` に `system_admin` が含まれる | User への直接付与と Group 由来の付与を同じ意味で扱うため |

`ResolveAuthentication` は現在も User と要求先テナントの一致を確認するが、共通判定でも要求先と所属先を明示的に検査する。

呼び出し側が事前条件へ暗黙に依存すると、認証解決の内部変更だけで制御面の境界が弱くなるためである。

### 真偽値ではなく検証済み User を返す

共通処理は `bool` ではなく、有効ロールを反映した検証済み User を返す。

監査イベントの記録や対象操作が主体 ID を必要とする場合に、認証と User 読み込みをもう一度行わずに済み、検証後に別の User 表現を参照する食い違いも避けられる。

未認証と認証保留は `ErrAdminAuthenticationRequired` とし、認証解決後の条件不一致は `ErrAdminAccessDenied` とする。

無効な User と別テナントのセッションは `ResolveAuthentication` が未認証として処理するため、その既存の応答規則は変えない。

### 既存判定の移行

| 現在の判定 | 移行後 |
| --- | --- |
| Tenancy の `requireSystemAdmin` | `RequireControlPlaneUser` を使用する |
| Audit の `system_admin`、`default`、`all_tenants` の組合せ | `RequireControlPlaneUser` が成功した場合だけ横断範囲を作る |
| Jobs の `RequireAdmin`、`actorMayReadAllTenants`、`scopeFor` | 制御面主体には `RequireControlPlaneUser` だけを要求し、それ以外の `admin` にはテナント内範囲だけを作る |
| SigningKeys と DataKeys の `requireSystemKeyHealthReader` | `RequireControlPlaneUser` を使用する |

Audit と Jobs のテナント内経路は通常の `admin` に開いたままにし、横断範囲へ切り替える箇所だけを共通判定へ通す。

### 拒否の観測

受け入れテストは、`default` 以外のテナントに属する `system_admin` が両方のヘルス API で拒否され、応答に別テナントの識別子や状態が含まれず、ヘルス取得ユースケースも呼ばれないことを確認する。

許可側では、`default` テナントの有効な User が直接または Group 経由で `system_admin` を持つ場合に横断ヘルスを取得でき、`admin` を併有しなくても Jobs の横断一覧、詳細参照、取消しを行えることを確認する。

## Plan

1. 3つのシナリオ、TypeSpec 文書コメント、関連する決定と用語集を制御面主体の条件へ揃える。
2. HTTP 境界で、`default` 以外の `system_admin` が両方の横断ヘルスを取得できる現在の挙動を観測し、受け入れ RED を確認する。
3. `RequireControlPlaneUser` の Unit RED を確認し、認証状態、有効状態、要求先、所属先、有効ロールの条件を実装する。
4. 5か所の制御面判定を共通処理へ移し、Jobs から余分な `admin` 条件を除いた許可経路と拒否経路を回帰テストで固定する。
5. `requireSystemAccount` を同じ条件へ揃え、制御面テナント外の `system_admin` が `/system` を開けないことを確認する。
6. 仕様生成物を再生成し、検査を通す。

## Tasks

- [x] T001 [Spec] 3つのシナリオ、TypeSpec 文書コメント、関連する決定と用語集を更新する。
- [x] T002 [Acceptance] 制御面テナント外の `system_admin` が両方の横断ヘルスを取得できることを HTTP 境界で観測し、RED を確認する。
- [x] T003 [App] `RequireControlPlaneUser` の Unit RED を確認してから共通判定を実装する。
- [x] T004 [App] Tenancy、Audit、Jobs、SigningKeys、DataKeys の制御面判定を共通処理へ移す。
- [x] T005 [App] `requireSystemAccount` を realm と有効ロールの組合せへ揃える。
- [x] T006 [Acceptance] 拒否応答に別テナントの情報が含まれず、横断取得処理が実行されないことを確認する。
- [x] T007 [Verify] 仕様生成物を再生成し、検査を通す。

## Verification

- `mise run test-go-race`
- `mise run test-ui-unit`
- `mise run test-ui-e2e`
- `mise run check-spec`
- `mise run check-ids`
- `mise run check-work-items`
- `mise run verify`

## Risk Notes

リスクは high とする。

この変更はテナント境界を越える情報開示を塞ぐため、条件を緩く実装すると漏えい経路が残り、厳しく実装すると正規のシステム運用者が鍵運用状態を観測できなくなる。

5か所を1つの判定へ集約するため、共通処理の誤りはテナント CRUD、監査、ジョブにも広がる。

各移行先では、制御面テナントの直接ロール、Group 由来ロール、別テナント、無効 User、認証保留を表にした回帰テストを先に置く。

制御面テナント外に既存の `system_admin` が保存されていても、この変更後は横断能力を得ない。

保存済みロールの扱いと新規割当ての禁止は [[wi-463-reserve-system-admin-role-to-control-plane]] が扱い、本作業項目は既存データの変更を行わない。

`reversibility` は reversible とする。

共通判定と認可条件は元へ戻せるが、戻すと同じ情報開示が再発する。

### Assumptions

- セキュリティ：`ResolveAuthentication` が要求先と User の所属先の一致および有効状態を検査するが、制御面判定も同じ条件を明示的に再検査する。
- 互換性：制御面テナント外の `system_admin` が横断ヘルスを取得できた挙動は欠陥であり、HTTP 403 への変更を互換性の修正として扱う。
- 移行：保存済みの User、Group、ロール割当ては変更せず、横断運用を続ける利用者は制御面テナントの User へ移す。
- 後退：コードを戻せば以前の挙動へ戻るが、テナント横断の情報開示も再発するため、後退は安全な恒久対策にならない。

## Completion

- **Completed At**: 2026-09-04
- **Summary**:
  `mise run spec-diff` は `REQ-DATAKEYS-006`、`REQ-JOBS-012`、`REQ-JOBS-013`、`REQ-SIGNINGKEYS-008`、`REQ-SIGNINGKEYS-009` の変更を返した。制御面主体を、有効な `default` テナント所属 User が `default` テナントの経路で有効ロール `system_admin` を持つ場合に限定する共通判定を追加し、署名鍵と DEK の横断ヘルス、Tenancy、Audit、Jobs、フロントエンドの `/system` ガードへ適用した。拒否時は横断収集を開始せず、正規の制御面主体は `admin` を併有しなくても Jobs の横断一覧、詳細参照、取消しを行える。
- **Primary Use Case Evidence**:
  - id: control-plane-signing-key-health
    unit_red: "`TestRequireControlPlaneUser` は `IsControlPlaneActor` と `RequireControlPlaneUser` が未定義のためコンパイルに失敗した（build failed）。"
    e2e_red: "`TestControlPlaneSigningKeyHealthRejectsSystemAdminOutsideControlPlaneTenant` は拒否期待に対して HTTP 200 を受け、応答にテナント ID、提供元、`active_kid`、鍵数、到達性が現れ、横断列挙も 1 回実行された。"
    unit_fault_injection: "`IsControlPlaneActor` から所属テナント条件を除くと、所属先 `acme`、要求先 `default` の独立ケースが誤って許可され、`TestIsControlPlaneActor` が失敗した。"
    e2e_fault_injection: "署名鍵ヘルスの配線を `RequireControlPlaneUser` からロールだけを解決する `ResolveAdminActor` へ戻すと、HTTP 200、情報露出、横断列挙 1 回を `TestControlPlaneSigningKeyHealthRejectsSystemAdminOutsideControlPlaneTenant` が検出した。"
  - id: control-plane-data-key-health
    unit_red: "`TestRequireControlPlaneUserRejectsRequestOutsideControlPlaneTenant` は `RequireControlPlaneUser` が未定義のためコンパイルに失敗した（build failed）。"
    e2e_red: "`TestControlPlaneDataKeyHealthRejectsSystemAdminOutsideControlPlaneTenant` は拒否期待に対して HTTP 200 を受け、応答に `acme` の有効版、状態、提供元が現れ、横断列挙も 1 回実行された。"
    unit_fault_injection: "`IsControlPlaneActor` から要求先テナント条件を除くと、`default` 所属の主体が `acme` 経路で誤って許可され、`TestIsControlPlaneActor` が失敗した。"
    e2e_fault_injection: "DEK ヘルスの配線を `RequireControlPlaneUser` から `ResolveAdminActor` へ戻すと、HTTP 200、情報露出、横断列挙 1 回を `TestControlPlaneDataKeyHealthRejectsSystemAdminOutsideControlPlaneTenant` が検出した。"
  - id: control-plane-job-oversight
    unit_red: "`TestIsControlPlaneActor` は対象の純粋判定が未定義のためコンパイルに失敗した（build failed）。"
    e2e_red: "`TestControlPlaneJobOversightNeedsNoAdminRole` は `system_admin` だけを持つ正規の制御面主体に HTTP 403 を返し、横断ジョブ一覧へ到達できなかった。"
    unit_fault_injection: "`IsControlPlaneActor` から `system_admin` 条件を除くと `admin` だけの主体が誤って制御面主体になり、`TestIsControlPlaneActor` が失敗した。"
    e2e_fault_injection: "Jobs の一覧配線を制御面主体にも `admin` を要求する `RequireAdmin` へ戻すと、HTTP 403 を `TestControlPlaneJobOversightNeedsNoAdminRole` が検出した。"
- **Change-Resistance Results**:
  `IsControlPlaneActor` の変更した純粋論理について、所属先、要求先、`system_admin`、有効状態、nil 防御を一つずつ除く 5 件の変異を与え、すべて `TestIsControlPlaneActor` で検出した。最初の所属先変異は要求先との一致検査に覆われて生き残ったため、所属先 `acme` と要求先 `default` を分離したケースを追加してから同じ変異を再実行し、検出できることを確認した。Jobs では制御面の横断範囲を無効にする変異を E2E の詳細参照 404 で、通常管理者のロール拒否を無効にする変異を `TestListJobsRequiresAdminRole` の誤った HTTP 200 で検出した。フロントエンドでは realm 条件と有効ロール条件をそれぞれ常時真にする 2 件の変異を `-guards.test.ts` が検出した。署名鍵、DEK、Jobs の各 HTTP 配線を旧判定へ戻す 3 件の変異も、それぞれ主要 E2E テストが検出した。すべての変異を復元後、対象試験と標準検証が通過した。限界として、純粋判定の各条件と主要配線の故障モデルは系統的に変異させたが、自動変異試験器による全構文変異ではなく、効果を持つアダプターの全故障を列挙したものでもない。
- **Verification Results**:
  - `mise run verify` - passed (exit 0)
  - `mise run test-go-race` - passed
  - `mise run test-ui-unit` - 675 pass、0 fail
  - `mise run test-ui-e2e` - 25 pass、0 fail
  - `mise run typecheck-ui` - passed
  - `mise run lint-go` - passed（0 issues）
  - `mise run check-spec` - passed
  - `mise run check-api-compat` - passed
  - `mise run spec-render` - passed
  - `mise run check-security-controls` - passed
  - `mise run check-links` - passed
  - `mise run check-work-items` - passed
  - `mise run check-ids` - passed
  - `mise run spec-diff` - `REQ-DATAKEYS-006`、`REQ-JOBS-012`、`REQ-JOBS-013`、`REQ-SIGNINGKEYS-008`、`REQ-SIGNINGKEYS-009` を変更
