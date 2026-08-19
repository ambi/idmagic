---
status: pending
authors: [tn]
risk: low
created_at: 2026-08-19
change_kind: bugfix
priority: p3
depends_on: []
affected_spec:
  - { path: spec/contexts/authentication/main.tsp, symbol: IdMagic.Authentication.Operations.StartBrowserMfaEnrollment }
---

# ブラウザー経路の MFA 登録が「既に登録済み」を「登録が許可されていない」に畳んでいるのを解く

## Motivation

`writeBrowserEnrollmentError` (`backend/oauth2/handlers_http/authorize_enrollment.go`) は `ErrMfaEnrollmentNotAllowed` と `ErrMfaAlreadyEnrolled` を同じ分岐で受け、どちらも 403 `mfa_enrollment_not_allowed` を返す。管理 API はこの 2 つを区別し、後者を 409 `mfa_already_enrolled` として返す (`backend/authentication/mfa/handlers_http/admin_mfa_enrollment_handler.go`)。

利用者から見ると意味がまるで違う。「登録が許可されていない」は管理者に問い合わせる状況で、「既に登録済み」は何もしなくてよい状況である。同じ code で返す限り、ログイン画面は 2 つを別の文面で案内できない。

`wi-382` は T009 でこの分岐を調べ、403 という status 自体は正しい (要求元セッションに対する認可判断である) と結論した。残っているのは code の畳み込みだけである。

## Scope

- ブラウザー経路で `ErrMfaAlreadyEnrolled` に固有の code を返す。status は管理 API と同じ 409 が妥当かを判断し、決めた側を契約に書く。
- `StartBrowserMfaEnrollment` / `ConfirmBrowserMfaEnrollment` の error union に対応するモデルを加える。
- ログイン画面の i18n 辞書に対応する文面を足す。

## Out of Scope

- `mfa_enrollment_not_allowed` そのものの status。403 のままとする (`wi-382` T009 の判断)。

## Verification

- `just verify`
- 手動確認: MFA 登録済みの user がブラウザー経路で登録を開始すると、「登録が許可されていない」とは別の code が返る。

## Risk Notes

ログイン途中の画面が返す code を増やすので、未知の code に落ちたときの既定文面が用意されていることを先に確かめる。fail-closed の側 (登録させない) は変えない。
