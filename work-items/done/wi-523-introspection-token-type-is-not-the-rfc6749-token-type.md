---
depends_on: []
status: completed
authors: [tn]
risk: low
reversibility: irreversible
created_at: 2026-09-09
priority: p3
change_kind: bugfix
evidence_policy: risk-based-v3
documentation_impact:
  level: upgrade_note
  reason: '公開済みの `/introspect` の応答から `token_type: "access_token"` / `"refresh_token"` が消え、アクセストークンには `Bearer` または `DPoP` が、リフレッシュトークンには何も入らなくなる。この値で分岐しているリソースサーバーは読み替えが要るので、リリースの読み手には互換性の変更として見える。'
  references:
    - { kind: release_note, path: docs/releases/changes/wi-523-introspection-token-type.md }
    - { kind: upgrade_note, path: docs/releases/upgrades/wi-523-introspection-token-type.md }
affected_spec:
  - { path: docs/contexts/oauth2/standards.md, requirement: RFC7662-INTROSPECT }
  - { path: spec/contexts/oauth2/models.tsp, symbol: IdMagic.Contract.IntrospectionResponse }
initial_context:
  specification:
    - docs/contexts/oauth2/standards.md#RFC7662-INTROSPECT
  typespec:
    - IdMagic.Contract.IntrospectionResponse
    - IdMagic.Contract.TokenResponse
  source:
    - backend/oauth2/ports/token_introspector.go
    - backend/shared/security/tokens_jose/jwt_signer.go
    - backend/oauth2/token/usecases/introspect_token.go
    - backend/oauth2/token/domain/token.go
    - backend/oauth2/domain/feature_aliases.go
    - backend/oauth2/handlers_http/token_handler.go
    - backend/oauth2/token/usecases/exchange_code.go
    - backend/oauth2/token/usecases/refresh_tokens.go
    - backend/oauth2/token/usecases/exchange_token.go
    - backend/oauth2/device/usecases/device_flow.go
    - backend/oauth2/approval/usecases/approval_flow.go
  tests:
    - backend/shared/http/server_http/token_presentation_standards_test.go
    - backend/oauth2/token/usecases/introspect_token_test.go
    - backend/oauth2/handlers_http/introspect_handler_test.go
  stop_before_reading:
    - frontend
    - backend/saml
    - backend/wsfederation
primary_use_cases:
  - id: introspect-bearer-token-type
    requirement: RFC7662-INTROSPECT
    observable_result: '制約なしのアクセストークンを内省したリソースサーバーが、`/token` の応答と同じ `token_type: "Bearer"` を受け取る。同じ発行で得たリフレッシュトークンの内省には `token_type` が無い。'
    unit_test: { path: backend/oauth2/token/usecases/introspect_token_test.go, name: TestIntrospectTokenReportsTheRFC6749PresentationType, task: test-go-race }
    e2e_test: { path: backend/shared/http/server_http/token_presentation_standards_test.go, name: TestIntrospectionAnswersAuthenticatedResourceServers, task: test-go-race }
    unit_fault_model: 内省の応答が提示形式を導かず、トークンの区分名 (`access_token` / `refresh_token`) をそのまま `token_type` に置く。
    e2e_fault_model: '`/introspect` が `/token` とは別の規則で `token_type` を決め、同じ 1 本のトークンについて 2 つの応答が食い違う。'
  - id: introspect-dpop-token-type
    requirement: RFC7662-INTROSPECT
    observable_result: 'DPoP で束縛したアクセストークンを内省したリソースサーバーが `token_type: "DPoP"` を受け取り、`cnf` を自分で解釈せずに提示形式を決められる。'
    unit_test: { path: backend/oauth2/token/domain/token_test.go, name: TestPresentationTokenTypeNamesTheRFC6749Vocabulary, task: test-go-race }
    e2e_test: { path: backend/shared/http/server_http/token_presentation_standards_test.go, name: TestSenderConstraintIsRecordedInCnfAndCheckedAtTheResource, task: test-go-race }
    unit_fault_model: 提示形式の判定が送信者制約の種別を見ず、束縛があれば一律に `DPoP` を返して mTLS 束縛のトークンまで `DPoP` と名乗る。
    e2e_fault_model: '内省の経路が `cnf` を読む前に `token_type` を決め、DPoP 束縛のトークンが `Bearer` として返る。'
---

# 内省の `token_type` が RFC 6749 §5.1 の種別ではなく、トークンの区分を名乗っている

## Motivation

[[wi-518-back-oauth2-token-presentation-standards-rows-with-tests]] が `RFC7662-INTROSPECT` にテストを対応付けているときに見つけた。

`backend/shared/security/tokens_jose/jwt_signer.go:257` は、アクセストークンの内省結果に `TokenType: "access_token"` を置く。`backend/oauth2/token/usecases/introspect_token.go:115` は、リフレッシュトークンの内省結果に `"refresh_token"` を置く。どちらも `/introspect` の応答の `token_type` としてそのまま出る。

RFC 7662 §2.2 は `token_type` を「Type of the token as defined in Section 5.1 of OAuth 2.0」と定めており、RFC 6749 §5.1 が定める値は `Bearer` である。DPoP による送信者制約付きトークンなら RFC 9449 §5 の `DPoP` になる。**製品が返しているのは、この語彙のどれでもない。**

同じ製品の中で `/token` の応答は正しい語彙を使っている。`backend/oauth2/handlers_http/token_handler.go:218` は `Bearer` と `DPoP` を出し分けている。**発行時と内省時で、同じ 1 本のトークンについて `token_type` が違う語彙で語られる。**

RFC 7662 に従うリソースサーバーは、内省の `token_type` を読んでトークンの提示形式を決める。`access_token` はどの提示形式でもないので、そのリソースサーバーは提示形式を決められない。送信者制約付きのトークンでは、`DPoP` を期待する経路が `access_token` を読むことになり、制約を無視する側へ倒れる。

`spec/contexts/oauth2/models.tsp:1019` は `token_type?: string` と宣言するだけで値を制約していないので、どの正典文書もこの語彙を定めていない。

## Scope

- `/introspect` の `token_type` を RFC 6749 §5.1 / RFC 9449 §5 の語彙 (`Bearer`、`DPoP`) に揃える。
- 揃えた語彙を `spec/contexts/oauth2/models.tsp` の `IntrospectionResponse.token_type` に宣言する。
- アクセストークンとリフレッシュトークンの区分を、リソースサーバーが読める形で残すかどうかを決める。RFC 7662 はこの区分を運ぶメンバーを定めていない。
- `backend/shared/http/server_http/token_presentation_standards_test.go` が現状の値を固定している注記を、決めた語彙へ書き換える。

## Out of Scope

- `/token` の応答の `token_type`。既に正しい語彙である。値も、TypeSpec の `TokenResponse.token_type` の宣言も変えない。
- `token_type_hint` の語彙。RFC 7009 §2.1 が別に定めており、`access_token` / `refresh_token` で正しい。
- api-tokens コンテキストの内省。同じ関数を通るので影響は受けるが、その行は
  [[wi-500-back-api-tokens-standards-rows-with-tests]] が固定している。

## Design

**提示形式は導出値であり、記録された値ではない。** トークンをどう提示するかは送信者制約から一意に決まる。制約が無ければ `Bearer`、DPoP 束縛なら `DPoP`、mTLS 束縛は RFC 8705 §3 のとおり `Bearer` である。この規則はいま製品の 6 か所に同じ 4 行で写されている (`token_handler.go`、`exchange_code.go`、`refresh_tokens.go`、`exchange_token.go`、`device_flow.go`、`approval_flow.go`)。内省へ 7 つ目の写しを足すと「`/token` と `/introspect` が一致する」という条件は、7 か所を目で読み比べる約束にしかならない。

そこで規則を 1 つの関数へ引き出す。

```go
// backend/oauth2/token/domain
func PresentationTokenType(sc *SenderConstraint) string
```

引数は送信者制約だけであり、時刻も乱数も設定も読まない。既存の 6 か所はこの関数の呼び出しへ置き換える。`/token` の応答の値は変わらない (振る舞いを変えない整理である)。内省もこの関数を通すので、一致は構造として保たれる。

**`ports.IntrospectionResult` から `TokenType` を落とす。** 提示形式を導けるのは `SenderConstraint` であり、それは同じ構造体が既に運んでいる。フィールドを残すと、JWT を検証する層が提示形式の語彙を知っていることになり、語彙が 2 か所に住む。落として、ユースケースが `PresentationTokenType(r.SenderConstraint)` で導く。本番でこのフィールドを書いているのは `jwt_signer.go` の 1 か所だけであり、読んでいるのは `introspect_token.go` の 1 か所だけである。

**リフレッシュトークンの内省は `token_type` を返さない。** RFC 6749 §5.1 が定める種別は、アクセストークンを保護リソースへどう提示するかを言う。リフレッシュトークンにはその意味の種別が無く、RFC 7662 §2.2 の `token_type` は OPTIONAL である。無い値を名乗るより、返さないほうが正確である。

区分そのものは落とす。RFC 7662 はこの区分を運ぶメンバーを定めておらず、製品にもこの値で分岐する経路は無い (`token_type` を読む本番コードは応答の組み立て以外に存在しない)。独自メンバーを新設する案 (`token_use` など) は採らない。公開する名前を 1 つ増やすのは取り消しにくい決定であり、それを正当化する読み手がいまはいないためである。内省を呼ぶ側は、自分が何を送ったかを知っているか、`token_type_hint` で言えるかのどちらかである。必要になったら、そのときの読み手を持って別の work item で足す。

**無効なトークンには `token_type` を付けない。** `PresentationTokenType(nil)` は `Bearer` を返すので、無条件に代入すると `active: false` の応答が `token_type: "Bearer"` を運ぶ。RFC 7662 §2.2 は無効なトークンに `active` 以外を返さないことを求める (`RFC7662-INACTIVE`)。既にある `if r.Active` の中で導出する。

**語彙は `IntrospectionResponse.token_type` の型として直接宣言する。** 名前付きの `enum` にはしない。`/token` の `TokenResponse.token_type` は Out of Scope なので、`enum` を作っても利用者は 1 か所しかない。公開する契約に名前を 1 つ増やすのは、共有する相手ができてからでよい。

この判断の途中で、道具の欠陥を 1 つ見つけた。`tools/check/src/spec-diff.ts` は追加された TypeSpec 宣言を `<path>:<name>` の形で数えるが、`tools/check/src/work-item-references.ts` は `affected_spec` の `symbol` を最後のドット区切りが宣言名であるものとして解決する。両方を満たす文字列が無いので、TypeSpec の宣言を追加した work item は、その追加を `affected_spec` で名乗れない。名乗り手がいない追加は開いている全記録へ配られるため、無関係な記録の `documentation_impact` が押し上げられる。個別の記録へ切り出す。

**却下した案**: 内省でも `access_token` を返し続け、standards.md の行のほうを製品に合わせる。RFC 7662 §2.2 が RFC 6749 §5.1 を名指しで参照している以上、行を書き換えても標準に従っていることにはならない。

## Plan

1. TypeSpec の `IntrospectionResponse.token_type` へ語彙を宣言し、`mise run check-spec` を通す。
2. E2E の RED を先に固定する。`token_presentation_standards_test.go` の 2 つのテストを、決めた語彙で読むよう書き換える。
3. `PresentationTokenType` を `backend/oauth2/token/domain` に置き、Unit RED → GREEN。
4. `ports.IntrospectionResult.TokenType` を落とし、内省のユースケースで導出する。Unit RED → GREEN。
5. `/token` 側の 6 か所を同じ関数へ寄せる。振る舞いは変えないので、既存のテストが GREEN のままであることが条件である。
6. リリースノートとアップグレードノートを書き、`mise run verify`。

## Tasks

- [x] T001 [Spec] `IntrospectionResponse.token_type` の型として語彙を宣言する。
  recipe: `mise run check-spec`
- [x] T002 [Acceptance] `token_presentation_standards_test.go` の 2 つのテストを決めた語彙で書き換え、E2E RED を観測する。
  recipe: `mise run test-go-test -- ./backend/shared/http/server_http TestIntrospectionAnswersAuthenticatedResourceServers`
- [x] T003 [Domain] `PresentationTokenType` を `backend/oauth2/token/domain` へ置き、`backend/oauth2/domain` から再公開する。
  recipe: `mise run test-go-test -- ./backend/oauth2/token/domain TestPresentationTokenTypeNamesTheRFC6749Vocabulary`
- [x] T004 [Use Cases] `ports.IntrospectionResult.TokenType` を落とし、内省のユースケースで `active` のときだけ導出する。
  recipe: `mise run test-go-test -- ./backend/oauth2/token/usecases TestIntrospectTokenReportsTheRFC6749PresentationType`
- [x] T005 [Adapters] `/token` の 6 か所を同じ関数へ寄せる。振る舞いは変えない。
  recipe: `mise run test-go-package -- ./backend/oauth2/handlers_http`
- [x] T006 [Docs] `docs/releases/changes/wi-523-introspection-token-type.md` と
  `docs/releases/upgrades/wi-523-introspection-token-type.md` を書く。
- [x] T007 [Verify] `mise run verify`。

## Verification

- 送信者制約なしのアクセストークンの内省が `Bearer` を返す。
- DPoP 束縛のアクセストークンの内省が `DPoP` を返す。
- 同じ 1 本のトークンについて、`/token` の応答と `/introspect` の応答の `token_type` が一致する。
- リフレッシュトークンの内省が `token_type` を返さない。
- `active: false` の内省の応答が `token_type` を運ばない。
- `mise run verify`

## Risk Notes

- **公開済みの応答の値を変える。** リソースサーバーが `access_token` を読んで分岐していれば壊れる。未リリースなので移行は不要だが、値の意味は取り消せない。
- **区分の情報を落とすと、内省だけではアクセストークンとリフレッシュトークンを見分けられなくなる。** Design で落とすことを決めた。読み手が現れたら別の work item で足す。
- **`/token` 側の 6 か所を寄せる整理が、寄せた先で振る舞いを変える。** 6 か所は同じ形に見えるが、参照している送信者制約の変数が違う。1 か所でも取り違えると、その grant のトークンだけ提示形式を誤って名乗る。grant ごとに既存のテストが GREEN のままであることを条件にする。
- **`active: false` に導出値が漏れる。** 提示形式は制約が無ければ `Bearer` を返すので、無条件の代入は無効なトークンの応答を汚す。`RFC7662-INACTIVE` を固定しているテストが対照になる。

## Completion

- **Completed At**: 2026-09-10
- **Summary**:
  `/introspect` の `token_type` が、トークンの区分名から RFC 6749 §5.1 の提示形式になった。
  制約なしのアクセストークンは `Bearer`、DPoP 束縛は RFC 9449 §5 の `DPoP`、mTLS 束縛は
  RFC 8705 §3 のとおり `Bearer` を名乗る。リフレッシュトークンと無効なトークンは何も名乗らない。
  `mise run spec-diff` は規範差分なしを返す。追加も削除もした宣言が無く、変わったのは
  `IntrospectionResponse.token_type` の型 (`string` から `"Bearer" | "DPoP"`) と doc だからである。
  規範の側では、本項目は既存の `RFC7662-INTROSPECT` を実装で満たしにいった記録である。
  実装は 3 つに分かれる。提示形式の規則を `backend/oauth2/token/domain` の
  `PresentationTokenType` へ 1 つだけ置き、`/token` に写されていた同じ 4 行 6 か所をそこへ寄せ、
  内省のユースケースが `active` のときだけ同じ関数で導く。`ports.IntrospectionResult` からは
  `TokenType` を落とした。提示形式は `SenderConstraint` から導けるので、JWT を検証する層が
  提示形式の語彙を知っている必要が無い。
  **規則を 1 つにしたことが効いた。** 「`/token` と `/introspect` が一致する」は、7 か所を
  読み比べる約束ではなく、同じ関数を通るという構造になった。E2E も約束ではなく、発行時に
  返った値と内省の値を突き合わせる形で読んでいる。
- **Primary Use Case Evidence**:
  - id: introspect-bearer-token-type
    unit_red: "TestIntrospectTokenReportsTheRFC6749PresentationType は 3 つの subtest で落ちた。制約なしのアクセストークンで token_type=\"\", want Bearer、DPoP 束縛で token_type=\"\", want DPoP、リフレッシュトークンで token_type=\"refresh_token\", want 空。"
    e2e_red: "TestIntrospectionAnswersAuthenticatedResourceServers は token_type=access_token, want Bearer と、内省の token_type=access_token, /token の token_type=\"Bearer\" と食い違っている と、リフレッシュトークンの内省が token_type=refresh_token を運んでいる の 3 件で落ちた。"
    unit_fault_injection: "導出を resp.TokenType = \"access_token\" に置き換え、リフレッシュ側へ TokenType: \"refresh_token\" を戻すと、同テストが token_type=\"access_token\", want Bearer と want DPoP で落ちた。"
    e2e_fault_injection: "同じ故障で TestIntrospectionAnswersAuthenticatedResourceServers が token_type=access_token, want Bearer と /token との食い違いで落ちた。"
  - id: introspect-dpop-token-type
    unit_red: "TestPresentationTokenTypeNamesTheRFC6749Vocabulary は undefined: PresentationTokenType でコンパイルできず落ちた。規則がまだどこにも無い状態である。"
    e2e_red: "TestSenderConstraintIsRecordedInCnfAndCheckedAtTheResource は DPoP 束縛トークンの内省の token_type=access_token, want DPoP と、mTLS 束縛トークンの内省の token_type=access_token, want Bearer で落ちた。"
    unit_fault_injection: "PresentationTokenType の条件を if sc != nil へ緩めて束縛があれば一律 DPoP を返すようにすると、同テストの mTLS 束縛 と 種別の無い制約 の 2 つが PresentationTokenType=\"DPoP\", want \"Bearer\" で落ちた。"
    e2e_fault_injection: "内省のユースケースが PresentationTokenType(nil) を呼ぶようにして cnf を読まないようにすると、TestSenderConstraintIsRecordedInCnfAndCheckedAtTheResource が DPoP 束縛トークンの内省の token_type=Bearer, want DPoP で落ちた。"
- **Verification Results**:
  - `mise run test-go-test -- ./backend/oauth2/token/domain TestPresentationTokenTypeNamesTheRFC6749Vocabulary` - passed
  - `mise run test-go-test -- ./backend/oauth2/token/usecases TestIntrospectTokenReportsTheRFC6749PresentationType` - passed
  - `mise run test-go-package -- ./backend/oauth2/token/usecases` - passed
  - `mise run test-go-changed` - passed（変更が package をまたいだあとに実行し、失敗 0）
  - `mise run check-spec` - passed（171 正準文書、156 規範、311 規則、744 例）
  - `mise run check-api-compat` - passed（破壊的変更なし）
  - `mise run spec-diff` - passed（規範差分なし）
  - `mise run check-work-items` - passed
  - `mise run check-ids` - passed
  - `mise run test-ui-e2e` - `N/A: 変更は /introspect の応答の値だけで、ブラウザーへ到達する経路が無い。frontend が読む token_type は WS-Federation のリライングパーティー設定であり、この応答とは無関係である。`
  - `mise run verify` - passed（終了コード 0 を直接確認）
