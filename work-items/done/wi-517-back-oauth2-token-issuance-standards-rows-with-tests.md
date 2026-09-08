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
    - backend/oauth2/handlers_http/token_handler.go
    - backend/oauth2/handlers_http/client_auth.go
    - backend/oauth2/token/usecases/exchange_code.go
    - backend/oauth2/token/usecases/exchange_token.go
    - backend/oauth2/token/usecases/refresh_tokens.go
    - backend/oauth2/usecases/resource_indicator.go
    - backend/shared/security/tokens_jose/jwt_signer.go
    - backend/oauth2/domain/delegation_mode.go
    - tools/check/standards-coverage-debt.json
  tests:
    - backend/shared/http/server_http/routes_e2e_test.go
    - backend/shared/http/server_http/authorization_request_standards_test.go
    - backend/oauth2/handlers_http/token_exchange_handler_test.go
    - backend/oauth2/handlers_http/client_auth_test.go
    - backend/oauth2/token/usecases/exchange_token_test.go
    - backend/oauth2/token/usecases/refresh_tokens_test.go
  stop_before_reading:
    - backend/saml
    - backend/wsfederation
    - backend/sourcing
    - frontend
---

# トークンの発行と交換が宣言する標準 15 行にテストを対応付け、標準の被覆台帳から外す

## Motivation

[[wi-499-back-oauth2-standards-rows-with-tests]] は `docs/contexts/oauth2/standards.md` の 80 行を引き取り、最初の節（`OAuth Client ID Metadata Document` の 7 行）を消化したうえで、残る 73 行を**行が共有する製品の入口**を単位に 7 件へ割った。本項目はそのうち 15 行を持つ。

15 行は、トークンエンドポイントが何を出すかを定める。JWT アクセストークンと ID トークンの claim と署名、`audience` の限定、リフレッシュトークンのローテーション、そしてトークン交換の委譲則がここに集まる。`RFC8693` の 4 行は `act` チェーンの入れ子と深さ制限を言っており、誤れば権限が交換のたびに広がる。

## Scope

- 次の 15 行を消化する。

| ID | Adoption | 節 |
|---|---|---|
| `RFC6749-CLIENT-CREDENTIALS` | optional | The OAuth 2.0 Authorization Framework |
| `RFC6749-PASSWORD-GRANT` | excluded | The OAuth 2.0 Authorization Framework |
| `RFC7523-CLIENT-ASSERTION` | optional | JSON Web Token Profile for OAuth 2.0 Client Authentication and Authorization Grants |
| `RFC9068-CLAIMS` | required | JSON Web Token Profile for OAuth 2.0 Access Tokens |
| `RFC9068-ASYMMETRIC-SIGNATURE` | required | JSON Web Token Profile for OAuth 2.0 Access Tokens |
| `RFC7518-SIGNATURE-ALGORITHMS` | required | JSON Web Algorithms |
| `RFC7519-REGISTERED-CLAIMS` | required | JSON Web Token |
| `OIDC-CORE-ID-TOKEN` | required | OpenID Connect Core 1.0 incorporating errata set 1 |
| `RFC8707-AUDIENCE` | required | Resource Indicators for OAuth 2.0 |
| `RFC8707-MCP-RESOURCE-BINDING` | required | Resource Indicators for OAuth 2.0 |
| `RFC9700-REFRESH-REPLAY` | required | Best Current Practice for OAuth 2.0 Security |
| `RFC8693-DELEGATION-DEFAULT` | required | OAuth 2.0 Token Exchange |
| `RFC8693-IMPERSONATION` | optional | OAuth 2.0 Token Exchange |
| `RFC8693-SUBJECT-TOKEN` | required | OAuth 2.0 Token Exchange |
| `RFC8693-DELEGATION-DEPTH` | required | OAuth 2.0 Token Exchange |

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

**観測の入口は トークンエンドポイント `/token` である。** [[wi-499-back-oauth2-standards-rows-with-tests]] が 7 行を測って出した結論は、消化の費用を支配するのが行数ではなく「行が共有する入口にハーネスを 1 つ組むこと」だという点にある。本項目の 15 行はこの入口を共有するので、ハーネスは 1 度組めば足りる。

この入口の観測は、`/token` へリクエストを送り、返ったトークンを復号して claim を直接読む形になる。到達できることだけでは足りない。[[wi-500-back-api-tokens-standards-rows-with-tests]] が測ったとおり、行が列挙している claim のうち認証が使わないものは、入口へ到達できるという観測では固定できない。

値の照合を言う行（`audience`、署名）は、壊れた文字列では観測できない。形式の検証で先に落ちるので、照合が無い実装でも同じ拒否になるからである。1 つの claim だけを差し替えてテナントの現行鍵で署名し直したトークンを作り、差し替えていない対照と対で読む。

行ごとの観測の形は `Adoption` が決める。`required` は宣言した振る舞いが正式な入口から到達できること、`optional` は提供しているならその振る舞い、`excluded` は提供していないことと拒否が防いだ効果、`partial` は採った範囲と採らなかった範囲の扱いを、それぞれ観測する。`excluded` の観測は 1 つの型に収まらない。行の `Statement` が製品の制約を書いているのか標準側の機能を書いているのかで観測が裏返るので、本項目の `excluded` の 1 件目でどちらかを決めてから残りへ広げる。

### ハーネスは `Register` が組んだスタック 1 つ（T002 で決めた）

15 行すべてを、`backend/shared/http/server_http` の `Register` が組み立てたスタックへ HTTP リクエストを出す形で観測した。テストは `backend/shared/http/server_http/token_issuance_standards_test.go` の 1 ファイルに置いた。

**テナント記録を本物にした。** 委譲深さの上限は `policy_tenancy.DelegationPolicyResolver` が Tenant Repository から解決する。解決器を代役へ差し替えると、`RFC8693-DELEGATION-DEPTH` について観測できるのは規則だけになり、その規則が `/token` へ配線されているかどうかは読めなくなる。上限そのものの規則（既定への退避をしないこと、上げる方向の上書きを丸めること）は `backend/oauth2/token/usecases/exchange_token_delegation_policy_test.go` が既に固定しているので、本項目はテナントが下げた上限が `/token` へ届くことだけを観測する。

**認可コードは正式な入口だけを通して取る。** 保存層へ直接置くと、交換の前提が製品の経路と違ってしまい、`RFC9068-*` と `OIDC-CORE-ID-TOKEN` が読む claim の出所が変わる。

### `excluded` の観測の型（T003、本文書で決めた）

`excluded` は `RFC6749-PASSWORD-GRANT` の 1 行だけである。標準側の機能を書いているので、[[wi-516-back-oauth2-authorization-request-standards-rows-with-tests]] と同じ向きの型を採った。

**利用者の正しい資格情報を載せた `grant_type=password` が通らないことと、通らなかった結果としてトークンが 1 つも出ていないことを対で読む。** 誤った資格情報で送ると、グラントを提供している実装でも同じ拒否になり、提供の有無を区別できない。

`optional` は `RFC6749-CLIENT-CREDENTIALS`、`RFC7523-CLIENT-ASSERTION`、`RFC8693-IMPERSONATION` の 3 行である。前 2 つは提供されていたので、その振る舞いを観測した。`RFC8693-IMPERSONATION` は形が違う。この行の `Statement` は「明示的に許可した場合だけ受け付ける」という**制約**であり、製品はその許可を持たない。したがって観測は「どの交換でも `sub` が入れ替わらず `act` が必ず載る」ことになる。制約を空虚に満たしているのではなく、**なりすましの形を作れる経路が無いことを、交換の出力から読んでいる**。規範の変更として切り出したものは無い。

## Plan

1. ~~15 行が共有する入口にハーネスを組み、`required` の 1 件目を通しで消化して型を決める。~~ 完了。Design の該当節。
2. ~~`optional` があれば、その 1 件目で「提供している」と言える根拠の形を決める。~~ 完了。3 行とも観測できた。
3. ~~`excluded` があれば、その 1 件目で観測の型を決める。~~ 完了。Design の該当節。
4. ~~残りを消化し、解決した id を台帳から外す。~~ 15 件すべてを外した（75 → 60）。
5. ~~宣言した採用を満たしていない行が見つかったら、欠陥の work item を切り出す。~~ 15 行とも満たされていた。切り出しは無い。

## Tasks

- [x] T001 [Acceptance] 消化する id を台帳から先に外し、`mise run check-spec` が当該 id ごとに `is declared, but no test names it` を報告することを観測する。
  15 件すべてについて観測した。Completion の Acceptance RED Evidence。
  recipe: `mise run check-spec`
- [x] T002 [Harness] `Register` が組み立てたスタックの `/token` へハーネスを組み、`required` の 1 件目で型を決める。
  `backend/shared/http/server_http/token_issuance_standards_test.go` に 6 テストを置いた。
  recipe: `mise run test-go-package -- ./backend/shared/http/server_http`
- [x] T003 [Type] `optional` と `excluded` の観測の型を、それぞれ 1 件目で決める。
  Design の「`excluded` の観測の型」節。`RFC8693-IMPERSONATION` だけは `optional` でありながら
  制約を書いている行なので、別の形で観測した。
- [x] T004 [Ledger] 残りを消化し、解決した id を `tools/check/standards-coverage-debt.json` から外す。
  15 件を外した（75 → 60）。
  recipe: `mise run check-spec`
- [x] T005 [Resistance] 行が言っている判断を production 側で崩し、対応するテストが落ちることを行ごとに観測する。
  20 件の変異を注入した。内訳は Completion の Change-Resistance Results。
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
- **値の照合を、壊れた文字列で観測してしまう。** 形式の検証で先に落ちるので、照合の有無を区別できない。1 要素だけを差し替えて再署名する。
- **台帳の同時編集。** 台帳は id 順に 1 エントリー 1 id なので、並行しても衝突はエントリー単位に収まる。自分が持つ id のエントリーだけを削除し、コメント配列と他の id には触れない。

## Completion

- **Completed At**: 2026-09-09
- **Summary**:
  `mise run spec-diff` は `no normative specification change against main` を返す。規範の変更は無い。

  `docs/contexts/oauth2/standards.md` のうち、トークンの発行と交換の入口に立つ **15 行**が、その行の
  `Statement` を区別できる入力と観測を持つテストを得て `tools/check/standards-coverage-debt.json` から
  消えた。台帳は 75 件から 60 件になった。テストは 1 ファイル
  （`backend/shared/http/server_http/token_issuance_standards_test.go`、6 テスト）で、`Register` が
  組み立てたスタックの `/token` へリクエストを出し、**返ったトークンの payload を復号して直接読む**。
  製品コードは 1 行も変わっていない。15 行とも宣言した採用を満たしており、欠陥の切り出しは無い。

  この 15 行が言っているのは「何を出すか」なので、応答が 200 であることや、そのトークンで保護リソースへ
  到達できることでは足りない。`jti` を落とす変異は「到達できる」という観測を生き残る。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-spec`（15 件を台帳から外し、テストを書く前に）
  - **Requirement**: N/A: 標準の被覆はテストの有無についての性質であり、製品の規範要求ではない。
  - **Observed Failure**: exit 1。`docs/contexts/oauth2/standards.md` の 15 行それぞれに
    `<ID> is declared, but no test names it. Cite the id from the test that exercises it, or list it in
    tools/check/standards-coverage-debt.json with a reason.`（10 行目 `RFC6749-CLIENT-CREDENTIALS` から
    244 行目 `OIDC-CORE-ID-TOKEN` まで 15 件）
  - **Detection Reason**: この検査は、宣言された id・テストが名指す id・台帳の 3 つを突き合わせる。台帳から
    外した id は、名指すテストが実在しない限り必ず報告される。15 件を消化した後の同じコマンドは
    `ok normative coverage (156 standard(s), 311 rule(s), 744 example(s), 225 id(s) named by a test)` を返す。
- **Unit RED Evidence**:
  - **Test**: 各行に対応付けたテストへの故障注入（Change-Resistance Results を参照）
  - **Requirement**: N/A: 行ごとの観測は標準の `Statement` に対応し、`REQ` 番号には対応しない。
  - **Observed Failure**: 15 行それぞれについて、対応する production の判断を崩すとそのテストが落ちる
    ことを観測した。内訳は Change-Resistance Results の表。
  - **Detection Reason**: `checkNormativeCoverage` は文字列の一致しか見ないので、id を書くだけで検査は
    通ってしまう。名指しが実在の検証に付いていることの担保は、production を崩したときにそのテストが
    落ちるという観測しかない。
  - **見つけた不足**: `RFC7523-CLIENT-ASSERTION` の有効期限を、`exp = 現在 - 1 分` で観測しようとして
    いた。**この事例は拒否されない。** 検証器は 60 秒の時計のずれを許容するので、境界ぎりぎりの値は
    「期限を見ていない実装」と「ずれの許容」を区別しない。当初これを製品の欠陥かと疑ったが、
    `client_assertion_verifier.go` の `clientAssertionClockSkewSeconds = 60` を読んで、テストの側の
    誤りだと分かった。`exp = 現在 - 10 分` へ変えたところ、期限の検査を外す変異を検出するようになった。
- **Change-Resistance Results**:
  15 行すべてについて、行が言っていることを production 側で崩し、対応するテストが落ちることを観測した。
  20 件の変異のうち 17 件は 1 か所を崩すだけで検出され、3 件は複数の防護をまとめて崩す必要があった。
  生き残りは無い。

  | 行 | 注入した故障 | 落ちたテストと観測 |
  |---|---|---|
  | `RFC6749-CLIENT-CREDENTIALS` | `/token` の `ClientType != Confidential` 検査を外す | `TestClientCredentialsIsConfidentialOnlyAndPasswordGrantIsNotOffered`: public クライアントにトークンが出る |
  | `RFC6749-PASSWORD-GRANT` | `password` をグラント種別・クライアントの宣言・`/token` の分岐の 3 か所すべてへ足す | 同テスト: `grant_type=password` にトークンが出る |
  | `RFC9068-CLAIMS` | `SignAccessToken` の claim から `jti` を落とす | `TestIssuedTokensCarryTheRegisteredClaimsAndAnAsymmetricSignature`: `jti` が無い |
  | `RFC7519-REGISTERED-CLAIMS` | 発行する `iss` の claim 名を変える | 同テスト: `iss` が無い |
  | `RFC9068-ASYMMETRIC-SIGNATURE` | header から `kid` を落とす | 同テスト: 公開鍵での検証ができない |
  | `RFC7518-SIGNATURE-ALGORITHMS` | header の `alg` を `HS256` にする | 同テスト: 対称鍵のアルゴリズムを名乗っている |
  | `OIDC-CORE-ID-TOKEN` | ID トークンから `auth_time` を落とす | 同テスト: 認証コンテキストが無い |
  | `RFC8707-AUDIENCE` / `RFC8707-MCP-RESOURCE-BINDING` | 認可コード交換で、解決した `resource` を `audience` にしない | `TestAccessTokenAudienceIsBoundToTheRequestedResource/認可コードの交換` |
  | `RFC9700-REFRESH-REPLAY` | 再利用の検知そのものを外す | `TestRefreshTokenRotatesAndReuseRevokesTheWholeFamily`: 使用済みの値が通る |
  | 〃 | 再利用を検知しても `RevokeFamily` を呼ばない | 同テスト: ローテーション後の値が検知の後も使える |
  | `RFC8693-DELEGATION-DEFAULT` | 発行する `sub` を現在の行為者へ置き換える | `TestTokenExchangeDelegatesByDefaultAndBoundsTheActorChain`: 委譲が元の利用者を保たない |
  | 〃 | 以前の行為者を `act` の内側へ入れ子にしない | 同テスト: 2 段目の `act` が入れ子でない |
  | `RFC8693-IMPERSONATION` | 発行するトークンから `act` を落とす | 同テスト: なりすましの形になっている |
  | `RFC8693-SUBJECT-TOKEN` | `verifyPS256AnyKey` が署名の検証結果を無視する | 同テスト `/claim_を差し替えた偽物` |
  | `RFC8693-DELEGATION-DEPTH` | `depth > maxDepth` の判定を外す | 同テスト `/テナントが下げた上限を超える交換は拒否される` |
  | `RFC7523-CLIENT-ASSERTION` | `iss == sub` の検査を外す | `TestClientAssertionVerifiesEveryDeclaredElement/sub_が別の値` |
  | 〃 | `aud` の検査を外す | 同テスト `/aud_が別の値` |
  | 〃 | `exp` の検査を外す | 同テスト `/exp_が過去` |
  | 〃 | `jti` の必須を外す | 同テスト `/jti_が無い` |
  | 〃 | `iss == client_id` の検査を外し、鍵の引き当ても `iss` ではなく `client_id` で行う | 同テスト `/iss_と_sub_がそろって別のクライアント` |

  この方法の限界: 3 つの行が単一の変異では倒れない。いずれも防護が重なっているためである。

  - **`RFC6749-PASSWORD-GRANT` は 3 か所に守られている。** `spec.GrantType.Valid()` の列挙、
    クライアントが宣言したグラント種別の照合、`/token` の分岐そのものである。実装が無いという最も強い
    形の `excluded` なので、単独の変異ではトークンを出す経路が生まれない。表の変異は「もし実装したら
    テストが捕まえる」ことを示すために 3 か所を同時に足したものである。
  - **`RFC7523-CLIENT-ASSERTION` の `iss == client_id` は 2 か所に守られている。** 検証器は鍵材料を
    `iss` で引くので、別のクライアントを名乗ると鍵が見つからずに落ちる。片方だけを外す変異は生き残る。
  - **`RFC8707-AUDIENCE` の「空でない audience」は、`resource` を指定しない経路では倒れない。**
    `Audiences` を空にしても、署名器が `client_id` へ退避するので観測が変わらない。この行を崩すには
    `resource` を指定する経路が要る（表の変異）。

  行としての被覆に穴は無いが、行に対応するテスト 1 つが production の 1 か所に対応しているわけではない。
- **Verification Results**:
  - `mise run check-spec` - passed（`ok normative coverage (156 standard(s), 311 rule(s), 744 example(s), 225 id(s) named by a test)`）
  - `mise run check-work-items` - passed
  - `mise run check-ids` - passed
  - `mise run test-go-package -- ./backend/shared/http/server_http` - passed
  - `mise run lint-go` - passed（`mapsloop` と `unparam` の指摘 3 件を直した）
  - `mise run verify` - passed
  - `mise run test-ui-e2e` - `N/A: 変更は Go のテスト 1 ファイルと tools の JSON、および work item だけで、ブラウザーへ到達する経路が無い。`
