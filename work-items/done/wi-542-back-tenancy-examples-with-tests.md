---
depends_on: [wi-565-make-backing-declared-examples-cheap]
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-12
priority: p3
change_kind: maintenance
spec_impact: { kind: none, reason: "宣言済みの具体例に、その id を名指しするテストを対応付ける作業である。シナリオも製品の振る舞いも変えない。テストが書けない具体例が見つかった場合、それは実装が具体例のとおりに振る舞っていないということなので、欠陥として個別の work item に切り出す。" }
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: 宣言済みの具体例と既存の検証を対応付ける作業であり、公開契約と運用手順を変えない。
  references: []
initial_context:
  specification:
    - docs/domain/tenancy/scenarios.feature.md#REQ-TENANCY-001
    - docs/domain/tenancy/scenarios.feature.md#REQ-TENANCY-002
    - docs/domain/tenancy/scenarios.feature.md#REQ-TENANCY-003
    - docs/domain/tenancy/scenarios.feature.md#REQ-TENANCY-004
    - docs/domain/tenancy/scenarios.feature.md#REQ-TENANCY-006
    - docs/domain/tenancy/scenarios.feature.md#REQ-TENANCY-007
    - docs/domain/tenancy/scenarios.feature.md#REQ-TENANCY-008
    - docs/domain/tenancy/scenarios.feature.md#REQ-TENANCY-009
    - docs/domain/tenancy/scenarios.feature.md#REQ-TENANCY-010
    - docs/domain/tenancy/scenarios.feature.md#REQ-TENANCY-011
    - docs/domain/tenancy/scenarios.feature.md#REQ-TENANCY-012
    - docs/domain/tenancy/scenarios.feature.md#REQ-TENANCY-015
    - docs/domain/tenancy/scenarios.feature.md#REQ-TENANCY-016
    - docs/domain/tenancy/scenarios.feature.md#REQ-TENANCY-018
    - docs/domain/tenancy/scenarios.feature.md#REQ-TENANCY-019
    - docs/domain/tenancy/scenarios.feature.md#REQ-TENANCY-020
    - docs/domain/tenancy/scenarios.feature.md#REQ-TENANCY-021
  typespec: []
  source:
    - backend/tenancy/handlers_http/branding_handler.go
    - backend/shared/notification/template/notifier.go
    - backend/shared/notification/template/locale.go
    - backend/shared/http/server_http/routes.go
    - frontend/vite.config.ts
  tests:
    - backend/tenancy/handlers_http/integration_endpoints_handler_test.go
    - backend/tenancy/handlers_http/admin_user_attribute_schema_handler_test.go
    - backend/tenancy/usecases/manage_tenants_test.go
    - backend/tenancy/handlers_http/branding_handler_test.go
    - backend/shared/http/server_http/tenant_routes_test.go
    - backend/shared/http/server_http/tenant_host_routes_test.go
    - backend/shared/http/support_http/tenant_cookie_test.go
    - backend/authentication/webauthn/handlers_http/request_rp_test.go
    - backend/shared/http/server_http/tenant_quota_csrf_test.go
    - backend/authentication/password/usecases/password_reset_test.go
    - backend/shared/notification/template/notifier_test.go
    - backend/shared/notification/template/render_test.go
    - backend/tenancy/usecases/manage_notification_templates_test.go
    - backend/tenancy/handlers_http/admin_notification_template_handler_test.go
    - backend/tenancy/handlers_http/admin_settings_handler_test.go
    - backend/tenancy/db_postgres/tenants_test.go
    - backend/authentication/password/usecases/password_expiry_test.go
    - backend/tenancy/handlers_http/admin_group_attribute_schema_handler_test.go
    - backend/tenancy/usecases/manage_delegation_depth_test.go
    - backend/oauth2/token/usecases/exchange_token_delegation_policy_test.go
    - frontend/src/features/admin-settings/AdminSettingsPage.test.tsx
    - frontend/src/features/admin-settings/BrandingTab.color-reset.test.tsx
    - frontend/src/features/admin-settings/DelegationPolicyTab.test.tsx
    - frontend/src/features/auth-flow/AuthFlowPages.test.tsx
    - frontend/src/components/Brand.test.tsx
  stop_before_reading:
    - docs/architecture
    - infra
---

# Tenancy が宣言する具体例 39 件にテストを対応付け、被覆台帳から外す

## Motivation

[[wi-496-burn-down-the-example-coverage-debt]] は具体例の被覆台帳の消化単位を Context と決め、測定のうえで残りを Context ごとの子 work item へ割った。本項目はそのうち `docs/domain/tenancy/scenarios.feature.md` が宣言する 39 件を引き取る。

親項目が `claim-mapping` の 3 件と `workloadidentity` の 13 件で測った結果は、**16 件のうち注記だけで済んだのは 4 件だけ**だというものである。3 件はテストが 1 つも無く、9 件は既存テストへ新しい観測を足す必要があった。件数は作業量の目安にならない。

## Scope

- 39 件を 1 件ずつ確認し、実装と一致する 37 件を次のいずれかに解決して `tools/check/example-coverage-debt.json` から外す。
  - 当の具体例を検証しているテストが実在する → そのテストに `//spec:covers EX-TENANCY-NNN-MM: <この具体例の何を固定しているか>` のディレクティブを足す。
  - 当の具体例を検証しているテストが無い → 書く。具体例が拒否なら、[[wi-392-refusal-tests-assert-the-absent-effect]] が定める形（拒否応答と、拒否が防いだ効果の双方を観測する）で書く。
  - 具体例が宣言されなくなっている → 台帳から外す（検査が落ちて教える）。
- 実装と一致しない 2 件は個別の work item へ切り出し、台帳へ `blocked_by` と `finding` を残す。
- 注記は「何を固定しているか」を書く。id だけの注記は禁止する。
- 実装が具体例のとおりに振る舞っていないことが分かった場合は、**本 work item では直さず欠陥として切り出す**。

## Out of Scope

- 他の Context が宣言する具体例。Context ごとに別の work item が持つ。
- `tools/check/standards-coverage-debt.json`。[[wi-495-burn-down-the-standards-coverage-debt]] が消化済みである。
- 既に台帳に載っていない拒否テストが、防いだ効果を観測していない件。[[wi-392-refusal-tests-assert-the-absent-effect]] が持つ。本項目が新しく書く拒否テストは、その規範を満たす形で書く。既存テストに注記を足すだけの件で、そのテストが効果の不在を見ていない場合は、注記を足したうえで wi-392 の対象として残す。
- シナリオと具体例の追加、削除、書き換え。具体例の記述が実装と食い違うことが判明した場合は規範の変更なので、別の work item が扱う。
- 見つかった実装の欠陥の修正。切り出した先で扱う。
- 行カバレッジ率の目標または閾値。
- `report-coverage-debt` の分類（`named`、`nearby`、`none`）を根拠にした台帳からの削除。分類は読む順を決める材料であり、台帳から外す根拠にはならない。

## 作業の進め方

[[wi-565-make-backing-declared-examples-cheap]] が、この作業の 1 件あたりの費用を下げる道具を用意した。使う。

- 読む順と観測点は `mise run spec-route -- <id>` が出す。規則、当の具体例の本文、契約が宣言する候補 operation とそのメソッド・パス・スコープ、同じ規則の隣の id を名指している既存テストが返る。**出力は読む順を決める材料であり、台帳から外す根拠にはしない。**
- 新しく書くテストは `backend/shared/http/testing_stack` の上に載せる。`Register` と同じ配線が option の合成で建ち、保存先は型付きの field から読み直せる。既存 fixture の全面移行はしない。触る必要が出た範囲だけ移す。
- 製品 Go を変更した場合、その変異は `mise run test-go-mutation -- <package-directory>` で読む。手で書く故障注入は、変異器が表現できない「配線を外す」「既定の分岐を差し替える」種類だけに残す。
- 実装が具体例と食い違って消化できない件は、台帳の当該行へ `blocked_by`（先に決着すべき work item）と `finding`（実装が実際に何を返すか）を書いて残す。散文にだけ書くと、次の読み手が同じ測定をやり直す。

## Design

### 証拠の境界

本項目は規範も製品ロジックも変えず、宣言済みの具体例とテストの対応を検証可能な形にする。
Acceptance RED は、台帳から対象行を外した時点の `mise run check-spec` とする。
検査が具体例 id を名指して落ちるため、対応する `//spec:covers` が無ければ完了できない。

Unit RED は N/A とする。
新しい単体ロジックを作らず、既存テストの観測を具体例と突き合わせる作業だからである。
代替の RED は同じ `mise run check-spec` であり、追加または補強した各テストは所属パッケージか UI テストファイルの絞り込みタスクで GREEN を確認する。

### readiness pass の結果

39 件のうち 37 件は、既存テストへの `//spec:covers` 追加か、既存テストの小さな観測補強で対応できる。

残る 2 件は、実装と具体例が一致しないため台帳に残す。

| 具体例 | 実測結果 | 所有する記録 |
| --- | --- | --- |
| `EX-TENANCY-004-02` | 越境したアセットは返らないが、応答は `InvalidRequestError` ではなく `404 not_found` である | [[wi-578-align-cross-tenant-branding-asset-refusal]] |
| `EX-TENANCY-004-03` | Vite gateway は realm 配下の `tenant-branding-assets` を backend へ転送するため、具体例が述べる取得失敗は再現しない | [[wi-579-align-branding-asset-gateway-example]] |

この 2 件は製品の公開契約または規範の変更を要する。
WI-542 では直さず、台帳の `blocked_by` と `finding` に測定結果を残す。

### テストの対応単位

一つの具体例が HTTP、保存、UI の複数境界を持つ場合は、同じ id を複数のテストから参照し、それぞれのディレクティブに固定する観測を限定して書く。
一つのテストだけへ全体を検証したような説明を置かない。

## Tasks

- [x] T001 [Acceptance] 37 件を台帳から外し、`mise run check-spec` が未対応 id を名指しで落とすことを確認する。
- [x] T002 [Handlers] REQ-TENANCY-001 から 004 までのうち、対応可能な 6 件を既存テストへ結び付ける。`mise run test-go-package -- ./backend/tenancy/handlers_http`。
- [x] T003 [Routing] REQ-TENANCY-006 から 012 までの 12 件を routing、cookie、WebAuthn、quota のテストへ結び付ける。`mise run test-go-package -- ./backend/shared/http/server_http` と各所属パッケージを使う。
- [x] T004 [Notification] REQ-TENANCY-015、016、018 の 9 件を password reset、通知テンプレート、描画のテストへ結び付ける。`mise run test-go-package -- ./backend/shared/notification/template`、`./backend/authentication/password/usecases`、`./backend/tenancy/usecases`、`./backend/tenancy/handlers_http` を使う。
- [x] T005 [Policy] REQ-TENANCY-019 から 021 までの 10 件を設定、永続化、グループ属性、委譲判定のテストへ結び付ける。各所属パッケージの `mise run test-go-package` を使う。
- [x] T006 [UI] 連携情報、branding、委譲深さの UI 観測へ具体例 id を対応付ける。`mise run test-ui-unit-file` で変更ファイルを確認する。
- [x] T007 [Triage] 2 件の食い違いを個別の work item へ切り出し、台帳へ `blocked_by` と `finding` を記録する。
- [x] T008 [Verify] `mise run verify` を通し、完了記録を作る。

## Verification

- `mise run check-spec` が、実装と一致する 37 件を `tools/check/example-coverage-debt.json` から外し、食い違う 2 件へ `blocked_by` と `finding` を残した状態で通る。台帳から外す前に同じ検査が対象 id を名指しで落とすことを観測する。
- 消化したテストの所属パッケージに対する `mise run test-go-package -- <package>`。
- `mise run verify`

## Risk Notes

- **注記だけを足して終わる。** `checkNormativeCoverage` は文字列の一致しか見ないので、読まずに id を貼れば件数は速く減る。減った件数は何も意味しない。注記に「この具体例の何を固定しているか」を書かせることで区別する。親項目の測定では、16 件のうち注記だけで済んだのは 4 件だけだった。
- **`nearby` に分類された件を、テストがある証拠として読む。** 分類が言っているのは「近傍に他の拒否テストがある」だけである。親項目の測定では、`claim-mapping` の 3 件はすべて `nearby` でありながら 2 件はテストが無かった。分類は読む順の材料にとどめる。
- **拒否である具体例のテストが、応答の字面だけを見て書かれる。** 新しく書く拒否テストには [[wi-392-refusal-tests-assert-the-absent-effect]] の規範が効く。注記へ「効果の不在を何で観測したか」を書き、後から区別できるようにする。
- **具体例の `Then` が複数あるのに、観測が 1 つで済まされる。** 親項目の測定では、`EX-WORKLOADIDENTITY-008-01` のようにイベントと実際の効果の双方を言う具体例が複数あった。`Then` の数だけ観測が要る。

## Completion

- **Completed At**: 2026-09-14
- **Summary**:
  `mise run spec-diff` は `main` に対して「no normative specification change」を返した。
  Tenancy が宣言する対象 39 件を実装と照合し、実装と一致する 37 件を `//spec:covers` を持つテストへ結び付け、被覆台帳から外した。
  足りなかった拒否経路、秘密値の不在、永続化値、locale fallback、UI 表示などの観測も補強した。
  実装と一致しない 2 件は [[wi-578-align-cross-tenant-branding-asset-refusal]] と [[wi-579-align-branding-asset-gateway-example]] に切り出し、台帳へ `blocked_by` と `finding` を残した。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-spec`
  - **Requirement**: REQ-TENANCY-001
  - **Observed Failure**: 対象 37 件を台帳から外した状態で、検査が 37 件すべてを名指しして落ちた。先頭の失敗は `EX-TENANCY-001-01 is declared, but no test names it.` だった。
  - **Detection Reason**: 台帳から外した具体例は、それぞれの id を名指すテストが無い限り検査を通らない。ディレクティブの説明と観測内容は検査だけでは保証できないため、各テストを具体例の `Then` と照合した。
- **Unit RED Evidence**:
  - **Test**: `mise run check-spec`（代替 RED）
  - **Requirement**: N/A: 製品ロジックと規範を変更せず、既存の観測と具体例を対応付ける保守作業である。
  - **Observed Failure**: Acceptance RED と同じ 37 件の未対応 id を検出した。
  - **Detection Reason**: 新しい単体ロジックが無いため、単体テストの失敗を作る対象が無い。代わりに、規範 id とテストの対応を検査する最小の受け入れ境界を RED にした。
- **Change-Resistance Results**:
  製品コードを変更していないため、変異テストと手書きの故障注入は N/A とした。
  既存テストへ観測を追加し、所属パッケージと変更影響範囲の Go テスト、および変更した UI テストファイルで検出能力を確認した。
  ブラウザーへ到達する製品経路は変更していないため、ブラウザー E2E は追加していない。
- **Verification Results**:
  - `mise run spec-diff` - passed（`main` に対する規範差分なし）
  - `mise run check-spec` - passed（157 standards、314 rules、757 examples、652 ids named）
  - `mise run check-work-items` - passed
  - 変更した 9 個の Go パッケージに対する `mise run test-go-package` - passed
  - 変更した 5 個の UI テストファイルに対する `mise run test-ui-unit-file` - passed
  - `mise run test-go-changed` - passed
  - `mise run lint-go` - passed（0 issues）
  - `mise run verify` - passed。初回は変更外の `AccountDataPage` の 1 件だけが失敗したが、対象ファイル単独では 5/5 件が通り、再実行では UI 695 件を含む全タスクが通った。
