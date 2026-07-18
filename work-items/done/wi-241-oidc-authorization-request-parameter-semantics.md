---
status: completed
authors: [tn]
risk: high
created_at: 2026-07-18
depends_on: []
change_kind: bugfix
initial_context:
  scl:
    OAuth2:
      - standards.RFC6749.RFC6749-AUTHORIZATION-CODE
      - standards.RFC7636.RFC7636-VERIFY
      - standards.RFC9207.RFC9207-ISS
      - standards.OpenIDConnectCore.OIDC-CORE-CODE-FLOW
      - models.AuthorizeParameters
      - interfaces.Authorize
      - scenarios.認可コードフローでアクセストークンと ID トークンを取得できる
  source:
    - backend/oauth2/adapters/http/validation.go
    - backend/oauth2/adapters/http/authorize_handler.go
    - backend/oauth2/adapters/http/authorize_completion.go
    - backend/oauth2/domain/authorization_request.go
    - backend/oauth2/usecases/authorize.go
    - backend/oauth2/usecases/push_authorization_request.go
  tests:
    - backend/oauth2/adapters/http/validation_test.go
    - backend/oauth2/adapters/http/authorize_handler_test.go
    - backend/oauth2/adapters/http/par_handler_test.go
  stop_before_reading:
    - frontend
affected_spec:
  - { context: OAuth2, kind: standard_requirement, standard: RFC6749, requirement: RFC6749-AUTHORIZATION-CODE }
  - { context: OAuth2, kind: standard_requirement, standard: RFC7636, requirement: RFC7636-VERIFY }
  - { context: OAuth2, kind: standard_requirement, standard: RFC9207, requirement: RFC9207-ISS }
  - { context: OAuth2, kind: standard_requirement, standard: OpenIDConnectCore, requirement: OIDC-CORE-CODE-FLOW }
  - { context: OAuth2, kind: model, element: AuthorizeParameters }
  - { context: OAuth2, kind: interface, element: Authorize }
  - { context: OAuth2, kind: scenario, element: 認可コードフローでアクセストークンと ID トークンを取得できる }
---

# OIDC authorization request の重複パラメータと prompt を仕様どおりに fail-closed で処理する

## Motivation

現在の `/authorize` は query / PAR の値を単一文字列へ縮約してから parse するため、同名の
`client_id`、`redirect_uri`、PKCE、`prompt` などが複数指定されても先頭値だけを採用する。
また `prompt` を単一文字列として比較しており、OIDC で定義される空白区切りの値集合を検証して
いない。例えば `prompt=login consent` は login / consent の双方を強制せず、`prompt=none login`
の禁止組合せや未知値も拒否されない。

認可要求は redirect URI、PKCE、同意、既存セッションを束ねる外部入力である。parameter pollution や
未対応 prompt を黙って成功扱いすると、RP が要求した対話・再認証・同意を満たさない authorization
code を発行し得る。SCL が採用する Authorization Code + OIDC Code Flow の入力契約として、意味論を
明示し fail-closed にする必要がある。

## Scope

- `spec/contexts/oauth2.yaml` の `AuthorizeParameters`、`Authorize`、OIDC Core / OAuth 2.0
  requirement と scenario に、単一値 parameter の重複拒否、`prompt` の token grammar、対応値、
  `none` の排他性、失敗時の redirect/error 形式を明記する。
- `/authorize` と `/par` の両方で、security-sensitive parameter（少なくとも `client_id`、
  `redirect_uri`、`response_type`、`scope`、`state`、`nonce`、PKCE、`prompt`、`max_age`、
  `acr_values`、`request_uri`、`authorization_details`）の重複を検出して拒否する。PAR で保存した
  request と front-channel に混在する parameter の優先規則も仕様化する。
- `prompt` を空白区切りの集合として parse し、重複 token、`none` と他 token の併用、未対応 token
  を `invalid_request` にする。対応する `login` と `consent` は同時指定でも各々の意味を適用し、
  `none` は UI / redirect を発生させず `login_required` または `consent_required` を返す。
- redirect URI を安全に確定できた後の authorization error は、state と RFC 9207 `iss` を保持して
  登録済み redirect URI へ返す。URI を確定できない parse / client validation failure は IdP 側で
  返す。
- HTTP / domain / usecase contract test に query と PAR の重複、prompt の全組合せ、既存同意、
  stale session、`prompt=none`、state / issuer を含む error redirect を追加する。

## Out of Scope

- `id_token_hint` による logout client 解決、session inventory、front/back-channel logout
  （[[wi-28-session-management-and-oidc-logout-completion]]）。
- JAR (JWT Secured Authorization Request)、request object、CIBA、implicit / hybrid flow。
- authorization_details の type schema / consent rendering（既存 RFC 9396 実装）。

## Plan

- raw `url.Values` を失う前に cardinality を検査する。許可する複数値 parameter を将来追加する場合も
  allowlist で明示し、安易に `Values.Get` の先頭値へ依存しない。
- prompt は string comparison でなく value object とし、入力の canonical token set と、
  `login` / `consent` / `none` の評価結果を分ける。`none` は他の prompt と共存しない。
- 認可 error の出力先は「登録済み redirect URI を確定済みか」で分岐する。エラーを返す都合で
  redirect URI の厳密一致・state・issuer mix-up 防御を後退させない。
- PAR は request URI を唯一の入力正本とし、許容される外側 parameter 以外を混在させない。単一使用・
  tenant isolation の既存保証を維持する。

## Tasks

- [x] T001 [SCL] authorization parameter の cardinality、prompt grammar / outcome、PAR 混在規則、error redirect scenario を定義し `just yaml-check-scl` と `just scl-render` を GREEN にした。
- [x] T002 [Domain] RED: `TestParsePromptTokens` を `ParsePromptTokens` 未定義で fail 確認（OIDC-CORE-CODE-FLOW）→ login+consent、none 排他、重複、未知値を fail-closed にする value object を実装して GREEN。
- [x] T003 [HTTP/Usecase] RED: `TestParseAuthorizeRequestRejectsDuplicateSecurityParameter` を旧 `values.Get` 実装で fail 確認（AuthorizeParameters）→ query/PAR raw cardinality 検査、request_uri 混在拒否、prompt canonicalization と安全に確定済み request の error redirect を GREEN。
- [x] T004 [Flow] 既存 handler contract test の prompt=none を state/`iss` 付き `login_required` redirect へ更新し、login/consent、max_age、既存 consent の回帰 test とともに GREEN。none の同意不足は UI を出さず `consent_required` を返すよう completion を分岐した。
- [x] T005 [Verify] `just test-go`、`just verify-go`、`just yaml-check`、`just scl-render` を実行した。

## Verification

- `just yaml-check`
- `just scl-render`
- `just test-go`
- `just verify-go`
- 手動: `prompt=login%20consent` が再認証後に同意を表示し、`prompt=none%20login` と duplicate
  `redirect_uri` が authorization code を発行しないことを確認する。
- 手動: 検証済み redirect URI を持つ `prompt=none` failure が `state` と `iss` 付きで RP へ戻り、
  未検証 URI の error は RP に redirect されないことを確認する。

## Risk Notes

既存 RP が未定義の prompt や重複 parameter に依存している場合、成功から明示エラーへ変わる。
しかし authorization request の曖昧解釈は redirect / PKCE / consent の境界を曖昧にするため許容しない。
raw input・PAR・error redirect を同じ contract test で固定し、登録済み redirect URI 以外へは一切
遷移しないことを回帰防止する。

## Completion

- **Completed At**: 2026-07-18
- **Summary**:
  OIDC prompt を一意な空白区切り token 集合として解析し、重複・未知 token・`none` との併用を
  `invalid_request` とした。`/authorize` と `/par` は security-sensitive parameter の重複を raw
  input のまま拒否し、PAR の `request_uri` と他の authorization parameter の混在を拒否する。
  `prompt=none` のログイン／同意不能時は UI を開始せず、登録済み redirect URI へ state と RFC 9207
  `iss` を保った error を返す。
- **Affected Guarantees State**:
  認可要求は同じ名前の security-sensitive parameter を曖昧に解釈せず、OIDC prompt の未対応値を
  成功扱いしない。PAR request URI は唯一の要求正本であり、認証・同意を対話なしで満たせない
  `prompt=none` は authorization code を発行しない。
- **Verification Results**:
  - `just yaml-check-scl` / `just yaml-check-work-items` / `just scl-render` — passed
  - `just test-go` — passed
  - `just verify-go` — passed (golangci-lint 0 issues + race test)
  - `just yaml-check` — passed
- **対応していないこと (ADR-121 の開示義務)**:
  - `id_token_hint` による logout client 解決、session inventory、front/back-channel logout、JAR、
    CIBA、implicit/hybrid flow、authorization_details の schema/consent UI は対象外のままである。
