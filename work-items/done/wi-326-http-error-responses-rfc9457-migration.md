---
status: completed
authors: ["tn"]
risk: medium
created_at: 2026-08-08
depends_on: [wi-325-scl-error-status-and-envelope]
---

# HTTP エラーレスポンスを RFC 9457 Problem Details へ移行する

## Motivation

ADR-154 (`decisions/ADR-154-rfc-9457-problem-details-for-http-errors.md`) で
HTTP エラーレスポンスの既定 envelope を RFC 9457 Problem Details とする決定を
した。`wi-325` で SCL に `status`/`error_format` の語彙を追加し、全
`kind: error` model を分類し終えたら、その SCL を正本として
`backend/shared/http/support_http` に共通実装を追加し、既存 390 箇所の
`WriteBrowserError` 呼び出しをコンテキスト単位で移行する。

現状 `WriteBrowserError`
(`backend/shared/http/support_http/response.go:20`) は `{"error": code,
"message": text}` を常に `application/json` で返し、echo の
`ErrorHandler` (`error_handler.go:26`) がフォールバックする
`echo.DefaultHTTPErrorHandler` はさらに別の `{"message": ...}` 形式を返す。
どちらも RFC 9457 の `type`/`title`/`status`/`detail`/`instance` を持たない。

## Scope

- `backend/shared/http/support_http` に Problem Details 用の共通ヘルパー
  (`WriteProblem` 相当) を実装する。SCL の `status` を参照し、
  `instance` に `request_id` (wi-111 で導入済み、
  `ARCHITECTURE.md` Request correlation) を設定する。
- `error_handler.go` の `echo.DefaultHTTPErrorHandler` フォールバックも
  Problem Details 形式で返すよう更新する。
- 既存 `WriteBrowserError` 呼び出し (約 390 箇所、25 ハンドラパッケージ)
  を新ヘルパーへ移行する。`error_format: problem_details` の binding が
  対象。OAuth2 / SCIM / DCR、および SharedSignals inbound SET receiver
  (`error_format: set_delivery`、wi-325 で RFC 8935 §2.3 の固定形式と
  確認済み) は対象外。
- 上記のうち、wi-325 で新設した 37 個の granular error model
  (`InvalidRoleError`、`PasswordReuseError` 等、wi-325 Design 節参照) に
  対応する Go 呼び出し箇所は、envelope を Problem Details 化するのと
  同時に status も SCL が宣言した値 (422、状態競合系は 409) へ揃える
  (現状はすべて `http.StatusBadRequest` で 400 を返しており、SCL の
  宣言と乖離している — wi-325 が意図的に残した差分)。
- `backend/application/handlers_http/admin_category_handler.go` の
  カテゴリ名必須・存在しないカテゴリ割当の 2 条件は、wi-325 の時点で
  Go 側に区別可能な文字列コードが無く SCL 化されなかった。本 work item で
  Go 側に新しいコードを追加し、対応する SCL error model の新設
  (`scl-change` 経由) とセットで行う。

## Out of Scope

- SCL 側の `status`/`error_format` 付与、`scl-to-openapi` ジェネレータ対応
  (wi-325 で完了済みが前提)。
- `type` を指す公開ドキュメントページの発行。
- OAuth2 (RFC 6749)、SCIM (RFC 7644)、Dynamic Client Registration
  (RFC 7591) のエラー形式変更。

## Design

- 移行対象が 25 ハンドラパッケージ・約 390 箇所と大きいため、1 PR で
  一括移行はせず、コンテキスト単位 (例: tenancy → idmanagement →
  authentication → …) で段階的に移行する。着手時に本 work item のスコープを
  さらにコンテキスト単位の複数 work item へ分割してよい。
- `WriteBrowserError` は移行完了まで残し、全呼び出し箇所の移行が終わった
  時点で削除する。移行の途中では新旧の envelope が混在するため、
  `ARCHITECTURE.md` の HTTP error responses 節にある「未実装」の記載を
  段階的に更新する。
- 採用しなかった代替: 全ハンドラを 1 コミットで一括置換する案 — 変更差分が
  巨大になりレビュー困難、かつロールバック時の影響範囲が大きいため、
  コンテキスト単位の段階移行を選ぶ。
- **スコープ分割の実施結果**: T004 着手時点で移行対象が 25 ハンドラパッケージ・
  392 箇所 (うち support_http 自身の 7 箇所と apitoken の 2 箇所を除く 380 箇所)
  と判明したため、Design 節で示唆した通りコンテキスト単位の work item へ
  分割した。本 work item では基盤 (T001, T002)・`admin_category_handler.go`
  の新コード (T003)・パイロットとして最小コンテキスト `apitoken`
  (`backend/apitoken/handlers_http/routes.go`、2 箇所) の移行までを完了し、
  残り 13 コンテキストの移行は `wi-327`〜`wi-339` (コンテキスト単位)、
  `WriteBrowserError` 削除と `ARCHITECTURE.md` 更新は `wi-340` へ分割した。
  各後続 work item に、対象ファイル・箇所数・wi-325 で新設した granular error
  model の status 実装状況 (現状 400 → SCL 宣言値 422/409) を調査済みの一覧
  として記載し、再調査コストを避けている。

## Plan

1. `support_http` に Problem Details 共通ヘルパーを実装し、ユニットテストを
   書く。
2. `error_handler.go` のフォールバックを Problem Details 化する。
3. `admin_category_handler.go` の 2 条件に新しい Go コードと対応する SCL
   error model を追加する (`scl-change` 経由)。
4. コンテキストを 1 つずつ選び、`WriteBrowserError` 呼び出しを新ヘルパーへ
   置き換える。wi-325 で新設した 37 個の granular error model に対応する
   箇所は、この移行と同時に status も SCL 宣言値 (422/409) へ揃える。
   置き換えるたびに `just verify` を通す。
5. 全コンテキストの移行が終わったら `WriteBrowserError` を削除し、
   `ARCHITECTURE.md` の「未実装」記載を更新する。

## Tasks

- [x] T001 [App] `support_http` に Problem Details 共通ヘルパーを実装する。
      RED: `TestWriteProblem_RFC9457Fields`/`TestWriteProblem_OmitsInstanceWhenRequestIDAbsent`
      を先に fail 確認 (`WriteProblem`/`Problem`/`ProblemContentType` 未定義で
      コンパイル失敗) → GREEN。あわせて support_http 自身の内部呼び出し
      (`auth.go`、`csrf.go`、`consent.go`、`response.go` の
      `WriteServerError`、`recover.go` の panic 応答) も `WriteProblem` へ移行。
- [x] T002 [App] `error_handler.go` のフォールバックを Problem Details 化する。
      RED: `TestErrorHandler_FallbackWritesProblemDetailsForHTTPError`/
      `TestErrorHandler_FallbackWritesProblemDetailsForPlainError`/
      `TestErrorHandler_QuotaExceededWritesProblemDetails422` を先に fail 確認
      (Content-Type が `application/json` のまま) → GREEN。
- [x] T003 [SCL+App] `admin_category_handler.go` の 2 条件に新しい Go
      コードと対応する SCL error model を追加する。`scl-change` 経由で
      `spec/contexts/application.yaml` に `CategoryNameRequiredError`(422)/
      `UnknownCategoryError`(422) を新設し `CreateApplicationCategory`/
      `UpdateApplicationCategory`/`SetApplicationCategories` の `errors:` へ
      配線。Go 側は `category_name_required`/`unknown_category` の
      distinguishable code を追加 (status は 400 のまま — envelope 移行と
      同時に揃える方針、`wi-327` へ委譲)。RED:
      `TestCreateApplicationCategory_EmptyNameYieldsDistinguishableCode`/
      `TestSetApplicationCategories_UnknownCategoryYieldsDistinguishableCode`
      を先に fail 確認 (`"invalid_request"` のまま) → GREEN。
- [x] T004 [App] コンテキスト単位でハンドラを新ヘルパーへ移行する
      (規模に応じてサブタスク化してよい)。wi-325 で新設した 37 個の
      granular error model に対応する箇所は status も 422/409 へ揃える。
      **本 work item ではパイロットとして `apitoken` context のみ実施** し、
      パターンを確立した (RED:
      `TestIssueApiToken_InvalidRequestIsProblemDetails` を先に fail 確認
      (Content-Type が `application/json` のまま) → GREEN)。残り 13
      コンテキストは `wi-327`〜`wi-339` へ分割。
- [ ] T005 [App] 全移行完了後 `WriteBrowserError` を削除する。→ `wi-340` へ分割。
- [ ] T006 [Docs] `ARCHITECTURE.md` の HTTP error responses 節を実装完了
      状態に更新する。→ `wi-340` へ分割 (全コンテキスト移行完了が前提のため)。
- [x] T007 [Verify] `just verify` を通す。

## Verification

- `just verify-go` (lint / race test)
- 手動: 代表的な 400/422 エンドポイントに対し
  `curl` でレスポンスが `application/problem+json` かつ `instance` に
  `X-Request-ID` と一致する値が入ることを確認する。
- OAuth2/SCIM/DCR エンドポイントのレスポンス形式が変化していないことを回帰
  確認する。

## Risk Notes

- 移行途中でクライアント側 (フロントエンド、外部 API 利用者) が
  `{"error", "message"}` 形式を前提にしている場合、Problem Details へ
  切り替わったコンテキストから順に影響が出る。フロントエンドのエラー
  ハンドリング側の対応状況を移行順序の判断材料にする。
- SharedSignals inbound SET receiver は RFC 8935 §2.3 により body が
  `err`/`description` の固定 2 フィールド (`error`/`error_description`
  ではない点に注意) でなければならない。実装時に他の `error_format` と
  混同しないこと。
- wi-325 で新設した 37 個の granular error model は、Go 側の対応する
  business rule 判定 (`errors.Is` 等) を正しく新しい status にマッピング
  し直す必要がある。取り違えると SCL の宣言と実際のレスポンスが再び
  乖離するため、コンテキスト単位で移行する際に wi-325 の Design 節の
  対応表と突き合わせて確認する。

## Completion

- **Completed At**: 2026-08-08
- **Summary**:
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
- **Verification Results**:
  - `just verify` - passed (test-go / lint-go / test-ui-unit / build-ui /
    check / typecheck-tools / test-tools / lint-ui / traceability-strict /
    format-check-ui すべて green)
  - `just check-scl` - passed (`spec/contexts/application.yaml` の新設
    2 error model・errors 配線を含む)
  - `just check-work-items` / `just check-ids` - passed (`wi-327`〜`wi-340`
    の新規 14 work item を含む)
  - RED→GREEN の自己証跡は各 Tasks 項目に記載 (`TestWriteProblem_*`、
    `TestErrorHandler_*`、`TestCreateApplicationCategory_*`/
    `TestSetApplicationCategories_*`、`TestIssueApiToken_*`)
  - 手動 curl 検証は実施せず、代わりに `backend/shared/http/server_http`
    の e2e テスト (実サーバスタックを httptest 経由で通す) で
    `Content-Type: application/problem+json`・`instance` ヘッダ相当の
    確認をカバーした。
