---
depends_on: []
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-12
priority: p2
change_kind: bugfix
evidence_policy: risk-based-v3
documentation_impact:
  level: release_note
  reason: これまで `fapi_2_security_profile` を選んでも何も変わらなかったクライアントが、PAR、非対称クライアント認証、送信者制約付きトークンを課されるようになる。プロファイルを選んだ既存のクライアントは設定を変えないと通らなくなるので、リリースの読み手に見える。
  references:
    - { kind: release_note, path: docs/releases/changes/wi-532.md }
affected_spec:
  - { path: docs/contexts/oauth2/standards.md, requirement: FAPI2-PROFILE-SELECTION }
  - { path: docs/contexts/oauth2/standards.md, requirement: FAPI2-PAR-PKCE }
  - { path: docs/contexts/oauth2/standards.md, requirement: FAPI2-CLIENT-AUTH }
  - { path: docs/contexts/oauth2/standards.md, requirement: FAPI2-SENDER-CONSTRAINT }
initial_context:
  specification:
    - docs/contexts/oauth2/standards.md#FAPI2-PROFILE-SELECTION
    - docs/contexts/oauth2/standards.md#FAPI2-PAR-PKCE
    - docs/contexts/oauth2/standards.md#FAPI2-CLIENT-AUTH
    - docs/contexts/oauth2/standards.md#FAPI2-SENDER-CONSTRAINT
    - docs/contexts/oauth2/decisions.md
    - docs/contexts/oauth2/scenarios.feature.md#REQ-OAUTH2-009
    - docs/contexts/oauth2/scenarios.feature.md#REQ-OAUTH2-010
  typespec:
    - IdMagic.Contract.FapiProfile
    - IdMagic.Contract.OAuth2Client
  source:
    - backend/oauth2/client/domain/client.go
    - backend/oauth2/client/usecases/register_client.go
    - backend/oauth2/authorization/usecases/authorize.go
    - backend/oauth2/handlers_http/token_handler.go
    - backend/shared/spec/policy.go
  tests:
    - backend/oauth2/client/domain/client_test.go
    - backend/oauth2/authorization/usecases/authorize_test.go
    - backend/shared/http/server_http/metadata_standards_test.go
  stop_before_reading:
    - backend/oauth2/db_postgres
    - backend/saml
    - backend/wsfederation
    - frontend
primary_use_cases:
  - id: fapi-client-must-use-par
    requirement: FAPI2-PAR-PKCE
    observable_result: プロファイルを選んだクライアントの直接の `/authorize` が `invalid_request` で拒否され、PAR 経由なら同じリクエストが認可コードまで進む。
    unit_test: { path: backend/oauth2/authorization/usecases/authorize_test.go, name: TestAuthorizeRequiresPARFromFapi2Clients, task: test-go-race }
    e2e_test: { path: backend/shared/http/server_http/fapi_security_profile_e2e_test.go, name: TestFapi2ClientCannotStartAnAuthorizationRequestWithoutPAR, task: test-go-race }
    unit_fault_model: Authorize が PAR の必須判定に `RequirePushedAuthorizationRequests` だけを読み、プロファイルの選択を無視する。
    e2e_fault_model: "`/authorize` のハンドラーが Authorize を通さず、リクエストをそのまま保存する。"
  - id: fapi-client-must-authenticate-asymmetrically
    requirement: FAPI2-CLIENT-AUTH
    observable_result: プロファイルを選んだクライアントを `client_secret_basic` で登録しようとすると `invalid_client_metadata` で拒否され、`private_key_jwt` なら登録できる。
    unit_test: { path: backend/oauth2/client/domain/client_test.go, name: TestOAuth2ClientRejectsSharedSecretAuthUnderFapi2, task: test-go-race }
    e2e_test: { path: backend/shared/http/server_http/fapi_security_profile_e2e_test.go, name: TestRegisterRefusesAFapi2ClientThatAuthenticatesWithASharedSecret, task: test-go-race }
    unit_fault_model: OAuth2Client の検証がクライアント認証方式とプロファイルの組み合わせを見ない。
    e2e_fault_model: "`/register` が Validate の失敗を握りつぶし、クライアントを保存する。"
  - id: fapi-client-tokens-are-sender-constrained
    requirement: FAPI2-SENDER-CONSTRAINT
    observable_result: プロファイルを選んだクライアントが DPoP 証明も mTLS 証明書も付けずに `/token` を叩くと拒否され、DPoP 証明を付けると `cnf` を持つアクセストークンが返る。
    unit_test: { path: backend/oauth2/client/domain/client_test.go, name: TestOAuth2ClientRequiresSenderConstraintEvidenceUnderFapi2, task: test-go-race }
    e2e_test: { path: backend/shared/http/server_http/fapi_security_profile_e2e_test.go, name: TestFapi2ClientCannotObtainAnUnconstrainedAccessToken, task: test-go-race }
    unit_fault_model: SenderConstraintSatisfied がプロファイルを見ずに常に true を返す。
    e2e_fault_model: "`/token` が送信者制約の判定を呼ばず、証拠の無い要求にもトークンを発行する。"
  - id: non-fapi-client-is-unaffected
    requirement: FAPI2-PROFILE-SELECTION
    observable_result: プロファイルを選んでいないクライアントが、PAR 無しの `/authorize`、共有シークレットでの登録、証拠の無い `/token` のいずれも従来どおり通す。
    unit_test: { path: backend/oauth2/client/domain/client_test.go, name: TestOAuth2ClientLeavesNonFapi2ClientsUnconstrained, task: test-go-race }
    e2e_test: { path: backend/shared/http/server_http/fapi_security_profile_e2e_test.go, name: TestNonFapi2ClientKeepsWorkingWithoutTheProfileConstraints, task: test-go-race }
    unit_fault_model: 追加制約が `FapiProfile` ではなく全クライアントに掛かる。
    e2e_fault_model: PAR、非対称クライアント認証、送信者制約のいずれかが製品の既定になる。
---

# `Fapi2SecurityProfile` を選んでも追加の制約が 1 つも掛からない

## Motivation

`docs/contexts/oauth2/standards.md` の FAPI 2.0 Security Profile 節は 4 行を宣言する。

| ID | Adoption | Statement |
|---|---|---|
| `FAPI2-PROFILE-SELECTION` | optional | `Fapi2SecurityProfile` を選択したクライアントだけに本プロファイルの追加制約を適用する。 |
| `FAPI2-PAR-PKCE` | optional | FAPI クライアントは PAR と S256 PKCE を使用する。 |
| `FAPI2-CLIENT-AUTH` | optional | FAPI クライアントは `private_key_jwt` または mTLS で認証する。 |
| `FAPI2-SENDER-CONSTRAINT` | optional | FAPI アクセストークンに DPoP または mTLS による送信者制約を付ける。 |

`docs/contexts/oauth2/decisions.md` も同じ向きの判断を 2 つ持つ。PKCE は「公開クライアントと FAPI 2.0 クライアントではデフォルトで必須」であり、PAR は「FAPI 2.0 クライアントで必須」である。

実装は `fapi_profile` を保存し、列挙として検証し、admin API と `/register` の応答へ書き戻し、管理 UI へ表示する。**しかしこの値を読んで制約を掛ける箇所が 1 つも無い。** 非テストの Go 全体で `fapi` を探すと、当たるのは認可規則の名前 `par_required_if_fapi` と `authorize.go` の予定を書いたコメントだけである。

規則の名前は FAPI を名乗るが、実装は `backend/shared/spec/policy.go:403` のとおり `Subject.Properties.RequirePAR` を読む。

```go
"par_required_if_fapi": func(r AuthZRequest) bool { return !r.Subject.Properties.RequirePAR || r.Context.ParUsed },
```

`RequirePAR` は `RequirePushedAuthorizationRequests` に由来し、`backend/oauth2/client/usecases/register_client.go:148` が入力の同名フラグをそのまま写す。`FapiProfile` からは導かれない。送信者制約の `DpopBoundAccessTokens` も、クライアント認証方式 `TokenEndpointAuthMethod` も同じで、いずれも `FapiProfile` と無関係に決まる。

したがって `fapi_2_security_profile` を選んだクライアントは、PAR 無しの `/authorize` を通し、`plain` PKCE を使い、`client_secret_basic` で認証し、送信者制約の無いアクセストークンを受け取れる。**プロファイルの選択は現状、表示専用のラベルである。**

これは宣言と実装の不一致であり、[[wi-508-amr-vocabulary-declaration-and-implementation-disagree]] と同じ形の欠陥である。宣言した採用 `optional` は「提供しているならその振る舞いを観測できる」ことを意味するが、提供されていない。

## Scope

- `Fapi2SecurityProfile` を選んだクライアントに、4 行が言う追加制約を適用する。
  - PAR を必須とし、PKCE を `S256` に限る。
  - クライアント認証を `private_key_jwt` または mTLS に限る。
  - アクセストークンに DPoP または mTLS による送信者制約を付ける。
- プロファイルを選んでいないクライアントが同じリクエストで通り続けることを、対にして観測する。制約が全クライアントへ漏れると、既存のクライアントが一斉に壊れる。
- 制約を掛ける層を決める。`par_required_if_fapi` が既にある認可規則の層に寄せるのか、クライアントの登録時点で派生フラグを立てるのかは着手時に決める。後者は保存された値と宣言の整合を後から崩しうる。
- 名前と実装が食い違っている `par_required_if_fapi` を、実装に合わせるか名前に合わせるかを決める。

## Out of Scope

- `standards.md` の 4 行の `Adoption` および `Strength` の変更。実装を宣言へ合わせるのが本項目であり、逆ではない。
- FAPI 2.0 Message Signing。`standards.md` は宣言していない。
- [[wi-293-request-object-jar-and-jarm-signed-authorization-messages]] が持つ JAR と JARM。
- 標準行へのテストの対応付けと台帳からの削除。[[wi-522-back-oauth2-client-profile-standards-rows-with-tests]] が持ち、本項目の完了を待つ。

## Design

### 制約を掛ける層: 保存された `FapiProfile` を要求の評価時に読む

登録時に `FapiProfile` から `RequirePushedAuthorizationRequests` などの派生フラグを立てる案は採らない。派生フラグは保存後に admin API から個別に落とせるので、`fapi_profile` が `fapi_2_security_profile` のまま制約だけが消えた状態を作れてしまう。宣言と実装が再びずれる余地を残す設計である。

代わりに、制約の判断そのものをドメインの述語として置き、要求を評価する各地点がその述語を呼ぶ。派生した状態を保存しないので、ずれが生まれる場所が無い。

```go
// backend/oauth2/client/domain/client.go
func (c OAuth2Client) UsesFapi2SecurityProfile() bool
func (c OAuth2Client) MustUsePushedAuthorizationRequests() bool
func (c OAuth2Client) SenderConstraintSatisfied(dpopJKT, mtlsThumbprint string) bool
```

`SenderConstraintSatisfied` は送信者制約の証拠を引数で受ける。DPoP 証明の鍵サムプリントと mTLS 証明書のサムプリントはどちらもトランスポートの事実であり、ドメインが自力で取りに行けるものではない。判断はドメインに、証拠の供給は入口に置くことで、`/token` のハンドラーを起動しない単体テストでもこの判断を読める。

クライアント認証方式の制約だけは述語ではなく `OAuth2Client` の不変項とする。方式は保存されたクライアント自身の属性なので、要求ごとに評価する必要が無く、`Validate` に置けば `/register`、admin の作成、admin の更新が 1 か所で塞がる。保存できたクライアントは必ず適合しているという形になる。

### 3 つの適用地点

| 行 | 地点 | 判断 |
|---|---|---|
| `FAPI2-PAR-PKCE` | `Authorize` | `MustUsePushedAuthorizationRequests` が真で PAR 未経由なら `invalid_request`。 |
| `FAPI2-CLIENT-AUTH` | `OAuth2Client.Validate` | プロファイル選択時、`private_key_jwt` と `tls_client_auth` 以外を拒否する。 |
| `FAPI2-SENDER-CONSTRAINT` | `/token` のハンドラー | `SenderConstraintSatisfied` が偽なら `invalid_request`。 |

**S256 PKCE は既に全クライアントへ掛かっている。** `Authorize` は `code_challenge_method != "S256"` を無条件に拒否する。`FAPI2-PAR-PKCE` が言う PKCE の側は既に満たされているので、本項目はここへ手を入れず、満たされていることを観測するだけにする。`decisions.md` が「従来の confidential クライアントでは任意」と書いている点とは食い違うが、これは制約が強い側への食い違いであり、プロファイルの選択が無効という本項目の欠陥とは別の話である。手を付けない。

送信者制約の判定を `/token` の入口に置くのは、DPoP の検証がグラント種別の分岐より前に 1 度だけ走るからである。分岐の後に置くと、グラントを足すたびに同じ判断を書き写すことになり、書き忘れた経路だけ制約が消える。

### `par_required_if_fapi`: 名前どおりの実装にする

この規則は `ActionAuthorizeInitiate` に属するが、この action を評価する呼び出し元は製品に無い。認可規則の表は SCL の `authorization/access` を写した宣言であり、`/authorize` の実際の判断は `Authorize` が持つ。つまりこの規則は生きた防御ではなく宣言である。

宣言であるからこそ、名前と中身が食い違っていることの害が大きい。読んだ人は「FAPI の制約は既にある」と誤認する。実測では、まさにこの名前のせいで [[wi-522-back-oauth2-client-profile-standards-rows-with-tests]] の判定に時間が掛かった。

`AuthZSubjectProps` に `Fapi2SecurityProfile` を足し、規則がそれを読むようにする。呼び出し元が無いので振る舞いは変わらないが、表が嘘をつかなくなる。規則を `par_required_if_client_requires_par` へ改名する案は採らない。`decisions.md` が宣言している意図は FAPI 由来であり、名前の側が正しい。

### 既存のシナリオ

`EX-OAUTH2-009-02` は「PAR 必須の FAPI クライアント」と書くが、これまで固定していたのは `RequirePAR` だけだった。本項目の後はプロファイルの選択だけでも同じ結果になるので、この例の文言は初めて字義どおりになる。例そのものは変えない。文言が指す状態が 2 通りになっただけで、期待する結果は変わらないからである。

## Plan

1. 4 行に対応する e2e と単体の RED を置き、いずれも現状では通ってしまう（＝制約が無い）ことを観測する。
2. ドメインの述語と `Validate` の不変項を足し、`Authorize` と `/token` から呼ぶ。
3. プロファイルを選んでいないクライアントが従来どおり通ることを、同じテストの中で対にして読む。
4. 認可規則の表の `par_required_if_fapi` を名前どおりの実装にする。
5. 故障注入で、4 行それぞれの判断を崩すと対応するテストが落ちることを観測する。

## Tasks

- [x] T001 [Acceptance] `backend/shared/http/server_http/fapi_security_profile_e2e_test.go` に 4 件の e2e を置き、
  制約が無いために RED になることを観測する。4 件とも RED になった。観測は Completion。
  recipe: `mise run test-go-package -- ./backend/shared/http/server_http`
- [x] T002 [Domain] `UsesFapi2SecurityProfile`、`MustUsePushedAuthorizationRequests`、
  `SenderConstraintSatisfied` と `Validate` の不変項を足した。
  recipe: `mise run test-go-package -- ./backend/oauth2/client/domain`
- [x] T003 [Use Cases] `Authorize` が `MustUsePushedAuthorizationRequests` を読む。
  recipe: `mise run test-go-package -- ./backend/oauth2/authorization/usecases`
- [x] T004 [Adapters] `/token` のハンドラーが、グラント種別の分岐より前で
  `SenderConstraintSatisfied` を読む。
  recipe: `mise run test-go-package -- ./backend/oauth2/handlers_http`
- [x] T005 [Declaration] 認可規則 `par_required_if_fapi` が `Fapi2SecurityProfile` を読む。
  recipe: `mise run check-security-controls`
- [x] T006 [Resistance] 6 件の故障注入を行い、いずれも対象のテストが落ちた。Completion。
- [x] T007 [Verify] `mise run verify`。

## Verification

- `fapi_2_security_profile` を選んだクライアントが、PAR 無しの `/authorize`、`plain` PKCE、共有シークレットによるクライアント認証、送信者制約の無いアクセストークンのいずれでも進めない。
- 同じリクエストを `none` のクライアントが送ると、これまでどおり通る。
- `mise run verify`

## Risk Notes

- **制約が全クライアントへ漏れる。** プロファイルを選んでいないクライアントが同じリクエストで通ることを対にして観測しないと、PAR と非対称クライアント認証が既定になった実装と区別できない。既存のクライアントが一斉に壊れる向きの失敗なので、選んでいない側の観測を先に置く。
- **保存された値と宣言がずれる。** 登録時に `FapiProfile` から `RequirePAR` などの派生フラグを立てる設計を採ると、保存後に admin API がその派生フラグだけを落とせてしまい、`fapi_profile` は `fapi_2_security_profile` のまま制約が消える。判断の時点を要求の評価側へ寄せるか、派生フラグを更新契約の不変項目にするかを決める。
- **`par_required_if_fapi` の名前が先に嘘をついている。** 規則の名前を信じて読むと、FAPI の制約が既にあると誤認する。本項目はこの名前を実装に合わせるか、名前どおりの実装にするかを決めるまで終わらない。
- **既存のシナリオが用語を混同している。** `EX-OAUTH2-009-02` は「PAR 必須の FAPI クライアント」と書くが、固定しているのは `RequirePAR` であってプロファイルではない。実装を宣言へ合わせるとき、このシナリオが何を指すかを併せて決める。

## Completion
- **Completed At**: 2026-09-12
- **Summary**:
  `fapi_profile` に `fapi_2_security_profile` を選んだクライアントへ、FAPI 2.0 Security Profile の
  追加制約 3 つが実際に掛かるようになった。認可リクエストは PAR 経由に限られ、クライアント認証は
  `private_key_jwt` と `tls_client_auth` に限られ、アクセストークンは DPoP か mTLS の送信者制約を
  必要とする。判断は保存された `FapiProfile` から毎回導き、派生フラグは保存しない。プロファイルを
  選んでいないクライアントの振る舞いは 1 つも変わらない。
  あわせて `/register` が `private_key_jwt` のクライアントを 1 件も登録できなかった欠陥を直した。
  登録前の検証に使う候補が必須項目 `UpdatedAt` を欠いており、鍵の有無によらず全件が
  `private_key_jwt requires non-empty inline jwks` で落ちていた。この欠陥は本項目の Scope の直上に
  あった。FAPI クライアントの認証方式を非対称に限る以上、その方式で登録できなければ制約が
  空虚になるためである。
- **Primary Use Case Evidence**:
  - id: fapi-client-must-use-par
    unit_red: TestAuthorizeRequiresPARFromFapi2Clients は実装より後に書いたので、着手前の RED ではない。
      配線を旧実装へ戻して観測した RED は「プロファイルを選んだクライアントの PAR 無しリクエストが
      通った」である。着手前に観測した RED は同じ行の e2e と、述語が無く未コンパイルだった
      TestOAuth2ClientRequiresPushedAuthorizationRequestsUnderFapi2 である。
    e2e_red: TestFapi2ClientCannotStartAnAuthorizationRequestWithoutPAR が失敗した。プロファイルを選んだ
      クライアントの直接の `/authorize` が 303 でログインへ進んでいた。
    unit_fault_injection: MustUsePushedAuthorizationRequests からプロファイルの項を外すと、単体と e2e の
      両方が落ちた。Authorize を旧実装へ戻す注入でも TestAuthorizeRequiresPARFromFapi2Clients が落ちた。
    e2e_fault_injection: Authorize を `RequirePushedAuthorizationRequests` だけを読む形へ戻すと、e2e が落ちた。
  - id: fapi-client-must-authenticate-asymmetrically
    unit_red: TestOAuth2ClientRejectsSharedSecretAuthUnderFapi2 は不変項が無く、3 方式とも検証を通った。
    e2e_red: TestRegisterRefusesAFapi2ClientThatAuthenticatesWithASharedSecret が失敗した。
      `client_secret_basic` の FAPI クライアントが 201 で登録された。
    unit_fault_injection: 不変項の判定を `true` に置き換えると、単体 (`none` の事例) と e2e の両方が落ちた。
    e2e_fault_injection: 同じ注入で `/register` が共有シークレットの FAPI クライアントを 201 で作り、e2e が落ちた。
  - id: fapi-client-tokens-are-sender-constrained
    unit_red: TestOAuth2ClientRequiresSenderConstraintEvidenceUnderFapi2 は述語が無く未コンパイルだった。
    e2e_red: TestFapi2ClientCannotObtainAnUnconstrainedAccessToken が失敗した。証拠の無い
      client_credentials に `cnf` を持たないアクセストークンが発行された。
    unit_fault_injection: SenderConstraintSatisfied を常に `true` にすると、単体と e2e の両方が落ちた。
    e2e_fault_injection: トークンエンドポイントの `/token` の分岐を到達不能にすると、e2e が落ちた。
  - id: non-fapi-client-is-unaffected
    unit_red: TestOAuth2ClientLeavesNonFapi2ClientsUnconstrained は述語が無く未コンパイルだった。
    e2e_red: TestNonFapi2ClientKeepsWorkingWithoutTheProfileConstraints が失敗した。対照側の
      `/authorize` が 303 を返すのに、テストが 302 と 200 しか成功と認めていなかった。
    unit_fault_injection: UsesFapi2SecurityProfile を常に `true` にすると、単体と e2e の両方が落ちた。
      制約が全クライアントの既定になった実装は、この 1 件で捕まる。
    e2e_fault_injection: 同じ注入で、プロファイルを選んでいないクライアントの PAR 無し `/authorize` が
      拒否され、e2e が落ちた。
- **Change-Resistance Results**:
  6 件の故障注入を行い、いずれも対象のテストが落ちた。内訳は述語 3 件 (PAR、クライアント認証、
  送信者制約)、選択そのもの 1 件 (`UsesFapi2SecurityProfile` を常に真)、配線 2 件 (`Authorize` を
  旧実装へ戻す、`/token` の分岐を到達不能にする) である。
  生き残った変異は無い。選択そのものを崩す注入を入れているので、「制約を全クライアントへ既定化する」
  という、この変更で最も起こりやすい間違いが個別の行のテストより先に捕まる。
  `/register` の `UpdatedAt` については、鍵を伴わない `private_key_jwt` の登録が依然として拒否される
  ことを対照に置いた。これが無いと、拒否の理由が鍵の有無ではなく候補の構造だった状態へ戻っても
  テストが気づかない。
- **Verification Results**:
  - `mise run verify` - passed
