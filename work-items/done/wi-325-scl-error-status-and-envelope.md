---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# SCL の error model へ HTTP status と envelope 形式を導入し RFC 9110 準拠の 400/422 に再分類する

SCL 言語 (`SPECIFICATION_CORE_LANGUAGE.md`) に `kind: error` model の
`status` field (MUST) と `kind: http` binding の `error_format` field
(`problem_details` 既定 | `oauth2` | `scim` | `set_delivery`) を追加し、
`tools/check/schemas/scl-v3.schema.json` と Go 側の strict decoder
(`backend/shared/spec/loader.go`) を対応するスキーマ変更で更新した。
19 の bounded context ファイルおよび `tools/ra`/`tools/scl-to-openapi`
自己記述 SCL、合計 21 ファイル・77 個の `kind: error` model すべてに
`status` を付与し (RFC 9110 に沿って 400/401/403/404/409/422/500/503 を
各 model の意味論に応じて割り当て、既存 Go 実装のステータスコードと
整合させた)、OAuth2 (12 interface, RFC 7591 DCR 含む)・SCIM (15
binding)・SharedSignals inbound SET receiver (1 binding, RFC 8935 の
固定形式を新設の `set_delivery` として表現) の `kind: http` binding へ
`error_format` を設定した。`tools/scl-to-openapi` を、単一の `default`
エラーレスポンスではなく HTTP status 別のレスポンス + `error_format` に
応じた content-type (`application/problem+json`/`application/json`/
`application/scim+json`) を生成するよう変更し (test-first: RED→GREEN)、
`just scl-render` で `spec/idmagic.openapi.json` 等の派生物を再生成した。
さらに、共通 `InvalidRequestError` (400) に紛れ込んでいた Go 側の
ビジネスルール違反コード (`invalid_role`、`password_reuse` 等、14 ファイル・
約40箇所) を Go handler → SCL interface の対応関係を洗い出した上で
37 個の専用 error model (`status: 422`、状態競合系は `409`) として
個別にモデリングし、6 コンテキストファイルの該当 interface の
`errors: [...]` へ配線した。ADR-154 は本 work item の実装により
`suggested` から `accepted` に更新した。
**未対応の範囲 (Out of Scope として明記)**:
- `admin_category_handler.go` の 2 条件 (カテゴリ名必須・存在しない
カテゴリ割当) のみ、Go 側に区別可能な文字列コードが無いため対象外
(Go 側の変更とセットで `wi-326` にて対応)。
- 新設した 37 個の granular error model のうち、Go 側の対応する
ビジネスルール違反コードは引き続き `WriteBrowserError(...,
http.StatusBadRequest, ...)` で 400 を返している。SCL は正しい
status (422/409) を宣言済みだが、実際の HTTP レスポンスが追従するのは
`wi-326` が該当呼び出し箇所を Problem Details へ移行するタイミングに
なる (SCL-first の意図した一時的な spec/実装間の乖離)。
- `backend/shared/http/support_http` の Problem Details 実装と約 390
箇所のハンドラ移行そのものは `wi-326` の scope。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
