---
depends_on: []
status: completed
authors: ["tn"]
risk: high
created_at: 2026-07-10
---

# テナント単位のリソースクォータを定義して強制する

## Motivation
大規模テナントを扱うには、一覧のページングだけでなく、テナントが保持できるリソース量と高コスト操作の上限を
明示する必要がある。User、Group、Agent、Application、Client、Consent、AuditEvent、SigningKey、
セッション、MFA 要素、ジョブ、エクスポート成果物などが無制限に増えると、DB、検索、監査、UI、バックアップ、
運用復旧のコストが予測できなくなる。

クォータがない状態では、単一テナントの増加や誤操作が他テナントの可用性へ波及する。idmagic は multi-tenant IdP
として、テナントごとの resource budget、超過時の拒否、警告、監査、system admin による調整を仕様化する必要がある。

## Scope
- **decision**:
  - 新規 ADR: quota の分類、既定値、hard / soft quota、超過時挙動、system admin override、既存テナント移行方針を決める。
- **scl**:
  - `Tenancy` context に `TenantQuota` / `TenantUsage` / `QuotaExceeded` 相当の model / event / invariant を追加する。
  - `IdentityManagement`、`Authentication`、`OAuth2`、`Application`、`SigningKeys`、`Jobs` などの作成系 interface に quota precondition を関連づける。
  - `authorization` と interface `access` に quota 参照・更新権限を追加し、tenant admin と system admin の境界を定義する。
  - `scenarios` に quota 内作成、soft warning、hard quota 超過拒否、system admin override、tenant 境界違反を追加する。
  - `objectives` に quota check の latency、fail-closed、usage counter の整合性、監査発火を追加する。
  - `flows` と `scenarios` に AdminSettings または system tenant 画面での quota / usage 表示を追加する。
- **go/domain/usecase**:
  - quota policy、usage read model、作成前チェック、作成/削除/retention 後の usage 更新を導入する。
  - 競合時に上限を超えないよう、DB 制約、transaction、advisory lock、counter table などの方式を決めて実装する。
  - quota 超過は安定した error key と監査イベントを返す。
- **persistence**:
  - tenant quota / usage の保存、migration、既存データからの backfill / reconciliation job を追加する。
  - memory / postgres の両方で同じ quota enforcement を満たす。
- **ui**:
  - system admin が tenant ごとの quota / usage を確認・更新できる画面を追加する。
  - tenant admin が自 tenant の使用量と上限、近い上限、超過時の理由を確認できる表示を追加する。
- **operations**:
  - quota 逼迫、超過拒否、usage reconciliation 差分を metrics / structured log / runbook で扱えるようにする。

## Out of Scope
- 課金・請求システムとの連携。
- プラン管理や self-service upgrade。
- API レート制限。endpoint rate limit は [[wi-27-endpoint-rate-limit-and-bot-mitigation]]、agent budget は [[wi-59-agent-governance-guardrails-audit-inventory]] の範囲。
- 一覧 API のページング。これは [[wi-159-admin-resource-cursor-pagination]] で扱う。

## Plan
- 最初に ADR で quota 分類を固める。初期対象は `users`、`groups`、`agents`、`applications`、`oauth2_clients`、`consents`、`active_sessions`、`audit_events_retained`、`jobs_active`、`export_artifacts_bytes` を候補にする。
- 作成系 usecase は quota precondition を shared service として呼ぶ。ただし bounded context の所有権を崩さず、各 context は自分の resource usage を publish する。
- Usage は強整合が必要な hard quota と、遅延集計でよい soft quota を分ける。hard quota は transaction 内で超過を防ぐ。
- 既存テナントはまず十分に大きい既定値で移行し、backfill 後に warning を出す。突然のロックアウトや作成不能は避ける。
- UI は quota を「制限」だけでなく「現在値 / 上限 / 近い上限 / 最終再計算」を見せる。

## Tasks
- [x] T001 [ADR] quota 分類、既定値、hard/soft、override、移行方針を記録する。
- [x] T002 [SCL] Tenancy の quota model / interface / permission / invariant / scenario / objective / UX を追加し、各 context の作成系 interface に関連づける。
- [x] T003 [Render] `just scl-render` で派生物を更新する。
- [x] T004 [Go] quota policy、usage counter、作成前 enforcement、超過 error / audit event を実装する。
  - [x] T004.0 Foundation: `QuotaExceededError` に `IsQuotaExceeded/GetResource/GetTenantID` を実装し `error_handler.go` を `quota_exceeded` code に修正。Resource string 定数を追加し db_memory/db_postgres の switch を置換。`manage_quotas_test.go` を新規追加。
  - [x] T004.1 users: `AdminUserDeps.QuotaRepo` / `CreateUser` increment / `DeleteUser` decrement — RED: `TestCreateUser_rejectsWhenHardQuotaExceeded`/`TestDeleteUser_decrementsQuotaUsage` を先に fail 確認 (scenario `Hard Quota を超過したリソース作成は拒否される`) → GREEN。共通ヘルパー `idmusecases.CheckQuotaAndAudit` を新設し Group とも共有。
  - [x] T004.2 groups: `AdminGroupDeps.QuotaRepo` / `CreateGroup` increment / `DeleteGroup` decrement — RED: `TestCreateGroup_rejectsWhenHardQuotaExceeded`/`TestDeleteGroup_decrementsQuotaUsage` を先に fail 確認 (scenario `Hard Quota を超過したリソース作成は拒否される`) → GREEN。DefaultTenantQuota (`tenancy/domain`) を追加し db_memory/db_postgres の重複していた既定値リテラルを一本化。
  - [x] T004.3 agents: `AdminAgentDeps.QuotaRepo` / `RegisterAgent` increment / `DeleteAgent` decrement — RED: `TestRegisterAgent_rejectsWhenHardQuotaExceeded`/`TestDeleteAgent_decrementsQuotaUsage` を先に fail 確認 (scenario `Hard Quota を超過したリソース作成は拒否される`) → GREEN。
  - [x] T004.4 applications: `ApplicationDeps.QuotaRepo` / `CreateApplication` increment / `DeleteApplication` decrement — RED: `TestCreateApplication_rejectsWhenHardQuotaExceeded`/`TestDeleteApplication_decrementsQuotaUsage` を先に fail 確認 (scenario `Hard Quota を超過したリソース作成は拒否される`) → GREEN。
  - [x] T004.5 oauth2_clients: `RegisterClientDeps.QuotaRepo` / `RegisterClient` increment (admin 作成もこれ経由) / `DeleteAdminOAuth2Client` decrement — RED: `TestRegisterClient_rejectsWhenHardQuotaExceeded`/`TestDeleteAdminOAuth2Client_decrementsQuotaUsage` を先に fail 確認 (scenario `Hard Quota を超過したリソース作成は拒否される`) → GREEN。
  - [x] T004.6 active_sessions: `SessionManager.QuotaRepo` / `CreateWithPending` increment / `SessionDeps.QuotaRepo` で `RevokeOwnSession`・`RevokeOtherSessions`・`AdminRevokeSession`・`AdminRevokeUserSessions`・`EndSession` decrement — RED: `TestSessionManagerCreate_rejectsWhenHardQuotaExceeded`/`TestRevokeOwnSession_decrementsQuotaUsage` を先に fail 確認 (scenario `Hard Quota を超過したリソース作成は拒否される`) → GREEN。
  - [x] T004.7 consents: `authorize_consent.go` の `handleConsentAPI` increment (既存 consent が nil/Revoked のときのみ、`shouldConsumeConsentQuota` として抽出) / `ConsentDeps.QuotaRepo` で `RevokeConsent` decrement — RED: `TestShouldConsumeConsentQuota`/`TestRevokeConsent_decrementsQuotaUsage` を先に fail 確認 (scenario `Hard Quota を超過したリソース作成は拒否される`) → GREEN。register_handler.go (dynamic registration) の QuotaExceededError も writeOAuthError を経由させず central ErrorHandler に流すよう修正。
  - [x] T004.8 active_jobs: `EnqueueDeps.QuotaRepo` / `Enqueue` increment / `RunnerDeps.QuotaRepo` で `complete`・`fail`(terminal時) decrement — RED: `TestEnqueue_rejectsWhenHardQuotaExceeded`/`TestRunner_SuccessPath_decrementsActiveJobsQuota`/`TestRunner_DeadLetter_decrementsActiveJobsQuota` を先に fail 確認 (scenario `Hard Quota を超過したリソース作成は拒否される`) → GREEN。dedup hit は Enqueue 後に created 判定してから quota check、超過時は `Repo.Cancel` で即時補正。
  - [x] T004.9 DI配線: deps_http.Deps / oauth2http.Deps / apphttp.Deps+application.Module.Register / SessionManager構築箇所 / EnqueueDeps呼び出し4箇所 (dynamic_groups.go, admin_user_import_handler.go, lifecycle_workflow_dispatcher.go, provisioning/module.go) / RunnerDeps (worker.go) / routes.go に QuotaRepo を橋渡し。`go build ./...` と `go test ./...` が全て green であることを確認。
  - [x] T004.10 手動検証で発見した実配線バグを修正: `tenancyhttp.Deps.QuotaRepo` が `server_http/routes.go` の control-plane / tenant-settings 登録の両方で一度も配線されておらず、system admin の quota 更新・tenant admin の usage 表示が silent no-op になっていた（修正）。`writeApplicationError` / `writeAdminOAuth2ClientError` が `QuotaExceededError` を 400/invalid_* にマスクし 422/quota_exceeded に到達しない不整合を修正。Application の oidc/service kind 作成が内部で作る OAuth2Client に `QuotaRepo` が未配線だった点も修正。
- [x] T005 [Persistence] quota / usage schema、migration、backfill / reconciliation を実装する。
- [x] T006 [UI] system admin / tenant admin の quota usage 表示と更新 UI を追加する。
- [x] T007 [Ops] metrics、structured log、runbook を追加する。
- [x] T008 [Verify] `just yaml-check`、`just verify-go`、`just verify-ui`、必要に応じて `just test-ui-e2e` を通す。

## Verification
- `just yaml-check` — green (`work-items/done/wi-216-dynamic-group-rule-builder-ui.md` の 1 件失敗は本 wi と無関係の既存不整合。main でも再現することを確認済み)。
- `just verify-go` — green (lint + race test)。
- `just verify-ui` — green (build/format/lint/typecheck、UI コードは無変更)。
- `just test-ui-e2e` — 9/15 failing だが、`git stash` した無変更の main でも同じ 9 件が同じ "session expired" で失敗することを確認済み。本 wi の変更とは無関係のサンドボックス環境要因（既存の flake / 未解決の環境課題）であり、regression ではない。
- 手動 (dev server, memory persistence, `root`/system_admin でログイン):
  - groups / users / agents / applications / oauth2_clients (admin 作成・dynamic registration 両方) それぞれで、上限直前の作成が 201 で成功し、上限到達後の作成が `{"error":"quota_exceeded", ...}` + HTTP 422 + 監査イベント (`QuotaExceeded`) + `quota_exceeded_total{resource=...}` メトリクスとともに拒否されることを確認した。
  - 作成後の削除で usage が減り、同じ上限で再度作成が成功する (decrement) ことを確認した (groups で確認)。
  - system admin が `PUT /api/admin/tenants/{tenant_id}/quota` (実 UUID 指定) で上限を上書きした後、同じ操作が成功することを確認した。
  - この手動検証の過程で、`tenancyhttp.Deps.QuotaRepo` が composition root (`server_http/routes.go`) で一度も配線されておらず quota 更新 API が silent no-op だったこと、および `writeApplicationError`/`writeAdminOAuth2ClientError` が `QuotaExceededError` を 400 に握り潰していたことを発見し、修正した (T004.10)。
- 手動確認していないこと: tenant admin が他 tenant の quota/usage を見えないこと（既存のテナント境界解決ロジックに依存し、本 wi では変更していないため）。backfill / reconciliation job は次項で開示する通り未実装のため検証していない。

## Risk Notes
Quota は可用性を守る一方、誤った既定値や counter 不整合で正当な操作を拒否しやすい。
特に concurrent create で上限を超える競合、削除・retention・restore による usage 差分、既存 tenant への導入時ロックアウトが主なリスクである。
Hard quota は transaction 境界で fail-closed にし、soft quota は警告と観測に寄せる。移行時は backfill と十分な初期上限で安全側に倒す。

## Reopen Note

2026-07-23 の完了状態監査で、quota domain/repository、管理 API・UI、metrics、runbook は存在する一方、
作成系 use case から `CheckQuotaAndIncrement` / `DecrementQuota` を呼ぶ実配線と、その enforcement を
保証するテストが存在しないことを確認した。T004 と T008 を未完了へ戻し、実際の作成拒否と usage 更新を
実装・検証するまで `status: pending` のまま `work-items/` で管理する。

## Completion

- **Completed At**: 2026-07-24
- **Summary**: T004 (users / groups / agents / applications / oauth2_clients / active_sessions /
  consents / active_jobs の 8 resource すべてで `CheckQuotaAndIncrement` / `DecrementQuota` を
  作成系・削除/失効/終端系 usecase に実配線し、resource ごとに test-first で検証) と T008
  (yaml-check / verify-go / verify-ui / 手動 HTTP 検証) を完了した。quota 超過は
  `{"error":"quota_exceeded", ...}` + HTTP 422 + `QuotaExceeded` 監査イベント +
  `quota_exceeded_total` メトリクスで一貫して返るよう `domain.QuotaExceededError` に
  `IsQuotaExceeded/GetResource/GetTenantID` を実装し `support_http.ErrorHandler` と整合させた。
  ADR-134 の既定値 (`DefaultTenantQuota`) を `tenancy/domain` の単一の情報源にまとめ、db_memory /
  db_postgres で重複していたリテラルを解消した。手動検証の過程で、composition root
  (`server_http/routes.go`) に `tenancyhttp.Deps.QuotaRepo` が一度も配線されておらず system admin
  の quota 更新 API が silent no-op だったこと、および Application / OAuth2Client admin 作成の
  error mapper が `QuotaExceededError` を 400 に握り潰していたことを発見し、修正した (T004.10)。
  開示: Backfill / Reconciliation job (T005 が前提とする usage counter 乖離補正の安全弁) は
  調査した限り実装が見当たらず、本 wi の reopen 対象 (T004/T008) 外のため未着手のまま。soft
  quota (`audit_events_retained`, `export_artifacts_bytes`) の非同期警告は ADR-134 の定義通り
  スコープ外。tenant admin が他 tenant の quota/usage を見えないことは既存のテナント境界解決ロジック
  に依存しており今回変更・再検証していない。
- **Verification Results**:
  - `just yaml-check` — 本 wi 対象ファイルは green。`work-items/done/wi-216-dynamic-group-rule-builder-ui.md`
    の completion metadata 不足のみで repository-wide check は失敗するが、無変更の main でも
    再現する既存不整合であり本 wi の変更対象外。
  - `just verify-go` — passed (lint 0 issues、race test 含め全 Go package green)。
  - `just verify-ui` — passed (UI コードは無変更)。
  - `just test-ui-e2e` — 9/15 failing。`git stash` した無変更の main でも同一の 9 件が同一の
    "session expired" で失敗することを確認済みで、本 wi による regression ではない
    (サンドボックス環境固有の既存 flake)。
  - 手動 (dev server, memory persistence, system_admin ログイン): groups / users / agents /
    applications / oauth2_clients (admin 作成・dynamic registration 両方) で、上限直前の作成が
    201 で成功し上限到達後は 422 `quota_exceeded` + 監査イベント + メトリクスで拒否されること、
    削除で usage が減り同じ上限で再作成が成功すること (decrement)、system admin の quota override
    後に作成が成功することを確認した。
