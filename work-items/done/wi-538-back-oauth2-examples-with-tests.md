---
depends_on: []
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-12
priority: p3
change_kind: maintenance
spec_impact: { kind: none, reason: "宣言済みの具体例に、その id を名指しするテストを対応付ける作業である。シナリオも製品の振る舞いも変えない。テストが書けない具体例が見つかった場合、それは実装が具体例のとおりに振る舞っていないということなので、欠陥として個別の work item に切り出す。" }
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: 宣言済みの具体例に検証を与える作業であり、公開契約も運用手順も変わらない。
  references: []
initial_context:
  specification:
    - docs/contexts/oauth2/scenarios.feature.md#REQ-OAUTH2-001
    - docs/contexts/oauth2/scenarios.feature.md#REQ-OAUTH2-002
    - docs/contexts/oauth2/scenarios.feature.md#REQ-OAUTH2-003
    - docs/contexts/oauth2/scenarios.feature.md#REQ-OAUTH2-004
  typespec: []
  source:
    - backend/oauth2/handlers_http/routes.go
    - backend/oauth2/handlers_http/admin_role_policy_handler.go
    - backend/apitoken/domain/token.go
    - tools/check/example-coverage-debt.json
    - tools/check/src/check-documents.ts
    - tools/check/src/normative-coverage.ts
  tests:
    - backend/oauth2/handlers_http/admin_role_policy_handler_test.go
    - backend/oauth2/handlers_http/refusal_effects_test.go
    - backend/oauth2/handlers_http/endpoint_refusal_effects_test.go
    - backend/shared/http/server_http/api_token_standards_test.go
    - backend/shared/http/server_http/saml_scope_examples_test.go
  stop_before_reading:
    - backend/oauth2/db_postgres
    - frontend
    - infra
---

# OAuth2 のスコープとロールポリシーが宣言する具体例 6 件にテストを対応付け、残る 99 件を規則群ごとに割る

## Motivation

[[wi-496-burn-down-the-example-coverage-debt]] は具体例の被覆台帳の消化単位を Context と決め、測定のうえで残りを Context ごとの子 work item へ割った。本項目はそのうち `docs/contexts/oauth2/scenarios.feature.md` が宣言する 105 件を引き取った。

親項目が `claim-mapping` の 3 件と `workloadidentity` の 13 件で測った結果は、**16 件のうち注記だけで済んだのは 2 件だけ**だというものである。残る 14 件は、既存テストへ新しい観測を足すか、テスト自体を書く必要があった。件数は作業量の目安にならない。

**105 件は全体の 18 % を占め、単独の Context として最大である。** 本項目はまず 1 規則群を通しで消化して 1 件あたりの所要を測り、**残りをさらに子 work item へ割るかどうかを自分で決める。** 親が測る前に割り方を決めないのと同じ理由で、この判断は件数ではなく測定に従う。

測定の結果、本項目は残りを 5 つの子 work item へ割った。根拠と割り方は Design に書く。

## Scope

- REQ-OAUTH2-002、REQ-OAUTH2-003、REQ-OAUTH2-004 が宣言する 6 件を 1 件ずつ確認し、次のいずれかに解決して `tools/check/example-coverage-debt.json` から外す。
  - 当の具体例を検証しているテストが実在する → そのテストに `// EX-OAUTH2-NNN-MM: <この具体例の何を固定しているか>` の注記を足す。
  - 当の具体例を検証しているテストが無い → 書く。具体例が拒否なら、[[wi-392-refusal-tests-assert-the-absent-effect]] が定める形（拒否応答と、拒否が防いだ効果の双方を観測する）で書く。
  - 具体例が宣言されなくなっている → 台帳から外す（検査が落ちて教える）。
- 注記は「何を固定しているか」を書く。id だけの注記は禁止する。
- 実装が具体例のとおりに振る舞っていないことが分かった場合は、**本 work item では直さず欠陥として切り出す**。
- 105 件をさらに分割するかどうかの判断を本項目が持つ。最初の 1 規則群を消化してから決め、Design へ書く。分割すると決めた場合、子 work item を起こすところまでを本項目が持つ。

## Out of Scope

- 分割した先の子 work item が引き取る 99 件。[[wi-559-back-oauth2-authorization-code-examples-with-tests]]、[[wi-560-back-oauth2-registration-and-logout-examples-with-tests]]、[[wi-561-back-oauth2-client-administration-examples-with-tests]]、[[wi-562-back-oauth2-backchannel-approval-examples-with-tests]]、[[wi-563-back-oauth2-delegation-and-agent-examples-with-tests]] が持つ。
- 他の Context が宣言する具体例。Context ごとに別の work item が持つ。
- `tools/check/standards-coverage-debt.json`。[[wi-495-burn-down-the-standards-coverage-debt]] が消化済みである。
- 既に台帳に載っていない拒否テストが、防いだ効果を観測していない件。[[wi-392-refusal-tests-assert-the-absent-effect]] が持つ。本項目が新しく書く拒否テストは、その規範を満たす形で書く。既存テストに注記を足すだけの件で、そのテストが効果の不在を見ていない場合は、注記を足したうえで wi-392 の対象として残す。
- シナリオと具体例の追加、削除、書き換え。具体例の記述が実装と食い違うことが判明した場合は規範の変更なので、別の work item が扱う。
- 見つかった実装の欠陥の修正。切り出した先で扱う。
- 行カバレッジ率の目標または閾値。
- `report-coverage-debt` の分類（`named`、`nearby`、`none`）を根拠にした台帳からの削除。分類は読む順を決める材料であり、台帳から外す根拠にはならない。

## Design

### 消化した規則群と観測の位置

最初の規則群として REQ-OAUTH2-001 から REQ-OAUTH2-004 を選んだ。この 4 規則はどれも「誰が何に届くか」を言っていて、観測の位置が既存のトークンスタックと一致するためである。

| 入口 | 具体例 | 観測の位置 |
| --- | --- | --- |
| OAuth2 の管理 API と API アクセストークンの粒度スコープ | EX-OAUTH2-003-01、003-02、003-03 | `backend/shared/http/server_http`（実トークンを発行して管理 API を叩く既存スタック） |
| account 同意 API | EX-OAUTH2-002-01 | `backend/authentication/handlers_http`（拒否の側を既に見ている fixture） |
| 管理 API のロールポリシー | EX-OAUTH2-004-01 | `backend/oauth2/handlers_http` |
| 認可コード + PKCE から account リソースサーバーまで | EX-OAUTH2-001-01 | 本項目では消化しない（下記） |

`Then` の数だけ観測を置く。既存テストが `Then` の一部しか見ていない場合は、足りない観測を足してから注記する。

### 分割の判断

**割る。** 105 件は 1 つの記録に収まらない。根拠は所要の測定である。

| 具体例 | 消化に要したもの |
| --- | --- |
| EX-OAUTH2-002-01 | 新規テスト 1 本と、主体を指定して同意を読み直す helper |
| EX-OAUTH2-003-01 | 新規テスト 1 本と、共有 fixture への保存先 2 つの追加と配線 |
| EX-OAUTH2-003-02 | 新規テスト 1 本 |
| EX-OAUTH2-003-03 | 新規テスト 1 本 |
| EX-OAUTH2-003-04 | 規範と実装の食い違い。台帳に残す |
| EX-OAUTH2-004-01 | 新規テスト 1 本 |

**6 件のうち、既存テストへの注記だけで済んだものは 1 件も無かった。** 親項目の測定 (16 件中 2 件) より悪い。しかもこの群は、観測の位置が 1 つの HTTP 境界に揃っていて、トークンを発行するスタックが既にある、最も条件の良い群である。認可コードの完全な流れ (REQ-OAUTH2-005)、バックチャネル承認の状態機械 (REQ-OAUTH2-041 から 043)、委譲と Agent (REQ-OAUTH2-044 から 050) は、いずれも fixture を建てるところから要る。

1 記録 1 コミットである以上、この所要で 100 件を 1 つの差分に入れると、読み手が差分から記録を復元できなくなる。

### 割り方

規則群の境界は、同じ fixture を必要とする範囲で引く。件数では引かない。

| 子 work item | 規則 | 件数 | 群としてまとめた理由 |
| --- | --- | --- | --- |
| [[wi-559-back-oauth2-authorization-code-examples-with-tests]] | REQ-OAUTH2-001、005〜014 | 20 | 認可コード + PKCE から `/token` までの 1 本の流れと、その上に載る PAR、DPoP、イントロスペクション、UserInfo、Discovery |
| [[wi-560-back-oauth2-registration-and-logout-examples-with-tests]] | REQ-OAUTH2-016〜027 | 17 | 動的登録、メタデータ取得、失効、ログアウト、デバイス認可。いずれもクライアントの生存期間とセッションの終了 |
| [[wi-561-back-oauth2-client-administration-examples-with-tests]] | REQ-OAUTH2-029〜037 | 10 | メタデータ、同意管理、クライアントとシークレットの管理 API |
| [[wi-562-back-oauth2-backchannel-approval-examples-with-tests]] | REQ-OAUTH2-041〜043 | 23 | 1 つの承認リクエストの状態機械を 3 方向から言う 3 規則 |
| [[wi-563-back-oauth2-delegation-and-agent-examples-with-tests]] | REQ-OAUTH2-044〜050 | 29 | 保護リソースの提示、委譲の深さ、Agent の再発行 |

`EX-OAUTH2-001-01` を wi-559 へ入れたのは、この具体例が `/token` での交換から account リソースサーバーの参照までを言っていて、REQ-OAUTH2-005 と同じ fixture を必要とするためである。同じ組み立てを 2 つの記録で別々に建てない。

`EX-OAUTH2-003-04` は台帳に残す。理由は下記の食い違いにあり、[[wi-564-name-the-refusal-a-cross-tenant-oauth-admin-token-gets]] が引き取る。

### 直さない食い違い

`EX-OAUTH2-003-04` は「トークンのテナントとリクエスト先のテナントが一致しない」ときの拒否を `AccessDeniedError` と言う。実装は 401 `invalid_token` を返す。

`acme` レルムで発行した `oauth-clients:read` と `oauth-clients:write` のトークンを `default` レルムの `/api/admin/v1/clients` へ提示して測った。参照も登録も 401 で、`WWW-Authenticate` は `Bearer error="invalid_token"`、`default` テナントのクライアント一覧は動かない。拒否そのものは効いていて、食い違っているのは型だけである。

これは写像の欠落ではない。管理発行トークンの照合は `FindByJTI(ctx, リクエスト先テナント, jti)` で行い、見つからなければ RFC 7662 に従って検証の内訳を漏らさない設計である。403 へ変えることは、拒否の理由を「このテナントには無いトークンである」と外へ伝えることになる。

[[wi-550-back-saml-examples-with-tests]] が `EX-SAML-005-03` について測ったものとまったく同じ食い違いであり、判断は 1 つで対象の具体例が 2 つある。SAML 側の判断は [[wi-558-name-the-refusal-a-cross-tenant-api-token-actually-gets]] が持つので、OAuth2 側はその結論を適用する記録を起こした。

## Plan

1. 台帳から 105 件すべてを外した状態で `mise run check-spec` を走らせ、105 行が 1 件ずつ id を名指して落ちることを見る。
2. 最初の規則群を入口ごとに消化し、消化した件だけを台帳から消す。
3. 消化した所要を測り、割るかどうかを決めて Design へ書く。
4. 割ると決めたら、子 work item を起こしてから本項目を完了させる。

## Tasks

- [x] T001 [Acceptance] `mise run check-spec` が OAuth2 の 105 件を名指しで落とすことを、消化前に観測する。
- [x] T002 [Adapters] 粒度スコープの 3 件 (EX-OAUTH2-003-01、003-02、003-03) を消化する。`mise run test-go-test -- ./backend/shared/http/server_http 'TestOAuth2(AdminOperationsFollowTheGranularScopes|ReadScopeCannotChangeOAuth2Clients|ScopeOfAnotherResourceCannotReachTheOperation)'`。
- [x] T003 [Adapters] account 同意の 1 件 (EX-OAUTH2-002-01) を消化する。`mise run test-go-test -- ./backend/authentication/handlers_http TestAccountConsentScopesAllowOnlyTheOwnersReadAndRevoke`。
- [x] T004 [Adapters] ロールポリシーの 1 件 (EX-OAUTH2-004-01) を消化する。`mise run test-go-test -- ./backend/oauth2/handlers_http TestAdminRolePoliciesListVisibleRolesPermissionsAndInterfaces`。
- [x] T005 [Decision] EX-OAUTH2-003-04 の食い違いを測り、台帳に残す判断を記録する。
- [x] T006 [Decision] 所要を測り、残る 99 件を規則群ごとの子 work item へ割る。
- [x] T007 [Verify] `mise run verify`。

## Verification

- `mise run check-spec` が、消化した 5 件を `tools/check/example-coverage-debt.json` から外した状態で通る。台帳から外す前に同じ検査が当の id を名指しで落とすことを観測する。
- 消化したテストの所属パッケージに対する `mise run test-go-package -- <package>`。
- `mise run check-work-items` が、起こした子 work item を含めて通る。
- `mise run verify`

## Risk Notes

- **注記だけを足して終わる。** `checkNormativeCoverage` は文字列の一致しか見ないので、読まずに id を貼れば件数は速く減る。減った件数は何も意味しない。注記に「この具体例の何を固定しているか」を書かせることで区別する。親項目の測定では、16 件のうち注記だけで済んだのは 4 件だけだった。
- **`nearby` に分類された件を、テストがある証拠として読む。** 分類が言っているのは「近傍に他の拒否テストがある」だけである。親項目の測定では、`claim-mapping` の 3 件はすべて `nearby` でありながら 2 件はテストが無かった。分類は読む順の材料にとどめる。
- **拒否である具体例のテストが、応答の字面だけを見て書かれる。** 新しく書く拒否テストには [[wi-392-refusal-tests-assert-the-absent-effect]] の規範が効く。注記へ「効果の不在を何で観測したか」を書き、後から区別できるようにする。
- **具体例の `Then` が複数あるのに、観測が 1 つで済まされる。** 親項目の測定では、`EX-WORKLOADIDENTITY-008-01` のようにイベントと実際の効果の双方を言う具体例が複数あった。`Then` の数だけ観測が要る。

## Completion

- **Completed At**: 2026-09-12
- **Summary**:
  `mise run spec-diff` は `no normative specification change against main` を返す。規範は 1 行も動いていない。
  変わったのは 2 つである。`docs/contexts/oauth2/scenarios.feature.md` の REQ-OAUTH2-002、REQ-OAUTH2-003、
  REQ-OAUTH2-004 が宣言する 6 件のうち 5 件が、その id を名指しするテストから到達されるようになったこと
  （台帳は 573 件から 568 件へ減った）。そして、残る 99 件の割り方が測定にもとづいて決まり、
  5 つの子 work item として存在するようになったことである。製品コードは 1 行も変えていない。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-spec`（`checkNormativeCoverage`）
  - **Requirement**: REQ-OAUTH2-003
  - **Observed Failure**: OAuth2 の 105 件を `tools/check/example-coverage-debt.json` から外した状態で exit 1。
    105 行それぞれが `EX-OAUTH2-NNN-MM is declared, but no test names it.` の形で id を 1 件ずつ名指しした
    （`EX-OAUTH2-001-01`、`EX-OAUTH2-002-01`、`EX-OAUTH2-003-01` …）。
    消化ごとの観測はこの 1 回で足りる。検査は id ごとに独立した行を出すので、
    105 行は 105 件それぞれについての観測である。
  - **Detection Reason**: この検査は宣言された id と、`backend` と `frontend` のテストファイルが名指しした id の
    集合を比べる。テストを書かずに台帳から外せば必ず落ちるので、「消化した」と「台帳から消した」を
    取り違えられない。実際、`oauth2_scope_examples_test.go` の注釈が `EX-OAUTH2-003-04` に言及していたとき、
    検査は「テストが名指ししているのに台帳に残っている」と落ちて、注釈と消化の取り違えを教えた。
- **Unit RED Evidence**:
  - **Test**: N/A: 製品コードを変えていない。宣言済みの具体例に観測を対応付ける作業なので、RED にできる内側のロジックが無い。
  - **Requirement**: N/A: 規範も製品コードも 1 行も変えていないため、単体境界に RED が無い。
  - **Observed Failure**: 代わりの検査として、書いた 3 本のテストが観測対象の入口で実際に落ちることを、
    製品側へ故障を注入して確かめた。結果は下の Change-Resistance Results に記録する。
    とくに `TestAdminRolePoliciesListVisibleRolesPermissionsAndInterfaces` は、
    `toAdminRolePolicyResponse` が interfaces を常に空で返すようにすると落ち、
    そのとき `backend/oauth2/handlers_http` の他のテストは 1 本も落ちなかった。
    既存テストがこの `Then` を見ていなかったことの直接の観測である。
  - **Detection Reason**: 単体境界が無いのは、変えたのがテストと台帳だけだからである。
    この作業で意味のある検出能力は「書いたテストが壊れた実装を落とすか」であり、
    それは故障注入でしか観測できない。
- **Change-Resistance Results**:
  リスクは medium なので、書いたテストが実際に何を検出するかを 5 つの故障注入で確かめた。5 件とも検出された。
  1. `requireAdminApiTokenScope` の先頭へ `return nil` を置いて粒度スコープを素通しにする →
     `TestOAuth2AdminOperationsFollowTheGranularScopes`、`TestOAuth2ReadScopeCannotChangeOAuth2Clients`、
     `TestOAuth2ScopeOfAnotherResourceCannotReachTheOperation` の 3 本が落ちる。
     `oauth-clients:read` での登録が 201 で通り、クライアントが実際に保存された。
  2. `requireAdminApiTokenScope` の契約引きを `AdminContractPath(c.Path())` から `c.Path()` へ戻して、
     レルム接頭辞付きのルートが契約と一致しないようにする → 同じ 3 本が落ちる。
     フェイルクローズ側へ倒れるので、許すはずの参照まで 403 になる。許す側と拒む側の双方を
     読んでいなければ、この故障は片方だけでは検出できない。
  3. `toAdminRolePolicyResponse` が interfaces を常に空配列で返すようにする →
     `TestAdminRolePoliciesListVisibleRolesPermissionsAndInterfaces` だけが落ちる
     (`AdminUserRead に GET /api/admin/v1/users の対応が無い: []`)。
     同パッケージの他のテストは 1 本も落ちなかった。
  4. `hasRequiredAccountScope` の先頭へ `return true` を置いて account スコープの判定を素通しにする →
     `TestAccountConsentScopesAllowOnlyTheOwnersReadAndRevoke` を含む 4 本が落ちる。
     `account:consents:write` だけのトークンで同意一覧が 200 で読めた。
  5. `ListConsentsForSub` の絞り込みから `consent.UserID == sub` を落とす →
     `TestAccountConsentScopesAllowOnlyTheOwnersReadAndRevoke` だけが落ちる。
     bob の同意まで返るので件数が 2 になる。主体の固定を読んでいるテストは、このパッケージでこれ 1 本だった。
- **Verification Results**:
  - `mise run verify` - passed
  - `mise run check-spec` - passed（OAuth2 の残りは 100 件。うち `EX-OAUTH2-003-04` は食い違いのため残置）
  - `mise run lint-go` - passed（0 issues）
  - `mise run test-ui-e2e` - 実行していない。製品のフロントエンドも製品の Go コードも 1 行も変わっておらず、
    差分はテストと台帳と work item だけである。

### 消化した具体例とテストの対応

| 具体例 | テスト |
| --- | --- |
| EX-OAUTH2-002-01 | `TestAccountConsentScopesAllowOnlyTheOwnersReadAndRevoke` |
| EX-OAUTH2-003-01 | `TestOAuth2AdminOperationsFollowTheGranularScopes` |
| EX-OAUTH2-003-02 | `TestOAuth2ReadScopeCannotChangeOAuth2Clients` |
| EX-OAUTH2-003-03 | `TestOAuth2ScopeOfAnotherResourceCannotReachTheOperation` |
| EX-OAUTH2-004-01 | `TestAdminRolePoliciesListVisibleRolesPermissionsAndInterfaces` |

### 台帳に残した 1 件

`EX-OAUTH2-003-04` は残した。理由は Design の「直さない食い違い」に書いた。
[[wi-564-name-the-refusal-a-cross-tenant-oauth-admin-token-gets]] が引き取る。

### 起こした子 work item

| 記録 | 規則 | 件数 |
| --- | --- | --- |
| [[wi-559-back-oauth2-authorization-code-examples-with-tests]] | REQ-OAUTH2-001、005〜014 | 20 |
| [[wi-560-back-oauth2-registration-and-logout-examples-with-tests]] | REQ-OAUTH2-016〜027 | 17 |
| [[wi-561-back-oauth2-client-administration-examples-with-tests]] | REQ-OAUTH2-029〜037 | 10 |
| [[wi-562-back-oauth2-backchannel-approval-examples-with-tests]] | REQ-OAUTH2-041〜043 | 23 |
| [[wi-563-back-oauth2-delegation-and-agent-examples-with-tests]] | REQ-OAUTH2-044〜050 | 29 |
| [[wi-564-name-the-refusal-a-cross-tenant-oauth-admin-token-gets]] | EX-OAUTH2-003-04 | 1 |

合計 100 件で、台帳に残る OAuth2 の件数と一致する。

### 親項目の測定への追加

親項目 [[wi-496-burn-down-the-example-coverage-debt]] は「注記だけで済むのは少数」と測った。OAuth2 では
**1 件も無かった**。6 件のうち 5 件はテストを新しく書き、1 件は規範との食い違いで残した。

そのうえで、この 6 件は OAuth2 のなかで最も条件が良い。観測の位置が 1 つの HTTP 境界に揃っていて、
実トークンを発行して管理 API を叩くスタック (`apiTokenStack`) と、account 同意の拒否を既に見ている
fixture (`consentRefusalFixture`) が既にあった。それでも共有 fixture へ保存先を 2 つ足す必要があり、
`AuthorizationDetailTypeRepository` と `McpResourceServerRepository` を `Register` へ配線した。

`report-coverage-debt` の分類は今回も当てにならなかった。分類は読む順の材料にとどまる。
