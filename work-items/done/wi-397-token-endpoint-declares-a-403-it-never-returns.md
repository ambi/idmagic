---
depends_on: []
status: completed
authors: [tn]
risk: low
created_at: 2026-08-23
priority: p2
change_kind: bugfix
evidence_policy: risk-based-v2
initial_context:
  specification: [docs/contexts/oauth2/standards.md]
  typespec:
    - IdMagic.OAuth2.Operations.Token
    - IdMagic.OAuth2.Operations.Authorize
  source:
    - backend/oauth2/handlers_http/errors.go
    - backend/oauth2/handlers_http/token_handler.go
    - backend/oauth2/handlers_http/authorize_handler.go
    - backend/oauth2/handlers_http/authorize_login.go
  tests:
    - backend/oauth2/handlers_http
  stop_before_reading: [frontend, backend/sourcing, backend/idmanagement]
affected_spec:
  - { path: spec/contexts/oauth2/main.tsp, symbol: IdMagic.OAuth2.Operations.Token }
  - { path: spec/contexts/oauth2/main.tsp, symbol: IdMagic.OAuth2.Operations.Authorize }
---

# Token endpoint が返さない 403 を契約が宣言している

## Motivation

[[wi-391-refusal-declaration-floor-and-reinventory]] の R4 が、契約とシナリオを突き合わせて見つけた食い違いである。

`spec/contexts/oauth2/main.tsp` の `Token` は `TokenError403` として `OAuthAccessDeniedError` を宣言している。しかし実装の `writeOAuthError` (`backend/oauth2/handlers_http/errors.go`) が状態コードを変えるのは `invalid_client` → 401 と `server_error` → 500 だけで、`access_denied` を含む残りはすべて 400 で返る。承認要求を利用者が拒否した場合 (CIBA) も、デバイス認可を拒否した場合も、返るのは 400 の `{"error":"access_denied"}` である。**`Token` が 403 を返す経路は存在しない。**

RFC 6749 §5.2 はトークンエンドポイントのエラー応答を 400 と定めており (`invalid_client` のみ 401 を許す)、RFC 8628 §3.5 の `access_denied` もこれに従う。**実装が標準どおりで、契約が誤っている**というのが現時点の判断である。

契約が返らない応答を宣言していると、クライアント実装は起こらない分岐を書き、生成された OpenAPI を読む相手は誤った期待を持つ。

## Scope

- `Token` の 403 宣言を実装と標準に合わせる。`OAuthAccessDeniedError` を 400 の本文へ移すか、`Token` から取り除くかを決める。
- 同じ宣言を持つ `Authorize` の 403 も、実際に返る経路があるかを確かめる。
- OpenAPI ベースラインの更新と、互換性検査の判断を記録する。

## Out of Scope

- `access_denied` を実際に 403 で返すよう実装を変えること。標準に反する。
- 管理 API の `AccessDeniedError` (Problem Details) の扱い。こちらは 403 で正しい。
- `Token` の 422 (`ClaimReleaseDeniedError`) と、`Authorize` / `Token` の 401 の宣言。実測の途中で、`Token` に 422 を返す経路が無いこと、`invalid_dpop_proof` が 401 の本文に宣言されながら `writeOAuthError` では 400 で返ること、`Authorize` の 401 が `invalid_client` の OAuth エラー形と `authentication_required` の Problem Details の 2 形を持つことが分かった。いずれも同じ種類の食い違いだが、宣言と実装の総当たりは [[wi-386-declared-status-code-audit]] が持つ。本 work item は 403 に限る。
- `enforceDefaultSignInPolicy` が 403 を書いたあとも処理を続け、認可コードを発行していた欠陥。`Authorize` の 403 を実測する過程で見つけたもので、[[wi-444-sign-in-policy-denial-still-issues-an-authorization-code]] が閉じた。本 work item は契約の宣言だけを扱う。

## Design

**`Token` は 403 を返さない。** `handleToken` がエラー応答を書く経路は `writeOAuthError` だけで、その状態コードの分岐は `invalid_client` → 401 と `server_error` → 500 の 2 つしかなく、残りはすべて 400 に落ちる。`access_denied` を作る経路は CIBA (`approval/usecases/approval_flow.go`) と デバイス認可 (`device/usecases/device_flow.go`) の 2 つで、どちらも `OAuthError` として `writeOAuthError` を通る。RFC 6749 §5.2 はトークンエンドポイントのエラー応答を 400 と定め、`invalid_client` にのみ 401 を許すので、実装が標準どおりで契約が誤っている。

したがって `TokenError403` は取り除く。ただし `OAuthAccessDeniedError` 自体は捨てない —— `/token` は実際に 400 でこれを返すので、`TokenError400Body` へ移す。取り除くだけでは「`access_denied` は `/token` から返らない」という別の誤りに置き換わる。

**`Authorize` は 403 を返す。ただし宣言と形が違う。** テナント既定のサインインポリシーが拒否したとき、`enforceDefaultSignInPolicy` が `support.WriteProblem(c, 403, "access_denied", ...)` を書く。これは `application/problem+json` の RFC 9457 Problem Details であって、契約が宣言している `application/json` + `OAuthError` ではない。`/authorize` の他の拒否は `redirect_uri` へのリダイレクトで返るなかで、これだけがブラウザーへ直接書かれる。

よって `Authorize` の 403 は消さず、`AccessDeniedError` (Problem Details) へ直す。同じ形を隣の `ResumeFederatedAuthorizationError403` が既に宣言しており、そちらに揃う。

なお [[wi-382-typespec-contract-wire-fidelity-and-doc-language]] は `AccessDeniedError` → `OAuthAccessDeniedError` の入れ替えを 2 件行っており、その 1 件がこの `Authorize` の 403 である。本 work item はそれを実測に基づいて戻す。

検討した代替案:

- **`Token` の 403 を残し、`access_denied` を 403 で返すよう実装を変える**: RFC 6749 §5.2 に反する。Out of Scope に置いた。
- **`Authorize` の 403 も宣言ごと消す**: 実測が 403 の到達を示しているので、消せば「返る応答が宣言に無い」という逆向きの誤りになる。採用しない。

## Plan

- 実装が返す状態コードを経路ごとに実測してから契約を直す。契約を先に直すと、別の経路が 403 を返していた場合に取り違える。実際にこの順序が効いた: `Authorize` は 403 を返しており、先に契約を直していれば消してしまうところだった。
- ベースラインは wi-397 の 3 点だけを反映する。`mise run update-api-baseline` は生成物を丸ごと凍結し直すので、本件と無関係な追加分まで飲み込む。

## Tasks

- [x] T001 [Survey] `Token` と `Authorize` の各エラー経路が実際に返す状態コードを実測した。
  `Token` の `access_denied` は `TestDeviceAuthorizationAPI/DeviceAPI_Deny` で 400 を実測。
  `Authorize` の 403 は `TestAuthorizeDeniedBySignInPolicyReturnsProblemDetails403` で Problem Details を実測。
- [x] T002 [Spec] 契約の 403 宣言を実測に合わせた。`TokenError403` を削除し `OAuthAccessDeniedError` を
  `TokenError400Body` へ移動、`AuthorizeError403` を `application/problem+json` + `AccessDeniedError` へ変更。
- [x] T003 [Verify] `mise run check-api-compat` の 3 件を分類し、ベースラインを該当箇所だけ更新した。

## Verification

- `mise run verify`
- `mise run check-api-compat`

## Risk Notes

リスクは low。契約の宣言を実測に合わせるだけで、実装の振る舞いは変えない。ただし公開済みの OpenAPI から応答が 1 つ消えるため、互換性検査が破壊的変更として扱う可能性がある。返らない応答の削除をどう扱うかを、ベースライン更新の判断として記録する。

## Completion

- **Completed At**: 2026-08-30
- **Summary**:
  `Token` から `TokenError403` を取り除き、`OAuthAccessDeniedError` を `TokenError400Body` へ移した。`/token` は RFC 6749 §5.2 のとおり `access_denied` を 400 で返しており、403 を返す経路は無い。`Authorize` の 403 は逆に実在したので消さず、宣言の形を実測に合わせた。`application/json` + `OAuthAccessDeniedError` から `application/problem+json` + `AccessDeniedError` へ変え、隣の `ResumeFederatedAuthorizationError403` と同じ形にした。これは [[wi-382-typespec-contract-wire-fidelity-and-doc-language]] が行った 2 件の入れ替えのうち 1 件を、実測に基づいて戻したことになる。`mise run spec-diff` は `no normative specification change against main` を返す。規範シナリオも標準行も動かしておらず、変わったのは TypeSpec が宣言する応答の集合と本文の形だけである。
- **Acceptance RED Evidence**:
  - **Test**: `TestAuthorizeDeniedBySignInPolicyReturnsProblemDetails403` (`backend/oauth2/handlers_http/authorize_handler_test.go`) と `TestDeviceAuthorizationAPI/DeviceAPI_Deny` (`backend/oauth2/handlers_http/device_handler_test.go`)
  - **Requirement**: N/A: 契約の宣言を実測へ合わせる変更であり、製品の振る舞いは変えていない。対応する `REQ-` シナリオは無く、根拠は RFC 6749 §5.2 と RFC 8628 §3.5 である。実際に失敗した代替の検査は `mise run check-api-compat` で、変更後に 3 件の破壊的変更を報告した (下記)。
  - **Observed Failure**: `check-api-compat: 3 breaking change(s)` — `GET /authorize 403: field 'error' removed`、`GET /authorize 403: field 'error_description' removed`、`POST /token 403: response status removed`。ベースラインの該当箇所を更新して解消した。
  - **Detection Reason**: この 2 つの検査は契約を直す前に実測を固定するために置いたもので、どちらも当初から通っている —— 実装は最初から正しく、誤っていたのは宣言の側だという判断そのものが、この 2 件が通ることで示される。契約が実装から離れたことを落とすのは `check-api-compat` で、宣言を消したり形を変えたりすれば必ず反応する。逆に、契約を直さずベースラインだけ更新しても、生成物が変わらないので何も起きない。
- **Unit RED Evidence**:
  - **Test**: `mise run check-spec` (TypeSpec のコンパイルと正本文書の検証)
  - **Requirement**: N/A: 上と同じ理由で、対応する `REQ-` シナリオを持たない。
  - **Observed Failure**: `TokenError403` を union から外したまま model 定義を残すと、参照されない model として残り続ける。定義と union の両方を消し、`OAuthAccessDeniedError` を `TokenError400Body` へ移してから `mise run check-spec` が通ることを確認した。生成物の側は `POST /token` の responses が `['200','400','401','403','422','429']` から `['200','400','401','422','429']` へ、`GET /authorize` の 403 の content-type が `application/json` から `application/problem+json` へ変わったことを、コンパイル済み OpenAPI から直接読んで確かめた。
  - **Detection Reason**: 契約だけの変更なので、内側の計算に単体境界が無い。代わりに、宣言の変化を生成された OpenAPI の responses 集合と content-type として直接読み取り、意図した 2 operation 以外が動いていないことを確認している。
- **Change-Resistance Results**:
  リスクは `low` のため必須ではないが、ベースライン更新の判断を誤らせる代表的な失敗を 1 つ実測した。`mise run update-api-baseline` をそのまま実行すると 1055 行 (1002 insertions) が動き、`check-api-compat` は通る。しかしその差分には Group CSV 取り込みの 6 経路 (`/api/admin/v1/groups/imports` 系と `/api/admin/v1/jobs` 系) が含まれており、これは本 work item と無関係な追加のドリフトである。`check-api-compat` は追加を破壊的変更として報告しないため、この混入は検査では捕まらない。HEAD の生成物とベースラインを直接突き合わせて、ドリフトが本変更より前から存在することを確認したうえで、ベースラインは該当する 2 operation と 3 つの schema だけを差し替えた。結果の差分は 26 行 (8 insertions / 18 deletions) で、`/authorize` 403 の content-type と schema、`/token` 403 の削除、`TokenError400Body` への `OAuthAccessDeniedError` 追加、`TokenError403Body` の削除、および 2 つの schema description のみである。
- **Verification Results**:
  - `mise run verify` - passed (exit 0)
  - `mise run check-api-compat` - `no breaking changes` (ベースライン更新後)
  - `mise run check-spec` - ok (148 document(s), 333 operation(s), 844 TypeSpec symbol(s))
  - `mise run test-go-package -- ./backend/oauth2/handlers_http/...` - ok
  - `mise run spec-diff` - `no normative specification change against main`
