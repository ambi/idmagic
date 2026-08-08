---
status: pending
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

- [ ] T001 [App] `support_http` に Problem Details 共通ヘルパーを実装する。
- [ ] T002 [App] `error_handler.go` のフォールバックを Problem Details 化する。
- [ ] T003 [SCL+App] `admin_category_handler.go` の 2 条件に新しい Go
      コードと対応する SCL error model を追加する。
- [ ] T004 [App] コンテキスト単位でハンドラを新ヘルパーへ移行する
      (規模に応じてサブタスク化してよい)。wi-325 で新設した 37 個の
      granular error model に対応する箇所は status も 422/409 へ揃える。
- [ ] T005 [App] 全移行完了後 `WriteBrowserError` を削除する。
- [ ] T006 [Docs] `ARCHITECTURE.md` の HTTP error responses 節を実装完了
      状態に更新する。
- [ ] T007 [Verify] `just verify` を通す。

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
