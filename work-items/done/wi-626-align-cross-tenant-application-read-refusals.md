---
depends_on: []
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-19
priority: p2
change_kind: bugfix
evidence_policy: risk-based-v3
documentation_impact:
  level: release_note
  reason: Application 管理 API 18 件が実際に返していた 404 application_not_found と category_not_found を、生成クライアントが型付きの応答として扱えるようになる。応答そのものは変わらない。
  references:
    - { kind: release_note, path: docs/releases/changes/wi-626-align-cross-tenant-application-read-refusals.md }
initial_context:
  specification:
    - docs/domain/application/scenarios.feature.md#REQ-APPLICATION-007
    - docs/domain/application/scenarios.feature.md#REQ-APPLICATION-008
    - docs/design/security/authorization.md
  typespec:
    - IdMagic.Application.Operations.GetAdminApplication
    - IdMagic.Application.Operations.GetApplicationIcon
  source:
    - backend/application/handlers_http/admin_application_handler.go
    - backend/shared/http/support_http/tenant_middleware.go
    - backend/shared/http/support_http/auth.go
    - backend/application/db_postgres/applications.go
    - backend/application/handlers_http/admin_category_handler.go
    - backend/application/handlers_http/application_provisioning.go
  tests:
    - backend/application/handlers_http/catalog_examples_test.go
    - backend/application/handlers_http/application_handler_test.go
    - backend/application/handlers_http/client_secret_refusal_effects_test.go
    - backend/application/handlers_http/not_found_statuses_test.go
    - backend/application/db_postgres/applications_test.go
    - tools/check/example-coverage-debt.json
  stop_before_reading:
    - frontend
    - backend/application/usecases
affected_spec:
  - { path: docs/domain/application/scenarios.feature.md, requirement: REQ-APPLICATION-007 }
  - { path: docs/domain/application/scenarios.feature.md, requirement: REQ-APPLICATION-008 }
  - { path: spec/contexts/application/main.tsp, symbol: IdMagic.Application.Operations.GetAdminApplication }
  - { path: spec/contexts/application/main.tsp, symbol: IdMagic.Application.Operations.GetApplicationIcon }
  - { path: spec/contexts/application/main.tsp, symbol: IdMagic.Application.Operations.UpdateAdminApplication }
  - { path: spec/contexts/application/main.tsp, symbol: IdMagic.Application.Operations.DeleteAdminApplication }
  - { path: spec/contexts/application/main.tsp, symbol: IdMagic.Application.Operations.UploadApplicationIcon }
  - { path: spec/contexts/application/main.tsp, symbol: IdMagic.Application.Operations.DeleteApplicationIcon }
  - { path: spec/contexts/application/main.tsp, symbol: IdMagic.Application.Operations.UpdateApplicationOidcConfig }
  - { path: spec/contexts/application/main.tsp, symbol: IdMagic.Application.Operations.RotateApplicationClientSecret }
  - { path: spec/contexts/application/main.tsp, symbol: IdMagic.Application.Operations.IssueApplicationClientSecret }
  - { path: spec/contexts/application/main.tsp, symbol: IdMagic.Application.Operations.RevokeApplicationClientSecret }
  - { path: spec/contexts/application/main.tsp, symbol: IdMagic.Application.Operations.UpdateApplicationWsFedConfig }
  - { path: spec/contexts/application/main.tsp, symbol: IdMagic.Application.Operations.UpdateApplicationSamlConfig }
  - { path: spec/contexts/application/main.tsp, symbol: IdMagic.Application.Operations.ListApplicationAssignments }
  - { path: spec/contexts/application/main.tsp, symbol: IdMagic.Application.Operations.AssignApplication }
  - { path: spec/contexts/application/main.tsp, symbol: IdMagic.Application.Operations.GetAppSignInPolicy }
  - { path: spec/contexts/application/main.tsp, symbol: IdMagic.Application.Operations.UpdateAppSignInPolicy }
  - { path: spec/contexts/application/main.tsp, symbol: IdMagic.Application.Operations.SetApplicationCategories }
  - { path: spec/contexts/application/main.tsp, symbol: IdMagic.Application.Operations.UpdateApplicationCategory }
  - { path: spec/contexts/application/main.tsp, symbol: IdMagic.Application.Operations.DeleteApplicationCategory }
primary_use_cases:
  - id: foreign-tenant-application-read-is-not-found
    requirement: REQ-APPLICATION-007
    observable_result: default テナントで作成した Application の id を別テナントの管理者が GetAdminApplication で取得すると、存在しない id と同じ 404 application_not_found になり、名称と OIDC 設定は返らない。
    unit_test: { path: backend/application/db_postgres/applications_test.go, name: TestApplicationRepositoryFindByIDIsScopedToTenant, task: test-go-race }
    e2e_test: { path: backend/application/handlers_http/catalog_examples_test.go, name: TestGetAdminApplicationFromAnotherTenantIsNotFound, task: test-go-race }
    unit_fault_model: 保存先の検索がテナント条件を落とし、Application id だけで別テナントの Application を返す。
    e2e_fault_model: 越境を存在しない id と区別できる状態コードまたは本文で拒否し、別テナントに同じ id があることを推測させる。
  - id: foreign-tenant-application-icon-is-not-found
    requirement: REQ-APPLICATION-008
    observable_result: default テナントでアップロードしたアイコンの application_id と id を別テナントの realm で取得すると、存在しない id と同じ 404 not_found になり、画像のバイト列は返らない。
    unit_test: { path: backend/application/db_postgres/applications_test.go, name: TestApplicationIconStoreFindIsScopedToTenant, task: test-go-race }
    e2e_test: { path: backend/application/handlers_http/catalog_examples_test.go, name: TestGetApplicationIconFromAnotherTenantIsNotFound, task: test-go-race }
    unit_fault_model: アイコンの保存先を要求先テナントを落として検索し、別テナントの画像を返す。
    e2e_fault_model: 越境を存在しない id と区別できる応答で拒否する。
  - id: foreign-tenant-application-admin-operations-are-not-found
    requirement: REQ-APPLICATION-007
    observable_result: default テナントの Application を別テナントの管理者が更新、削除、アイコン、プロトコル設定、資格情報、割り当て、サインイン方針、カテゴリー設定の各 API 操作で指定すると、どれも存在しない id と同じ 404 application_not_found になり、default テナントの Application は変わらない。
    unit_test: { path: backend/application/db_postgres/applications_test.go, name: TestApplicationRepositoryFindByIDIsScopedToTenant, task: test-go-race }
    e2e_test: { path: backend/application/handlers_http/not_found_statuses_test.go, name: TestApplicationAdminOperationsAnswerAnotherTenantsApplicationAsMissing, task: test-go-race }
    unit_fault_model: 保存先の検索がテナント条件を落とし、Application id だけで別テナントの Application を返す。
    e2e_fault_model: エラーヘルパーが application_not_found を宣言と異なる状態コードで書くか、Application の検索がテナントを無視して別テナントの Application を変更する。
  - id: foreign-tenant-application-category-is-not-found
    requirement: REQ-APPLICATION-007
    observable_result: default テナントのカテゴリーを別テナントの管理者が更新または削除すると、存在しない id と同じ 404 category_not_found になり、default テナントのカテゴリーは変わらない。
    unit_test: { path: backend/application/db_postgres/applications_test.go, name: TestApplicationCategoryRepositoryFindByIDIsScopedToTenant, task: test-go-race }
    e2e_test: { path: backend/application/handlers_http/not_found_statuses_test.go, name: TestApplicationCategoryOperationsAnswerAnotherTenantsCategoryAsMissing, task: test-go-race }
    unit_fault_model: カテゴリーの検索がテナント条件を落とし、別テナントのカテゴリーを返す。
    e2e_fault_model: エラーヘルパーが category_not_found を宣言と異なる状態コードで書くか、カテゴリーの検索がテナントを無視して別テナントのカテゴリーを変更する。
---

# テナントをまたぐ Application 参照の拒否契約を実装と一致させる

## Motivation

`EX-APPLICATION-007-03` は、別テナントの管理者が同じ Application id を指定した参照を `InvalidRequestError` で拒否すると定める。

`handleGetApplication` は解決済みテナントを検索キーに含め、見つからない場合は存在を隠す `404 application_not_found` を返す。

`EX-APPLICATION-008-03` も同様に、別テナントの `application_id` と id でのアイコン取得を `InvalidRequestError` で拒否すると定める。

`handleGetApplicationIcon` は `404 not_found` を返す。

テナント境界の拒否そのものはどちらも成立しており、越境した内容は返らない。

食い違うのは公開するエラー契約である。

さらに `GetAdminApplication` の TypeSpec は 200、400、401、403 だけを宣言しており、実装が返す 404 を宣言していない。

一方 `GetApplicationIcon` は 404 `ApplicationIconNotFoundError` を宣言しているため、こちらは具体例だけが実装と契約の双方から外れている。

## Scope

- 越境した Application 参照とアイコン取得について、`InvalidRequestError` と存在を隠す `404` のどちらを公開契約にするか決める。
- 決定に従い、`EX-APPLICATION-007-03`、`EX-APPLICATION-008-03`、TypeSpec、実装、テストを同じ契約へそろえる。
- `GetAdminApplication` が返す 404 を契約へ反映するか、実装を宣言済みの状態へ寄せるかを決める。
- 拒否応答に加え、別テナントの Application の名称、設定、アイコン内容が返らないことを検証する。
- `GetAdminApplication` 以外の Application 管理 API が実際に返す未宣言の 404（`application_not_found`、`category_not_found`）を TypeSpec へ宣言し、存在しない対象と別テナントの対象が同じ応答になることを HTTP の境界で固定する。

## Out of Scope

- Application 以外の Context が持つ越境参照のエラー契約。同じ判断は [[wi-564-name-the-refusal-a-cross-tenant-oauth-admin-token-gets]] と [[wi-573-agent-admin-api-answers-with-undeclared-statuses]] が別途扱う。
- 割り当ての主体を検査しない欠落。[[wi-625-application-assignment-accepts-any-subject]] が扱う。
- アイコンの保存形式、配信 URL の形式、キャッシュ方針の変更。
- 404 以外の未宣言の応答。資格情報、WS-Federation、SAML の各 API 操作が 400 で返す拒否は、宣言済みの 400 に含まれる。
- `UnassignApplication` の応答。存在しない Application に対しても冪等に 204 を返し、別テナントの Application と区別できないため、宣言を変えない。

## Design

判断点は、存在を隠すテナント境界の原則を優先して `404` を規範へ反映するか、既存の `InvalidRequestError` の字面を優先して実装を変えるかである。

前者は現行実装と情報非開示の意図を保てる。

後者は具体例の字面を保てるが、別テナントに同じ id があるかを推測させない応答であることを別途確認する必要がある。

`GetApplicationIcon` は既に契約と実装が 404 で一致しているため、規範だけを動かす選択に対して費用が低い。

`GetAdminApplication` は契約が 404 を宣言していないため、どちらを選んでも TypeSpec を変更する。

この判断は公開契約を変えるため、実装前に決定する。

前者を採る。

- `docs/design/security/authorization.md` は、別テナントのリソースを存在しないものとして扱い、権限のない対象と存在しない対象を同じレスポンスにすると定める。
- 2 つの具体例と TypeSpec の doc は、既に「存在しないものとして扱う」と述べており、`InvalidRequestError` の字面はこれと矛盾する。
- 同じ判断を tenancy の branding アセット（`EX-TENANCY-004-02`）と Agent 管理 API が採っており、どちらも存在しない id と同じ 404 を規範にした。

| 対象 | 変更 |
| --- | --- |
| `EX-APPLICATION-007-03` | 存在しない id と同じ 404 `application_not_found` で応答し、名称と OIDC 設定を含まないと述べる |
| `EX-APPLICATION-008-03` | 存在しない id と同じ 404 `not_found` で応答し、画像の内容を含まないと述べる |
| `GetAdminApplication` | 404 `ApplicationNotFoundError`（`type` = `urn:idmagic:error:application_not_found`）を宣言する |
| `GetApplicationIcon` | 変更しない。既に 404 `ApplicationIconNotFoundError` を宣言している |
| handler | 変更しない。既に両方とも要求先テナントで検索し、見つからなければ 404 を返す |
| `tools/check/example-coverage-debt.json` | 2 件の項目を削除する |

ドメインの型と作用の境界は変えない。
テナント境界を担うのは、handler が渡す `support.RequestTenantID(c)` と、`ApplicationRepo.FindByID(ctx, tenantID, id)` および `ApplicationIconStore.Find(ctx, tenantID, applicationID, id)` である。

単体テストは本番の保存先である PostgreSQL 実装で、別テナントの `FindByID` と `Find` が nil を返すことを固定する。

E2E テストは `httpadapter.Register` で組み立てた router に default と globex の 2 テナントを登録する。
default の管理者が作成した Application を、globex の管理者が globex の realm で取得する。
応答が存在しない id の応答とステータス、`Content-Type`、本文ともに一致し、名称（`redirect_uris` の値も同じ語を含む）、Application id、アイコンのバイト列を含まないことを表明する。
同じ realm では取得できることを先に確かめ、拒否が URL の誤りではないことを示す。
アイコン取得は認証を要求しない公開 API なので、globex の realm の URL へそのまま要求する。

### 残りの Application 管理 API の 404

存在しない id を `:id` を取る各 API 操作へ送って観測すると、次の API 操作が未宣言の 404 を返した。
`writeApplicationError` と `writeCategoryError` は `mise run check-status-drift` が追跡しないヘルパーなので、検査は通っていた。

| 追加する宣言 | API 操作 |
| --- | --- |
| 404 `ApplicationNotFoundError` | UpdateAdminApplication、DeleteAdminApplication、UploadApplicationIcon、DeleteApplicationIcon、RotateApplicationClientSecret、IssueApplicationClientSecret、RevokeApplicationClientSecret、ListApplicationAssignments、AssignApplication、GetAppSignInPolicy、UpdateAppSignInPolicy、SetApplicationCategories |
| 既存の 404 の union に `ApplicationNotFoundError` | UpdateApplicationOidcConfig、UpdateApplicationWsFedConfig、UpdateApplicationSamlConfig |
| 404 `ApplicationCategoryNotFoundError`（`type` = `urn:idmagic:error:category_not_found`） | UpdateApplicationCategory、DeleteApplicationCategory |

実装は変えない。
境界テストは、default テナントの Application とカテゴリーを globex の管理者が各 API 操作で指定した応答が、存在しない id の応答と一致する 404 であることを表明する。
変更系の API 操作を試した後に、default テナントの Application とカテゴリーが変わっていないことも表明する。

## Plan

1. 同種のテナント境界拒否がほかの Context で採る契約を確認する。
2. 2 つの具体例と TypeSpec の応答契約を決定する。
3. 正式な HTTP 入口から越境参照を試すテストを RED にする。
4. 必要な仕様と実装をそろえ、拒否応答と情報非開示を検証する。

## Tasks

- [x] T001 [Decision] 越境した Application 参照とアイコン取得の公開エラー契約を決める。
- [x] T002 [Spec] `EX-APPLICATION-007-03`、`EX-APPLICATION-008-03`、TypeSpec を決定済みの契約へそろえる。
- [x] T003 [Acceptance] HTTP 入口で拒否応答と、Application 設定およびアイコン内容の不在を検証する。
- [x] T004 [App] 必要な場合は handler の応答写像を変更する。
- [x] T005 [Verify] 仕様、契約、テナント境界の検査を通し、2 件を被覆台帳から外す。
- [x] T006 [Spec] 残りの Application 管理 API 17 件に、実際に返す 404 を宣言する。
- [x] T007 [Acceptance] 17 件の 404 と、別テナントの対象が存在しない対象と同じ応答になることを HTTP の境界で固定する。

## Verification

- `mise run check-spec`
- `mise run check-contract-drift`
- `mise run check-api-compat`
- `mise run test-go-package -- ./backend/application/handlers_http`
- `mise run verify`

## Risk Notes

テナント境界の拒否を誤ると、別テナントの Application の存在または内容が漏れる。

ステータスとエラー型だけでなく、応答本文に対象の名称、プロトコル設定、アイコンのバイト列が含まれないことを観測する。

契約を `InvalidRequestError` へ寄せる選択は、id の存在を推測できる応答になりやすい。

その場合は、存在する id と存在しない id が同じ応答を返すことを観測に含める。

## Completion

- **Completed At**: 2026-09-24
- **Summary**:
  `mise run spec-diff` は REQ-APPLICATION-007 と REQ-APPLICATION-008 のシナリオ変更と、TypeSpec 宣言 `ApplicationNotFoundError` と `ApplicationCategoryNotFoundError` の追加を報告した。
  `EX-APPLICATION-007-03` と `EX-APPLICATION-008-03` は、別テナントの id による取得を `InvalidRequestError` で拒否するという記述から、存在しない id を指定したときと同じ 404（`application_not_found`、`not_found`）で応答し、名称、OIDC 設定、画像の内容を含まないという記述へ変わった。
  Application 管理 API 18 件に、実際に返していた 404 を宣言した。Application を指定する 16 件は `ApplicationNotFoundError`、カテゴリーを指定する 2 件は `ApplicationCategoryNotFoundError` である。
  製品コードは変更していない。`tools/check/example-coverage-debt.json` から 2 件の項目を削除し、API ガイドラインの適用状況とリリースノートを追記した。
- **Primary Use Case Evidence**:
  - id: foreign-tenant-application-read-is-not-found
    unit_red: 保存先は変更前から要求先テナントで絞り込んでおり、TestApplicationRepositoryFindByIDIsScopedToTenant は書いた時点で GREEN だった。実装前に失敗を観測した代替検査は、例の被覆負債を削除した後の `mise run check-spec` であり、`EX-APPLICATION-007-03 is declared, but no test names it` で失敗した。
    e2e_red: TestGetAdminApplicationFromAnotherTenantIsNotFound は旧具体例の 400 を期待した段階で、実際の 404 `application_not_found` を観測して失敗した。
    unit_fault_injection: 生成済みクエリ `GetApplicationByID` の条件を `(tenant_id = $1 OR TRUE)` に変えると、単体テストが別テナントからの検索で Application が返ることを検出して失敗した。
    e2e_fault_injection: メモリ保存先の `FindByID` がテナントを無視すると、別テナントの取得が 200 で名称と設定を返し、E2E テストが失敗した。越境だけを 400 `invalid_request` で返す故障と、同じ 404 でも `detail` だけを変える故障も検出した。
  - id: foreign-tenant-application-icon-is-not-found
    unit_red: TestApplicationIconStoreFindIsScopedToTenant は書いた時点で GREEN だった。代替検査は同じ `mise run check-spec` であり、`EX-APPLICATION-008-03 is declared, but no test names it` で失敗した。
    e2e_red: TestGetApplicationIconFromAnotherTenantIsNotFound は旧具体例の 400 を期待した段階で、実際の 404 `not_found` を観測して失敗した。
    unit_fault_injection: 生成済みクエリ `GetApplicationIcon` の条件を `(a.tenant_id = $1 OR TRUE)` に変えると、単体テストが失敗した。
    e2e_fault_injection: handler の検索テナントを既定テナントに固定すると、別 realm の取得が 200 で GIF を返し、E2E テストが失敗した。越境だけを 400 で返す故障も検出した。
  - id: foreign-tenant-application-admin-operations-are-not-found
    unit_red: "N/A: 保存先は変更前から要求先テナントで絞り込んでおり、単体テストは上のユースケースと共有する。"
    e2e_red: "N/A: 16 件の 404 は実装済みであり、TestApplicationAdminOperationsAnswerAnotherTenantsApplicationAsMissing は既存の振る舞いを宣言どおりに固定した。実装前に観測した失敗は、存在しない id を各 API 操作へ送る探索で、宣言にない 404 が 15 件返ったことである。"
    unit_fault_injection: 上のユースケースと同じ故障で、単体テストが失敗した。
    e2e_fault_injection: "`writeApplicationError` の 404 を 400 に変えると 15 件、`writeCategoryError` の `application_not_found` を 400 に変えると SetApplicationCategories のサブテストが失敗した。メモリ保存先の `FindByID` がテナントを無視すると、16 件のサブテストに加え、default テナントの Application が変わったことの表明も失敗した。"
  - id: foreign-tenant-application-category-is-not-found
    unit_red: "N/A: カテゴリーの保存先は変更前から要求先テナントで絞り込んでおり、TestApplicationCategoryRepositoryFindByIDIsScopedToTenant は書いた時点で GREEN だった。"
    e2e_red: "N/A: 2 件の 404 は実装済みであり、TestApplicationCategoryOperationsAnswerAnotherTenantsCategoryAsMissing は既存の振る舞いを宣言どおりに固定した。"
    unit_fault_injection: 生成済みクエリ `GetApplicationCategoryByID` の条件を `(tenant_id = $1 OR TRUE)` に変えると、単体テストが失敗した。
    e2e_fault_injection: "`writeCategoryError` の `category_not_found` を 400 に変えると 2 件のサブテストが失敗した。メモリ保存先のカテゴリー検索がテナントを無視すると、2 件のサブテストとカテゴリーが変わったことの表明が失敗した。"
- **Change-Resistance Results**:
  - `handleGetApplication` の検索テナントだけを既定テナントに固定する故障は、E2E テストを失敗させなかった。`GetSignInPolicy` が要求先テナントで Application の存在をもう一度確かめ、404 を返すためである。越境の内容は漏れない。
  - 境界テストの初回実行では、別テナントからの要求の前後で Application の詳細が異なった。差分は保存されていないサインイン方針を読むたびに現在時刻で合成する `created_at` と `updated_at` だけであり、越境の書き込みではなかった。テストは方針を先に保存してから比べる。
  - `mise run test-go-mutation -- backend/application/handlers_http` は 295 個の変異を試し、229 個を検出し、57 個が生き残った。`handleGetApplication`、`handleGetApplicationIcon`、`requireApp` の条件への変異 10 個はすべて検出した。生き残った変異は `buildApplicationResponse`、`handleListApplications`、プロトコル設定更新などにあり、この変更の対象外である。
  - PostgreSQL を使う単体テストはサンドボックス内では embedded-postgres を起動できず、黙ってスキップされた。サンドボックス外で実行して上の障害注入を観測した。
- **Verification Results**:
  - `mise run check-spec` - passed（負債項目の削除直後は 2 件の被覆欠落で failed）
  - `mise run check-contract-drift`、`mise run check-api-compat`、`mise run check-status-drift` - passed
  - `mise run test-go-package -- ./backend/application/handlers_http` - passed
  - `mise run test-go-test -- ./backend/application/db_postgres ...` - passed（サンドボックス外、embedded-postgres が起動した状態）
  - `mise run lint-go` - 0 issues
  - `mise run verify` - passed
  - `mise run test-ui-e2e` - 未実行。フロントエンドはこれらのエラーコードを参照しておらず、変更はブラウザーへ届かない。
- **Left Undone**:
  - `mise run check-status-drift` は、`writeApplicationError` と `writeCategoryError` の中まで読めないままである。写像は HTTP の境界テストで固定した。
