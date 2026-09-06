---
depends_on: []
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-06
priority: p2
change_kind: maintenance
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: 標準の行にも製品の振る舞いにも変更が無く、増えたのはテストと注記だけなので、リリースの読み手に見えるものが無い。見つけた 2 件の欠陥は切り出した先が自分のリリース文書を持つ。
  references: []
spec_impact: { kind: none, reason: "宣言済みの標準行に、その id を名指しするテストを対応付ける作業である。standards.md の行そのものも製品の振る舞いも変えない。テストが書けない行が見つかった場合、それは製品が宣言した採用を満たしていないということなので、欠陥として個別の work item に切り出す。" }
initial_context:
  specification:
    - docs/contexts/api-tokens/standards.md
    - docs/contexts/api-tokens/internals.md
    - docs/contexts/api-tokens/README.md
  typespec: []
  source:
    - backend/apitoken/usecases/usecases.go
    - backend/apitoken/domain/token.go
    - backend/apitoken/handlers_http/routes.go
    - backend/apitoken/module.go
    - backend/shared/http/support_http/auth.go
    - backend/shared/http/support_http/admin_scope.go
    - backend/shared/http/support_http/csrf.go
    - backend/shared/http/support_http/tenant_middleware.go
    - backend/shared/http/server_http/routes.go
    - backend/shared/security/tokens_jose/jwt_signer.go
    - backend/shared/security/tokens_jose/dpop_verifier.go
    - backend/oauth2/handlers_http/token_handler.go
    - backend/oauth2/handlers_http/client_auth.go
    - backend/oauth2/token/usecases/introspect_token.go
    - backend/oauth2/token/usecases/revoke_token.go
    - tools/check/src/normative-coverage.ts
    - tools/check/src/check-specifications.ts
    - tools/check/standards-coverage-debt.json
  tests:
    - backend/shared/http/support_http/admin_scope_test.go
    - backend/shared/http/support_http/auth_test.go
    - backend/shared/http/server_http/tenant_routes_test.go
    - backend/shared/http/server_http/control_plane_boundary_test.go
    - backend/apitoken/handlers_http/handlers_test.go
    - backend/sourcing/scim/handlers_http/scim_test.go
  stop_before_reading:
    - docs/contexts/oauth2/standards.md
    - backend/oauth2/authorization
    - backend/saml
    - backend/wsfederation
    - frontend
---

# ApiTokens が宣言する標準 11 行にテストを対応付け、標準の被覆台帳から外す

## Motivation

[[wi-495-burn-down-the-standards-coverage-debt]] は標準の被覆台帳へ受入集合を入れ、消化の単位を所有文書と決めた。本項目はそのうち `docs/contexts/api-tokens/standards.md` の 11 行を引き取る。この文書は 12 行のうち 11 行が名指しを持たない。

11 行が扱うのは API アクセストークンの提示、内省、失効、そして送信者拘束である。この防護は他の Context のスコープ検査より手前にあり、ここが素通りすれば以降の検査は攻撃者の名乗った主体に対して働く。件数の少なさは重要度の低さを意味しない。

## Scope

- 次の 11 行を消化する。

| ID | Adoption |
|---|---|
| `RFC6750-API-TOKEN-HEADER` | required |
| `RFC6750-API-TOKEN-QUERY` | excluded |
| `RFC7009-API-TOKEN-REVOKE` | required |
| `RFC7009-API-TOKEN-UNKNOWN` | required |
| `RFC7662-API-TOKEN-INACTIVE` | required |
| `RFC7662-API-TOKEN-INTROSPECT` | required |
| `RFC9068-API-TOKEN-CLAIMS` | required |
| `RFC9068-API-TOKEN-SIGNATURE` | required |
| `RFC9449-API-TOKEN-DPOP` | optional |
| `RFC9700-API-TOKEN-AUDIENCE` | required |
| `RFC9700-API-TOKEN-SENDER-CONSTRAINT` | optional |

- 1 行ごとに、その行の `Statement` を区別できる入力と観測を持つテストを対応付け、`// <ID>: <この行の何を固定しているか>` の注記を足す。
- 観測の形は行の `Adoption` に従う。型は [[wi-495-burn-down-the-standards-coverage-debt]] の Design が定める。
- 消化した id を `tools/check/standards-coverage-debt.json` から外す。

## Out of Scope

- 他の文書が宣言する標準行。文書ごとに別の work item が持つ。OAuth2 の同名に見える行（`RFC6750-AUTHORIZATION-HEADER` など）は別の入口を指す別の行であり、[[wi-499-back-oauth2-standards-rows-with-tests]] が持つ。
- `standards.md` の行の追加、削除、`Adoption` および `Strength` の変更。
- 見つかった実装の欠陥の修正。欠陥として切り出した先で扱う。**T005 と T006 で 2 件見つけたので、[[wi-510-introspection-ignores-the-managed-token-lifecycle-record]] と [[wi-511-dpop-proof-htu-at-protected-resources-is-not-the-target-uri]] へ切り出した。**
- 受入集合の導入と、台帳ファイルそのものの削除。親の [[wi-495-burn-down-the-standards-coverage-debt]] が持つ。
- 具体例の被覆負債。[[wi-496-burn-down-the-example-coverage-debt]] が持つ。
- id を名指しする文字列があるだけを根拠にした台帳からの削除。

## Design

`required` の 8 行は、宣言した振る舞いが製品の正式な入口から到達できることを観測する。API トークンの検証はミドルウェアの位置にあるため、入口は保護されたエンドポイントであり、トークン検証関数の単体テストでは代わりにならない。関数が正しくても配線されていなければ素通りする。

`RFC6750-API-TOKEN-QUERY` は `excluded` である。クエリ文字列でのトークン送信を提供しないという行なので、観測は「提供していないこと」になる。クエリに正しいトークンを載せた要求が認証されないこと、および状態が変わっていないことを対で観測する。ヘッダーに載せた同じトークンが通ることを先に確かめないと、そのトークンが最初から無効だった実装と区別できない。

`optional` の 2 行（`RFC9449-API-TOKEN-DPOP`、`RFC9700-API-TOKEN-SENDER-CONSTRAINT`）は、提供しているならその振る舞いを観測する。提供していないなら、その行の `Adoption` が誤っているということなので規範の変更として切り出す。この 2 行は同じ送信者拘束の機構を別の角度から書いているので、実装の有無は 1 度読めば両方に答えが出る。

### 入口の選択: 組み立て済みのスタック 1 つ（T001 で決めた）

11 行のうち 10 行は、`backend/shared/http/server_http` の `Register` が組み立てたスタックへ HTTP 要求を出す形で観測した。テストは `backend/shared/http/server_http/api_token_standards_test.go` の 1 ファイルに置いた。

**Context ごとにテストを置く案を採らなかったのは、行が言っている防護がどれも配線に宿るからである。** `apitoken/usecases.Service.AuthenticateClaims` は audience もスコープも正しく照合するが、それが管理 API の手前に立っているかどうかは `routes.go` の組み立てが決める。実際、この Context には**正しく書かれていて配線だけ落ちている関数**が現に 1 つある（Design の「見つけた欠陥 1」）。関数の単体テストは、まさにその欠陥を素通りさせる。

保護されたエンドポイントには `GET /realms/default/api/admin/v1/users`（`users:read`）と `POST /realms/default/api/admin/v1/users/{sub}/disable`（`users:write`）を使う。後者は `excluded` の行が要求する「拒否が防いだ効果」を保存先から読み直すために要る。参照だけでは、拒否しながら状態を変える実装を見分けられない。

発行だけは HTTP から行えない。`POST /api/admin/v1/api-tokens` は対話セッション限定の operation として契約が宣言しており、API アクセストークンでは到達できない（`admin_scope_test.go` が既に固定している）。そこでテストは、`Register` へ渡したのと同じ repository の上に組んだ `apitoken/usecases.Service` から発行し、提示だけを HTTP から行う。行が言っているのは提示・内省・失効の側なので、この切り分けで観測は欠けない。

### 値だけが違う正しいトークンを作る

`RFC9700-API-TOKEN-AUDIENCE` と `RFC9068-API-TOKEN-SIGNATURE` は、壊れた文字列では観測できない。形式の検証で先に落ちるので、照合が無い実装でも同じ拒否になるからである。

そこでテストは `resignManaged` を持つ。発行済みトークンの payload を復号し、指定した claim だけを差し替えて、**テナントの現行署名鍵で署名し直す**。差し替えないで再署名したトークンが管理 API へ届くことを毎回の対照に置くので、拒否の理由が差し替えた 1 つの値であることが分かる。`RFC9068-API-TOKEN-SIGNATURE` では逆に、claim をそのままにして同じ `kid` を名乗る別の鍵で署名する。

### `excluded` の観測の型（本文書で決めた）

[[wi-495-burn-down-the-standards-coverage-debt]] は `excluded` の型を各子 work item がその文書の 1 件目で決めるとした。本文書の `excluded` は `RFC6750-API-TOKEN-QUERY` の 1 行だけであり、次の型を採った。

**その標準が定める提示の形を使った要求が、参照でも変更でも通らず、変更については状態も変わらないことを観測する。対照として、同じ 1 本のトークンを製品が受け付ける形で提示すると両方が通ることを併せて読む。**

対照が要るのは、拒否の一点だけでは「その提示の形を受け付けない」と「そのトークンがそもそも無効」を区別できないからである。変更の側では、`Origin` と double-submit CSRF トークンを満たしたうえで送る。満たさないと、拒否の理由がクエリの扱いではなくブラウザー検証になり、行とは別のものを観測してしまう。

[[wi-501-back-authentication-standards-rows-with-tests]] が `NIST63B4-PASSWORD-MINIMUM` で採った型（「採用した実装なら拒否する入力が受理されること」）とは形が逆になる。あちらの行は標準が課す制約を採らないことを言い、こちらの行は標準が許す機能を提供しないことを言っているからである。**`excluded` に 1 つの型は無い。行の `Statement` が製品の制約を書いているのか標準側の機能を書いているのかで、観測は裏返る。**

### 見つけた欠陥 1: 内省がライフサイクル記録を見ない（`RFC7662-API-TOKEN-INACTIVE` は消化できない）

この行だけは、行を満たすテストが書けない。

`DELETE /api/admin/v1/api-tokens/{id}` で失効させたトークンを `/introspect` へ提示すると、`active=true` と全 claim が返る。`routes.go:305` が作る `apiTokenService` は `ApiTokenAuthenticator` と `ManagedTokenRevoker` としては配線されているが、`TokenIntrospector` としては配線されていない。`/introspect` が使うのは生の `JWTSigner` で、その失効判定は `AccessTokenDenylist` と Agent の revocation epoch だけである。管理コンソールの失効は記録に `revoked_at` を立てるだけなので、どちらにも載らない。

**`apitoken/usecases.Service.IntrospectAccessToken` は、この重ね合わせのために書かれていて、production のどこからも呼ばれていない。** 同じトークンについて、自前の入口（拒否する）と `/introspect`（有効と申告する）が別の答えを出している。

行は失効の経路を限定していないので、この状態では行を満たしたことにならない。台帳には残し、理由を「投入時からある」から見つけた内容と切り出し先へ書き換えた。[[wi-510-introspection-ignores-the-managed-token-lifecycle-record]] が持つ。

内省が非活性を返す 4 通り（未知、`/revoke` を通した失効、期限切れ、レルム不一致）についてのテストは書いてあり、`active` 以外の鍵を 1 つも返さないことまで観測している。ただし行を名指してはいない。`checkNormativeCoverage` は台帳と名指しの両方に載る id を拒否するので、名指しは [[wi-510-introspection-ignores-the-managed-token-lifecycle-record]] の修正と同時に足す。

### 見つけた欠陥 2: 保護リソースの `htu` が target URI でない

`RFC9449-API-TOKEN-DPOP` を観測している途中で見つけた。`support_http/auth.go:150` は保護リソースの DPoP 証明を `RequestHTU(c, "")` と照合する。base が空文字なので期待値はパスだけになる。同じ関数を `/token` と `/userinfo` は `RequestHTU(c, d.Issuer)` で呼び、絶対 URL を期待する。RFC 9449 §4.2 と `spec/contexts/oauth2/models.tsp:607` の `htu: url` に従う適合クライアントは絶対 URL を送るので、**送信者制約付きの API アクセストークンは適合クライアントからは 1 度も使えない**。

この行そのものは満たされている。行が言っているのは 7 つの要素を検証することであり、7 つとも検証されている。期待値の形は行が述べていない。したがって行は消化し、欠陥は [[wi-511-dpop-proof-htu-at-protected-resources-is-not-the-target-uri]] へ切り出した。テスト側には、無傷の証明がパスを使っているのは現状に合わせただけで正しさの判断ではない、という注記を残した。

## Plan

1. ~~`RFC6750-API-TOKEN-HEADER` から着手し、保護されたエンドポイントを 1 つ選んで、`required` の観測の型を決める。~~ 完了。型は Design の「入口の選択」節。
2. ~~`RFC9068-API-TOKEN-SIGNATURE` と `RFC9068-API-TOKEN-CLAIMS` を進める。~~ 完了。
3. ~~`RFC9700-API-TOKEN-AUDIENCE` を進める。値だけが違う正しい形式のトークンで確かめる。~~ 完了。`resignManaged` を置いた。
4. ~~`RFC7662-*` と `RFC7009-*` の内省と失効を進める。失効は失効前の成功と対で観測する。~~ 3 行を消化。`RFC7662-API-TOKEN-INACTIVE` は消化できないと判断した。
5. ~~`RFC6750-API-TOKEN-QUERY` の `excluded` を進める。~~ 完了。型は Design の「`excluded` の観測の型」節。
6. ~~`RFC9449-API-TOKEN-DPOP` と `RFC9700-API-TOKEN-SENDER-CONSTRAINT` の実装の有無を確かめる。~~ 両方とも実装がある。観測した。
7. ~~解決した id を台帳から外す。~~ 完了。10 件を外し、1 件は理由を書き換えて残した。

## Tasks

- [x] T001 [Acceptance] `RFC6750-API-TOKEN-HEADER` を保護されたエンドポイントから観測し、`required` の型を決める。
  `backend/shared/http/server_http/api_token_standards_test.go` に `Register` が組み立てたスタックを組み、
  `TestApiTokenIsAcceptedOnlyFromTheAuthorizationHeaderScheme` を置いた。同じ 1 本のトークンをスキームだけ
  変えて 5 通り提示する。型は Design の「入口の選択」節。
  recipe: `mise run test-go-test -- ./backend/shared/http/server_http TestApiTokenIsAcceptedOnlyFromTheAuthorizationHeaderScheme`
- [x] T002 [Acceptance] `RFC9068-API-TOKEN-SIGNATURE` / `RFC9068-API-TOKEN-CLAIMS` / `RFC9700-API-TOKEN-AUDIENCE` を消化する。
  同ファイルの `TestManagedApiTokenIsSignedWithTheTenantAccessTokenKey`、
  `TestManagedApiTokenCarriesTheRFC9068Claims`、`TestApiTokenIsBoundToTheIssuingRealmAudience`。
  署名は、テナントの現行公開鍵で検証が通ること、通常の OAuth アクセストークンと `kid` が同じであること、
  同じ `kid` を名乗る別鍵の署名が届かないことの 3 つを読む。
  recipe: `mise run test-go-package -- ./backend/shared/http/server_http`
- [x] T003 [Acceptance] `RFC7662-API-TOKEN-INTROSPECT` / `RFC7662-API-TOKEN-INACTIVE` / `RFC7009-API-TOKEN-REVOKE` / `RFC7009-API-TOKEN-UNKNOWN` を消化する。
  3 行を消化した。`TestApiTokenIntrospectionReturnsTheIssuedTokenClaims`、
  `TestRevokingAManagedApiTokenTakesEffectImmediately`、
  `TestRevokingAnUnknownApiTokenIsAnIndistinguishableNoOp`。
  **`RFC7662-API-TOKEN-INACTIVE` は消化できなかった。** Design の「見つけた欠陥 1」。
  recipe: `mise run test-go-package -- ./backend/shared/http/server_http`
- [x] T004 [Acceptance] `RFC6750-API-TOKEN-QUERY` を、ヘッダーでの成功と対にして消化する。
  `TestApiTokenIsNotAcceptedFromTheQueryString`。参照と変更の両方をクエリで試し、変更については
  対象の利用者が無効化されていないことを保存先から読み直す。`excluded` の型は Design の該当節。
  recipe: `mise run test-go-test -- ./backend/shared/http/server_http TestApiTokenIsNotAcceptedFromTheQueryString`
- [x] T005 [Inventory] `RFC9449-API-TOKEN-DPOP` と `RFC9700-API-TOKEN-SENDER-CONSTRAINT` の実装の有無を確かめ、観測するか切り出すかを決める。
  両方とも実装がある。`TestDPoPBoundApiTokenVerifiesEveryProofElement` が 7 要素を 1 つずつ崩して観測し、
  `TestApiTokenSenderConstraintIsChosenAtIssuance` が制約なしと制約ありの 2 本を比べる。
  途中で `htu` の欠陥を見つけ、[[wi-511-dpop-proof-htu-at-protected-resources-is-not-the-target-uri]] へ切り出した。
  recipe: `mise run test-go-package -- ./backend/shared/http/server_http`
- [x] T006 [Defect] 宣言した採用を満たしていない行が見つかったら、欠陥の work item を切り出す。
  2 件切り出した。[[wi-510-introspection-ignores-the-managed-token-lifecycle-record]] と
  [[wi-511-dpop-proof-htu-at-protected-resources-is-not-the-target-uri]]。
- [x] T007 [Ledger] 解決した id を `standards-coverage-debt.json` から外す。
  10 件を外し（121 → 111）、`RFC7662-API-TOKEN-INACTIVE` は理由を書き換えて残した。
  recipe: `mise run check-spec`
- [x] T008 [Verify] `mise run verify`。

## Verification

- 本項目が持つ 11 件のうち 10 件が `tools/check/standards-coverage-debt.json` から消えている。残る 1 件
  （`RFC7662-API-TOKEN-INACTIVE`）は理由の欄が「投入時からある」から、欠陥の内容と切り出し先を名指す文へ
  書き換わっている。
- 注記を足した各テストについて、対応する production の判断を崩すとそのテストが落ちる。
- `mise run check-spec`
- `mise run verify`

## Risk Notes

- **手前の検証で落ちて後段の照合が確かめられない。** 理由ごとに 1 要素だけを崩したトークンを作る。崩した要素以外が有効であることは、正しいトークンが通ることで先に確認する。
- **`optional` の 2 行が「未実装だから観測できない」で止まる。** 止めない。実装が無いなら行の `Adoption` が誤っているので、規範の変更として切り出す。台帳へ残す場合は、理由を `present when the check was introduced` から具体的な理由へ書き換える。
- **台帳の同時編集。** 自分が持つ id のエントリーだけを削除し、コメント配列と他の id には触れない。
- **拒否の理由が、行とは別の防護になっている。** クエリ文字列の要求は Authorization ヘッダーを持たないので、ブラウザー検証（`Origin` と CSRF）が先に立つ。満たしてから送らないと、観測しているのが行ではなく CSRF になる。

## Completion

- **Completed At**: 2026-09-07
- **Summary**:
  `mise run spec-diff` は `no normative specification change against main` を返す。規範の変更は無い。
  差分は `docs/contexts/api-tokens/standards.md` の 12 行のうち、名指しを持たなかった 11 行に対する被覆の
  状態である。10 行がその行の `Statement` を区別できる入力と観測を持つテストを得て
  `tools/check/standards-coverage-debt.json` から消え、台帳は 121 件から 111 件になった。
  残る `RFC7662-API-TOKEN-INACTIVE` は、管理コンソールから失効させたトークンを `/introspect` が有効と
  申告するため行を満たすテストが書けない。台帳には残したうえで、理由の欄を「投入時からある」から欠陥の
  内容と切り出し先を名指す文へ書き換えた。テストは 1 ファイル（`api_token_standards_test.go`、11 テスト）で、
  `Register` が組み立てたスタックへ HTTP 要求を出す。製品コードは 1 行も変わっていない。
  見つけた欠陥 2 件を [[wi-510-introspection-ignores-the-managed-token-lifecycle-record]] と
  [[wi-511-dpop-proof-htu-at-protected-resources-is-not-the-target-uri]] へ切り出した。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-spec`（新しいテストファイルを退避し、10 件を台帳から外した状態で）
  - **Requirement**: N/A: 標準の被覆はテストの有無についての性質であり、製品の規範要求ではない。
  - **Observed Failure**: exit 1。`docs/contexts/api-tokens/standards.md` の 10 行それぞれについて
    `<ID> is declared, but no test names it. Cite the id from the test that exercises it, or list it in
    tools/check/standards-coverage-debt.json with a reason.`（9 行目 `RFC6750-API-TOKEN-HEADER` から
    55 行目 `RFC9700-API-TOKEN-SENDER-CONSTRAINT` まで 10 件）
  - **Detection Reason**: この検査は、宣言された id・テストが名指す id・台帳の 3 つを突き合わせる。台帳から
    外した id は、名指すテストが実在しない限り必ず報告される。つまり「台帳を縮めた」という主張は、テストを
    書かずには通せない。10 件を消化した後の同じコマンドは
    `ok normative coverage (156 standard(s), 311 rule(s), 744 example(s), 174 id(s) named by a test)` を返す。
- **Unit RED Evidence**:
  - **Test**: 各行に対応付けたテストへの故障注入（Change-Resistance Results を参照）
  - **Requirement**: N/A: 行ごとの観測は標準の `Statement` に対応し、`REQ` 番号には対応しない。
  - **Observed Failure**: 注記を足した 10 行それぞれについて、対応する production の判断を崩すとその
    テストが落ちることを観測した。内訳は Change-Resistance Results の表。17 件の変異をすべて検出した。
  - **Detection Reason**: `checkNormativeCoverage` は文字列の一致しか見ないので、id を書くだけで検査は
    通ってしまう。名指しが実在の検証に付いていることの担保は、production を崩したときにそのテストが
    落ちるという観測しかない。したがって本項目ではこれを Unit RED の代わりに置いた。
  - **見つけた不足**: `RFC9068-API-TOKEN-CLAIMS` は、当初「管理 API へ到達できる」ことだけで観測しようと
    していた。しかし `iat` を落とす変異はこの観測を生き残る。管理発行トークンの `authTime` は
    `resolveAuthnContext` が現在時刻で上書きするので、認証は `iat` を 1 度も読まないからである。**行が
    列挙している claim のうち、認証が使わないものは入口の観測では固定できない。** 復号した payload を
    直接読む形へ変えたところ、この変異を検出するようになった。
- **Change-Resistance Results**:
  10 行すべてについて、行が言っていることを production 側で崩し、対応するテストが落ちることを観測した。
  17 件の変異はすべて検出された（生き残りは無い）。

  | 行 | 注入した故障 | 落ちたテストと観測 |
  |---|---|---|
  | `RFC6750-API-TOKEN-HEADER` | `authorizationToken` の受け付けるスキームへ `basic` を追加 | `TestApiTokenIsAcceptedOnlyFromTheAuthorizationHeaderScheme/Basic_scheme`: 利用者一覧が 200 で返る |
  | 〃 | スキーム名を `Bearer` / `DPoP` に固定し、比較を `EqualFold` から `==` へ | 同テスト `/lowercase_bearer`: 同じスキームの小文字が 401 |
  | `RFC6750-API-TOKEN-QUERY` | `authorizationToken` が `access_token` クエリパラメーターへ fallback | `TestApiTokenIsNotAcceptedFromTheQueryString`: クエリのトークンで参照へ到達 |
  | `RFC9068-API-TOKEN-CLAIMS` | `SignAccessToken` の claim から `iat` を削除 | `TestManagedApiTokenCarriesTheRFC9068Claims`: `iat = <nil>` |
  | `RFC9068-API-TOKEN-SIGNATURE` | `typ` を `at+jwt` から `JWT` へ | `TestManagedApiTokenIsSignedWithTheTenantAccessTokenKey`: `typ = JWT` |
  | 〃 | `verifyPS256AnyKey` が `rsa.VerifyPSS` の結果を無視 | 同テスト: 同じ `kid` を名乗る別鍵のトークンが 200 で到達 |
  | `RFC9700-API-TOKEN-AUDIENCE` | `AuthenticateClaims` から `aud` と記録の照合を削除 | `TestApiTokenIsBoundToTheIssuingRealmAudience/another_realm's_API_audience`: 別レルムの audience で到達 |
  | `RFC9700-API-TOKEN-SENDER-CONSTRAINT` | `Issue` が渡された `dpop_jkt` を `SenderConstraint` にしない | `TestApiTokenSenderConstraintIsChosenAtIssuance`: 制約ありのトークンが証明なしで通る |
  | `RFC9449-API-TOKEN-DPOP` | `proof.JKT != res.SenderConstraint.JKT` の照合を削除 | `TestDPoPBoundApiTokenVerifiesEveryProofElement/thumbprint_of_another_key`: 別鍵の証明で到達 |
  | 〃 | `verifyDPoP` の `htu` 比較を無効化 | 同テスト `/htu_of_another_resource`: 別リソース向けの証明で到達 |
  | 〃 | リプレイ判定 `if !isNew` を無効化 | 同テスト: 同じ `jti` の証明が 2 回通る |
  | 〃 | `ath` の必須化と比較を無効化 | 同テスト `/ath_of_another_access_token`: 別トークンに結び付いた証明で到達 |
  | `RFC7662-API-TOKEN-INTROSPECT` | 内省の応答から `scope` を落とす | `TestApiTokenIntrospectionReturnsTheIssuedTokenClaims`: `scope = <nil>` |
  | 〃 | `handleIntrospect` からクライアント認証を削除 | 同テスト: 認証なしの内省が 200 と全 claim を返す |
  | `RFC7009-API-TOKEN-REVOKE` | `revokeAccessToken` から `ManagedTokenRevoker` の呼び出しを削除 | `TestRevokingAManagedApiTokenTakesEffectImmediately`: 記録に `revoked_at` が立たない |
  | 〃 | `AccessTokenDenylist.Add` を削除 | 同テスト: 失効させたトークンの内省が `active=true` |
  | `RFC7009-API-TOKEN-UNKNOWN` | 未知のトークンの失効でエラーを返す | `TestRevokingAnUnknownApiTokenIsAnIndistinguishableNoOp/unparsable`: 200 ではなく 500 |

  この方法の限界: `RFC6750-API-TOKEN-HEADER` について、受け付けるスキームから `dpop` を落とす変異は
  `TestApiTokenIsAcceptedOnlyFromTheAuthorizationHeaderScheme` を殺さない（実測）。同テストは Bearer と
  非スキームの区別しか読んでいないからである。この変異は
  `TestApiTokenSenderConstraintIsChosenAtIssuance` と `TestDPoPBoundApiTokenVerifiesEveryProofElement` が
  落として捕まえる。行としての被覆に穴は無いが、行に対応するテスト 1 つでは閉じていない。**行を 1 つの
  テストに閉じ込めると、その行が別の行と共有している機構を二重に書くことになる**ので、ここは分けたままに
  した。表の 17 件も、行と 1 対 1 ではなく、行が言っている判断ごとに 1 件を当てている。
- **Verification Results**:
  - `mise run check-spec` - passed（`ok normative coverage (156 standard(s), 311 rule(s), 744 example(s), 174 id(s) named by a test)`）
  - `mise run check-work-items` - passed
  - `mise run check-ids` - passed
  - `mise run test-go-package -- ./backend/shared/http/server_http` - passed
  - `mise run verify` - passed
  - `mise run test-ui-e2e` - `N/A: 変更は Go のテスト 1 ファイルと tools の JSON、および work item だけで、ブラウザーへ到達する経路が無い。`
