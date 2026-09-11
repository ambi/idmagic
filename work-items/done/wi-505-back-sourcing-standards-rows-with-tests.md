---
depends_on: []
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-06
priority: p2
change_kind: maintenance
spec_impact: { kind: none, reason: "宣言済みの標準行に、その id を名指しするテストを対応付ける作業である。standards.md の行そのものも製品の振る舞いも変えない。テストが書けない行が見つかった場合、それは製品が宣言した採用を満たしていないということなので、欠陥として個別の work item に切り出す。" }
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: 標準の行にも製品の振る舞いにも変更が無く、増えたのはテストと注記だけなので、リリースの読み手に見えるものが無い。
  references: []
initial_context:
  specification:
    - docs/contexts/sourcing/standards.md
    - docs/contexts/provisioning/standards.md
  typespec: []
  source:
    - backend/sourcing/scim/handlers_http/routes.go
    - backend/sourcing/scim/handlers_http/handlers.go
    - backend/sourcing/scim/domain/scim_models.go
    - backend/sourcing/scim/domain/discovery.go
    - backend/sourcing/scim/domain/mutation.go
    - backend/sourcing/scim/usecases/users.go
    - backend/shared/http/support_http/tenant_middleware.go
    - backend/shared/http/support_http/auth.go
    - tools/check/src/check-documents.ts
    - tools/check/src/normative-coverage.ts
    - tools/check/standards-coverage-debt.json
  tests:
    - backend/sourcing/scim/handlers_http/scim_test.go
    - backend/sourcing/scim/handlers_http/resource_contract_test.go
    - backend/sourcing/scim/domain/discovery_test.go
    - backend/provisioning/client_scim/conformance_test.go
  stop_before_reading:
    - backend/provisioning/client_scim/client.go
    - backend/sourcing/scim/domain/filter.go
    - backend/sourcing/scim/db_postgres
    - frontend
---

# Sourcing が宣言する標準 5 行にテストを対応付け、標準の被覆台帳から外す

## Motivation

[[wi-495-burn-down-the-standards-coverage-debt]] は標準の被覆台帳へ受入集合を入れ、消化の単位を所有文書と決めた。本項目はそのうち `docs/contexts/sourcing/standards.md` の 5 行を引き取る。この文書は 8 行のうち 5 行が名指しを持たない。

**同じ SCIM の規範を扱う Provisioning は 13 行のうち 12 行が名指しを持つ。** [[wi-238-scim-inbound-list-query-conformance]] の適合作業が id を名指すテストを書いたからである。Sourcing は同じ RFC 7643 / RFC 7644 を、下流へ送る Provisioning とは逆に、SCIM のサービス提供者として受け取る側から採用しているが、その適合作業を経ていない。つまりこの 5 行は、規範が難しいのではなく、作業がまだ来ていないだけである。Provisioning 側の 12 行が消化の実例として読める。

## Scope

- 次の 5 行を消化する。

| ID | Adoption |
|---|---|
| `RFC7643-SERVICE-PROVIDER-CONFIG` | required |
| `RFC7644-RESOURCE-OPERATIONS` | required |
| `RFC7644-BEARER-AUTHORIZATION` | required |
| `RFC7644-ERROR-RESPONSE` | required |
| `RFC7643-ENTERPRISE-EXTENSION` | partial |

- 1 行ごとに、その行の `Statement` を区別できる入力と観測を持つテストを対応付け、`// <ID>: <この行の何を固定しているか>` の注記を足す。
- 観測の形は行の `Adoption` に従う。型は [[wi-495-burn-down-the-standards-coverage-debt]] の Design が定める。
- 消化した id を `tools/check/standards-coverage-debt.json` から外す。

## Out of Scope

- 他の文書が宣言する標準行。文書ごとに別の work item が持つ。Provisioning が持つ同じ RFC の行は、既に 12 行が名指しを持ち、残る 1 行は [[wi-507-back-provisioning-standards-rows-with-tests]] が持つ。
- `standards.md` の行の追加、削除、`Adoption` および `Strength` の変更。
- 見つかった実装の欠陥の修正。欠陥として切り出した先で扱う。
- 受入集合の導入と、台帳ファイルそのものの削除。親の [[wi-495-burn-down-the-standards-coverage-debt]] が持つ。
- 具体例の被覆負債。[[wi-496-burn-down-the-example-coverage-debt]] が持つ。
- SCIM の適合範囲そのものの拡張。
- id を名指しする文字列があるだけを根拠にした台帳からの削除。

## Design

**Provisioning 側の 12 行がどう消化されたかを先に読む。** 同じ RFC の行が既に名指しを持っているので、注記の書き方と観測の置き場所はそこに実例がある。Provisioning は下流へ送る側なので、その入口は組み立てた要求であり、テストは擬似の下流サーバーが受け取ったものを読む。Sourcing は受け取る側なので、入口は `/scim/v2/...` の HTTP 経路そのものになる。入口は逆向きだが、行の `Statement` を区別するという条件は同じである。

`RFC7643-ENTERPRISE-EXTENSION` だけが `partial` である。`partial` の観測は 2 つ要る。採用した範囲の振る舞いと、採用していない範囲がどう扱われるか、つまり拒否するのか単に提供しないのかである。片方だけでは、全部を採用している実装とも、何も採用していない実装とも区別できない。行の `Statement` が採用の境界をどこに引いているかを読み、その境界の両側を観測する。

`RFC7644-ERROR-RESPONSE` は誤り応答の形を宣言する行である。観測は、誤りが起きたときに SCIM の誤り応答の形（`schemas`、`status`、`scimType`、`detail`）で返ることであり、HTTP のステータスだけではない。ステータスだけを観測すると、SCIM の形になっていない応答と区別できない。

`RFC7644-BEARER-AUTHORIZATION` は拒否の行である。認可の無い、または不正な Bearer トークンでの要求が拒否されること、および**その拒否が防いだ効果**、つまり対象リソースが変わっていないことを対で観測する。ステータスだけでは、変更してから拒否を返す実装と区別できない。

### T001 の棚卸し

既存のテストを読み、行ごとに「どこまで区別できているか」を書き出した。

| ID | 既存テストが届いている範囲 | 届いていない範囲 |
|---|---|---|
| `RFC7643-SERVICE-PROVIDER-CONFIG` | `TestScimInboundProvisioning` の 3 番目の区画が `authenticationSchemes` が空でないことと `oauthbearertoken` を含むことを観測している。行の `Statement` はこの 2 つだけを言っているので、**区別はできている** | 名指しが無い。観測が 6 区画からなる長いテストの途中にあり、行に対応する試験として読めない |
| `RFC7644-RESOURCE-OPERATIONS` | User の作成・PATCH・削除 (`TestScimInboundProvisioning`)、User と Group の置換 (`TestScimUpdateUserFullReplace`、`TestScimGroupResourceContract`)、Group の作成と PATCH (`TestScimGroupSync`) が HTTP 境界にある | **Group の削除と Group の単体参照が HTTP 境界に無い。** `DeleteGroup` / `GetGroup` を観測しているのは use case のテストだけで、経路は `TestScimRoutesRequireOperationScope` が存在しない id で状態符号を見るに留まる。行は「操作を提供する」と言っており、提供の単位は製品の入口である |
| `RFC7644-BEARER-AUTHORIZATION` | 未登録トークンの 401 と challenge (`TestScimInboundProvisioning`)、15 経路それぞれの scope 不足の 403 と `insufficient_scope` (`TestScimRoutesRequireOperationScope`) | **拒否が防いだ効果を 1 件も観測していない。** どれも状態符号と header 止まりで、変更してから拒否を返す実装と区別できない。Authorization header の無い要求と、**行が言う「テナント単位」**、つまり別テナントのトークンでの要求にテストが無い |
| `RFC7644-ERROR-RESPONSE` | 失敗の応答本体から `scimType` を読むテストが多数ある | **SCIM の誤り応答の形を観測していない。** `schemas`、`status`、`detail`、`Content-Type` を読むテストが 1 件も無く、`scimType` だけを持つ素の JSON を返す実装と区別できない。`domain.NewScimError` 自体の単体テストはあるが、それは形を作る関数の試験であって、失敗が製品の入口からその形で出ることの観測ではない |
| `RFC7643-ENTERPRISE-EXTENSION` | 採用した側は厚い。`TestScimCreateUserEnterpriseExtension`、`TestScimPatchUserEnterpriseExtension`、`TestScimEnterpriseExtensionDiscovery`、`TestEnterpriseUserSchemaAttributes` が employeeNumber / department / manager の CRUD・PATCH・Discovery を観測する。Discovery は `costCenter` / `division` / `organization` を広告しないことまで見ており、**境界の両側が揃っている唯一の観測である** | **振る舞いの側に境界が無い。** 採用外の拡張属性を載せた要求が、CRUD で落とされるのか PATCH で拒否されるのかを観測していない。名指しはすべて `REQ-SOURCING-007` であって標準 id ではない |

5 行のうち、既存の観測が行の `Statement` を区別できているのは `RFC7643-SERVICE-PROVIDER-CONFIG` の 1 行だけだった。

### 採用の境界の両側（T005 が決めた）

`RFC7643-ENTERPRISE-EXTENSION` の採用外の属性は、入口によって扱いが違う。この非対称そのものが境界である。

| 入口 | 採用外の属性 (`costCenter`、`division`、`organization`) の扱い |
|---|---|
| Discovery (`/Schemas`、`/ResourceTypes`) | 広告しない |
| 作成 (POST) と置換 (PUT) | 拒否せず、保存も応答もしない。要求は 201 / 200 で通り、採用した 3 属性だけが往復する |
| PATCH | `invalidPath` の 400 で拒否する。PATCH の path は allowlist に閉じているため |

片側だけでは、拡張を全部採用している実装とも、何も採用していない実装とも区別できない。3 つを 1 つのテストで続けて観測する。

### 意図した RED 検査

- **Acceptance RED**: `mise run check-spec`。5 件を `tools/check/standards-coverage-debt.json` から先に外し、テストを書く前の状態で走らせる。5 行それぞれについて `<ID> is declared, but no test names it.` が出ることを観測する。製品の規範要求に対応する受入境界はこの作業に無いので、Requirement は N/A である。
- **Unit RED**: 各行に対応付けたテストへの故障注入。行が言っていることを production 側で崩し、対応するテストが落ちることを 1 件ずつ観測する。`checkNormativeCoverage` は文字列の一致しか見ないので、名指しが実在の検証に付いていることの担保はこの観測しかない。recipe は `mise run test-go-test -- ./backend/sourcing/scim/handlers_http <Test>` と `mise run test-go-package -- ./backend/sourcing/...`。

## Plan

1. Provisioning の 12 行がどう名指されているかを読み、注記の書き方と置き場所を揃える。
2. `RFC7643-SERVICE-PROVIDER-CONFIG` を、生成された `ServiceProviderConfig` を読む形で消化する。
3. `RFC7644-RESOURCE-OPERATIONS` を消化する。
4. `RFC7644-BEARER-AUTHORIZATION` を、拒否が防いだ効果まで含めて消化する。
5. `RFC7644-ERROR-RESPONSE` を、応答本体の形まで読む形で消化する。
6. `RFC7643-ENTERPRISE-EXTENSION` を、採用の境界の両側で消化する。
7. 解決した id を台帳から外す。

## Tasks

- [x] T001 [Inventory] Provisioning 側の 12 行の名指しを読み、注記の書き方と置き場所を決める。
  Provisioning は `conformance_test.go` と `client_test.go` に、行 1 つにつき 1 つの試験を置き、`// <ID>: <何を固定しているか>` を試験の直前に書いている。Sourcing でも同じ形を採り、置き場所は行が指す入口に合わせて `backend/sourcing/scim/handlers_http/standards_test.go` に集めた。5 行すべてが SCIM の HTTP 入口を指すので、1 ファイルで足りる。既存テストが行の一部を固定している 2 箇所には、その範囲を書いた注記を足した。棚卸しの結果は Design の「T001 の棚卸し」節で、**注記だけで済む行は 1 つも無かった**。
- [x] T002 [Acceptance] `RFC7643-SERVICE-PROVIDER-CONFIG` と `RFC7644-RESOURCE-OPERATIONS` を消化する。
  `TestScimServiceProviderConfig_AdvertisesTheBearerAuthenticationScheme` と `TestScimResourceOperations_UsersAndGroupsSupportCreateReadReplaceDelete`。後者は User と Group の 4 操作を HTTP 境界で通し、置換は**別の要求で読み直して**応答の組み立てで終わっていないことを確かめる。再実行の recipe は `mise run test-go-test -- ./backend/sourcing/scim/handlers_http <Test>`。
- [x] T003 [Acceptance] `RFC7644-BEARER-AUTHORIZATION` を、拒否が防いだ効果まで消化する。
  `TestScimBearerAuthorization_RefusesAndLeavesTheResourceUnchanged`。5 通りの資格情報 (header 無し、Bearer 以外、未発行、別テナント、書き込みスコープ無し) それぞれで、置換・無効化・削除の 3 要求を送り、拒否と challenge を見たうえで**リソースを読み直して 3 つとも届いていないこと**を確かめる。経路ごとのスコープ対応は `TestScimRoutesRequireOperationScope` が持ち、そちらにも注記を足した。
- [x] T004 [Acceptance] `RFC7644-ERROR-RESPONSE` を、応答本体の形まで消化する。
  `TestScimErrorResponse_FailuresCarryTheScimErrorBody`。認証・認可・構文・パス・フィルター・不在・衝突の 7 通りの失敗について、`schemas`・`status`・`detail` と `Content-Type: application/scim+json` を読む。
- [x] T005 [Acceptance] `RFC7643-ENTERPRISE-EXTENSION` を、採用の境界の両側で消化する。
  `TestScimEnterpriseExtension_AdoptsOnlyTheDeclaredSubset`。境界の形は Design の「採用の境界の両側」節。採用外の属性の扱いが入口ごとに違う (Discovery は広告せず、作成は黙って落とし、PATCH は拒否する) ことが分かったので、3 つとも観測している。
- [x] T006 [Ledger] 解決した id を `standards-coverage-debt.json` から外す。
  5 件を削除し、台帳は 11 件から 6 件になった。コメント配列と他の id には触れていない。
- [x] T007 [Verify] `mise run verify`。
  exit 0。詳細は Completion の Verification Results。

## Verification

- 本項目が持つ 5 件が `tools/check/standards-coverage-debt.json` から 1 件残らず消えている。
- 注記を足した各テストについて、対応する production の判断を崩すとそのテストが落ちる。
- `mise run check-spec`
- `mise run verify`

## Completion

- **Completed At**: 2026-09-12
- **Summary**:
  `mise run spec-diff` は `no normative specification change against main` を返す。規範の変更は無い。
  差分は `docs/contexts/sourcing/standards.md` の 5 行に対する被覆の状態である。5 行すべてが
  その行の `Statement` を区別できる入力と観測を持つテストを得て `tools/check/standards-coverage-debt.json`
  から消え、台帳は 11 件から 6 件になった。名指しを持つ id は 276 件から 281 件へ増えた。
  新設したテストは Go 5 件（部分試験を数えると 24 件）で、**製品コードは 1 行も変わっていない**。
  **既存テストが行を区別できていたのは 5 行のうち 1 行だけだった。** `RFC7643-SERVICE-PROVIDER-CONFIG`
  は `TestScimInboundProvisioning` の途中で `authenticationSchemes` を読んでおり、観測としては足りていた。
  残る 4 行は、Group の削除と単体参照が HTTP 境界に無く、拒否が防いだ効果を 1 件も観測しておらず、
  誤り応答の形を `scimType` 以外まったく読んでおらず、採用の境界が Discovery 側にしか無かった。
  **欠陥は 1 件も見つからなかった。** 5 行はいずれも宣言した採用を満たしている。
  **起票時の Motivation が入出力を取り違えていたので直した。** Sourcing は SCIM のサービス提供者として
  受け取る側であり、下流へ送るのは Provisioning である。入口の向きは、どこにテストを置くかを決める前提
  なので、そのままにはできない。
  既存テストを 1 件整理した。`TestScimInboundProvisioning` の 3 番目の区画にあった
  `authenticationSchemes` の確認は、行に対応する試験へ移した。同じ判断を強さの違う 2 つのテストが
  観測する状態を残さないためである。同区画はトークンが受け付けられることの確認として残る。
  **標準の行が言っていないことを 1 つ見つけた。** User の DELETE は soft delete なので、削除後も
  `GET /scim/v2/Users/{id}` は 200 を返す（Group は実削除なので 404 になる）。`RFC7644-RESOURCE-OPERATIONS`
  の `Statement` は「4 操作を提供する」までしか言っておらず、削除後の到達性を宣言していないので、
  この非対称は行に対する欠陥ではない。行の `Statement` を SCIM の適合範囲まで広げるかは規範の変更であり、
  本項目の Out of Scope である。テストは行が言う範囲、つまり削除が User の lifecycle に及んだことを観測する。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-spec`（5 件を台帳から外し、テストを書く前の状態で）
  - **Requirement**: N/A: 標準の被覆はテストの有無についての性質であり、製品の規範要求ではない。
  - **Observed Failure**: exit 1。`docs/contexts/sourcing/standards.md` の 5 行それぞれについて
    `<ID> is declared, but no test names it. Cite the id from the test that exercises it, or list it in
    tools/check/standards-coverage-debt.json with a reason.`（9 行目 `RFC7643-SERVICE-PROVIDER-CONFIG`、
    11 行目 `RFC7643-ENTERPRISE-EXTENSION`、19 行目 `RFC7644-RESOURCE-OPERATIONS`、
    21 行目 `RFC7644-BEARER-AUTHORIZATION`、22 行目 `RFC7644-ERROR-RESPONSE`）。報告はこの 5 行だけだった。
  - **Detection Reason**: 検査は、宣言された id・テストが名指す id・台帳の 3 つを突き合わせる。台帳から
    外した id は、名指すテストが実在しない限り必ず報告される。消化後の同じコマンドは
    `ok normative coverage (156 standard(s), 311 rule(s), 746 example(s), 281 id(s) named by a test)`
    を返す（276 → 281）。
- **Unit RED Evidence**:
  - **Test**: 各行に対応付けたテストへの故障注入（Change-Resistance Results を参照）
  - **Requirement**: N/A: 行ごとの観測は標準の `Statement` に対応し、`REQ` 番号には対応しない。
  - **Observed Failure**: 12 件の故障すべてを、対応するテストが検出した。テナント越えだけは 1 箇所の注入では
    検出されず、下表のとおり 2 箇所を同時に崩す必要があった。
  - **Detection Reason**: `checkNormativeCoverage` は文字列の一致しか見ないので、id を書くだけで検査は
    通ってしまう。名指しが実在の検証に付いていることの担保は、production を崩したときにそのテストが
    落ちるという観測しかない。
- **Change-Resistance Results**:
  5 行 12 観測すべてについて、行が言っていることを production 側で崩し、対応するテストが落ちることを
  観測した。故障は注入のたびに `git diff --stat` で着弾を確かめ、観測後に元へ戻している。

  | 行 | 注入した故障 | 落ちたテストと観測 |
  |---|---|---|
  | `RFC7643-SERVICE-PROVIDER-CONFIG` | 応答を書く直前に `config.AuthenticationSchemes = nil` を入れる | `TestScimServiceProviderConfig_AdvertisesTheBearerAuthenticationScheme`: `authenticationSchemes` が `nil` |
  | 〃 | 広告する方式の `Type` を `httpbasic` にする | 同テスト: `oauthbearertoken` の方式が見つからない |
  | `RFC7644-RESOURCE-OPERATIONS` | `GET /scim/v2/Users/:id` の経路を外す | `…/User`: 参照が 200 のはずが 404 |
  | 〃 | `DELETE /scim/v2/Groups/:id` の経路を外す | `…/Group`: 削除が 204 のはずが 404 |
  | 〃 | `UpdateGroup` が `group.Name` を書かず Save もせず、応答にだけ要求の `displayName` を載せる | `…/Group`: **応答側は通り、読み直しだけが落ちる**（`Lifecycle` のまま） |
  | 〃 | `DeleteUser` が `Lifecycle.Status` を書き換えない | `…/User`: 削除後の状態が `active` のまま |
  | `RFC7644-BEARER-AUTHORIZATION` | テナント照合を 2 箇所とも崩す（`FindByJTI` の `token.TenantID != tenantID` と handler の `reqTenantID != principal.TenantID`） | `…/a token issued to another tenant`: 別テナントのトークンで置換・無効化・削除が通り、`userName` と `active` が変わる |
  | 〃 | `authenticate` のスコープ検査を無効化する | `…/a token without the write scope`: 403 のはずが 200 / 204 で、リソースが変わる |
  | 〃 | `handleDeleteUser` が `authenticate` の**前に** `DeleteUser` を呼ぶ | 4 つの拒否の事例すべて: **状態符号と challenge は正しいまま、`active` だけが false になる** |
  | `RFC7644-ERROR-RESPONSE` | `NewScimError` が `schemas` を載せない | `TestScimErrorResponse_FailuresCarryTheScimErrorBody`: 7 事例すべてで `schemas` が `nil` |
  | 〃 | `writeScimError` が `Content-Type` を設定しない | 同テスト: `application/json` になる |
  | 〃 | `writeScimError` が `detail` に空文字を渡す | 同テスト: `detail` が空 |
  | `RFC7643-ENTERPRISE-EXTENSION` | `EnterpriseUserSchema` が `costCenter` を広告する | `…/Discovery advertises exactly the adopted attributes`: 広告が 4 属性になる |
  | 〃 | `userPatchAttrs` に `costcenter` を足す | `…/an unadopted attribute is refused by PATCH`: 400 invalidPath のはずが 200 |
  | 〃 | `toScimUser` が `ext["costCenter"]` を載せる | `…/an unadopted attribute is neither stored nor returned`: 採用外の属性が応答に出る |
  | 〃 | `ParseUserWrite` が `department` を読まない | `…/the adopted attributes are stored and returned`: `department` が往復しない |

  **テナント越えは 1 箇所の注入では検出されなかった。** handler の `reqTenantID != principal.TenantID` だけを
  外しても、`Service.Authenticate` が要求のテナントで token 記録を引くので、別テナントのトークンは
  そこで落ちる。テナント単位という性質は独立した 2 箇所が持っており、上表の観測は両方を崩す形で
  取っている。これは二重の防御であって、テストが弱いのではない。
- **Verification Results**:
  - `mise run verify` - passed（2026-09-12 に取得、exit 0）
  - `mise run check-spec` - passed
    (`ok normative coverage (156 standard(s), 311 rule(s), 746 example(s), 281 id(s) named by a test)`)
  - `mise run lint-go` - passed（0 issues）
  - `mise run test-go-package -- ./backend/sourcing/...` - passed
  - `mise run spec-diff` - `no normative specification change against main`
  - `mise run test-ui-e2e` - N/A: 変更は Go のテスト 3 ファイルと台帳だけで、製品コードもフロントエンドも
    1 行も動いていない。ブラウザーへ届く経路が無い。

## Risk Notes

- **`partial` を片側だけで消化する。** `RFC7643-ENTERPRISE-EXTENSION` は採用の境界を持つ行なので、採用した側だけを観測すると全部採用している実装と区別できない。境界の両側を観測する。
- **拒否をステータスだけで観測する。** `RFC7644-BEARER-AUTHORIZATION` は、変更してから拒否を返す実装をステータスでは区別できない。対象リソースを読み直す。
- **Provisioning の名指しを流用して Sourcing の入口を観測しない。** 同じ RFC でも入口が逆向きなので、Provisioning のテストが Sourcing の行を区別することはない。
- **台帳の同時編集。** 自分が持つ id のエントリーだけを削除し、コメント配列と他の id には触れない。
