---
status: completed
archived_from: f38047e82cc3eb840e45081034d4cc4bb0ff3114
---

# HTTP エラーレスポンスを RFC 9457 Problem Details へ移行する

`backend/shared/http/support_http` に RFC 9457 Problem Details 共通ヘルパー
`WriteProblem`/`Problem`/`ProblemContentType` (`problem.go`) を test-first
(RED→GREEN) で実装した。`type` は `urn:idmagic:error:<code>`、`title` は
code を人間可読化した文字列、`instance` は `logging.RequestIDFromContext`
経由の request_id (wi-111)。support_http 自身の内部呼び出し (`auth.go` の
`WriteAccessTokenError`/`WriteAdminAccessError`、`csrf.go`、`consent.go`、
`response.go` の `WriteServerError`、`recover.go` の panic 応答) も
あわせて移行した。`error_handler.go` の `echo.DefaultHTTPErrorHandler`
フォールバックを独自の `problemFallback` に置き換え、
status 別の committed guard・`*echo.HTTPError` の message 抽出・
status text からのコード導出 (`problemCodeForStatus`) を実装した。
`admin_category_handler.go` の「カテゴリ名必須」「存在しないカテゴリ割当」
2 条件に distinguishable code (`category_name_required`/
`unknown_category`) を追加し、`scl-change` 経由で
`spec/contexts/application.yaml` に対応する `kind: error` model
(`CategoryNameRequiredError`/`UnknownCategoryError`、いずれも status: 422)
を新設して `CreateApplicationCategory`/`UpdateApplicationCategory`/
`SetApplicationCategories` の `errors:` へ配線した (Go 側の status は
wi-325 の 37 モデルと同じ理由で、対象コンテキストの envelope 移行と
同時に揃える方針のため 400 のまま残した)。
移行対象が 25 ハンドラパッケージ・392 箇所 (support_http 自身の 7 箇所と
この work item で移行した `apitoken` の 2 箇所を除くと残り 380 箇所) と
判明したため、Design 節記載の通りコンテキスト単位の work item へ分割した:
パイロットとして `apitoken` context (`backend/apitoken/handlers_http/routes.go`)
を実際に移行しパターンを確立し、残り 13 コンテキストは `wi-327`
(application)、`wi-328` (audit)、`wi-329` (authentication)、`wi-330`
(idgovernance)、`wi-331` (idmanagement)、`wi-332` (oauth2)、`wi-333`
(provisioning)、`wi-334` (saml)、`wi-335` (sharedsignals)、`wi-336`
(signingkeys)、`wi-337` (tenancy)、`wi-338` (workloadidentity)、`wi-339`
(wsfederation) へ分割した。各 work item には対象ファイル・箇所数・
wi-325 で新設した granular error model の現状 status (実装時の変更対象)
を調査済みの一覧として記載し、再調査コストを避けている。`WriteBrowserError`
の削除と `ARCHITECTURE.md` の記載更新 (元 T005/T006) は全コンテキスト移行
完了が前提のため `wi-340` (13 work item すべてに依存) へ分割した。
移行に伴い、`WriteAccessTokenError`/`WriteAdminAccessError` の envelope
変更に依存していた既存 e2e テスト
(`backend/shared/http/server_http/routes_e2e_test.go` の
`TestAccountContextRejectsStaleBearerToken`、
`backend/audit/handlers_http/admin_audit_event_handler_test.go` の
`TestAdminAuditEventsRequiresAdminRole`) のアサーションを新 envelope に
更新した。`/api/auth/account` の未認証応答
(`backend/idmanagement/user/handlers_http/account_handler.go`、まだ未移行)
を検証する 2 テストは意図的に旧 envelope のアサーションのまま残した
(誤って新形式へ書き換えたが、実装がまだ追従していないコードパスだと
気づいて元に戻した — `wi-331` で移行される)。
**対応していないこと (Out of Scope として明記)**:
- 残り 13 コンテキスト (`application`/`audit`/`authentication`/
`idgovernance`/`idmanagement`/`oauth2`/`provisioning`/`saml`/
`sharedsignals`/`signingkeys`/`tenancy`/`workloadidentity`/
`wsfederation`) の `WriteBrowserError` 呼び出し・約 380 箇所は未移行。
`wi-327`〜`wi-339` で扱う。
- `WriteBrowserError` 自体の削除、`ARCHITECTURE.md` の「未実装」記載の
更新は `wi-340` へ持ち越し。
- `admin_category_handler.go` の新設 2 code (`category_name_required`/
`unknown_category`) は distinguishable code の追加のみ完了、envelope と
status の実装追従は `wi-327` (application context) で行う。
- 元々の Scope 通り OAuth2 (RFC 6749)/SCIM (RFC 7644)/DCR (RFC 7591)、
SharedSignals inbound SET receiver (RFC 8935 §2.3、`error_format:
set_delivery`) の応答形式は変更していない。
- 実装中に判明した追加の乖離: SharedSignals inbound SET receiver
(`handleReceiveSecurityEvent`) は現在 `{error, message}` を返しており、
RFC 8935 §2.3 が MUST で要求する `{err, description}` フィールド名にも
実は追従できていない (`wi-335` の Risk Notes に記録し、別途 work item化を
検討することとした)。

全文は `git show f38047e82cc3eb840e45081034d4cc4bb0ff3114:<このファイルのパス>` で読む。
