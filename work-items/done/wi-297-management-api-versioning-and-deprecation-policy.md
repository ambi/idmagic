---
status: completed
archived_from: 3220367f30e9cf53d981104b4544fab54a7334c5
---

# 管理 API の版管理と非推奨化ポリシーを定義し、外部契約として保証する

[[ADR-156-api-versioning-and-deprecation-policy]] で互換の定義・パス版管理
(`/api/admin`・`/api/account`、現行パス=v1)・同時2版・非推奨から最低12か月・
`Deprecation`/`Sunset` ヘッダ・internal 境界 (API access token 到達可能性で判定) を決定した。
全 324 interface に `stability` (stable 158 / internal 166) を付与し、`System` に
`InterfaceStability`/`Deprecation` glossary と非推奨 scenario を追加、`BackendErrorResponse`
のエラーコードを契約として `stable` にした。Go 側は `Echo.OnAddRoute` フックで
`/api/admin`・`/api/account` 配下の全 route を `/v1/` へ自動ミラーする
`RegisterVersionAliases`(既存 handler の登録コードは無改修) と、SCL の
`deprecated_since`/`sunset_at` を読んで応答ヘッダを付与する `DeprecationHeadersMiddleware`
を追加。`tools/scl-to-openapi` は同じ `/v1/` ミラーと `x-scl-stability` 等の拡張 field を
生成に反映。新設 `tools/check-api-compat` は `$ref` 解決付き再帰 schema diff で
ADR-156 の破壊的変更定義 (フィールド削除・改名・型変更・必須化・既定値変更・エラーコード削除・
path/operation/parameter/response 削除) を検知し、`spec/idmagic.openapi.baseline.json`
(このWI完了時点を最初のリリースベースラインとして固定) との差分を `just check-api-compat`
として CI (`idmagic-ci.yaml`) と `just verify` に組み込んだ。README に運用手順を追記。
**対応していないこと (Out of Scope の通り)**: 既存 API の実際の破壊的変更・形状変更は
行っていない (現行パスは無変更で全既存テストが green)。OAuth2/OIDC/SAML/WS-Federation/
SCIM/SharedSignals のプロトコル endpoint は IdMagic 独自の版管理の対象外のまま
(discovery が正本、[[ADR-011-discovery-as-derived-artifact]])。SDK/クライアントライブラリ、
Terraform provider 等の IaC 統合、rate limit/quota は本 WI の範囲外。SCL 自体の
`spec_version` (3.0) は変更していない。`beta` stability は仕組みとして用意したが、
該当する interface は現時点で無い。

全文は `git show 3220367f30e9cf53d981104b4544fab54a7334c5:<このファイルのパス>` で読む。
