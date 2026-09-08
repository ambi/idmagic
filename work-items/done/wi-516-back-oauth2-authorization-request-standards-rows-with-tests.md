---
depends_on: []
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-09
priority: p2
change_kind: maintenance
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: 標準の行にも製品の振る舞いにも変更が無く、増えたのはテストと注記だけなので、リリースの読み手に見えるものが無い。
  references: []
spec_impact: { kind: none, reason: "宣言済みの標準行に、その id を名指しするテストを対応付ける作業である。standards.md の行そのものも製品の振る舞いも変えない。テストが書けない行が見つかった場合、それは製品が宣言した採用を満たしていないということなので、欠陥として個別の work item に切り出す。" }
initial_context:
  specification:
    - docs/contexts/oauth2/standards.md
  typespec: []
  source:
    - backend/oauth2/handlers_http/authorize_handler.go
    - backend/oauth2/handlers_http/authorize_completion.go
    - backend/oauth2/handlers_http/authorize_consent.go
    - backend/oauth2/handlers_http/validation.go
    - backend/oauth2/authorization/usecases/authorize.go
    - backend/oauth2/authorization/usecases/push_authorization_request.go
    - backend/oauth2/authorization/domain/redirect_uri.go
    - backend/oauth2/authorization/domain/authorization_details.go
    - backend/oauth2/usecases/authorization_details.go
    - backend/oauth2/token/usecases/exchange_code.go
    - backend/oauth2/token/usecases/exchange_token.go
    - tools/check/standards-coverage-debt.json
  tests:
    - backend/oauth2/handlers_http/authorize_handler_test.go
    - backend/oauth2/handlers_http/par_handler_test.go
    - backend/oauth2/handlers_http/refusal_effects_test.go
    - backend/oauth2/handlers_http/register_handler_test.go
    - backend/oauth2/handlers_http/consent_withdrawal_standards_test.go
    - backend/oauth2/handlers_http/token_exchange_handler_test.go
    - backend/oauth2/authorization/domain/authorization_details_test.go
  stop_before_reading:
    - backend/saml
    - backend/wsfederation
    - backend/sourcing
    - frontend
---

# 認可リクエストが宣言する標準 15 行にテストを対応付け、標準の被覆台帳から外す

## Motivation

[[wi-499-back-oauth2-standards-rows-with-tests]] は `docs/contexts/oauth2/standards.md` の 80 行を引き取り、最初の節（`OAuth Client ID Metadata Document` の 7 行）を消化したうえで、残る 73 行を**行が共有する製品の入口**を単位に 7 件へ割った。本項目はそのうち 15 行を持つ。

15 行は、認可リクエストを受け取ってから認可コードを渡すまでに立つ判断である。`redirect_uri` の厳密一致、PKCE、`request_uri` の 1 回限りの使用、`authorization_details` の検証、`iss` の付与がここに集まる。ここが素通りすれば、以降のトークン発行はすべて攻撃者の選んだ宛先とクライアントに対して働く。

## Scope

- 次の 15 行を消化する。

| ID | Adoption | 節 |
|---|---|---|
| `RFC6749-AUTHORIZATION-CODE` | required | The OAuth 2.0 Authorization Framework |
| `RFC6749-IMPLICIT` | excluded | The OAuth 2.0 Authorization Framework |
| `RFC7636-VERIFY` | required | Proof Key for Code Exchange by OAuth Public Clients |
| `RFC7636-S256` | required | Proof Key for Code Exchange by OAuth Public Clients |
| `RFC7636-PLAIN` | excluded | Proof Key for Code Exchange by OAuth Public Clients |
| `RFC9126-PAR` | optional | OAuth 2.0 Pushed Authorization Requests |
| `RFC9126-SINGLE-USE` | required | OAuth 2.0 Pushed Authorization Requests |
| `RFC9207-ISS` | required | OAuth 2.0 Authorization Server Issuer Identification |
| `RFC9396-REGISTERED-TYPES` | required | OAuth 2.0 Rich Authorization Requests |
| `RFC9396-MONOTONIC-NARROWING` | required | OAuth 2.0 Rich Authorization Requests |
| `RFC9396-SCOPE-PRECEDENCE` | required | OAuth 2.0 Rich Authorization Requests |
| `RFC9700-REDIRECT-MATCH` | required | Best Current Practice for OAuth 2.0 Security |
| `RFC9700-AUTHORIZATION-CODE` | required | Best Current Practice for OAuth 2.0 Security |
| `OIDC-CORE-HYBRID-IMPLICIT` | excluded | OpenID Connect Core 1.0 incorporating errata set 1 |
| `RFC7591-REDIRECT-URI` | required | OAuth 2.0 Dynamic Client Registration Protocol |

- 1 行ごとに、その行の `Statement` を区別できる入力と観測を持つテストを対応付け、`// <ID>: <この行の何を固定しているか>` の注記を足す。
- 観測の形は行の `Adoption` に従う。型は [[wi-495-burn-down-the-standards-coverage-debt]] の Design が定める。
- 消化した id を `tools/check/standards-coverage-debt.json` から外す。

## Out of Scope

- 他の work item が持つ標準行。[[wi-499-back-oauth2-standards-rows-with-tests]] の Design にある分割表が所属を定める。
- `standards.md` の行の追加、削除、`Adoption` および `Strength` の変更。行の内容が現状と食い違うと判明した場合は規範の変更であり、別の work item が扱う。
- 見つかった実装の欠陥の修正。欠陥として切り出した先で扱う。
- 受入集合の導入と、台帳ファイルそのものの削除。[[wi-495-burn-down-the-standards-coverage-debt]] が持つ。
- 具体例の被覆負債。[[wi-496-burn-down-the-example-coverage-debt]] が持つ。
- id を名指しする文字列があるだけを根拠にした台帳からの削除。

## Design

**観測の入口は 認可エンドポイント `/authorize` と Pushed Authorization Requests エンドポイント である。** [[wi-499-back-oauth2-standards-rows-with-tests]] が 7 行を測って出した結論は、消化の費用を支配するのが行数ではなく「行が共有する入口にハーネスを 1 つ組むこと」だという点にある。本項目の 15 行はこの入口を共有するので、ハーネスは 1 度組めば足りる。

この入口の観測は、認可リクエストを HTTP で送り、応答（リダイレクト先、エラーコード）と保存された認可リクエストの両方を読む形になる。拒否の行では、拒否したうえで認可リクエストが保存されていないことを対で読む。保存してから拒否する実装は後続の経路へ材料を残すので、応答だけでは足りない。

`excluded` の 3 行（`RFC6749-IMPLICIT`、`RFC7636-PLAIN`、`OIDC-CORE-HYBRID-IMPLICIT`）はいずれも標準側の機能を書いているので、その機能を使うリクエストが通らないことと、通らなかった結果としてトークンもコードも発行されていないことを対で読む。

行ごとの観測の形は `Adoption` が決める。`required` は宣言した振る舞いが正式な入口から到達できること、`optional` は提供しているならその振る舞い、`excluded` は提供していないことと拒否が防いだ効果、`partial` は採った範囲と採らなかった範囲の扱いを、それぞれ観測する。`excluded` の観測は 1 つの型に収まらない。行の `Statement` が製品の制約を書いているのか標準側の機能を書いているのかで観測が裏返るので、本項目の `excluded` の 1 件目でどちらかを決めてから残りへ広げる。

### ハーネスは `Register` が組んだスタック 1 つ（T002 で決めた）

15 行すべてを、`backend/shared/http/server_http` の `Register` が組み立てたスタックへ HTTP リクエストを出す形で観測した。テストは `backend/shared/http/server_http/authorization_request_standards_test.go` の 1 ファイルに置いた。

**同じパッケージにある `routes_e2e_test.go` の `newServer` を使わなかった。** あちらは `AuthzDetailTypeRepo` と `McpResourceServerRepo` を配線していないので、`RFC9396-*` の 3 行とトークン交換が、その配線ごと観測できない。ハーネスは同じ形で組み直し、2 つの登録簿を足した。

**入口は 2 つに広がった。** `/authorize` と `/par` に加えて、`/token`（認可コードの交換とトークン交換）と `/register` を通す。3 行の `Statement` がそこを名指しているからである。

- `RFC6749-AUTHORIZATION-CODE` は「認可エンドポイントとトークンエンドポイントで提供し」と書いている。`/authorize` が 303 を返すことだけでは、コードがトークンにならない実装を通してしまう。
- `RFC7636-VERIFY` は「トークンリクエストの `code_verifier` を認可時の `code_challenge` と照合し」と書いている。照合そのものは `/token` にある。
- `RFC9396-MONOTONIC-NARROWING` は「後続の交換は権限を狭めることだけを許し」と書いている。交換は `/token` のトークン交換グラントにある。
- `RFC7591-REDIRECT-URI` は登録の要件なので `/register` にある。

**入口が行の `Statement` で決まるという [[wi-499-back-oauth2-standards-rows-with-tests]] の分割の基準は、ここでは 1 つの HTTP 経路ではなく 1 つのハーネスとして効いた。** 4 つの経路は同じスタックの上にあるので、ハーネスは 1 度組めば足りている。分割の単位が経路そのものだったなら、この 15 行は 4 つに割れていたはずである。

### `excluded` の観測の型（T003、本文書で決めた）

3 行とも標準側の機能を書いている（Implicit Grant を提供する、`plain` を許可する、Implicit / Hybrid Flow を提供する）ので、[[wi-500-back-api-tokens-standards-rows-with-tests]] の `RFC6750-API-TOKEN-QUERY` と同じ向きの型を採った。

**その機能を使うリクエストが通らないことと、通らなかった結果として資格情報が 1 つも出ていないことを対で読む。**

`RFC7636-PLAIN` では、`code_challenge_method=plain` を RFC 7636 が定める正しい組（`plain` では challenge が verifier そのもの）で送る。壊れた値で送ると、method を読んでいない実装でも同じ拒否になり、`plain` の扱いを区別できない。

`optional` は `RFC9126-PAR` の 1 行だけである。提供されていたので、その振る舞いを観測した。規範の変更として切り出したものは無い。

## Plan

1. ~~15 行が共有する入口にハーネスを組み、`required` の 1 件目を通しで消化して型を決める。~~ 完了。Design の「ハーネスは `Register` が組んだスタック 1 つ」節。
2. ~~`optional` があれば、その 1 件目で「提供している」と言える根拠の形を決める。~~ 完了。`RFC9126-PAR` は提供されていた。
3. ~~`excluded` があれば、その 1 件目で観測の型を決める。~~ 完了。Design の「`excluded` の観測の型」節。
4. ~~残りを消化し、解決した id を台帳から外す。~~ 15 件すべてを外した（90 → 75）。
5. ~~宣言した採用を満たしていない行が見つかったら、欠陥の work item を切り出す。~~ 15 行とも満たされていた。切り出しは無い。

## Tasks

- [x] T001 [Acceptance] 消化する id を台帳から先に外し、`mise run check-spec` が当該 id ごとに `is declared, but no test names it` を報告することを観測する。
  15 件すべてについて観測した。Completion の Acceptance RED Evidence。
  recipe: `mise run check-spec`
- [x] T002 [Harness] `Register` が組み立てたスタックへハーネスを組み、`required` の 1 件目で型を決める。
  `backend/shared/http/server_http/authorization_request_standards_test.go` に 11 テストを置いた。
  型は Design の該当節。
  recipe: `mise run test-go-package -- ./backend/shared/http/server_http`
- [x] T003 [Type] `optional` と `excluded` の観測の型を、それぞれ 1 件目で決める。
  `excluded` の型は `RFC6749-IMPLICIT` で決め、`RFC7636-PLAIN` と `OIDC-CORE-HYBRID-IMPLICIT` へ広げた。
  `optional` は `RFC9126-PAR` の 1 行だけで、提供されていた。
- [x] T004 [Ledger] 残りを消化し、解決した id を `tools/check/standards-coverage-debt.json` から外す。
  15 件を外した（90 → 75）。
  recipe: `mise run check-spec`
- [x] T005 [Resistance] 行が言っている判断を production 側で崩し、対応するテストが落ちることを行ごとに観測する。
  21 件の変異を注入した。内訳は Completion の Change-Resistance Results。
- [x] T006 [Defect] 宣言した採用を満たしていない行が見つかったら、欠陥の work item を切り出す。
  15 行とも満たされていた。切り出しは無い。
- [x] T007 [Verify] `mise run verify`。

## Verification

- 本項目が持つ 15 件が `tools/check/standards-coverage-debt.json` から 1 件残らず消えている。
- 注記を足した各テストについて、対応する production の判断を崩すとそのテストが落ちる。
- `mise run check-spec`
- `mise run verify`

## Risk Notes

- **注記だけを足して終わる。** 名指しの文字列があれば検査は通るので、読まずに id を貼れば件数は減る。注記に「何を固定しているか」を書かせ、崩して落ちることを T005 で確かめる。
- **ハーネスが本番とずれる。** 入口を通すという方針は、その入口が製品と同じ部品でできているときだけ意味を持つ。[[wi-500-back-api-tokens-standards-rows-with-tests]] はここを一度間違え、製品の欠陥でないものを欠陥として起票した。組み立ては `cmd/internal/bootstrap` が作る形に合わせる。
- **1 行が 2 つのことを言っている。** `Statement` に動詞が 2 つあれば観測も 2 つ要る。
- **拒否の理由が、行とは別の防護になっている。** 認可リクエストは検証の順序が長い。1 要素だけを崩したリクエストを作り、崩していない対照が通ることを先に確かめる。
- **台帳の同時編集。** 台帳は id 順に 1 エントリー 1 id なので、並行しても衝突はエントリー単位に収まる。自分が持つ id のエントリーだけを削除し、コメント配列と他の id には触れない。

## Completion

- **Completed At**: 2026-09-09
- **Summary**:
  `mise run spec-diff` は `no normative specification change against main` を返す。規範の変更は無い。

  `docs/contexts/oauth2/standards.md` のうち、認可リクエストの入口に立つ **15 行**が、その行の
  `Statement` を区別できる入力と観測を持つテストを得て `tools/check/standards-coverage-debt.json` から
  消えた。台帳は 90 件から 75 件になった。テストは 1 ファイル
  （`backend/shared/http/server_http/authorization_request_standards_test.go`、11 テスト）で、
  `Register` が組み立てたスタックへ HTTP リクエストを出す。製品コードは 1 行も変わっていない。
  15 行とも宣言した採用を満たしており、欠陥の切り出しは無い。

  **ハーネスは 1 つで足りたが、通した経路は 4 つになった。** `/authorize` と `/par` に加えて、
  `RFC6749-AUTHORIZATION-CODE`、`RFC7636-VERIFY`、`RFC9396-MONOTONIC-NARROWING` が `/token` を、
  `RFC7591-REDIRECT-URI` が `/register` を名指しているからである。
  [[wi-499-back-oauth2-standards-rows-with-tests]] の分割の基準は「行が共有する入口」だが、ここで効いた
  単位は 1 つの HTTP 経路ではなく **1 つのハーネス**だった。経路そのものを単位にしていたら、この 15 行は
  4 件へ割れていたはずである。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-spec`（15 件を台帳から外し、テストを書く前に）
  - **Requirement**: N/A: 標準の被覆はテストの有無についての性質であり、製品の規範要求ではない。
  - **Observed Failure**: exit 1。`docs/contexts/oauth2/standards.md` の 15 行それぞれに
    `<ID> is declared, but no test names it. Cite the id from the test that exercises it, or list it in
    tools/check/standards-coverage-debt.json with a reason.`（9 行目 `RFC6749-AUTHORIZATION-CODE` から
    246 行目 `OIDC-CORE-HYBRID-IMPLICIT` まで 15 件）
  - **Detection Reason**: この検査は、宣言された id・テストが名指す id・台帳の 3 つを突き合わせる。台帳から
    外した id は、名指すテストが実在しない限り必ず報告される。15 件を消化した後の同じコマンドは
    `ok normative coverage (156 standard(s), 311 rule(s), 744 example(s), 210 id(s) named by a test)` を返す。
- **Unit RED Evidence**:
  - **Test**: 各行に対応付けたテストへの故障注入（Change-Resistance Results を参照）
  - **Requirement**: N/A: 行ごとの観測は標準の `Statement` に対応し、`REQ` 番号には対応しない。
  - **Observed Failure**: 15 行それぞれについて、対応する production の判断を崩すとそのテストが落ちる
    ことを観測した。内訳は Change-Resistance Results の表。
  - **Detection Reason**: `checkNormativeCoverage` は文字列の一致しか見ないので、id を書くだけで検査は
    通ってしまう。名指しが実在の検証に付いていることの担保は、production を崩したときにそのテストが
    落ちるという観測しかない。
  - **見つけた不足 1**: `RFC9396-SCOPE-PRECEDENCE` を、1 回目より広い `scope` を要求するリクエストで
    観測しようとしていた。この形では「構造化 detail は過去の scope 同意で代替しない」規則を外す変異が
    生き残る。**同意を求め直す理由が scope の不足になってしまい、detail の扱いを 1 度も観測していな
    かった**からである。`scope` を 1 回目とまったく同じにし、違いを `authorization_details` の有無だけに
    絞ったところ、この変異を検出するようになった。
  - **見つけた不足 2**: `RFC9126-SINGLE-USE` の「短命」を、応答の `expires_in` だけで観測しようとして
    いた。この形では保存する寿命を 90 秒から 1 日へ伸ばす変異が生き残る。**広告する `expires_in` と
    保存する `ExpiresAt` は別々に決まっているので、応答の数字は実際の寿命の証拠にならない。**
    保存された記録の `ExpiresAt - IssuedAt` が広告した秒数と一致することを併せて読む形へ変えたところ、
    この変異を検出するようになった。
- **Change-Resistance Results**:
  15 行すべてについて、行が言っていることを production 側で崩し、対応するテストが落ちることを観測した。
  21 件の変異のうち 19 件は 1 か所を崩すだけで検出され、2 件は複数の防護をまとめて崩す必要があった。
  生き残りは無い。

  | 行 | 注入した故障 | 落ちたテストと観測 |
  |---|---|---|
  | `RFC6749-AUTHORIZATION-CODE` | `validateAuthorizationParameterCardinality` の `len != 1` を `len < 1` へ | `TestAuthorizationCodeGrantSpansBothEndpointsAndRejectsDuplicatedParameters` の重複 5 事例 |
  | 〃 | 重複検査の対象集合から `state` を落とす | 同テスト `/重複した_state` |
  | 〃 | `/token` の認可コード交換が常に `code_verifier` 必須で落ちる | 同テスト: 対照のコード交換が失敗する |
  | `RFC6749-IMPLICIT` / `OIDC-CORE-HYBRID-IMPLICIT` | `Authorize` の `response_type != "code"` 検査を外す | `TestImplicitAndHybridResponseTypesAreRefused` の 6 事例すべて |
  | `RFC7636-VERIFY` | `exchange_code` の `VerifyPKCES256` の結果を無視 | `TestPKCEVerifierIsCheckedAtTheTokenEndpointAndDuplicatesAreRefused`: 誤った verifier で交換できる |
  | 〃 | 重複検査を外す（上と同じ変異） | 同テスト `/重複した_code_challenge` ほか |
  | `RFC7636-S256` / `RFC7636-PLAIN` | `Authorize` の `CodeChallengeMethod != "S256"` 検査を外す | `TestOnlyS256CodeChallengeMethodIsAccepted` の 4 事例すべて |
  | `RFC9126-PAR` | `/par` がクライアント認証の失敗を無視して `client_id` をそのまま使う | `TestPushedAuthorizationRequestIsAuthenticatedShortLivedAndSingleUse`: 認証の無い PAR が受理される |
  | 〃 | `PARResult` から `ExpiresIn` を落とす | 同テスト: `expires_in=0` |
  | `RFC9126-SINGLE-USE` | `/authorize` が `PARStore.Consume` ではなく `Find` を呼ぶ | 同テスト: 2 回目の `request_uri` が通る |
  | 〃 | 保存する寿命を 90 秒から 24 時間へ伸ばす | 同テスト: 広告した `expires_in` と保存した寿命が食い違う |
  | `RFC9207-ISS` | 認可レスポンスの URL から `iss` を落とす | `TestIssuerIdentifierAccompaniesAuthorizationResponsesAndRedirectedErrors`: `iss=""` |
  | 〃 | 認可エラーの URL の `iss` を空にする | 同テスト: エラー側の `iss` が空 |
  | `RFC9396-REGISTERED-TYPES` | 未登録の `type` を通す | `TestAuthorizationDetailsAreValidatedAgainstRegisteredTypes` の 3 事例 |
  | 〃 | `ValidateAgainstType` の結果を無視 | 同テスト `/許可されない_action` と `/必須フィールドが無い` |
  | 〃 | `details` の 1 件目だけ検証して残りを素通しする | 同テスト `/妥当なものと未登録の型を混ぜる` |
  | `RFC9396-MONOTONIC-NARROWING` | 交換時の `DetailsSubsetOf` の結果を無視 | `TestAuthorizationDetailsCanOnlyNarrowAcrossExchange/金額の上限を上げる` |
  | 〃 | 認可コード交換で発行するトークンから `authorization_details` を落とす | 同テスト: 発行したトークンが同意した detail を運ばない |
  | `RFC9396-SCOPE-PRECEDENCE` | 構造化 detail があっても過去の scope 同意で自動承認する | `TestCoarseScopeConsentDoesNotCoverStructuredAuthorizationDetails` |
  | `RFC9700-REDIRECT-MATCH` | `RedirectURIAllowed` を前方一致へ緩める | `TestRedirectURIMustMatchARegisteredValueExactly` の `/末尾スラッシュ`、`/クエリを足した`、`/パスを足した` |
  | 〃 | 照合そのものを外す | 同テストの 5 事例すべて |
  | `RFC9700-AUTHORIZATION-CODE` | `parseAuthorizeRequest` が `code_challenge` の欠落を既定値で補う | `TestAuthorizationCodeGrantSpansBothEndpointsAndRejectsDuplicatedParameters/code_challenge_が無い`: PKCE 無しの要求が 303 で先へ進む |
  | `RFC7591-REDIRECT-URI` | 登録時の `redirect_uris` 必須を 3 か所すべてから外す | `TestDynamicRegistrationRequiresARedirectURIForTheCodeGrant` の 2 事例 |

  この方法の限界: 2 つの行が単一の変異では倒れない。

  - **`RFC7591-REDIRECT-URI` は 3 か所に守られている。** リクエストスキーマの `Min(1)`、
    `register_client.go` の「redirect 系グラントには `redirect_uris` を要求する」、
    `OAuth2Client.Validate` の同名の規則である。どれか 1 つを外す変異は生き残る（実測）。
  - **`RFC9700-AUTHORIZATION-CODE` の `code_challenge` 必須も同様である。** `zog` の struct shape は
    キーが入力に無ければ `.Required()` の有無にかかわらず落ちるので、`.Required()` を外す変異も
    `Authorize` の空文字検査を外す変異も、単独では観測を変えない。**この行を崩すには、欠落を
    「値がある」状態へ変える必要がある**（表の既定値で補う変異）。

  行としての被覆に穴は無いが、行に対応するテスト 1 つが production の 1 か所に対応しているわけではない。
  表の 21 件も、行と 1 対 1 ではなく、行が言っている判断ごとに 1 件を当てている。
- **Verification Results**:
  - `mise run check-spec` - passed（`ok normative coverage (156 standard(s), 311 rule(s), 744 example(s), 210 id(s) named by a test)`）
  - `mise run check-work-items` - passed
  - `mise run check-ids` - passed
  - `mise run test-go-package -- ./backend/shared/http/server_http` - passed
  - `mise run lint-go` - passed（gofumpt の指摘 1 件を `mise run format-go` で直した）
  - `mise run verify` - passed
  - `mise run test-ui-e2e` - `N/A: 変更は Go のテスト 1 ファイルと tools の JSON、および work item だけで、ブラウザーへ到達する経路が無い。`
