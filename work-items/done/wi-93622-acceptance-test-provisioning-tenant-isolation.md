---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-21
priority: p2
depends_on: []
change_kind: bugfix
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: 既存の越境参照の応答を HTTP の境界で固定するテストと fixture の追加であり、利用者から観測できる振る舞いと公開契約は変わらない。
  references: []
initial_context:
  specification:
    - docs/domain/provisioning/scenarios.feature.md#REQ-PROVISIONING-015
  typespec:
    - Contract.ProvisioningConnectionNotFoundError
    - Contract.ProvisioningDeliveryNotFoundError
  source:
    - backend/provisioning/handlers_http/handlers.go
    - backend/provisioning/handlers_http/routes.go
    - backend/provisioning/usecases/admin.go
    - backend/provisioning/db_memory/repositories.go
    - backend/shared/http/support_http/auth.go
    - backend/authentication/session/usecases/session_manager.go
    - backend/shared/http/testing_stack/stack.go
  tests:
    - backend/shared/http/server_http/provisioning_api_token_scope_test.go
    - backend/provisioning/db_postgres/repositories_test.go
    - tools/check/example-coverage-debt.json
  stop_before_reading:
    - frontend
    - backend/provisioning/client_scim
    - backend/provisioning/source_idmanagement
affected_spec:
  - { path: docs/domain/provisioning/scenarios.feature.md, requirement: REQ-PROVISIONING-015 }
primary_use_cases:
  - id: foreign-tenant-provisioning-read-is-not-found
    requirement: REQ-PROVISIONING-015
    observable_result: default テナントの接続と配信を acme の管理者が acme の realm で GetProvisioningConnection と GetProvisioningDelivery により参照すると、どちらも存在しない id と同じ 404 provisioning_not_found になり、接続先 URL と配信の内容は返らない。
    unit_test: { path: backend/provisioning/db_postgres/repositories_test.go, name: TestProvisioningTaskRepository_Find_ScopesToTenant, task: test-go-race }
    e2e_test: { path: backend/shared/http/server_http/provisioning_tenant_isolation_test.go, name: TestForeignTenantAdminSeesProvisioningAsMissing, task: test-go-race }
    unit_fault_model: 配信の保存先の検索がテナント条件を落とし、配信 id だけで別テナントの配信を返す。
    e2e_fault_model: ハンドラーが要求先テナントではなく固定のテナントで検索して別テナントの接続または配信を返すか、越境を存在しない id と区別できる応答で拒否する。接続と配信の一方だけが境界を守る実装も含む。
---

# テナントをまたぐ Provisioning 管理参照を公開 HTTP 境界で NotFound にする

## 動機

`EX-PROVISIONING-015-01` は tenant-b の管理者が tenant-a の接続または配信を参照した場合に NotFound を返すと宣言する。
リポジトリの tenant 分離テストはあるが、管理者の認証・テナント解決・HTTP 応答までを通した観測がない。

## 対象範囲

- tenant-a の Provisioning 接続と配信を、tenant-b の認証済み管理者が HTTP で参照する受け入れテストを追加する。
- 接続と配信の双方で NotFound と情報非開示を検証する。
- 現行の公開境界が別の結果を返す場合は、契約に合わせて修正する。

## 対象外

- API トークンのクロステナント拒否。[[wi-37560-return-access-denied-for-cross-tenant-provisioning-api-tokens]] が扱う。
- 他 Context のテナント境界テスト。
- 接続と配信の変更系 API 操作の越境。この規範は参照だけを述べる。

## 設計

この規範の actor は API トークンではなく tenant-b の管理者である。
テストは両テナントに管理者セッションを作り、正式な realm 入口を通す。保存層だけのテストは middleware や応答変換を迂回するため受け入れ証拠にしない。

| 要素 | 内容 |
| --- | --- |
| fixture | `testing_stack` に `(*Stack).AdminSessionCookie(t, realm) *http.Cookie` を足す。realm のテナントに `admin` ロールの利用者を置き、製品と同じ `SessionManager.Create` でセッションを作る。`AuthnResolver` が要るので `WithAuthorizationCodeFlow` を前提にする |
| tenant-a | default テナント。接続は保存先へ直接置き、配信は同じ接続に属するものを 1 件置く |
| tenant-b | acme テナント。同じ application id と delivery id で参照する |
| 対照 | default の管理者は同じ URL で 200 を得る。拒否が URL や配線の誤りではないことを示す |
| 情報非開示 | 越境の応答は、acme で存在しない id を参照したときの応答と状態コード、`type`、`detail` が一致し、接続先 URL と配信 id を本文に含まない |

テナント境界を担うのは、ハンドラーが渡す `support.RequestTenantID(c)`、`usecases.GetConnection(ctx, deps, tenantID, applicationID)` と `usecases.GetDelivery(ctx, deps, tenantID, applicationID, deliveryID)`、そして保存先の `Find(ctx, tenantID, …)` である。
型と作用の境界は変えない。

単体テストは本番の保存先である PostgreSQL 実装で、別テナントからの配信の `Find` が nil を返すことを固定する。接続の `Find` は `TestProvisioningConnectionRepository_RegisterFindDelete` が既に固定している。

## 計画

1. tenant ごとの管理者セッションを組み立てる既存 fixture を選ぶ。
2. 接続と配信の NotFound RED を確認する。
3. 必要な認証・テナント配線またはハンドラーを修正し検証する。

## タスク

- [x] T001 [Readiness] 複数 tenant の管理者セッション fixture を決める。
- [x] T002 [Acceptance] `EX-PROVISIONING-015-01` の接続・配信参照 RED を確認する。
- [x] T003 [Adapter] NotFound と情報非開示を実装または確認する。
- [x] T004 [Verify] 対象パッケージと `mise run verify` を実行する。

選んだ実行手順:

- RED、GREEN、障害注入: `mise run test-go-test -- ./backend/shared/http/server_http TestForeignTenantAdminSeesProvisioningAsMissing`
- 保存先: `mise run test-go-test -- ./backend/provisioning/db_postgres TestProvisioningDeliveryRepository_Find_ScopesToTenant`（embedded-postgres のためサンドボックス外）
- パッケージ: `mise run test-go-package -- ./backend/shared/http/server_http`

## 検証

- `mise run test-go-package -- ./backend/shared/http/server_http`
- `mise run check-spec`
- `mise run verify`

## リスク

別 tenant のリソースが見えるとテナント境界を破る。接続と配信を別々に観測し、一方だけのフィルタでは成立しないようにする。

## 完了

- **Completed At**: 2026-09-24
- **Summary**:
  `mise run spec-diff` は main に対して規範仕様の変更がないと報告した。
  `EX-PROVISIONING-015-01` を、default テナントの接続と配信を acme の管理者がセッションで参照する HTTP の境界テストで被覆した。越境の参照は接続と配信のどちらも、存在しない id と同じ 404 `provisioning_not_found` になり、接続先 URL と配信 id を返さない。
  現行実装は既に境界を守っていたため、製品コードは変更していない。
  `testing_stack` に、realm のテナントの管理者セッションを製品と同じ `SessionManager` で作る `AdminSessionCookie` を足した。PostgreSQL の配信の `Find` が別テナントで nil を返すことを単体テストで固定し、`tools/check/example-coverage-debt.json` から項目を削除した。
- **Primary Use Case Evidence**:
  - id: foreign-tenant-provisioning-read-is-not-found
    unit_red: "N/A: 保存先は変更前から要求先テナントで絞り込んでおり、TestProvisioningDeliveryRepository_Find_ScopesToTenant は書いた時点で GREEN だった。代替検査として、境界テストを外して被覆負債を削除した状態の `mise run check-spec` が `EX-PROVISIONING-015-01 is declared, but no test names it` で失敗した。"
    e2e_red: "N/A: 越境の参照は変更前から 404 を返しており、TestForeignTenantAdminSeesProvisioningAsMissing は書いた時点で GREEN だった。失敗の観測は上の `mise run check-spec` と、下の障害注入による。"
    unit_fault_injection: 生成済みクエリ `FindProvisioningDelivery` の条件を `(tenant_id=$1 OR TRUE)` に変えると、単体テストが別テナントからの検索で配信が返ることを検出して失敗した。
    e2e_fault_injection: "`handleGetConnection` の検索テナントを既定テナントに固定すると接続のサブテストだけが 200 と接続先 URL を観測して失敗し、`handleGetDelivery` を同様に固定すると配信のサブテストだけが失敗した。メモリ保存先の配信検索がテナントを無視したうえで、越境を 400 `invalid_request` で拒否する故障と、同じ 404 で `detail` だけを変える故障も、配信のサブテストが検出した。"
- **Change-Resistance Results**:
  - 障害注入は上の 5 件で、すべて検出した。注入は作業ツリーで行い、元に戻したことを `git diff --stat` で確かめた。
  - `mise run test-go-mutation` は実行していない。変更した Go はテストと試験用 fixture だけであり、変異させる製品コードがない。
  - PostgreSQL を使う単体テストはサンドボックス内では embedded-postgres を起動できないため、サンドボックス外で実行した。
- **Verification Results**:
  - `mise run check-spec` - passed（境界テストを外した状態では 1 件の被覆欠落で failed）
  - `mise run test-go-test -- ./backend/provisioning/db_postgres TestProvisioningDeliveryRepository_Find_ScopesToTenant` - passed（サンドボックス外）
  - `mise run test-go-package -- ./backend/shared/http/server_http` - passed
  - `mise run lint-go` - 0 issues
  - `mise run verify` - passed
  - `mise run test-ui-e2e` - 未実行。変更はテストと試験用 fixture だけで、ブラウザーへ届かない。
- **Left Undone**:
  - 接続と配信の変更系 API 操作の越境は、この規範の範囲外として扱っていない。
