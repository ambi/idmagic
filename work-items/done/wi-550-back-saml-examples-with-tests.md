---
depends_on: []
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-12
priority: p3
change_kind: maintenance
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: 宣言済みの具体例に検証を与える作業であり、併せて直す欠陥も、契約が既に宣言している 400 を 500 の代わりに返すようにするものである。公開契約も運用手順も変わらない。
  references: []
initial_context:
  specification:
    - docs/contexts/saml/scenarios.feature.md#REQ-SAML-001
    - docs/contexts/saml/scenarios.feature.md#REQ-SAML-002
    - docs/contexts/saml/scenarios.feature.md#REQ-SAML-003
    - docs/contexts/saml/scenarios.feature.md#REQ-SAML-004
    - docs/contexts/saml/scenarios.feature.md#REQ-SAML-005
    - docs/contexts/saml/scenarios.feature.md#REQ-SAML-006
    - docs/contexts/saml/scenarios.feature.md#REQ-SAML-007
    - docs/contexts/saml/scenarios.feature.md#REQ-SAML-008
  typespec:
    - IdMagic.Saml.Operations.RegisterSamlServiceProvider
    - IdMagic.Saml.Operations.ListSamlServiceProviders
    - IdMagic.Saml.Operations.DeleteSamlServiceProvider
    - IdMagic.Saml.Operations.CreateSamlIdentityProviderProfile
    - IdMagic.Saml.Operations.UpdateSamlIdentityProviderProfile
    - IdMagic.Saml.Operations.DeleteSamlIdentityProviderProfile
    - IdMagic.Saml.Operations.PublishSamlMetadata
    - IdMagic.Saml.Operations.DownloadSamlSigningCertificate
    - IdMagic.Saml.Operations.SamlSingleSignOn
  source:
    - backend/saml/handlers_http
    - backend/saml/usecases
    - backend/saml/domain
    - backend/saml/db_memory
    - backend/shared/http/support_http/admin_scope.go
    - backend/shared/http/support_http/auth.go
    - tools/check/example-coverage-debt.json
    - tools/check/src/normative-coverage.ts
  tests:
    - backend/saml/handlers_http
    - backend/saml/domain
    - backend/shared/http/support_http/admin_scope_test.go
    - backend/shared/http/server_http/api_token_standards_test.go
    - frontend/src/features/admin-saml-idp-profiles/AdminSamlIDPProfilesPages.test.tsx
  stop_before_reading:
    - backend/saml/db_postgres
    - backend/application
    - infra
affected_spec:
  - { path: docs/contexts/saml/scenarios.feature.md, requirement: REQ-SAML-001 }
  - { path: docs/contexts/saml/scenarios.feature.md, requirement: REQ-SAML-002 }
  - { path: docs/contexts/saml/scenarios.feature.md, requirement: REQ-SAML-003 }
  - { path: docs/contexts/saml/scenarios.feature.md, requirement: REQ-SAML-004 }
  - { path: docs/contexts/saml/scenarios.feature.md, requirement: REQ-SAML-005 }
  - { path: docs/contexts/saml/scenarios.feature.md, requirement: REQ-SAML-006 }
  - { path: docs/contexts/saml/scenarios.feature.md, requirement: REQ-SAML-007 }
  - { path: docs/contexts/saml/scenarios.feature.md, requirement: REQ-SAML-008 }
  - { path: spec/contexts/saml/main.tsp, symbol: IdMagic.Saml.Operations.RegisterSamlServiceProvider }
---

# Saml が宣言する具体例 18 件にテストを対応付け、被覆台帳から外す

## Motivation

[[wi-496-burn-down-the-example-coverage-debt]] は具体例の被覆台帳の消化単位を Context と決め、測定のうえで残りを Context ごとの子 work item へ割った。本項目はそのうち `docs/contexts/saml/scenarios.feature.md` が宣言する 18 件を引き取る。

親項目が `claim-mapping` の 3 件と `workloadidentity` の 13 件で測った結果は、**16 件のうち注記だけで済んだのは 4 件だけ**だというものである。3 件はテストが 1 つも無く、9 件は既存テストへ新しい観測を足す必要があった。件数は作業量の目安にならない。

## Scope

- 18 件を 1 件ずつ確認し、次のいずれかに解決して `tools/check/example-coverage-debt.json` から外す。
  - 当の具体例を検証しているテストが実在する → そのテストに `// EX-SAML-NNN-MM: <この具体例の何を固定しているか>` の注記を足す。
  - 当の具体例を検証しているテストが無い → 書く。具体例が拒否なら、[[wi-392-refusal-tests-assert-the-absent-effect]] が定める形（拒否応答と、拒否が防いだ効果の双方を観測する）で書く。
  - 具体例が宣言されなくなっている → 台帳から外す（検査が落ちて教える）。
- 注記は「何を固定しているか」を書く。id だけの注記は禁止する。
- **実装が具体例のとおりに振る舞っていない件のうち、修正が局所で済むものは本項目で直す。** 判定の基準は、直す対象が既存の写像や規則の欠落であり、設計上の判断をやり直さずに済むことである。基準に当てはまらないものは欠陥として切り出す。

## Out of Scope

- 他の Context が宣言する具体例。Context ごとに別の work item が持つ。
- `tools/check/standards-coverage-debt.json`。[[wi-495-burn-down-the-standards-coverage-debt]] が消化済みである。
- 既に台帳に載っていない拒否テストが、防いだ効果を観測していない件。[[wi-392-refusal-tests-assert-the-absent-effect]] が持つ。本項目が新しく書く拒否テストは、その規範を満たす形で書く。既存テストに注記を足すだけの件で、そのテストが効果の不在を見ていない場合は、注記を足したうえで wi-392 の対象として残す。
- シナリオと具体例の追加、削除、書き換え。具体例の記述が実装と食い違い、かつ実装のほうが正しいと判断した場合は規範の変更なので、別の work item が扱う。
- 設計上の判断をやり直す必要がある欠陥の修正。切り出した先で扱う。
- `backend/application` が持つ、同じリポジトリ保存経路に対する重複したエラー写像の整理。
- 行カバレッジ率の目標または閾値。
- `report-coverage-debt` の分類（`named`、`nearby`、`none`）を根拠にした台帳からの削除。分類は読む順を決める材料であり、台帳から外す根拠にはならない。

## Design

### 消化の対象と観測の位置

18 件は 4 つの入口に分かれる。入口ごとに 1 つのテストファイルへ寄せ、具体例 1 件に注記 1 つを対応させる。

| 入口 | 具体例 | 観測の位置 |
| --- | --- | --- |
| ブラウザーの SAML エンドポイント | EX-SAML-001-01、002-01、003-01、003-02、006-01〜06、007-01、008-01 | `backend/saml/handlers_http`（`httpadapter.Register` が組み立てた `/saml/*`） |
| SAML の管理 API | EX-SAML-004-01、004-02、004-03 | `backend/saml/handlers_http`（`/api/admin/v1/saml/*`） |
| API アクセストークンの粒度スコープ | EX-SAML-005-01、005-02、005-03 | `backend/shared/http/server_http`（実トークンを発行して管理 API を叩く既存スタック） |
| 管理画面 | EX-SAML-004-01 の画面遷移 | `frontend/src/features/admin-saml-idp-profiles` |

`Then` の数だけ観測を置く。既存テストが `Then` の一部しか見ていない場合は、足りない観測を足してから注記する。

### 直す欠陥

`POST /api/admin/v1/saml/service-providers` は、`dedicated` プロファイルを 2 つ目の SP へ割り当てる要求に対して 500 を返す。リポジトリが返す `samldomain.ErrDedicatedIDPProfileCardinality` を `handleUpsertServiceProvider` が写像せずそのまま返すためで、契約 `RegisterSamlServiceProvider` は 500 を宣言していない。

同じパッケージの `writeIDPProfileError` が既にこの写像を持ち、`backend/application` の同じ保存経路も 400 `invalid_request` に写像している。したがって修正は写像の欠落を埋めるだけであり、設計上の判断をやり直さない。

欠陥は adapter だけにある。基数の規則そのものは `SamlIdentityProviderProfile.Validate` が持ち、`db_memory` のリポジトリが保存前に適用していて、どちらも既にテストがある。したがって単体境界に RED は無く、RED は HTTP 境界にだけ現れる。`change_kind` を `bugfix` ではなく `maintenance` にしてあるのは、この項目の意味的な変更が「宣言済みの具体例にテストを対応付ける」ことであり、欠陥の修正がその過程で見つかった写像 1 行の欠落だからである。

### 直さない食い違い

EX-SAML-005-03 は「トークンのテナントとリクエスト先のテナントが一致しない」ときの拒否を `AccessDeniedError` と言う。実装は 401 `invalid_token` を返す。これは写像の欠落ではない。管理発行トークンの照合は `FindByJTI(ctx, リクエスト先テナント, jti)` で行い、見つからなければ RFC 7662 に従って検証の内訳を漏らさず `active: false` を返す設計であり、`usecases.go` にその旨の注記が置かれている。403 へ変えることは、拒否の理由を「このテナントには無いトークンである」と外へ伝えることになる。

具体例の記述のほうを直すのが筋なので、規範の変更として切り出す。EX-SAML-005-03 は台帳に残す。

## Plan

1. 入口ごとに、具体例 1 件につき当該 id を名指しした検査が落ちることを確認してから台帳の当該行を消す。
2. 欠陥は先に RED を観測してから直す。
3. 台帳の削除は 1 件ずつ行い、`mise run check-spec` がその id を名指しで落とすことを削除の前に見る。

## Tasks

- [x] T001 [Acceptance] `mise run check-spec` が SAML の 18 件を名指しで落とすことを、消化前に観測する。
- [x] T002 [Adapters] ブラウザー入口の 11 件を消化する。
- [x] T003 [Adapters] 管理 API の 3 件を消化し、EX-SAML-004-02 の欠陥を RED → GREEN で直す。
- [x] T004 [Adapters] 粒度スコープの 2 件を消化する。
- [x] T005 [UI] 管理画面のテストに EX-SAML-004-01 を注記する。
- [x] T006 [Verify] `mise run verify`。

## Verification

- `mise run check-spec` が、`docs/contexts/saml/scenarios.feature.md` の消化済みの件を `tools/check/example-coverage-debt.json` から外した状態で通る。台帳から外す前に同じ検査が当の id を名指しで落とすことを、消化ごとに観測する。
- 消化したテストの所属パッケージに対する `mise run test-go-package -- <package>`。
- `mise run test-ui-unit-file -- frontend/src/features/admin-saml-idp-profiles/AdminSamlIDPProfilesPages.test.tsx`
- `mise run verify`

## Risk Notes

- **注記だけを足して終わる。** `checkNormativeCoverage` は文字列の一致しか見ないので、読まずに id を貼れば件数は速く減る。減った件数は何も意味しない。注記に「この具体例の何を固定しているか」を書かせることで区別する。親項目の測定では、16 件のうち注記だけで済んだのは 4 件だけだった。
- **`nearby` に分類された件を、テストがある証拠として読む。** 分類が言っているのは「近傍に他の拒否テストがある」だけである。親項目の測定では、`claim-mapping` の 3 件はすべて `nearby` でありながら 2 件はテストが無かった。分類は読む順の材料にとどめる。
- **拒否である具体例のテストが、応答の字面だけを見て書かれる。** 新しく書く拒否テストには [[wi-392-refusal-tests-assert-the-absent-effect]] の規範が効く。注記へ「効果の不在を何で観測したか」を書き、後から区別できるようにする。
- **具体例の `Then` が複数あるのに、観測が 1 つで済まされる。** 親項目の測定では、`EX-WORKLOADIDENTITY-008-01` のようにイベントと実際の効果の双方を言う具体例が複数あった。`Then` の数だけ観測が要る。
- **欠陥の修正が、テストを緑にするための最短経路に流れる。** 拒否の応答形を変える修正は、拒否そのものを弱めても緑になる。EX-SAML-004-02 のテストは、応答が 400 であることと、2 つ目の SP が一覧に現れないことの双方を読む。

## Completion

- **Completed At**: 2026-09-12
- **Summary**:
  `mise run spec-diff` は `no normative specification change against main` を返す。規範は 1 行も動いていない。
  変わったのは、`docs/contexts/saml/scenarios.feature.md` が宣言する 18 件のうち 17 件が、
  その id を名指しするテストから到達されるようになったことと、
  `POST /api/admin/v1/saml/service-providers` が IdP プロファイルの基数違反へ返す状態コードが 500 から
  契約どおりの 400 `invalid_request` になったことである。台帳は 598 件から 581 件へ減った。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-spec`（`checkNormativeCoverage`）
  - **Requirement**: REQ-SAML-001
  - **Observed Failure**: 同じ検査が REQ-SAML-001 から REQ-SAML-008 までの具体例を同時に名指しした。SAML の 18 件を `tools/check/example-coverage-debt.json` から外した状態で exit 1。
    18 行それぞれが `EX-SAML-NNN-MM is declared, but no test names it.` の形で id を名指しした。
  - **Detection Reason**: この検査は宣言された id と、テスト本文が名指しした id の集合を比べる。
    テストを書かずに台帳から外せば必ず落ちるので、「消化した」と「台帳から消した」を取り違えられない。
- **Unit RED Evidence**:
  - **Test**: `TestAdminServiceProviderRefusesASecondBindingToADedicatedProfile`
    (`backend/saml/handlers_http/scenario_examples_test.go`)
  - **Requirement**: REQ-SAML-004
  - **Observed Failure**: `2 つ目の割り当て status=500 body={"type":"urn:idmagic:error:internal_server_error"...}, want 400`。
    サーバーログには `unhandled request error: dedicated SAML identity provider profile can be assigned to only one service provider` が出ていた。
  - **Detection Reason**: 契約 `RegisterSamlServiceProvider` は 400 `InvalidRequestError` を宣言し、500 を宣言していない。
    テストは状態コードと problem の型に加えて、拒否のあと 2 つ目の SP が一覧に現れないことを読む。
    拒否を書いてから保存する実装は、状態コードだけでは見分けられない。
    基数の規則そのものは `SamlIdentityProviderProfile.Validate` と `db_memory` のリポジトリが既に持っていて、
    どちらも既存テストで緑だった。欠陥は adapter のエラー写像だけにあり、単体境界に RED は無かった。
- **Change-Resistance Results**:
  リスクは medium なので、消化したテストが実際に何を検出するかを、代表的な 5 つの故障注入で確かめた。5 件とも検出された。
  1. `handleUpsertServiceProvider` の写像を外して元の実装へ戻す →
     `TestAdminServiceProviderRefusesASecondBindingToADedicatedProfile` が 500 で落ちる（上の Unit RED そのもの）。
  2. メタデータの公開を `certs[:1]` にして移行期間中の旧証明書を落とす →
     `TestSamlSigningCertificateIsTheActiveCredentialAndMetadataCarriesEveryTrustedOne` が落ちる。
  3. `issueResponse` の署名鍵解決をプロファイル指定からデフォルト固定へ変える →
     `TestSamlSSOIssuesWithTheAssignedProfileEntityIDAndCredentials` が
     `profile-a の証明書で assertion 署名が検証できない` で落ちる。
  4. `SignInService.Issue` のリプレイ判定の結果を無視する →
     `TestSamlSSORefusesAReplayedAuthnRequestIDAndIssuesNoSecondAssertion` が 2 回目の 200 で落ちる。
  5. `requireAdminApiTokenScope` を素通しにする →
     `TestSamlServiceProviderOperationsFollowTheGranularScopes` と
     `TestSamlReadScopeCannotChangeServiceProviders` が、拒否されるはずの登録が 201 で通ることで落ちる。
  6. `db_memory` の `FindIDPProfileByID` をテナント非依存にする →
     `TestSamlProfileEndpointsRefuseAnUnknownOrForeignProfileID` が、別テナントのプロファイルの
     メタデータが 200 で返ることで落ちる。
- **Verification Results**:
  - `mise run verify` - passed
  - `mise run check-spec` - passed（SAML の残りは `EX-SAML-005-03` の 1 件）
  - `mise run test-ui-e2e` - 実行していない。製品のフロントエンドは 1 行も変わっておらず、
    変えた管理 API を画面は呼んでいない（`rg 'saml/service-providers' frontend/src` は 0 件）。

### 消化した具体例とテストの対応

| 具体例 | テスト |
| --- | --- |
| EX-SAML-001-01 | `TestSamlSigningCertificateIsTheActiveCredentialAndMetadataCarriesEveryTrustedOne` |
| EX-SAML-002-01 | `TestSamlSSOIssuesWithTheAssignedProfileEntityIDAndCredentials` |
| EX-SAML-003-01 | `TestSamlDedicatedProfilePublishesItsOwnEndpointsAndSigningCredential` |
| EX-SAML-003-02 | `TestSamlProfileEndpointsRefuseAnUnknownOrForeignProfileID` |
| EX-SAML-004-01 | `TestAdminManagesSharedAndDedicatedIDPProfilesEndToEnd`、`AdminSamlIDPProfilesPages.test.tsx` |
| EX-SAML-004-02 | `TestAdminServiceProviderRefusesASecondBindingToADedicatedProfile` |
| EX-SAML-004-03 | `TestAdminIDPProfileDeletionIsRefusedWhileReferencedOrDefault` |
| EX-SAML-005-01 | `TestSamlServiceProviderOperationsFollowTheGranularScopes` |
| EX-SAML-005-02 | `TestSamlReadScopeCannotChangeServiceProviders` |
| EX-SAML-006-01 | `TestSamlSSO_SPInitiatedAuthenticatedIssuesPostForm` |
| EX-SAML-006-02 | `TestSamlSSOFailsClosedOnEveryInvalidRequestDimension` |
| EX-SAML-006-03 | `TestSamlSSORejectsUnparsableAndUnverifiableAuthnRequests` |
| EX-SAML-006-04 | `TestSamlWebBrowserSSOFailsClosedOnUnsupportedRequestParameters` |
| EX-SAML-006-05 | `TestSamlWebBrowserSSOReturnsNoPassiveWhenLoginIsRequired` |
| EX-SAML-006-06 | `TestSamlSSORefusesAReplayedAuthnRequestIDAndIssuesNoSecondAssertion` |
| EX-SAML-007-01 | `TestSamlSSOFailsClosedOnEveryInvalidRequestDimension` |
| EX-SAML-008-01 | `TestSamlSSO_ForceAuthnWithStaleSessionRedirectsToLogin` |

### 台帳に残した 1 件

`EX-SAML-005-03` は残した。トークンのテナントとリクエスト先のテナントが一致しないとき、
具体例は `AccessDeniedError` と言うが、製品は 401 `invalid_token` を返す。
`acme` レルムで発行した `saml:read` と `saml:write` のトークンを `default` レルムの
`/api/admin/v1/saml/service-providers` へ提示して測った。参照も登録も 401 で、保存先には何も残らない。
拒否そのものは効いていて、食い違っているのは型だけである。

401 の側には理由がある。管理発行トークンの照合は `AuthenticateClaims` が
`FindByJTI(ctx, リクエスト先テナント, jti)` で行い、見つからなければ RFC 7662 に従って検証の内訳を
漏らさず `active: false` を返す。403 へ変えれば「このトークン自体は有効だが、ここでは使えない」ことを
提示者へ伝えることになる。写像の欠落ではなく規範と実装のどちらを正とするかの判断なので、
本項目の Out of Scope に当たる。[[wi-558-name-the-refusal-a-cross-tenant-api-token-actually-gets]] が扱う。

### 親項目の測定への追加

親項目 [[wi-496-burn-down-the-example-coverage-debt]] は「注記だけで済むのは少数」と測った。SAML でも同じだった。
18 件のうち、既存テストへ注記を足すだけで済んだのは `EX-SAML-006-01` の 1 件だけである
（それも `InResponseTo` の観測を 1 つ足した）。
既存テストがあっても観測が足りなかったのが 3 件（`006-04`、`006-05`、`008-01`）、
テストを新しく書いたのが 12 件、規範との食い違いで残したのが 1 件、
[[wi-487-back-saml-declared-refusals-with-effect-tests]] が済ませていたのが 3 件である。

`report-coverage-debt` の分類は今回も当てにならなかった。SAML は `named` 1 件、`nearby` 17 件と出ていたが、
実際にはテストを新しく書く必要のある件が 12 件あった。分類は読む順の材料にとどまる。
