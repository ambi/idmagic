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
    - backend/oauth2/handlers_http/userinfo_handler.go
    - backend/oauth2/handlers_http/token_handler.go
    - backend/oauth2/handlers_http/client_auth.go
    - backend/oauth2/token/usecases/introspect_token.go
    - backend/oauth2/token/usecases/revoke_token.go
    - backend/oauth2/token/usecases/exchange_code.go
    - backend/shared/security/tokens_jose/dpop_verifier.go
    - backend/shared/security/tokens_jose/jwt_signer.go
    - backend/shared/security/certificates_mtls/mtls_client_cert.go
    - backend/shared/http/support_http/tenant_middleware.go
    - tools/check/src/normative-coverage.ts
    - tools/check/standards-coverage-debt.json
  tests:
    - backend/shared/http/server_http/api_token_standards_test.go
    - backend/shared/http/server_http/token_issuance_standards_test.go
  stop_before_reading:
    - backend/saml
    - backend/wsfederation
    - backend/sourcing
    - frontend
---

# トークンの提示・内省・失効・送信者制約が宣言する標準 13 行にテストを対応付け、標準の被覆台帳から外す

## Motivation

[[wi-499-back-oauth2-standards-rows-with-tests]] は `docs/contexts/oauth2/standards.md` の 80 行を引き取り、最初の節（`OAuth Client ID Metadata Document` の 7 行）を消化したうえで、残る 73 行を**行が共有する製品の入口**を単位に 7 件へ割った。本項目はそのうち 13 行を持つ。

13 行は、発行済みのトークンをどう受け取り、どう無効にするかを定める。提示の形、内省の応答、失効の即時性、そして DPoP と mTLS による送信者制約がここに集まる。6 行が `optional` であり、13 行の半分近くが「提供していること」以外を観測する行である。

## Scope

- 次の 13 行を消化する。

| ID | Adoption | 節 |
|---|---|---|
| `RFC6750-AUTHORIZATION-HEADER` | required | The OAuth 2.0 Authorization Framework Bearer Token Usage |
| `RFC6750-QUERY-TOKEN` | excluded | The OAuth 2.0 Authorization Framework Bearer Token Usage |
| `RFC7662-INTROSPECT` | required | OAuth 2.0 Token Introspection |
| `RFC7662-INACTIVE` | required | OAuth 2.0 Token Introspection |
| `RFC7009-REVOCATION-ENDPOINT` | required | OAuth 2.0 Token Revocation |
| `RFC7009-UNKNOWN-TOKEN` | required | OAuth 2.0 Token Revocation |
| `RFC7800-CONFIRMATION` | optional | Proof-of-Possession Key Semantics for JSON Web Tokens |
| `RFC8705-CLIENT-AUTH` | optional | OAuth 2.0 Mutual-TLS Client Authentication and Certificate-Bound Access Tokens |
| `RFC8705-CERT-BOUND` | optional | OAuth 2.0 Mutual-TLS Client Authentication and Certificate-Bound Access Tokens |
| `RFC9449-PROOF` | optional | OAuth 2.0 Demonstrating Proof of Possession |
| `RFC9449-ATH` | optional | OAuth 2.0 Demonstrating Proof of Possession |
| `RFC9700-SENDER-CONSTRAINT` | optional | Best Current Practice for OAuth 2.0 Security |
| `OIDC-CORE-USERINFO` | required | OpenID Connect Core 1.0 incorporating errata set 1 |

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

**観測の入口は ベアラー保護リソースと、内省・失効の各エンドポイント である。** [[wi-499-back-oauth2-standards-rows-with-tests]] が 7 行を測って出した結論は、消化の費用を支配するのが行数ではなく「行が共有する入口にハーネスを 1 つ組むこと」だという点にある。本項目の 13 行はこの入口を共有するので、ハーネスは 1 度組めば足りる。

この入口の観測は、保護されたエンドポイントへトークンを提示する形になる。この防護はミドルウェアの位置にあるので、検証関数の単体テストでは代わりにならない。関数が正しくても配線されていなければ素通りする。

`optional` の 6 行は、提供しているならその振る舞いを観測する。提供していないなら、その行の `Adoption` が誤っているということなので規範の変更として切り出す。「未実装だから観測できない」で止めない。

行ごとの観測の形は `Adoption` が決める。`required` は宣言した振る舞いが正式な入口から到達できること、`optional` は提供しているならその振る舞い、`excluded` は提供していないことと拒否が防いだ効果、`partial` は採った範囲と採らなかった範囲の扱いを、それぞれ観測する。`excluded` の観測は 1 つの型に収まらない。行の `Statement` が製品の制約を書いているのか標準側の機能を書いているのかで観測が裏返るので、本項目の `excluded` の 1 件目でどちらかを決めてから残りへ広げる。

### ハーネスは `Register` が組んだスタック 1 つ（T002 で決めた）

13 行すべてを、`backend/shared/http/server_http` の `Register` が組み立てたスタックへ HTTP リクエストを出す形で観測した。テストは `backend/shared/http/server_http/token_presentation_standards_test.go` の 1 ファイル（9 テスト）に置いた。ベアラー保護リソースとして使うのは `/userinfo` である。

**保護リソースは `/userinfo` にした。** 13 行のうち `OIDC-CORE-USERINFO` はこのエンドポイントを名指しし、`RFC8705-CERT-BOUND` と `RFC9449-ATH` は「保護リソースでの照合」を言う。`/userinfo` は Bearer と DPoP の両スキームを受け、mTLS サムプリントと DPoP 証明の両方をここで照合する。管理 API 側の保護リソースは DPoP 証明の `htu` にパスを要求しており（[[wi-511-dpop-proof-htu-at-protected-resources-is-not-the-target-uri]]）、適合クライアントの形で観測できない。`/userinfo` は `RequestHTU(c, d.Issuer)` を期待するので絶対 URL で通る。

**トークンは正式な入口だけを通して取る。** 保存層へ直接置くと、提示の前提が製品の経路と違ってしまい、`cnf` の出所も内省が読む値も変わる。認可コードは `/authorize` とログイン API を通して取り、送信者制約は `/token` へ証明または証明書を出して付ける。

**`DpopReplayStore` と `AccessTokenDenylist` を配線した。** どちらも nil なら該当の検査そのものが飛ぶ。持たせずに観測すると、検査が働いた結果と配線が無い結果を区別できない。

### `optional` と `excluded` の観測の型（T003、本文書で決めた）

`excluded` は `RFC6750-QUERY-TOKEN` の 1 行だけである。標準側の機能を書いているので、[[wi-516-back-oauth2-authorization-request-standards-rows-with-tests]] と [[wi-517-back-oauth2-token-issuance-standards-rows-with-tests]] と同じ向きの型を採った。**ヘッダーで通ることを確かめた同じ 1 本を、クエリパラメーターで送ると通らないことと、通らなかった結果として利用者の claim が 1 つも返っていないことを対で読む。** 別のトークンで比べると、最初から無効だった実装と区別できない。

`optional` は 6 行あり、6 行とも提供されていた。規範の変更として切り出したものは無い。観測は次の 2 つに分かれる。

- **提供の有無が選べる行**（`RFC7800-CONFIRMATION`、`RFC9700-SENDER-CONSTRAINT`）は、制約なし・DPoP・mTLS の 3 本を同じ入口で作って比べる。制約ありの 1 本だけを見ても、常に制約を付ける実装と区別できない。
- **提供している機能の中身を言う行**（`RFC8705-CLIENT-AUTH`、`RFC8705-CERT-BOUND`、`RFC9449-PROOF`、`RFC9449-ATH`）は、行が挙げる要素ごとに 1 要素だけを崩す。崩していない要素が有効であることは、無傷の提示が通ることで先に確かめる。

**値の照合を言う行は、壊れた入力では観測できない。** 形式の検証で先に落ちるので、照合が無い実装でも同じ拒否になる。mTLS の 2 行は、別の Subject DN を持つ**正しい形式の**証明書を作って照合だけを差にした。DPoP の `ath` も、別のアクセストークンの正しいハッシュを載せた証明で読む。

**`RFC9449-PROOF` はトークンエンドポイント側で観測した。** `jti` のリプレイは「同じ入力の 2 回目」でしか読めないが、認可コードの交換は 1 度成功した時点でコードが消えるので、2 回目の拒否がリプレイ検知によるものかコード消費によるものか区別できない。`client_credentials` なら同じリクエストを 2 度出せる。`ath` はトークンエンドポイントでは要求されない（RFC 9449 §4.3: 束縛先のアクセストークンがまだ存在しない）ので、保護リソース側で観測した。

## Plan

1. ~~13 行が共有する入口にハーネスを組み、`required` の 1 件目を通しで消化して型を決める。~~ 完了。Design の該当節。
2. ~~`optional` があれば、その 1 件目で「提供している」と言える根拠の形を決める。~~ 完了。6 行とも提供されており、観測できた。
3. ~~`excluded` があれば、その 1 件目で観測の型を決める。~~ 完了。Design の該当節。
4. ~~残りを消化し、解決した id を台帳から外す。~~ 13 件すべてを外した（60 → 47）。
5. ~~宣言した採用を満たしていない行が見つかったら、欠陥の work item を切り出す。~~ 13 行とも満たされていた。ただし `RFC7662-INTROSPECT` を観測する途中で、行が名指ししていない欠陥を 1 件見つけたので [[wi-523-introspection-token-type-is-not-the-rfc6749-token-type]] を切り出した。

## Tasks

- [x] T001 [Acceptance] 消化する id を台帳から先に外し、`mise run check-spec` が当該 id ごとに `is declared, but no test names it` を報告することを観測する。
  13 件すべてについて観測した。Completion の Acceptance RED Evidence。
  recipe: `mise run check-spec`
- [x] T002 [Harness] ベアラー保護リソースと、内省・失効の各エンドポイント にハーネスを組み、`required` の 1 件目で型を決める。
  `backend/shared/http/server_http/token_presentation_standards_test.go` に 9 テストを置いた。
  recipe: `mise run test-go-package -- ./backend/shared/http/server_http`
- [x] T003 [Type] `optional` と `excluded` の観測の型を、それぞれ 1 件目で決める。
  Design の「`optional` と `excluded` の観測の型」節。
- [x] T004 [Ledger] 残りを消化し、解決した id を `tools/check/standards-coverage-debt.json` から外す。
  13 件を外した（60 → 47）。
  recipe: `mise run check-spec`
- [x] T005 [Resistance] 行が言っている判断を production 側で崩し、対応するテストが落ちることを行ごとに観測する。
  20 件の変異を注入した。内訳は Completion の Change-Resistance Results。
- [x] T006 [Defect] 宣言した採用を満たしていない行が見つかったら、欠陥の work item を切り出す。
  13 行とも満たされていた。行の外で見つけた 1 件を
  [[wi-523-introspection-token-type-is-not-the-rfc6749-token-type]] として切り出した。
- [x] T007 [Verify] `mise run verify`。

## Verification

- 本項目が持つ 13 件が `tools/check/standards-coverage-debt.json` から 1 件残らず消えている。
- 注記を足した各テストについて、対応する production の判断を崩すとそのテストが落ちる。
- `mise run check-spec`
- `mise run verify`

## Risk Notes

- **注記だけを足して終わる。** 名指しの文字列があれば検査は通るので、読まずに id を貼れば件数は減る。注記に「何を固定しているか」を書かせ、崩して落ちることを T005 で確かめる。
- **ハーネスが本番とずれる。** 入口を通すという方針は、その入口が製品と同じ部品でできているときだけ意味を持つ。[[wi-500-back-api-tokens-standards-rows-with-tests]] はここを一度間違え、製品の欠陥でないものを欠陥として起票した。組み立ては `cmd/internal/bootstrap` が作る形に合わせる。
- **1 行が 2 つのことを言っている。** `Statement` に動詞が 2 つあれば観測も 2 つ要る。
- **`optional` の 6 行が「未実装だから観測できない」で止まる。** 止めない。実装が無いなら行の `Adoption` が誤っているので、規範の変更として切り出す。台帳へ残す場合は、理由を `present when the check was introduced` から具体的な理由へ書き換える。
- **台帳の同時編集。** 台帳は id 順に 1 エントリー 1 id なので、並行しても衝突はエントリー単位に収まる。自分が持つ id のエントリーだけを削除し、コメント配列と他の id には触れない。

## Completion

- **Completed At**: 2026-09-09
- **Summary**:
  `mise run spec-diff` は `no normative specification change against main` を返す。規範の変更は無い。

  `docs/contexts/oauth2/standards.md` のうち、発行済みトークンの提示・内省・失効・送信者制約に立つ
  **13 行**が、その行の `Statement` を区別できる入力と観測を持つテストを得て
  `tools/check/standards-coverage-debt.json` から消えた。台帳は 60 件から 47 件になった。テストは
  1 ファイル（`backend/shared/http/server_http/token_presentation_standards_test.go`、9 テスト）で、
  `Register` が組み立てたスタックの `/token` でトークンを作り、`/userinfo`、`/introspect`、`/revoke`
  へ提示する。製品コードは 1 行も変わっていない。

  この 13 行が言っているのは「受け取ったトークンをどう扱うか」なので、観測は必ず**同じ 1 本のトークンを
  条件だけ変えて提示する**形になる。別のトークンで比べると、拒否がトークンの側の欠陥によるものか提示の
  側の条件によるものか区別できない。失効と `excluded` の 2 種類は、応答に加えて**拒否が防いだ効果**を
  読む。応答だけを見るテストは、200 を返しながら他クライアントのトークンを失効させる実装を見逃す。

  `optional` の 6 行は 6 行とも提供されており、規範の変更として切り出したものは無い。行の外で
  `RFC7662-INTROSPECT` を観測する途中に見つけた 1 件を
  [[wi-523-introspection-token-type-is-not-the-rfc6749-token-type]] として切り出した。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-spec`（13 件を台帳から外し、テストを書く前に）
  - **Requirement**: N/A: 標準の被覆はテストの有無についての性質であり、製品の規範要求ではない。
  - **Observed Failure**: exit 1。`docs/contexts/oauth2/standards.md` の 13 行それぞれに
    `<ID> is declared, but no test names it. Cite the id from the test that exercises it, or list it in
    tools/check/standards-coverage-debt.json with a reason.`（20 行目 `RFC6750-AUTHORIZATION-HEADER` から
    245 行目 `OIDC-CORE-USERINFO` まで 13 件）
  - **Detection Reason**: この検査は、宣言された id・テストが名指す id・台帳の 3 つを突き合わせる。台帳から
    外した id は、名指すテストが実在しない限り必ず報告される。13 件を消化した後の同じコマンドは
    `ok normative coverage (156 standard(s), 311 rule(s), 744 example(s), 238 id(s) named by a test)` を返す。
- **Unit RED Evidence**:
  - **Test**: 各行に対応付けたテストへの故障注入（Change-Resistance Results を参照）
  - **Requirement**: N/A: 行ごとの観測は標準の `Statement` に対応し、`REQ` 番号には対応しない。
  - **Observed Failure**: 13 行それぞれについて、対応する production の判断を崩すとそのテストが落ちる
    ことを観測した。内訳は Change-Resistance Results の表。
  - **Detection Reason**: `checkNormativeCoverage` は文字列の一致しか見ないので、id を書くだけで検査は
    通ってしまう。名指しが実在の検証に付いていることの担保は、production を崩したときにそのテストが
    落ちるという観測しかない。
  - **見つけた不足**: `RFC7662-INTROSPECT` の観測で、内省の `token_type` が `Bearer` を返すものとして
    書いていた。**返るのは `access_token` である。** RFC 7662 §2.2 は `token_type` を RFC 6749 §5.1 の
    種別と定めているので、これは製品側の欠陥だが、この行の `Statement` は `token_type` の語彙を名指して
    いない。行が言う「許可されたメタデータを返す」ことの観測は、アクセストークンとリフレッシュトークンで
    値が分かれることで足りるので、テストは現状の値を注記付きで固定し、語彙の是正を
    [[wi-523-introspection-token-type-is-not-the-rfc6749-token-type]] へ切り出した。
- **Change-Resistance Results**:
  13 行すべてについて、行が言っていることを production 側で崩し、対応するテストが落ちることを観測した。
  20 件の変異はいずれも 1 か所を崩すだけで検出された。生き残りは無い。

  | 行 | 注入した故障 | 落ちたテストと観測 |
  |---|---|---|
  | `RFC6750-AUTHORIZATION-HEADER` | `/userinfo` が `Authorization` の `Bearer ` を認めない | `TestBearerTokenIsAcceptedOnlyFromTheAuthorizationHeader`: 対照のヘッダー提示が 401 になる |
  | `RFC6750-QUERY-TOKEN` | ヘッダーが無いときクエリの `access_token` へ退避する | 同テスト `/query_parameter_instead_of_the_header`: クエリ提示で UserInfo へ到達する |
  | `OIDC-CORE-USERINFO` | UserInfo の応答から `sub` を落とす | `TestUserInfoReturnsTheSubjectForAnOpenIDScopedToken`: `sub` が空になる |
  | `RFC7662-INTROSPECT` | 内省の応答の `scope` を落とす | `TestIntrospectionAnswersAuthenticatedResourceServers`: 発行したトークンの `scope` と一致しない |
  | 〃 | `/introspect` と `/token` のクライアント認証の失敗を無視する | 同テスト `/no_client_authentication`: 認証なしの内省が全メタデータを返す |
  | `RFC7662-INACTIVE` | `IntrospectionResponse.JTI` の `omitempty` を外す | `TestIntrospectionRevealsNothingAboutInactiveTokens`: 無効なトークンの本文が `active` 以外を運ぶ |
  | `RFC7009-REVOCATION-ENDPOINT` | 失効時に `AccessTokenDenylist.Add` を呼ばない | `TestRevocationIsOfferedToAuthenticatedClients`: 失効させたトークンが保護リソースへ到達する |
  | 〃 | `/revoke` がクライアント認証に失敗しても Basic の利用者名を主体として受け入れる | 同テスト: 誤った資格情報の失効要求が受理される |
  | `RFC7009-UNKNOWN-TOKEN` | リフレッシュトークンの所有者照合を外す | `TestRevokingAnUnknownOrForeignTokenIsAnIndistinguishableNoOp`: 他クライアントのリフレッシュトークンが失効する |
  | 〃 | アクセストークンの所有者照合を外す | 同テスト: 他クライアントのアクセストークンが失効する |
  | 〃 | 所有者でない要求へ `invalid_request` を返す | 同テスト `/refresh_token_owned_by_another_client`: 応答が自分のトークンの失効と区別できる |
  | `RFC7800-CONFIRMATION` | 署名時に `cnf` claim を載せない | `TestSenderConstraintIsRecordedInCnfAndCheckedAtTheResource`: DPoP トークンに `cnf` が無い |
  | `RFC9700-SENDER-CONSTRAINT` | `/userinfo` の送信者制約の検査ごと飛ばす | 同テスト: DPoP 束縛トークンが証明なしで通る |
  | `RFC8705-CLIENT-AUTH` | 登録済み Subject DN との照合を外す | `TestMutualTLSAuthenticatesTheClientAndBindsTheAccessTokenToItsCertificate`: 別の DN の証明書にトークンが出る |
  | `RFC8705-CERT-BOUND` | 認可コードの交換で mTLS サムプリントを送信者制約にしない | 同テスト: `cnf` が無い |
  | 〃 | `/userinfo` のサムプリント照合を、提示された証明書自身との比較にする | 同テスト `/another_valid_certificate`: 別の証明書でリソースへ到達する |
  | `RFC9449-PROOF` | 証明の `htu` の照合を外す | `TestDPoPProofElementsAreVerifiedAtTheTokenEndpointAndTheProtectedResource/htu_of_another_endpoint` |
  | 〃 | `jti` のリプレイ検知の結果を無視する | 同テスト: 同じ `jti` の証明が 2 回通る |
  | `RFC9449-ATH` | `ath` が無ければアクセストークンのハッシュで補う | 同テスト `/no_ath`: `ath` を持たない証明で保護リソースへ到達する |
  | 〃 | `ath` の照合を、提示された値自身との比較にする | 同テスト `/ath_of_another_access_token` |

  この方法の限界: 表の変異は 1 か所ずつだが、行に対応するテスト 1 つが production の 1 か所に対応して
  いるわけではない。`RFC7662-INTROSPECT` の認証の変異は `/token` と `/introspect` の両方の
  `authenticateTokenClient` 呼び出しに当たる。両者は同じ 1 行なので、分けて崩すことはできない。
- **Verification Results**:
  - `mise run check-spec` - passed（`ok normative coverage (156 standard(s), 311 rule(s), 744 example(s), 238 id(s) named by a test)`）
  - `mise run check-work-items` - passed（`ok 521 file(s)` / `ok 521 work-item dependency record(s)`）
  - `mise run check-ids` - passed（`ok 521 record id(s)`）
  - `mise run test-go-package -- ./backend/shared/http/server_http` - passed
  - `mise run format-go` - passed
  - `mise run lint-go` - passed（`httpNoBody` と `unparam` の指摘 3 件を直した）
  - `mise run verify` - passed
  - `mise run test-ui-e2e` - `N/A: 変更は Go のテスト 1 ファイルと tools の JSON、および work item だけで、ブラウザーへ到達する経路が無い。`
