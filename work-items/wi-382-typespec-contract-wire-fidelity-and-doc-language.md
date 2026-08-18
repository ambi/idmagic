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

実装側はまだ移行の途中である。`WriteBrowserError` (レガシーな `{"error": code, "message": msg}` を `application/json` で返す) の呼び出しが 420 箇所残っており、`wi-327`〜`wi-339` の context 単位の移行と、それらを束ねる `wi-340` が未完了である。つまり今のエラー契約は、media type が実装より先に進み、本体が実装より遅れているという二重の食い違いになっている。

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

## Scope
- 1〜4 について、契約の記述を実際の線上表現に合わせる。`spec/**/*.tsp` の変更に限り、実装・受理する値・返す値は変えない。
  - `<Op>HttpRequest` / `<Op>HttpResponse` を operation 単位で handler と突き合わせ、架空のラッパーを取り除く。実在する封筒はそのまま残す。
  - RFC 9457 の `ProblemDetails` モデルを 1 つ導入し、空の `*Error` モデル 84 個が実際のエラー本体を記述するようにする。
  - 大文字始まりの enum 値 165 個を、標準または Go の定数から写した実際の値に置き換える。
  - `unknown` 126 箇所を、実在する型またはモデルに置き換える。置き換えられないものは、なぜ任意の JSON なのかを `@doc` に書く。
- 5 について、`spec/**/*.tsp` の `@doc` 1870 件を英語にする。
- 凍結 OpenAPI baseline を、今回の修正後の生成物で更新する。

## Out of Scope
- 実装の変更。契約が実装に合わせるのであって、逆ではない。線上表現そのものに直したい点があれば別 work item を立てる。
- 文字列長上限の網羅。`models.tsp` の string プロパティ 873 個のうち 776 個に `@maxLength` が無く、Go 側には zog の上限があるものが多い。`wi-128` の系統に属する独立した作業である。
- `models.tsp` が 21 context すべてで単一の `IdMagic.Contract` namespace を共有している構造の解消。Tenancy が所有する `UserAttributeDef` が `claim-mapping/models.tsp` に置かれているのはこの構造の症状だが、生成される component 名が変わるため、線上表現の修正とは分けて扱う。
- operation が宣言するステータスコード集合と handler が実際に返すものの突き合わせ (`GetAdminUser` は 400 / 403 を宣言するが handler は 404 を返す)。本 work item の 5 つとは別の軸の監査であり、規模も読めていない。
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

### エラー本体と Problem Details 移行の順序
T002 だけは「実装が今返しているもの」をそのまま写せない。`WriteProblem` へ移行済みの context は RFC 9457 の `Problem` を返し、未移行の 420 箇所は `{"error": ..., "message": ...}` を返すので、実装が 1 つの答えを持っていない。

それでも契約側は Problem Details で書く。transport wrapper は既に全 operation で `@header contentType: "application/problem+json"` を宣言しており、契約はこの envelope に踏み込み済みだからである。T002 が書くのはその宣言の本体であって、新しい約束ではない。実装側の残りは `wi-327`〜`wi-340` の担当で、それが終わった時点で食い違いが閉じる。

したがって T002 は `wi-340` より先に着手してよいが、T002 の後・`wi-340` の完了前は、未移行 context のエラー本体だけが契約と食い違う状態が残る。`wi-340` の Verification でこの整合を確認する。`depends_on` は置かない。置くと enum・ラッパー・`@doc` まで 13 件の移行チェーンに縛られ、独立に進められる作業が止まるためである。

### baseline の扱い
`spec/idmagic.openapi.baseline.json` は既に架空のラッパーを凍結している (`PUT /api/admin/v1/tenant/user_attribute_schema` の要求本体が `{"request": {...}}` として入っている)。したがって修正後の生成物は `just check-api-compat` から破壊的変更として報告される。

これは baseline がサーバーの受理しない本体を凍結していたためであって、実在するクライアントは 1 件も壊れない。`{"request": {...}}` を送っていたクライアントは存在しえない。commit skill は通常の commit で baseline を更新しないと定めているので、baseline の更新は本 work item の最後の task として単独で行い、何を凍結し直したのかを commit 本文に書く。

### 進め方
食い違いの重さと、後の作業への影響の大きさで並べる。enum とエラー本体は他の task が触るファイルと重ならないので先に片づけ、最も広いラッパーの棚卸しをその後に置く。`@doc` の英語化は全ファイルに触るので、構造の変更が終わってから最後に行う。そうしないと翻訳した文が直後の構造変更で書き換わる。

## Plan
1. enum 値 (task T001)。context 単位で完結し、他の task と衝突しない。
2. エラー本体 (T002)。`ProblemDetails` の導入と 84 モデルの書き換え。
3. `unknown` (T003)。
4. ラッパーの棚卸し (T004〜T006)。327 operation を context 単位に分けて進める。
5. `@doc` の英語化 (T007)。
6. baseline の更新 (T008)。

## Tasks
- [ ] T001 [Spec] 大文字始まりの値を持つ enum 46 個を、標準または Go の定数から写した実際の値に置き換える。`spec-where` で各 enum の Go 側の対応を確認し、標準由来のものは RFC 番号を `@doc` に残す。
- [ ] T002 [Spec] RFC 9457 の `ProblemDetails` モデルを導入し、空の `*Error` モデル 84 個が `support_http.Problem` と同じ本体を記述するようにする。各エラーの識別は `type` URN 接尾辞で行い、Go の error code と一致させる。実装側に残る `WriteBrowserError` 420 箇所は `wi-327`〜`wi-340` の担当なので触らない。
- [ ] T003 [Spec] `: unknown` 126 箇所を実在する型・モデルに置き換える。置き換えられないものは理由を `@doc` に書く。
- [ ] T004 [Inventory] 392 個の `<Op>HttpRequest` / `<Op>HttpResponse` を、対応する handler の本体と 1 対 1 で突き合わせた表を作る。架空のラッパーと実在する封筒を区別する。
- [ ] T005 [Spec] T004 の表に従って架空のラッパーを取り除く。context 単位に分け、各 context で `just check-spec` を通す。
- [ ] T006 [Verify] 修正後の OpenAPI から、代表的な operation の要求／応答本体が `frontend/src/api/*.ts` の送受信と一致することを確認する。
- [ ] T007 [Spec] `spec/**/*.tsp` の `@doc` 1870 件を英語にする。SPECIFICATION.md の日本語散文と同じ内容を二言語で持たないよう、翻訳ではなく契約の記述として書く。
- [ ] T008 [Verify] `just spec-render` の生成物で `spec/idmagic.openapi.baseline.json` を更新し、`just check-api-compat` を通す。差分の要約を Completion に残す。

## Verification
- `just check-spec`
- `just check-api-compat` (T008 の前は破壊的変更を報告する。T008 で baseline を更新してから通す)
- `just verify`
- 手動確認: `PUT /api/admin/v1/tenant/user_attribute_schema` と `GET /api/admin/v1/audit_events` について、生成された OpenAPI の要求／応答本体が `frontend/src/api/admin.ts` の送受信と一致する。
- 手動確認: 403 応答の本体が `type` / `title` / `status` を持ち、`backend/shared/http/support_http/problem.go` の `Problem` と一致する。

## Risk Notes
実装に触らないので、動作は変わらない。危険はすべて「契約を直したことが破壊的変更に見える」ことに集中する。

`check-api-compat` は架空のラッパーを凍結した baseline と比較するため、T005 の後に大量の findings を出す。これを理由に修正を縮めてはならない。baseline が記述していた本体はサーバーが一度も受理したことのないもので、それに依存するクライアントは存在しえない。逆に、T008 で baseline を更新する際に、本当に壊れる変更 (enum 値の変更は実在するクライアントに影響しうる) を「どうせ全部 findings が出るから」と一緒に飲み込まないよう、T005 と T008 の間で findings を一度読み切る。

enum 値の修正 (T001) だけは性質が違う。契約を読んで `"AuthorizationCode"` を送るよう組んだクライアントは、サーバーがそれを受理しないので既に動いていない。しかし逆向き、つまり契約を信じて生成した型で `"authorization_code"` を弾く検証を書いたクライアントはありうる。標準由来の値は標準どおりであることを、Go の定数と RFC の両方で確認してから直す。

`@doc` の英語化 (T007) は 1870 件あり、機械的な翻訳では意味を落とす。既存の日本語が説明している fail-closed の条件や既定値の根拠を落とさないこと。落とすくらいなら、その `@doc` は context の SPECIFICATION.md へ移してから短い英語に置き換える。
