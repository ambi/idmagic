---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# TypeSpec が記述する契約を、実際の線上表現と公開文書の言語方針に合わせる

契約が記述する要求・応答本体、エラー本体、enum の値、`unknown`、`@doc` の言語を、サーバーが実際に受理し返すものと AGENTS.md の言語表に合わせた。`just spec-diff` が示す意味の差は、名前付きモデルが 31 件増え、失われた宣言が 1 件も無いことである。増えた 31 件の内訳は、横断的なエラー本体 (`ProblemDetails`、`DependencyStatus`)、標準が形を定める接点の本体 (SCIM 11 件、`ClientRegistrationResponse`)、実在する要求本体 (`SamlServiceProviderRequest` / `SamlIdentityProviderProfileRequest` / `WsFedRelyingPartyRequest` / `IdentityProviderConnectionRequest`)、T009 が要求した新しいエラー (`AuthenticationRequiredError` / `SessionNotFoundError` / `UserNotFoundError` / `ActiveJobQuotaExceededError` / `SecurityEventTokenTooLargeError` / `SecurityEventStreamNotFoundError`)、接点ごとに分けたエラー (`OAuthInvalidRequestError` / `OAuthAccessDeniedError`)、実体のあった応答 (`AdminRolePolicy` 系 3 件、`UserImportJobRef` / `UserImportResult`)。消えた宣言はすべて `<Op>HttpRequest` / `<Op>HttpResponse` の架空のラッパーで、`spec-diff` は transport wrapper として除外するため差分に現れない。
T009 の 3 つの判断:
- **`invalid_attribute` は 422 が正しい。** 属性スキーマへの適合は「解析できた内容が業務規則に違反する」に当たる (`docs/SPECIFICATION.md` HTTP error responses)。契約に 422 を書き、account endpoint が 400 を返している実装の修正は `wi-383` に切り出した。
- **`mfa_enrollment_not_allowed` は 403 と 422 の双方が正しい。** ブラウザー経路の 403 は要求元セッションに対する認可判断、管理 API の 422 は対象 user に対して bypass を発行できないという業務規則違反で、条件が違う。両方の union にモデルを置いた。残る欠陥はブラウザー経路が `ErrMfaAlreadyEnrolled` を同じ code に畳んでいることだけで、`wi-384` に切り出した。
- **`quota_exceeded` は status ではなく code を分ける。** 429 (実行中ジョブ数の上限、待てば通る) と 422 (テナント資源クォータ、業務規則違反) はどちらも正しい status で、1 つの code が 2 概念を指しているのが欠陥である。契約に `ActiveJobQuotaExceededError` を分けて置き、`type` がいまは同じ URN であることを `@doc` に明記した。code の分割は `wi-383` に切り出した。
`check-api-compat` の 448 findings はすべて分類して読み切った。架空のラッパーの除去が 336 件 (`request` / `response` / 資源名のフィールド削除と、全プロパティが path/query だった要求本体の削除)、path/query パラメータの要求本体からの除去が 63 件、二重封筒の解消が 21 件、enum 値の修正が 15 件、JSON でない本体の型変更が 8 件、接点ごとにモデルを分けたことによる error code の入れ替えが 10 件 (`InvalidRequestError` → `OAuthInvalidRequestError` 8 件、`AccessDeniedError` → `OAuthAccessDeniedError` 2 件)。実在するクライアントを壊すものは 1 件も無い。baseline が凍結していた本体はサーバーが一度も受理したことのないもので、enum の旧値もサーバーが受理したことがない。
作業中に判明した副次的な事実を 2 つ記録する。ひとつは `spec-diff` の宣言抽出が行頭に固定されておらず、英語になった `@doc` の散文 (`the union of the roles` など) を宣言として拾っていたこと。同じコミットで正規表現を行頭固定にし、回帰テストを足した。もうひとつは、テナント解決 middleware が routing の手前で返す 404 `{"error": "tenant_not_found"}` が第 4 のエラー本文であり、どの operation も宣言していないこと。operation の外なので本 work item では書かず、`wi-386` に記録した。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
