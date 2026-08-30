---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-08-19
change_kind: bugfix
priority: p2
depends_on: [wi-385-typespec-go-struct-drift-check]
evidence_policy: risk-based-v2
initial_context:
  source:
    - tools/check/src/contract-drift.ts
    - tools/check/src/check-contract-drift.ts
    - backend/shared/http/support_http/auth.go
    - backend/shared/http/support_http/problem.go
    - backend/shared/http/support_http/error_handler.go
    - backend/shared/http/support_http/csrf.go
    - backend/shared/http/support_http/rate_limit.go
    - backend/shared/http/support_http/tenant_middleware.go
    - backend/authentication/deps_http/account_helpers.go
    - backend/oauth2/handlers_http/errors.go
    - backend/oauth2/handlers_http/register_handler.go
    - backend/idmanagement/user/handlers_http/admin_user_handler.go
  specification:
    - docs/api-rules.md
  tests:
    - tools/check/src
  stop_before_reading: [frontend, docs/contexts, load, infra]
affected_spec:
  - { path: spec/contexts/oauth2/main.tsp, symbol: IdMagic.OAuth2.Operations.RegisterClient }
  - { path: spec/contexts/oauth2/main.tsp, symbol: IdMagic.OAuth2.Operations.Authorize }
  - { path: spec/contexts/sourcing/main.tsp, symbol: IdMagic.Sourcing.Operations.CreateScimUser }
  - { path: spec/contexts/authentication/main.tsp, symbol: IdMagic.Authentication.Operations.CompleteFederatedLogin1 }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.GetAdminUser }
---

# operation が宣言するステータスコード集合と handler が実際に返すものを総なめで突き合わせる

## Motivation

`wi-382` は本体の形を直したが、ステータスコード集合は 5 件だけを閉じ、全面的な突き合わせを Out of Scope に置いた。その 5 件を直す過程で、宣言と実装がずれている operation が他にもあることが確かめられている。

- `RegisterClient` は `Success_200` を宣言するが handler は 201 を返す。SCIM の `CreateScimUser` / `CreateScimGroup` も同じく 200 宣言で 201 実装である。
- `/userinfo` は `invalid_token` を 401 として宣言するが、`writeOAuthError` は `invalid_client` と `server_error` 以外をすべて 400 にする。
- `AuthorizeError403` は宣言されているが、`/authorize` の access_denied は redirect_uri へのリダイレクトで返るため到達しない。
- `CompleteFederatedLogin` は 200 を宣言するが handler は 303 リダイレクトしか返さない。`StartFederatedLogin` も同じ。
- `GetAdminUser` は 400 / 403 を宣言するが handler は 404 を返す。
- テナント解決 middleware は routing の手前で 404 `{"error": "tenant_not_found"}` を返す。これは `application/json` の第 4 のエラー本文であり、どの operation も宣言していない。

いずれも 1 件ずつは小さいが、333 operation を総なめしないと全体が読めない。

## Scope

- 全 operation について、宣言するステータスコード集合と handler が到達しうるコードを突き合わせた表を作る。表は人手の一覧ではなく `mise run check` が毎回作り直す検査とし、`wi-385` の突き合わせの鎖 (operationId → 経路 → handler) の上に乗せる。
- 差分を「契約が足りない」「契約が余っている」「実装が仕様どおりでない」に分類する。
- 契約側を直せるものは直す。実装の変更を要するものは分類だけ残し、別 work item に切り出す。
- middleware が operation の手前で返す応答をどう契約に書くか (共通の応答として宣言するか、書かないと決めるか) を決め、根拠を `docs/api-rules.md` に残す。

## Out of Scope

- 要求・応答の本体の形。`wi-382` が閉じており、再発防止は `wi-385` が担う。403 の本体 union が `InsufficientScopeError` を 2 operation でしか挙げていない件も本体の問題なので、ここでは直さない。
- 追えなかった operation を 0 件にすること。`wi-385` と同じ理由で上限値も置かない。被覆を毎回の出力に書くことで沈黙を避ける。
- 同一 `operationId` を 2 つの path が名乗っている件 (`CompleteFederatedLogin`, `PublishSamlMetadata`, `DownloadSamlSigningCertificate`, `SamlSingleSignOn`)。契約の識別子の問題であってステータス集合の問題ではない。

## Design

### 何を「到達しうるコード」と呼ぶか

先に決めておかないと表が作れない。応答を書けるのは echo の context を持つ関数だけなので、`*echo.Context` を引数に取る関数 (522 個) だけを読む。その中で状態を決める呼び出しは 2 種類しかない。

1. **応答書き込みそのもの** (`c.JSON` / `c.NoContent` / `c.Redirect` / `c.String` / `c.Blob` / `c.XML` / `echo.NewHTTPError` …)。呼び出しの実引数に `http.StatusXxx` が定数で書かれていれば、それがそのコードである。
2. **共有ヘルパー**。`WriteProblem(c, http.StatusNotFound, …)` のように定数を渡していれば呼び出し側で決まる。定数を渡していない場合だけ、ヘルパーの側を読む必要がある。

ヘルパーには性質の違う 2 種類がある。この区別が本 work item の要である。

- **guard**: 要求そのものから拒否を決める。`VerifyBrowserRequest` (403)、`CheckRateLimit` (429)、`requireAdmin` / `requireWorkflowAdmin` の類、`WriteAdminAccessError` / `WriteAccessTokenError` (401 と 403)。呼び出し側の直前に立っている判定なので、**guard が書きうるコードはその呼び出し地点で全部到達しうる**。したがって辿ってよい。
- **error mapper**: use case が返したエラー値から応答を決める。`WriteAccountError`、`writeOAuthError`、provisioning の `writeError` など。どの分岐に入るかは use case が返すエラーで決まり、handler の字面には現れない。実測すると `ListMyConsents` が `WriteAccountError` を通じて `mfa_already_enrolled` の 409 を「到達しうる」ことになってしまう。**mapper は辿らない。**

mapper の見分けは字面で行う。**引数に `error` を取るか、本体が `errors.Is` / `errors.As` / `errors.AsType` で分岐するヘルパーは mapper とみなす。** `WriteAdminAccessError` と `WriteAccessTokenError` は `error` を取るが、その `error` は直前の `RequireAdmin` が作ったものでどちらの分岐も必ず到達しうるので、明示的な例外として guard 側に置く。

この判定は保守的な側に倒れる。mapper を辿らないことで、handler が実際に返しうるコードを取りこぼす。取りこぼしは「宣言が余っている」の誤検出になるので、**mapper を 1 つでも呼ぶ operation は「全部は読めなかった」として記録し、そこでは「宣言が余っている」を報告しない。**

### 検出する 2 つ

| 規則 | 内容 | 近似の向き |
| --- | --- | --- |
| **S1 宣言が足りない** | handler と guard が書くコードを契約が宣言していない | 過少近似。読めたものだけを数えるので、報告は必ず本物 |
| **S2 宣言が余っている** | 契約が宣言するコードを、全部読めた handler がどこにも書かない | 全部読めた operation に限る。読み残しがあれば報告しない |

S2 には共通の error handler が書く分を除く。`support_http.ErrorHandler` は handler が素の `err` を返したときに 401 / 403 / 422 / 500 を書きうるが、その `err` が出るかどうかは use case 側の話で、handler の字面からは読めない。この 4 つを S2 の対象から外すことで、S2 の報告を「本当に到達しない」に限る。

### 契約に何を書くと決めるか

`docs/api-rules.md` に節を足して次を決める。決めないと S1 の 400 件が「直すべき差分」なのか「書かないと決めた差分」なのか判定できない。

- **operation は、自身の handler と guard が書くステータスをすべて宣言する。** `@body` の規則 (サーバーが実際に返すものを書く) をステータス行にも及ぼすだけである。
- **共通の error handler が写像しなかったエラーに対して書く 500 は宣言しない。** どの operation でも同じに出るうえ、呼び出し側が operation ごとに変える対応が無い。逆に handler が固有のコードを添えて自分で書く 5xx (`webauthn_unavailable` の 503 など) は operation 固有の結果なので宣言する。
- **テナント解決 middleware の 404 `tenant_not_found` は宣言しない。** routing の手前で返るのでどの operation の応答でもなく、operation ごとに書けば同じ operation の 404 と衝突して読めなくなる。api-rules 側に 1 箇所だけ書く。
- **401 は 403 と同じ guard の 2 つの分岐なので、403 を宣言する operation は 401 も宣言する。** 現状 403 は 272 operation、401 は 10 operation にしか無く、これが差分の最大の塊である。片方だけ宣言することは、呼び出し側に「サインインしていない」を「権限が無い」として扱わせることになる。

401 の本体は 3 系統ある `docs/api-rules.md` のエラー本文の区分そのままで、汎用 API は Problem Details、OAuth 2.0 / OIDC は `OAuthError`、SCIM は `ScimProtocolError` である。汎用 API の 401 は operation ごとに変わらないので、**共有の応答モデルを 1 つ置いて参照する。** 217 operation に 8 行ずつ同じモデルを書き下ろすのは、同じ事実を 217 回書くことでしかない。403 を operation ごとに持つのはそちらの本体が operation ごとに違うから (`AccessDeniedError` / `MfaEnrollmentNotAllowedError` / `InsufficientScopeError`) で、`check-security-controls` の R4 がその本体名で scenario と突き合わせている。401 にはその違いが無い。

### 実測 (検査を実ツリーに当てた結果)

333 operation のうち 332 が経路まで解決し、126 が「全部読めた」に入る。

- **S1 宣言が足りない**: 401×217、404×41、503×35、400×14、204×13、500×10、303×8、201×8、403×7、409×3、410×1、304×1。
- **S2 宣言が余っている**: 400×29、200×10、422×9、403×3、429×3、201×2、401×1。

検討した代替案:

- **403 の宣言を全 operation から外し、api-rules に 1 箇所だけ書く**: 契約は小さくなるが、`check-security-controls` の R4 が `union <Op>Error403Body` を読んで「契約が約束する拒否を scenario が宣言しているか」を突き合わせているので、この join が黙って消える。セキュリティ検査を弱めるので採らない。
- **共有の error handler の 500 を全 operation に宣言する**: 「サーバーが返すものを書く」を字義どおりに取ればこうなるが、333 回書いて呼び出し側の分岐は 1 つも増えない。宣言の意味を「呼び出し側が分岐すべきもの」に置き、根拠を api-rules に残す。
- **Go の型検査器で mapper の分岐到達性を解く**: use case が返すエラー値まで追えれば S2 の被覆が上がる。`wi-385` の Follow-up が同じ判断を保留しており、そちらに合流させる。

## Plan

1. `docs/api-rules.md` に「Declared status codes」節を足し、上の 4 つを決める。
2. `status-drift.ts` を単体テストから作る。読み取りの段 (context を持つ関数、応答書き込みの定数、guard と mapper の区別、読み残しの記録) と 2 つの検出規則をそれぞれ RED に置く。
3. `check-status-drift.ts` で入力を集めて終了コードを決め、`mise run check` に組み込む。
4. S1 / S2 を分類しながら契約を直す。401 は共有応答モデルで、それ以外は operation ごとに。
5. 実装側の変更を要するものは別 work item に切り出す。

## Tasks

- [x] T001 [Spec] `docs/api-rules.md` に「Declared status codes」節を足し、operation ごとに宣言するもの、
  共通の error handler の 500、テナント解決 middleware の 404、401 と 403 の対を決めた。
- [x] T002 [Test] `tools/check/src/status-drift.test.ts` を RED に置いた (24 件)。読み取りの各段と 2 つの
  検出規則、および「読めなかったこと」の記録をそれぞれ固定した。
- [x] T003 [Tooling] `status-drift.ts` (純関数) と `check-status-drift.ts` (入力収集と終了コード) を実装した。
- [x] T004 [Acceptance] 契約を直す前に `mise run check-status-drift` が
  `S1 RegisterClient: handleRegisterClient writes 201` で落ちることを実測した。Motivation が名指しした 6 件のうち
  検査の射程に入る 5 件がすべて出た。
- [x] T005 [Spec] S1 275 件と S2 18 件を分類して契約を直した。401 は共有応答モデル 1 つを 203 operation から参照し、
  残りは operation ごとに応答モデルを足した。到達しない宣言 37 件を消し、ベースラインへ 1 件ずつ反映した。
- [x] T006 [Tooling] `mise run check` に `check-status-drift` を組み込んだ。
- [x] T007 [Verify] `mise run verify`、`mise run check`、`mise run check-api-compat`。

## Verification

- `mise run check-spec`
- `mise run check-api-compat`
- `mise run verify`
- 手動確認: 監査表に 333 operation すべてが載り、突き合わせられなかった行が理由付きで数えられている。

## Risk Notes

宣言に無いステータスを契約へ足すのは非破壊だが、宣言にあって実装が返さないステータスを消すのは、生成クライアントの分岐を壊しうる。消す側は 1 件ずつ、到達不能であることを実装から示してから行う。S2 を「全部読めた operation」に限り、共通 error handler が書きうる 4 つを対象から外すのは、この確認を字面の側でも成立させるためである。

## Completion

- **Completed At**: 2026-08-30
- **Summary**:
  `mise run spec-diff` は 26 の TypeSpec 宣言の追加と 1 件の削除を返す。追加は 401 の共有応答
  `AuthenticationRequiredResponse` と、404 / 409 / 410 / 503 の本体になる 25 のエラーモデルである。削除は
  `FederatedLoginStartResponse` で、`StartFederatedLogin` が 303 リダイレクトしか返さないと確かめた結果、
  それを指していた 200 の宣言ごと消えた。
  `mise run check-status-drift` を足し、`mise run check` に組み込んだ。operation ごとに、生成 OpenAPI が宣言する
  ステータス集合と、handler および guard が書くコードを突き合わせる。**検査が見つけた 293 件はすべて解決した。**
  許可リストも閾値も置いていない。内訳は、宣言が足りない 275 件 (401×217、404×41、503×35、400×14、204×13、
  500×10、303×8、201×8、403×7、409×3、410×1、304×1) と、宣言が余っている 18 件である。
  `docs/api-rules.md` に「Declared status codes」節を足し、何を宣言し何を宣言しないかを決めた。共通の
  error handler が写像しなかったエラーに対して書く 500 と、テナント解決 middleware が routing の手前で返す
  404 `tenant_not_found` は operation ごとに宣言しない。前者はどの operation でも同じに出て呼び出し側の対応が
  変わらず、後者はそもそもどの operation の応答でもないためである。401 は 403 と同じ guard の 2 つの分岐なので、
  403 を宣言する operation は 401 も宣言する。
  実ツリーは 333 operation 中 82 を全部読み、248 は読み残しあり、3 は経路または handler に届いていない。この
  3 つの数は毎回の出力に書く。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-status-drift` を、契約を直す前のツリーに対して実行した。
  - **Requirement**: N/A: リポジトリの検査ツールと契約の記述であり、対応する規範的な製品要件を持たない。代わりに
    失敗したのは、本 work item の Motivation が名指しした食い違いである。
  - **Observed Failure**: `fail  S1 RegisterClient: handleRegisterClient writes 201, which the contract does not declare`
    ほか 274 件、`fail  S2 GetAdminUser: the contract declares 400, which HandleGetAdminUser never writes` ほか
    17 件 (exit 1)。Motivation の 6 件のうち、`RegisterClient` / `CreateScimUser` / `CreateScimGroup` の 201、
    `CompleteFederatedLogin` と `StartFederatedLogin` の 303、`GetAdminUser` の 404 と余分な 400 がすべて出た。
  - **Detection Reason**: 検査を足しただけで受け入れとせず、Motivation が人手で見つけた食い違いを検査が独立に
    再発見することを見た。修正後に `RegisterClient` の宣言を 201 から 200 へ戻すと
    `fail  S1 RegisterClient` が再び出て exit 1 になり、戻すと exit 0 になる。
- **Unit RED Evidence**:
  - **Test**: `tools/check/src/status-drift.test.ts` (24 件)
  - **Requirement**: N/A: 上と同じ理由で、対応する `REQ-` シナリオを持たない。
  - **Observed Failure**: 最初は `error: Cannot find module './status-drift.ts'` (実装が存在しない)。以降は
    読み取りの段ごとに RED を置いた。
  - **Detection Reason**: この検査の正しさは、ほぼ全部が「何を辿り、何を辿らないか」に乗っている。そこで
    guard を辿ること (`VerifyBrowserRequest` の 403、`requireAdmin` から `WriteAdminAccessError` への連鎖)、
    error mapper を辿らないこと (`WriteAccountError` の 409 を account operation に付けない)、
    `isNotFound(err)` のように `errors.Is` を使わない写像も mapper と見なすこと、応答そのものを他へ渡す
    handler (`ServeHTTP(c.Response(), ...)`) を読み残しとすること、同名の関数が 2 つあるときどちらも読まない
    ことを、それぞれ固定した。**読めなかった場合を「合格」にしないことを同じ強さで主張する**: 読み残しのある
    operation では「宣言が余っている」を報告しないことと、経路が無い場合・handler が曖昧な場合が未解決として
    数えられることを検査で固定した。
- **Change-Resistance Results**:
  読み取り規則を 2 か所で壊し、どちらも検査の結論が変わることを実測した。
  - **error mapper を guard として辿るようにする** (`isGuard` を常に真にする): 0 件が **228 件**になる。中身は
    設計が予測したとおりで、`fail S1 ListMyConsents: handleListAccountConsents writes 400, 401, 404, 409, 503`
    のように、MFA の `mfa_already_enrolled` (409) が consent 一覧の応答として報告される。mapper を辿らない規則が
    効いていることの直接の証拠である。
  - **読み残しのある operation でも S2 を報告する** (`unread` の gate を外す): S2 が 0 件から **116 件**になる。
    `S2 GenerateRecoveryCodes: the contract declares 400, which HandleGenerateRecoveryCodes never writes` は、
    実際には辿らなかった写像の先に 400 があるので偽陽性である。「読めなかったものを到達不能と呼ばない」が
    効いていることの証拠である。
  - **契約側を 1 件戻す**: `RegisterClientSuccess_201` を `Success_200` に戻すと `fail S1 RegisterClient` で
    exit 1、戻すと exit 0。
  実装の過程でも、この検査を実ツリーに当てることが方法として機能した。**最初の 3 版はいずれも使い物にならず、
  当てて初めてそれが分かった。** 呼び出し関係を無条件に推移閉包すると 332 operation 中 331 が「あらゆる
  ステータスを返しうる」ことになり、`*echo.Context` を持つ関数に絞ってなお、共有の error mapper を辿るせいで
  404 が 199 operation、409 が 129 operation に付いた。guard と mapper を分ける規則はこの実測から出たもので、
  単体テストだけでは出ていない。
  Motivation の 6 件のうち **1 件は誤りだった**。「`AuthorizeError403` は到達しない」は成立しない。
  `handleAuthorize` は `enforceDefaultSignInPolicy` を呼び、そこがサインインポリシーの拒否を 403 で書く。
  宣言は残した。検査はこれを報告していない (403 は共通 error handler も書きうるので S2 の対象外) ので、
  handler を読んで確かめた。
- **Verification Results**:
  - `mise run verify` - passed
  - `mise run check` - passed (`check-status-drift` を含む)
  - `mise run check-status-drift` - `ok  declared status codes (0 finding(s); 82/333 operation(s) read in full, 248 read in part, 3 not reached: handler-ambiguous=2, route-not-found=1)`
  - `mise run check-api-compat` - 修正直後は 37 件の破壊的変更。すべて「サーバーが返さないステータスの削除」で、
    24 件は検査が到達不能を示したもの、13 件は成功の書き込みが 201 か 204 だけであることを handler から確かめた
    ものである。該当する 37 の (path, method, status) だけをベースラインから外して `no breaking changes`。
    ベースライン全体の更新はしていない。
  - `mise run test-tools` / `lint-tools` / `typecheck-tools` - passed (status-drift の単体 24 件を含む)
  - `mise run spec-diff` - 26 declarations added, 1 removed

## Follow-up

- `wi-453`: `/userinfo` の `invalid_token`。契約は 401 を宣言し RFC 6750 もそれを要求するが、`writeOAuthError`
  が 400 にしている。契約ではなく実装が誤っている 1 件で、Scope の「実装の変更を要するものは別 work item」に当たる。
- `wi-454`: 403 の本体。403 を書く guard は 3 つあり `invalid_origin` / `csrf_failed` / `insufficient_scope` /
  `access_denied` の 4 コードを返すが、272 operation のうち 270 が `AccessDeniedError` だけを名乗る。
- `wi-455`: 1 つの `operationId` が複数の operation を名乗っている件と、Go の handler 名の衝突。後者は本 work item の
  未解決 2 件 (`handler-ambiguous`) の原因でもある。

読み残し 248 operation を減らすには、error mapper の分岐がどの operation から到達しうるかを解く必要があり、
それには use case が返すエラー値を追う型解決が要る。`wi-385` の Follow-up が `go/packages` の導入を同じ理由で
保留しているので、そちらに合流させる。
