---
depends_on: []
status: pending
authors: [tn]
risk: medium
created_at: 2026-08-18
change_kind: bugfix
priority: p1
affected_spec:
  - { path: spec/contexts/oauth2/models.tsp, symbol: IdMagic.Contract.OAuthErrorCode }
  - { path: spec/contexts/application/models.tsp, symbol: IdMagic.Contract.GrantType }
  - { path: spec/contexts/application/models.tsp, symbol: IdMagic.Contract.TokenEndpointAuthMethod }
  - { path: spec/contexts/claim-mapping/models.tsp, symbol: IdMagic.Contract.AttrVisibility }
  - { path: spec/contexts/identity-management/models.tsp, symbol: IdMagic.Contract.AttributeType }
  - { path: spec/contexts/sharedsignals/models.tsp, symbol: IdMagic.Contract.AccessDeniedError }
  - { path: spec/contexts/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.GetTenantUserAttributeSchema }
  - { path: spec/contexts/audit/main.tsp, symbol: IdMagic.Audit.Operations.ListAdminAuditEvents }
  - { path: spec/contexts/oauth2/models.tsp, symbol: IdMagic.Contract.RateLimitedError }
  - { path: spec/contexts/sharedsignals/main.tsp, symbol: IdMagic.SharedSignals.Operations.ReceiveSecurityEvent }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.UpdateUserProfile }
  - { path: spec/contexts/authentication/main.tsp, symbol: IdMagic.Authentication.Operations.StartBrowserMfaEnrollment }
---

# TypeSpec が記述する契約を、実際の線上表現と公開文書の言語方針に合わせる

## Motivation
`wi-381` は `UserAttributeDef` が 10 フィールド中 2 つしか宣言していないことを直した。その調査の副産物として、同じ種類の食い違いが TypeSpec 全体に広がっていることが分かった。個別のモデルの記述漏れではなく、TypeSpec を書くときの習慣そのものに原因がある。

現状の規模は次のとおりである (21 context、691 model、106 enum、327 operation)。

### 1. 要求／応答本体に存在しないラッパーが 1 段挟まっている
`main.tsp` は operation ごとに `<Op>HttpRequest` / `<Op>HttpResponse` を定義する。この 392 個のうち 320 個は単一プロパティで、そのプロパティ名がそのまま JSON の 1 段目のキーとして OpenAPI に出る。

```tsp
model GetTenantUserAttributeSchemaHttpResponse {
  schema: IdMagic.Contract.TenantUserAttributeSchemaResponse;
}
```

契約はこれを `{"schema": {"tenant_id": ..., "attributes": [...]}}` と記述するが、handler は `support.NoStoreJSON(c, 200, toUserAttributeSchemaResponse(schema))` を返すので実際の本体は `{"tenant_id": ..., "attributes": [...]}` である。要求側も同じで、`UpdateTenantUserAttributeSchemaHttpRequest { request: ... }` に対し handler が読むのは `{"attributes": [...]}` であって `{"request": {...}}` ではない。`ListAdminAuditEvents` の `response:`、`GetAdminUser` の `user:` も同様に架空である。

ただし一律ではない。`ListGroupExportsHttpResponse { exports: DataExportJob[] }` のような集合の封筒は実在する (`listAdminGroups` は本当に `{"groups": [...]}` を受け取る)。単一プロパティのラッパーのうち、キー名が `request` / `response` / `body` / `schema` / `result` の総称であるものが 153 個、資源名であるものが 167 個で、後者は実在するものと架空のものが混ざっている。機械的な一括除去はできず、operation 単位で handler と突き合わせるほかない。

これは公開 OpenAPI から生成したクライアントが 1 件も動かないことを意味する、最も重い食い違いである。

### 2. エラー本体が空オブジェクトとして記述されている
`*Error` モデル 86 個のうち 84 個が本体を持たない (`model AccessDeniedError {}`)。空でない 2 個 (`UserImportRowError` / `DataExportError`) は HTTP エラー本体ではない。

一方で transport wrapper は `@header contentType: "application/problem+json"` を宣言している。実際に返るのは `backend/shared/http/support_http` の `Problem` (RFC 9457) で、`type` / `title` / `status` / `detail?` / `instance?` を持つ。契約は media type だけ RFC 9457 を名乗り、本体は「プロパティの無いオブジェクト」と記述している。TypeSpec には `Problem` に相当するモデルが存在しない。

実装側の移行は `wi-327`〜`wi-340` で完了した。`WriteBrowserError` (レガシーな `{"error": code, "message": msg}` を `application/json` で返す) は関数ごと削除され、汎用 API はすべて `support_http.Problem` を返す。したがってエラー本体はもう「実装が 2 つの答えを持つ」状態ではなく、契約が写すべき対象は 1 つに定まっている。

ただし 84 個すべてが Problem Details なのではない。移行の過程で、標準または独自形状が理由で意図的に Problem Details にしていない応答が 2 種類あることが確定した。

| モデル | 実際に返るもの | 理由 |
| --- | --- | --- |
| `RateLimitedError` (429) | `application/json` の `{"error": "rate_limited", "retry_after_seconds": n, "message": ...}` と `Retry-After` ヘッダー | `support_http.WriteRateLimited` の独自形状。`retry_after_seconds` を運ぶため Problem Details にしていない |
| `SecurityEventRejectedError` (400) | `application/json` の `{"error": code, "message": ...}` | 受信エンドポイントは RFC 8935 が形を定める接点で、Problem Details を適用しない (`spec/SPECIFICATION.md` HTTP error responses) |

`RateLimitedError` の transport wrapper は 7 operation すべてで `@header contentType: "application/problem+json"` を宣言しており、これは実際の media type と違う。T002 でこの 2 つを一括で `ProblemDetails` に書き換えると、いま正しく「例外」である応答に誤った契約を与えることになる。

### 3. enum の値が線上の値と違う
値が大文字始まりの enum が 46 個、値にして 165 個ある。TypeSpec は member 名をそのまま値に複製しており、線上の値と一致していない。

| TypeSpec | 実際の値 | 出典 |
| --- | --- | --- |
| `GrantType.AuthorizationCode: "AuthorizationCode"` | `authorization_code` | RFC 6749 |
| `GrantType.DeviceCode: "DeviceCode"` | `urn:ietf:params:oauth:grant-type:device_code` | RFC 8628 |
| `TokenEndpointAuthMethod.ClientSecretBasic: "ClientSecretBasic"` | `client_secret_basic` | OpenID Connect Discovery |
| `OAuthErrorCode.InvalidRequest: "InvalidRequest"` | `invalid_request` | RFC 6749 §5.2 |
| `ResponseMode.FormPost: "FormPost"` | `form_post` | OAuth 2.0 Form Post Response Mode |
| `AttrVisibility.Private: "Private"` | `private` | `backend/idmanagement/domain/enums.go` |
| `AttributeType.String: "String"` | `string` | 同上 |
| `UserStatus.Active: "Active"` | `active` | 同上 |

`GrantType.DeviceCode` が URN であることが示すとおり、snake_case への機械的な変換では済まない。標準が定める値は標準から、独自の値は Go の定数から、1 つずつ写す必要がある。`CodeChallengeMethod.S256` と `SignatureAlgorithm.PS256` / `ES256` のように既に正しいものもある。

### 4. `unknown` が型の代わりに使われている
`: unknown` が 126 箇所ある。`LoginSession.id` と `MfaEnrollmentBypass.id` は不透明な `string`、`AttributeValue.number` は数値、`AccountProfileResponse.attributes` は既に他所で使われている `Record<AttributeValue>` である。契約を読む側から見ると、これらは「任意の JSON 値」と書かれているに等しい。

### 5. `@doc` の言語が AGENTS.md の言語表と食い違っている
`@doc` は 1870 件中 1833 件 (98%) が日本語である。AGENTS.md の言語表は `spec/**/*.tsp` を doc コメント込みで English と定めている。`@doc` は生成される OpenAPI の `description` になり、ローカライズを経ずにリポジトリの外へ出るので、表の規定と、表の末尾にある「ローカライズされずリポジトリの外へ出るものは英語」という原則の両方が英語を指している。実態のほうが規定から外れている。

### 6. operation が宣言するエラーの集合に、実装が返すものが入っていない
`wi-327`〜`wi-340` の Problem Details 移行は、各 call site の status を仕様の宣言値へ揃える作業を伴った。その過程で、**揃える先が契約に無い** ために据え置いた箇所が残っている。いずれも operation は宣言されているが、その operation の error union に該当のモデルが入っていない。

| 実装が返すもの | 契約が宣言しているもの | 影響 |
| --- | --- | --- |
| `UpdateUserProfile` (`PATCH /api/account/v1/profile`) が `invalid_attribute` を 400 で返す | 400 に `InvalidRequestError`、403 に `AccessDeniedError` のみ。`InvalidUserAttributeError` はどちらにも無い | 同じ code を管理 API (`UpdateAdminUser`) は 422 で返す。同一 code が endpoint によって 400 と 422 に分かれたまま |
| 同 operation が 401 (`authentication_required`) と 404 (`user_not_found` / `session_not_found`) を返す | 400 / 403 のみ | 未宣言の status |
| `StartBrowserMfaEnrollment` / `ConfirmBrowserMfaEnrollment` が `mfa_enrollment_not_allowed` を 403 で返す | 403 union は `AccessDeniedError`。`MfaEnrollmentNotAllowedError` (422) を持つのは `IssueMfaEnrollmentBypass` だけ | 同じ code が管理 API では 422、ブラウザ経路では 403。さらにこの分岐は `ErrMfaAlreadyEnrolled` (管理 API では 409) も同じ status で扱っている |
| `StartUserCsvExport` などが `quota_exceeded` を 429 で返す | 400 / 403 / 422 のみ。`QuotaExceededError` は 422 側にあり、これはテナントの資源クォータを指す | 「実行中ジョブ数の上限」と「テナント資源クォータ」という別概念が 1 つの code を共有している |
| `ReceiveSecurityEvent` が 413 (`security_event_token_too_large`) と 404 (`ssf_stream_not_found`) を返す | 400 のみ | 未宣言の status |

`wi-331` / `wi-332` は「揃える先の宣言が無い」ことを理由にこれらの status を据え置いた。判断は変えなくてよいが、契約側が沈黙している状態は残っている。決めるべきことは 2 つある。同じ code を別 status で返している 3 件について、**どちらが正しいのかを決めて契約に書く** こと。そして実装が返す status を union に加えることである。

`ReceiveSecurityEvent` は要求本体も食い違っている。契約は `{stream_id, token}` という JSON オブジェクトを記述するが、handler が読むのは compact serialization された SET そのもの (JSON オブジェクトですらない) である。1 の架空のラッパーと同じ棚卸しの対象だが、封筒を剥がすだけでは済まない唯一の例である。

## Scope
- 1〜4 について、契約の記述を実際の線上表現に合わせる。`spec/**/*.tsp` の変更に限り、実装・受理する値・返す値は変えない。
  - `<Op>HttpRequest` / `<Op>HttpResponse` を operation 単位で handler と突き合わせ、架空のラッパーを取り除く。実在する封筒はそのまま残す。
  - RFC 9457 の `ProblemDetails` モデルを 1 つ導入し、空の `*Error` モデル 84 個が実際のエラー本体を記述するようにする。`RateLimitedError` と `SecurityEventRejectedError` は Problem Details ではないので、それぞれの実際の形を書き、transport wrapper の `contentType` も実際の media type に直す。
  - 大文字始まりの enum 値 165 個を、標準または Go の定数から写した実際の値に置き換える。
  - `unknown` 126 箇所を、実在する型またはモデルに置き換える。置き換えられないものは、なぜ任意の JSON なのかを `@doc` に書く。
- 5 について、`spec/**/*.tsp` の `@doc` 1870 件を英語にする。
- 6 について、`wi-327`〜`wi-340` が列挙して残した 5 件の食い違いを閉じる。実装が返す status を operation の error union に加え、同じ code を別 status で返している 3 件はどちらが正しいかを決めて契約に書く。決めた結果が実装の変更を要するなら、その実装変更だけを別 work item に切り出す。
- 凍結 OpenAPI baseline を、今回の修正後の生成物で更新する。

## Out of Scope
- 実装の変更。契約が実装に合わせるのであって、逆ではない。線上表現そのものに直したい点があれば別 work item を立てる。
- 文字列長上限の網羅。`models.tsp` の string プロパティ 873 個のうち 776 個に `@maxLength` が無く、Go 側には zog の上限があるものが多い。`wi-128` の系統に属する独立した作業である。
- `models.tsp` が 21 context すべてで単一の `IdMagic.Contract` namespace を共有している構造の解消。Tenancy が所有する `UserAttributeDef` が `claim-mapping/models.tsp` に置かれているのはこの構造の症状だが、生成される component 名が変わるため、線上表現の修正とは分けて扱う。
- operation が宣言するステータスコード集合と handler が実際に返すものの **全面的な** 突き合わせ (`GetAdminUser` は 400 / 403 を宣言するが handler は 404 を返す)。327 operation を総なめする監査であり、規模が読めていない。Scope に入れたのは 6 に列挙した 5 件だけで、これらは `wi-327`〜`wi-340` が実装を触りながら 1 件ずつ確認して記録した既知の集合である。総なめは、この work item が終わってから独立の監査として起票する。
- SharedSignals 受信エンドポイントの本体が RFC 8935 §2.3 の `{err, description}` ではなく `{error, message}` になっている件 (`wi-335` Risk Notes)。これは実装の変更であり、「契約が実装に合わせる」という本 work item の原則の外にある。ここで契約に書くのは実装が今返している `{error, message}` のほうで、RFC への追従は別 work item を立てる。契約を先に正しい形にしてしまうと、どちらが正なのか読めなくなる。
- `WriteRateLimited` の 429 を Problem Details にするかどうかの判断。`retry_after_seconds` を RFC 9457 の拡張メンバーとして持たせる設計判断を含むため、契約の記述合わせとは別の変更である。ここでは今の形をそのまま記述する。
- TypeSpec のモデルと Go の構造体を機械的に突き合わせる検査の追加 (`wi-381` の Out of Scope を引き継ぐ)。今回の作業を将来の再発から守る唯一の手段なので、完了後にすぐ起票する。
- work item の `affected_spec.symbol` に残る `Product.*` 接頭辞 (7 件)。TypeSpec に存在しない namespace を指しているが、work item 側の記録の問題である。

## Design
### 突き合わせの基準
契約は、サーバーが実際に受理し返すものを記述する。基準は次の順に取る。

1. その値を標準が定めているなら標準 (OAuth2 / OIDC / RFC 9457 / SCIM)。
2. それ以外は Go の実装。構造体の JSON タグ、`support_http` の書き出し、`backend/shared/spec` の定数。
3. 判断が割れるときは、`frontend/src/types.ts` と `frontend/src/api/*.ts` が実際に送受信しているものを第三の証拠として使う。

`omitempty` の付いたフィールドだけを optional にし、真偽値の既定は Go の zero value から写す。実装が値を必須としているなら契約に既定値を書かない。この規則は `wi-381` で `UserAttributeDef` に適用したものと同じで、今回は残り全体へ広げる。

### 却下した案
- **ラッパーを一括除去する。** 集合の封筒は実在するため、架空のものと区別できない。operation 単位の突き合わせを避けられない。
- **enum 値を機械的に snake_case へ変換する。** `GrantType.DeviceCode` の実値は URN であり、変換規則では出てこない。
- **`*Error` モデルに `type` / `title` / `status` を個別に持たせる。** 84 個に同じ 5 フィールドを複製することになる。`ProblemDetails` を 1 つ定義し、各エラーはその `type` URN 接尾辞 (= Go の `code`) を `@doc` で特定する形にする。
- **`@doc` を日本語のままにして AGENTS.md を変更する。** `@doc` は OpenAPI の `description` として生成物に出るため、言語表の「ローカライズされずリポジトリの外へ出るものは英語」に該当する。実態ではなく規定のほうが正しい。

### エラー本体は 3 種類ある
起票時点では、T002 だけが「実装が今返しているもの」をそのまま写せない task だった。`WriteProblem` 移行済みの context と未移行の context が混在し、実装が 1 つの答えを持っていなかったためである。`wi-327`〜`wi-340` の完了でこの前提は消えた。実装は接点ごとに 1 つの答えを持つので、T002 も他の task と同じく「写す」だけでよい。

写す先は 3 種類ある。混ぜないこと。

1. **汎用 API** — `support_http.Problem` (`type` / `title` / `status` / `detail?` / `instance?`)、`application/problem+json`。空の `*Error` モデルのほとんどがこれで、`ProblemDetails` を 1 つ定義して参照させる。
2. **標準が形を定める接点** — OAuth2 / SCIM / DCR の各標準形式と、SharedSignals 受信エンドポイントの `{error, message}` (RFC 8935 に追従できていない現状の形をそのまま書く)。`application/json`。
3. **独自形状** — `WriteRateLimited` の 429 だけ。`{error, retry_after_seconds, message}` と `Retry-After` ヘッダー。`application/json`。

1 と 2 の境界は `spec/SPECIFICATION.md` の HTTP error responses 節が持っている。同じパッケージの中でも接点ごとに引かれる境界なので、モデル名やファイルの所属ではなく、その応答を受け取るのが標準クライアントかブラウザーかで判断する。

`depends_on` は空のままでよい。移行チェーンは完了しており、この work item を止めるものは無い。

### baseline の扱い
`spec/idmagic.openapi.baseline.json` は既に架空のラッパーを凍結している (`PUT /api/admin/v1/tenant/user_attribute_schema` の要求本体が `{"request": {...}}` として入っている)。したがって修正後の生成物は `just check-api-compat` から破壊的変更として報告される。

これは baseline がサーバーの受理しない本体を凍結していたためであって、実在するクライアントは 1 件も壊れない。`{"request": {...}}` を送っていたクライアントは存在しえない。commit skill は通常の commit で baseline を更新しないと定めているので、baseline の更新は本 work item の最後の task として単独で行い、何を凍結し直したのかを commit 本文に書く。

### 進め方
食い違いの重さと、後の作業への影響の大きさで並べる。enum とエラー本体は他の task が触るファイルと重ならないので先に片づけ、最も広いラッパーの棚卸しをその後に置く。`@doc` の英語化は全ファイルに触るので、構造の変更が終わってから最後に行う。そうしないと翻訳した文が直後の構造変更で書き換わる。

## Plan
1. enum 値 (task T001)。context 単位で完結し、他の task と衝突しない。
2. エラー本体 (T002)。`ProblemDetails` の導入と 84 モデルの書き換え。
3. エラー集合の穴埋め (T009)。T002 と同じ union を触るので直後に置く。
4. `unknown` (T003)。
5. ラッパーの棚卸し (T004〜T006)。327 operation を context 単位に分けて進める。
6. `@doc` の英語化 (T007)。
7. baseline の更新 (T008)。

## Tasks
- [ ] T001 [Spec] 大文字始まりの値を持つ enum 46 個を、標準または Go の定数から写した実際の値に置き換える。`spec-where` で各 enum の Go 側の対応を確認し、標準由来のものは RFC 番号を `@doc` に残す。
- [ ] T002 [Spec] RFC 9457 の `ProblemDetails` モデルを導入し、空の `*Error` モデル 84 個が `support_http.Problem` と同じ本体を記述するようにする。各エラーの識別は `type` URN 接尾辞で行い、Go の error code と一致させる。`RateLimitedError` (429、独自形状) と `SecurityEventRejectedError` (RFC 8935 の接点) は `ProblemDetails` を参照させず、実際の本体を書いたうえで transport wrapper の `contentType` を `application/json` に直す。
- [ ] T003 [Spec] `: unknown` 126 箇所を実在する型・モデルに置き換える。置き換えられないものは理由を `@doc` に書く。
- [ ] T004 [Inventory] 392 個の `<Op>HttpRequest` / `<Op>HttpResponse` を、対応する handler の本体と 1 対 1 で突き合わせた表を作る。架空のラッパーと実在する封筒を区別する。
- [ ] T005 [Spec] T004 の表に従って架空のラッパーを取り除く。context 単位に分け、各 context で `just check-spec` を通す。
- [ ] T006 [Verify] 修正後の OpenAPI から、代表的な operation の要求／応答本体が `frontend/src/api/*.ts` の送受信と一致することを確認する。
- [ ] T007 [Spec] `spec/**/*.tsp` の `@doc` 1870 件を英語にする。SPECIFICATION.md の日本語散文と同じ内容を二言語で持たないよう、翻訳ではなく契約の記述として書く。
- [ ] T008 [Verify] `just spec-render` の生成物で `spec/idmagic.openapi.baseline.json` を更新し、`just check-api-compat` を通す。差分の要約を Completion に残す。
- [ ] T009 [Spec] Motivation 6 の 5 件を閉じる。`UpdateUserProfile` に 401 / 404 と `InvalidUserAttributeError`、`StartBrowserMfaEnrollment` / `ConfirmBrowserMfaEnrollment` に `MfaEnrollmentNotAllowedError`、export 系 operation に 429、`ReceiveSecurityEvent` に 413 / 404 を宣言する。同じ code を別 status で返している 3 件 (`invalid_attribute` が 400 と 422、`mfa_enrollment_not_allowed` が 403 と 422、`quota_exceeded` が 429 と 422) は、契約に書く前にどちらが正しいかを決める。`quota_exceeded` は 2 つの別概念が 1 つの code を共有しているので、status ではなく code を分ける結論もありうる。実装の変更が要るという結論になった分だけを別 work item に切り出し、この work item では契約に書ける分を書く。
- [ ] T010 [Verify] `ReceiveSecurityEvent` の要求本体が compact serialization された SET そのものであることを契約に反映する (T004 の表に載せ、T005 で直す)。`{stream_id, token}` という JSON オブジェクトは実在しない。

## Verification
- `just check-spec`
- `just check-api-compat` (T008 の前は破壊的変更を報告する。T008 で baseline を更新してから通す)
- `just verify`
- 手動確認: `PUT /api/admin/v1/tenant/user_attribute_schema` と `GET /api/admin/v1/audit_events` について、生成された OpenAPI の要求／応答本体が `frontend/src/api/admin.ts` の送受信と一致する。
- 手動確認: 403 応答の本体が `type` / `title` / `status` を持ち、`backend/shared/http/support_http/problem.go` の `Problem` と一致する。
- 手動確認: 429 応答 (`WriteRateLimited`) と `POST /ssf/streams/{stream_id}/events` の拒否応答が、生成された OpenAPI では `application/problem+json` ではなく `application/json` として記述されている。
- `grep -rn "WriteProblem\|writeSecurityEventReceiverError\|WriteRateLimited" backend --include="*.go"` で、契約が記述する 3 種類のエラー本体以外の書き出し経路が増えていないことを確認する。

## Risk Notes
実装に触らないので、動作は変わらない。危険はすべて「契約を直したことが破壊的変更に見える」ことに集中する。

`check-api-compat` は架空のラッパーを凍結した baseline と比較するため、T005 の後に大量の findings を出す。これを理由に修正を縮めてはならない。baseline が記述していた本体はサーバーが一度も受理したことのないもので、それに依存するクライアントは存在しえない。逆に、T008 で baseline を更新する際に、本当に壊れる変更 (enum 値の変更は実在するクライアントに影響しうる) を「どうせ全部 findings が出るから」と一緒に飲み込まないよう、T005 と T008 の間で findings を一度読み切る。

enum 値の修正 (T001) だけは性質が違う。契約を読んで `"AuthorizationCode"` を送るよう組んだクライアントは、サーバーがそれを受理しないので既に動いていない。しかし逆向き、つまり契約を信じて生成した型で `"authorization_code"` を弾く検証を書いたクライアントはありうる。標準由来の値は標準どおりであることを、Go の定数と RFC の両方で確認してから直す。

T002 と T009 は同じ union を触る。T002 は「84 個を一括で `ProblemDetails` にする」と読める task なので、例外の 2 つ (`RateLimitedError` / `SecurityEventRejectedError`) を巻き込みやすい。この 2 つはいま実装と一致している数少ない箇所であり、一括置換で壊すと、契約を直す作業が正しい記述を壊したことになる。置換の前に対象一覧から除外しておくこと。

T009 の「どちらが正しいか」は契約の記述ではなく設計判断である。決めずに実装の現状を書き写すと、同じ code が endpoint によって別 status を持つことを契約が追認してしまう。決められないなら、その 1 件だけを T009 から外して別 work item に送るほうがよい。

`@doc` の英語化 (T007) は 1870 件あり、機械的な翻訳では意味を落とす。既存の日本語が説明している fail-closed の条件や既定値の根拠を落とさないこと。落とすくらいなら、その `@doc` は context の SPECIFICATION.md へ移してから短い英語に置き換える。
