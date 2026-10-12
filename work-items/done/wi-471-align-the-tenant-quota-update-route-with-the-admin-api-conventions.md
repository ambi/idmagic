---
status: completed
authors: [tn]
risk: high
reversibility: irreversible
created_at: 2026-09-03
change_kind: bugfix
priority: p1
depends_on: [wi-467-enforce-csrf-on-tenant-quota-update]
affected_spec:
  - { path: docs/modules/tenancy/quota/README.md, requirement: REQ-TENANCY-012 }
  - { path: docs/modules/tenancy/quota/README.md, requirement: REQ-TENANCY-046 }
  - { path: docs/modules/system/http-api/README.md, requirement: REQ-SYSTEM-023, impact: conforms }
  - { path: docs/requirements/scenarios.feature.md, requirement: REQ-PLATFORM-005, impact: conforms }
  - { path: spec/modules/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.UpdateTenantQuota }
evidence_policy: risk-based-v4
documentation_impact:
  level: none
  references: []
  reason: "初回公開前である。既存データの内部 ID は維持し、UI は API と同時に realm 送信へ変更する。独自の呼び出し元の UUID は realm へ差し替える必要がある。"
initial_context:
  specification:
    - docs/modules/tenancy/quota/README.md#REQ-TENANCY-012
    - docs/modules/tenancy/lifecycle/README.md#REQ-TENANCY-014
    - docs/modules/tenancy/README.md
    - docs/modules/tenancy/design/README.md
    - docs/modules/system/http-api/README.md#REQ-SYSTEM-023
    - docs/requirements/scenarios.feature.md#REQ-PLATFORM-005
    - docs/development/specification-first-workflow.md
    - docs/formats/work-item-format.md
    - docs/development/testing.md
    - docs/development/coding-style.md
    - docs/development/writing-language.md
    - docs/development/release.md
    - docs/formats/specification-format.md
    - docs/design/application/api-guidelines.md
    - docs/design/application/design-guidelines.md
    - docs/design/security/authorization.md
    - docs/design/security/threat-model.md
  typespec:
    - IdMagic.Tenancy.Operations.UpdateTenantQuota
    - IdMagic.Contract.TenantQuotaUpdateRequest
  source:
    - backend/tenancy/handlers_http/admin_tenant_handler.go
    - backend/shared/http/support_http/response.go
    - backend/shared/http/support_http/error_handler.go
    - frontend/src/api/admin.ts
    - frontend/src/features/system-tenants/SystemTenantsPage.tsx
  tests:
    - backend/tenancy/handlers_http/admin_tenant_handler_test.go
    - backend/tenancy/handlers_http/control_plane_contract_test.go
    - backend/tenancy/handlers_http/control_plane_audit_test.go
    - backend/tenancy/handlers_http/implicit_rules_test.go
    - backend/shared/http/server_http/control_plane_audit_test.go
    - backend/shared/http/server_http/tenant_quota_csrf_test.go
    - frontend/src/features/system-tenants/SystemTenantsPage.test.tsx
  stop_before_reading:
    - backend/tenancy/db_postgres
    - backend/authentication/session
primary_use_cases:
  - id: quota-realm-storage
    requirement: REQ-TENANCY-046
    observable_result: "realm に対応する内部 ID の上書きだけが変わり、応答と監査イベントが保存先に一致する。"
    boundary: acceptance
    test:
      path: backend/tenancy/handlers_http/tenant_quota_contract_test.go
      name: TestTenantQuotaResolvesRealmAndReturnsStoredQuota
      task: test-go-race
    fault_model: "realm を解決せずにそのまま保存またはイベントへ渡す。成功応答から no-store を落とす。"
  - id: quota-browser-save
    requirement: REQ-TENANCY-046
    observable_result: "新規テナントの上限を管理画面から変更し、再読み込みして保存された値を確認できる。"
    boundary: e2e
    test:
      path: frontend/tests/e2e/tenant-quota.spec.ts
      name: system console saves tenant quotas by realm
      task: test-ui-e2e
    fault_model: "管理画面だけ UUID を送信し続けるか、gateway と API の接続で realm が失われる。"
    reason: "正式なブラウザー入口と gateway・API の組み立てを通る保存と再読み込みは下位境界では検証できない。"
---

# テナントクォータ更新経路を Tenancy 管理 API の規約へそろえる

## Motivation

`UpdateTenantQuota` は、Tenancy の制御面ハンドラーのうち唯一、共通の規約をどれも通らずに書かれている。

第一に、テナントの識別子を解決しない。

兄弟の経路はすべてパス引数を `resolveTenantByRealm` で realm から UUID へ解決するが、クォータ更新だけは受け取った文字列をそのまま `QuotaRepo.SetQuota` へ渡す。

`tenant_quotas.tenant_id` は `tenants(id)` を参照する UUID 列なので、同じパス引数が経路によって realm と UUID の二つの意味を持つ状態になっている。

管理画面もこの不整合をそのまま反映しており、属性更新と停止には `tenant.realm` を、クォータ更新には `tenant.id` を送っている。

兄弟の経路と同じ規約で呼ぶ API クライアントは、realm を送った時点でデータベース層の型エラーになる。

第二に、拒否を応答へ写像しない。

兄弟の経路は `WriteAdminAccessError` を通して 401 と 403 の Problem Details を書くが、クォータ更新は `requireSystemAdmin` が返した番兵エラーをそのまま返す。

この番兵エラーは HTTP 状態を持たないため、共通のエラーハンドラーが 500 として扱い、権限のない要求の拒否が「未処理の要求エラー」として記録される。

仕様は 403 を宣言し、この操作については 401 を宣言していないため、宣言された拒否がどちらも発生しない。

第三に、存在しないテナントに対して 404 を返さず、`writeTenantError` を通らない。

第四に、内部エラーの文字列をレスポンスボディへ連結しており、データベースのエラーメッセージが呼び出し元へ出る。

第五に、リクエストボディを `support.DecodeJSON` ではなく `c.Bind` で読み、応答を `support.NoStoreJSON` ではなく `c.JSON` で書いている。

起票後に本文復号は `support.DecodeJSON` へ変更され、現在の共通仕様では未知フィールドを無視し、64 KiB の上限を適用する。
成功応答にはまだ管理応答の `Cache-Control: no-store` がない。

そしてこの経路を名指しする HTTP テストは一つもない。

規約から外れていること自体は、[[wi-467-enforce-csrf-on-tenant-quota-update]] の CSRF 欠落と [[wi-470-record-control-plane-state-changes-in-the-audit-log]] の監査記録欠落と同じ根を持つが、識別子と応答契約の不統一はそれらのどちらでも直らない。

## Scope

- クォータ更新のテナント識別子を realm として解決し、兄弟の制御面経路と同じ意味にそろえる。
- 存在しないテナントを `writeTenantError` 経由の 404 で返す。
- 認証と認可の拒否を `WriteAdminAccessError` へ通し、宣言どおりの 401 と 403 の Problem Details にする。
- 保存失敗の応答から内部エラーの文字列を外す。
- リクエストボディを `support.DecodeJSON` で読み、共通仕様どおり未知フィールドを無視し、本文長の上限を維持する。
- 成功応答を `support.NoStoreJSON` で書き、`Cache-Control: no-store` を付ける。
- 仕様へ 401 と 404 を宣言し、パス引数 `tenant_id` が realm を指すことを doc に明示する。
- 管理画面のクォータ更新呼出しを `tenant.realm` へ変える。
- 成功、権限なし、未認証、存在しないテナント、未知フィールドを含む本文の各経路を HTTP 受け入れテストで固定する。

## Out of Scope

- 未知フィールドの拒否。現行の共通仕様 REQ-PLATFORM-005 は無視を要求するため、既存の挙動を維持する。
- クォータ更新の `Origin` と CSRF トークンの検証。
  [[wi-467-enforce-csrf-on-tenant-quota-update]] が扱う。
- クォータ更新の監査イベント発行。
  [[wi-470-record-control-plane-state-changes-in-the-audit-log]] が扱う。
- システムコンソールに要求する認証強度と再認証時刻。
  [[wi-468-system-console-privileged-session-assurance]] が扱う。
- クォータ値そのものの範囲検証、および同一テナントへの同時更新の調停。
- 上限の適用規則、Hard と Soft の区分、利用量の集計方法を変えること。
- ほかの管理 API のパス引数命名を一括で見直すこと。

## Design

### 着手時の照合と要件差分

前提の wi-467 と監査対応の wi-470 は完了している。
401 の宣言、`RequireControlPlaneUser` と `WriteAdminAccessError`、`support.DecodeJSON` は実装済みなので維持する。
今回採用する残りの案は、realm の解決、404、内部エラーの秘匿、no-store、UI の realm 送信である。
UUID への統一と二種類の識別子の受理は下記の理由で対象外とする。

- REQ-TENANCY-012 に、成功応答は `Cache-Control: no-store` を伴うことを追加する。
- REQ-TENANCY-046 を追加する。System 管理者が realm を指定して上限を更新したとき、Tenancy は、その realm に属するテナントの上書きを保存し、保存先と `TenantQuotaUpdated` のテナントを内部 ID で識別する。
- REQ-TENANCY-046 は、不在の realm には本文の妥当性によらず 404 `tenant_not_found` を返し、保存とイベント発行を行わないことも定める。
- 保存と解決の内部エラーは既存の REQ-SYSTEM-023 に適合させ、共通の `WriteServerError` で記録と秘匿を行う。
- `UpdateTenantQuota` へ 404 を追加し、`tenant_id` が realm を指すことを明示する。401 は既存の宣言を維持する。

操作は `handleUpdateTenantQuota(*echo.Context) error` のまま、既存の `resolveTenantByRealm(*echo.Context, string) (*domain.Tenant, error)` で対象を解決する。
保存ポートは `QuotaRepository.SetQuota(context.Context, string, *domain.TenantQuota) error`、イベント出力は `Deps.Emit` を使う。
モジュール、依存、永続化形式と確定の境界は変えない。
CSRF、認証と認可、realm 解決、本文復号、既存の非負検証、保存、イベント、応答の順にする。

予定ファイルを指定した `report-contract-evidence --changed --base-revision main` は、TypeSpec 変更を含むため未解析の 282 操作を保守的に選択した。
参照のない操作は 0 件だった。
契約テストの成功は別に収集し、今回の変更操作と監査・CSRF の呼び出し元を検証する。
未知フィールドは起票時の記述と現行の REQ-PLATFORM-005 が矛盾していた。
利用者へ確認したが回答はなく、既存の観測結果を変えない共通仕様の維持を採用した。
未知フィールドの拒否は対象外とし、無視する挙動と 64 KiB の上限を既存規範へ適合するテストで固定する。

分類 (a) は realm と no-store、分類 (c) はパス文字列の直接保存と内部エラーの開示であり、今回の対象範囲で是正する。
その他の観点では、上書き全体の置換、非負検証、使用量の保持、保存成功後だけのイベントを維持する。
別テナントの `system_admin` を制御面へ提示する fixture は、既存の `Authenticator.Authenticate` が所属と要求先の不一致を未認証として扱うため 401 になる。
この既存の認証境界を維持し、通常の制御面管理者は 403 とする。両方とも保存とイベントの無変更を観測する。
重複キー、大小文字、エラーの具体的な英文は共通の復号器と応答処理に委ね、独自の細部を固定しない（分類 (b)）。
同時更新、再試行と冪等性の追加、欠けた依存の構築条件は対象外とする。
未公開のため告知は必須ではなく、データ移行も不要である。UI と API は同時に変更し、独自の呼び出し元は UUID を realm へ差し替える。
後退は UI と API を同時に戻す必要があり、DB の内容は内部 ID のまま維持できる。

パス引数の意味は realm にそろえる。

制御面のテナント経路は、作成、属性更新、正規ロケーション切替、停止、再開のすべてが realm を受け取っており、多数派に合わせるほうが、公開済みの URL 形と管理画面の遷移の両方に対して破壊が小さい。

逆に全経路を UUID へそろえる案は採らない。

realm は URL 上でテナントを表すための識別子であり、UUID を要求すると、運用者は上限を調整するためだけに一覧を引いて内部 ID を控える必要が生じる。

UUID も受け付けて両方を解決する案も採らない。

同じ位置に二つの識別子空間を許すと、realm が UUID の形をとる将来のテナントで曖昧になり、拒否の理由も「存在しない realm」と「存在しない ID」に分岐して増える。

この変更は、現在 UUID を送っている呼び出し元を壊す。

壊れる呼び出し元は管理画面と、この経路だけを UUID で呼ぶよう作られた自動化に限られ、いずれも realm を持っているため、移行は送る値の差し替えで済む。

拒否の写像は、番兵エラーを直接返す形をやめ、兄弟の経路と同じ `WriteAdminAccessError` の一本道にする。

共通のエラーハンドラー側で番兵エラーを 403 へ写像する案は採らない。

その写像を増やすと、ハンドラーが拒否を書かずに処理を続ける実装が検査を通ってしまい、応答だけが正しく副作用が残る誤りを見えなくする。

順序は CSRF 検証、認証と認可、識別子解決、本文の復号、保存とする。

識別子解決を本文の復号より前に置くのは、リクエストボディの妥当性が対象テナントの存在を観測する手段にならないようにするためである。

## Plan

1. realm を送る要求が失敗し、権限のない要求が 500 になる現在の挙動を HTTP 境界で観測し、RED を確認する。
2. 仕様へ 401 と 404 を宣言し、パス引数の意味を doc に明示して生成物を再生成する。
3. ハンドラーを共通ヘルパーの規約へそろえ、拒否、404、内部エラー本文、復号、応答ヘッダーを直す。
4. 管理画面のクォータ更新を realm へ切り替える。
5. 成功と各拒否経路を受け入れテストで固定し、拒否時に保存ポートが呼ばれないことを確認する。
6. 状態コードと契約のドリフト検査を通す。

## Tasks

- [x] T001 [Acceptance] `test-go-test` の `TestTenantQuotaResolvesRealmAndReturnsStoredQuota`、`TestTenantQuotaRefusesMissingRealm`、`TestTenantQuotaHidesSaveFailure` で、誤った保存先、no-store の欠落、不在の 200/400、内部エラー開示を RED として観測した。401/403 は既に修正済みであり既存の拒否テストを維持した。要件は REQ-TENANCY-012、REQ-TENANCY-046、REQ-SYSTEM-023。
- [x] T002 [Spec] 401 を維持し、404 と realm の説明を追加した。REQ-TENANCY-012 に no-store、REQ-TENANCY-046 に realm の保存先と不在の拒否を追加し、`check-spec` が成功した。
- [x] T003 [App] `resolveTenantByRealm` の結果を保存とイベントへ渡し、`writeTenantError` の 404 と `WriteServerError` の秘匿へ統一した。T001 の同じ HTTP テストで GREEN を確認した。
- [x] T004 [App] 既存の `DecodeJSON` を維持し、`NoStoreJSON` に切り替えた。`TestTenantQuotaRefusesInvalidBody` と `TestTenantQuotaIgnoresUnknownFields` が REQ-PLATFORM-005 の上限と前方互換性を確認した。
- [x] T005 [UI] `tenant.realm` を送信する。`test-ui-unit-file` で UUID の送信を RED として観測し、切替後に 4 テストが成功した。`test-ui-e2e-file -- tests/e2e/tenant-quota.spec.ts` でも保存通知が出ない RED と、保存後の再読で上限 321 が見える GREEN を確認した。
- [x] T006 [Acceptance] 新規の契約テストと既存の `TestUpdateTenantQuotaHTTPContractRefusesAuthenticationAndScope` が成功、401、403、404、未知フィールド、過大・不正な本文を検査した。HTTP 境界で状態・本文・イベントを観測し、同じ規則を別の単体境界へ複製しない。
- [x] T007 [Unit] `TestTenantQuotaRefusesNonSystemAdmin`、`TestTenantQuotaRefusesMissingRealm`、`TestTenantQuotaRefusesInvalidBody` と既存の認証・CSRF テストで、保存ポートの未呼び出し、保存値とイベントの無変更を確認した。ドメインとユースケースの内部計算は変えないため独立した単体 RED は該当なしとし、HTTP アダプターの失敗を代替証拠とした。
- [x] T008 [Verify] `render-docs` と `generate-contract` が成功し、追跡対象の経路メタデータは変わらなかった。契約、状態コード、セキュリティ制御、操作参照、仕様影響と仕様検査が成功した。本文と状態コードの静的解析ができない操作は、HTTP の契約テストの実行結果で補う。

## Verification

- 局所の RED/GREEN: `mise run test-go-test -- ./backend/tenancy/handlers_http 'TestTenantQuota.*'`。HTTP 変換と保存・イベントを同じ境界で観測する。ドメイン計算を変えないため単体 RED は重複させない。
- UI の RED/GREEN: `mise run test-ui-unit-file -- src/features/system-tenants/SystemTenantsPage.test.tsx`。外向き HTTP 要求の realm、方式、本文、CSRF を確認する。
- 正式な入口: `mise run test-ui-e2e-file -- tests/e2e/tenant-quota.spec.ts`。ブラウザーから作成、上限保存、再読を確認する。
- 逆依存と契約証拠: `mise run test-go-changed -- main`。チェックポイントも含む main からの差分に対し、Tenancy の呼び出し元と共有ルーター、監査・CSRF の既存テストを race 付きで検査する。
- Go の静的検査: `mise run lint-go`。
- UI の形式、型、規則: `mise run format-check-ui`、`mise run typecheck-ui`、`mise run lint-ui`。
- 変更耐性: `mise run check-go-mutation-tool` と `mise run test-go-mutation -- backend/tenancy/handlers_http 2 off /tmp/idmagic-wi-471-mutation.json main`。realm→ID、no-store、秘匿のヘルパー配線は手動で故障注入する。
- 仕様と生成: `mise run check-spec`、`mise run render-docs`、`mise run generate-contract`、`mise run check-published-api-compat`。
- 契約と拒否: `mise run check-status-drift`、`mise run check-contract-drift`、`mise run check-security-controls`、`mise run check-operation-references`、`mise run check-spec-impact`。
- 完了記録: `mise run spec-diff`、`mise run check-work-items`、`mise run check-links`。
- 共有コードと保存形式は変えず、参照不足もなかったため全リポジトリの集約検査は選ばない。初回公開前なので公開済みベースラインとの互換性は公開条件による検査に任せる。

## Risk Notes

パス引数の意味を変えるため、現在 UUID を送っている呼び出し元は変更の時点で 404 を受け取る。

管理画面の切り替えを同じ変更に含め、移行が必要な呼び出し元を変更記録へ明示する。

拒否の写像を直すと、これまで 500 として観測されていた要求が 401 と 403 になる。

監視の閾値やアラートが 5xx の発生を前提にしていないことを確認する。

未知フィールドは現行の共通仕様に従って無視するため、余分なキーを送る自動化の既存挙動を維持する。

公開されたパス引数の意味と宣言される状態コードが変わり、外部の呼び出し元が送る値を変更する必要があるため、`reversibility` は irreversible とする。

## 完了

- **完了日**: 2026-10-12
- **要約**:
  `spec-diff` は REQ-TENANCY-046 の追加、REQ-TENANCY-012 の変更、`UpdateTenantQuota` の変更を示した。
  クォータ更新を realm で解決し、内部 ID へ保存・監査し、不在は本文の妥当性によらず 404 にする。
  成功応答へ no-store を付け、保存失敗は共通処理で記録して内部文字列を秘匿する。
  管理画面は realm を送信し、独自の呼び出し元は UUID を realm へ差し替える必要がある。
  401、403、CSRF、本文の上限と監査発行は既存処理を維持した。
- **主要ユースケースの証拠**:
  - id: quota-realm-storage
    red: 実装前に TestTenantQuotaResolvesRealmAndReturnsStoredQuota が保存先の users が 100 のまま、no-store が空で失敗した。
    fault_injection: 保存先と監査の対象を realm へ変更し、また no-store を外したそれぞれの木で、同じテストが保存値、イベントの対象、ヘッダーの不一致を検出して失敗した。
  - id: quota-browser-save
    red: API の realm 解決を実装後、UUID を送り続ける UI でブラウザーの保存通知を待つテストが失敗した。
    fault_injection: UI のクォータ呼び出しを UUID へ戻すと、system console saves tenant quotas by realm が保存通知を待って失敗した。
- **受け入れ RED の証拠**:
  - **テスト**: `TestTenantQuotaResolvesRealmAndReturnsStoredQuota`、`TestTenantQuotaRefusesMissingRealm`、`TestTenantQuotaHidesSaveFailure`。
  - **要件**: REQ-TENANCY-046
  - **観測した失敗**: realm の保存先の上限が変わらず、不在が 200/400 となり、保存エラーが本文へ出た。
  - **検出できる理由**: 解決した内部 ID の読み戻し、不在への Problem Details、内部文字列の不在を HTTP 境界で区別する。
  `mise run test-go-test -- ./backend/tenancy/handlers_http 'TestTenantQuota.*'` が実装前に失敗した。
  `TestTenantQuotaResolvesRealmAndReturnsStoredQuota` は内部 ID の上限が 100 のまま、no-store が空であることを検出した（REQ-TENANCY-012、REQ-TENANCY-046）。
  `TestTenantQuotaRefusesMissingRealm` は不在や UUID の指定に 200、壊れた本文には 400 が返り、期待する 404 と異なることを検出した（REQ-TENANCY-046）。
  `TestTenantQuotaHidesSaveFailure` は `internal_error` と保存先のエラー文字列が返ることを検出した（REQ-SYSTEM-023）。
  realm と内部 ID を別の値にし、両方の保存値とイベントを読むため、成功コードだけを保つ誤実装も区別できる。
  UI の単体テストは `/tenants/tenant-1/quota` が送られ、期待する `/tenants/acme/quota` と異なることを検出した。
  ブラウザーのテストも、API 修正後に UI が UUID を送る状態で保存通知を待つ検査が失敗し、UI 修正後には保存と再読が成功した。
- **単体 RED の証拠**:
  - **テスト**: `SystemTenantsPage.test.tsx` の `saves the quota by realm while retaining the internal selection ID`。
  - **要件**: REQ-TENANCY-046
  - **観測した失敗**: クォータ更新の要求先が `/tenants/tenant-1/quota` で、期待する `/tenants/acme/quota` と異なった。
  - **検出できる理由**: 外向き HTTP 境界へ送る URL と本文を観測するため、API が正しくても UI が内部 ID を送る故障を検出できる。
  ドメインとユースケースの計算、型と保存契約は変更していないため、独立した単体境界は該当しない。
  HTTP アダプターの契約テストの RED と、UI が出す要求値の単体 RED を代替証拠とする。
  保存ポートの未呼び出し、保存値とイベントの無変更は、認証、ロール、CSRF、不在と不正な本文の各拒否で観測した。
- **変更耐性の結果**:
  比較基準は実装前の main（`607a4ffe4`）。
  `check-go-mutation-tool` の canary は通常とキャッシュ実行で期待する lived と not viable を分類した。
  `test-go-mutation -- backend/tenancy/handlers_http 2 off /tmp/idmagic-wi-471-mutation.json main` は変更行の条件否定 2 件（realm 解決エラーと保存エラー）を両方検出した。
  生存、未被覆、構築不能、時間超過はなかった。
  ツールが表現できない配線には、保存先を realm にする、イベント対象を realm にする、NoStoreJSON を外す、保存エラーを detail に含める、の 4 故障を一つずつ注入した。
  前三者は `TestTenantQuotaResolvesRealmAndReturnsStoredQuota`、最後は `TestTenantQuotaHidesSaveFailure` が失敗して検出した。
  全故障を戻した後に局所テストが成功した。
  UI の UUID 送信は単体とブラウザーの RED で検出した。
  変更していない Go 行と同時更新の検出能力は今回の調査から判断しない。
- **検証結果**:
  最終差分に対する `test-go-changed -- main` は race 付きで 1,689 テスト成功、失敗・スキップ・未実行 0 件だった。
  `test-go-test` の対象契約テストは子テストを含め 14 件成功した。
  UI の対象ファイルは 4 テスト、ブラウザーの保存と再読は 1 テスト成功した。
  Go lint、UI の形式・lint・型検査、`check-spec`、`check-spec-impact`、状態コード・本文のドリフト、セキュリティ制御、操作参照の検査が成功した。
  ドリフト検査には静的解析の対象外が残るため、それを成功した操作数へ含めず、今回の操作は実行済みの HTTP 契約テストで確認した。
  公開互換性の検査は未公開のため比較を省略した。
  `render-docs` と `generate-contract` が成功し、追跡対象の経路メタデータに差分はなく、生成 OpenAPI と HTML は追跡していない。
  七つの視点で差分をレビューし、既存の解決・拒否・応答ヘルパーの再利用、早期 return、局所的な変換、保存とイベントの観測、公開範囲と依存の維持を確認した。
- **対象外と未実施**:
  未知フィールドの拒否は現行の REQ-PLATFORM-005 に合わせて対象外とし、無視する挙動を維持した。
  同時更新、値域の拡張、認証強度、CSRF と監査機構の拡張は対象外である。
  DB 形式は変わらず、データ移行は不要。Go 全体、UI 全体、全ブラウザースイートと集約 verify は変更範囲に必要な検査へ絞ったため実行していない。
  対象範囲に残した未実施事項はない。
