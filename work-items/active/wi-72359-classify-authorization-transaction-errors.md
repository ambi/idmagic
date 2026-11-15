---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-11-15
priority: p1
group: spec-conformance
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/system/features/http-api.md, requirement: REQ-SYSTEM-023 }
---

# 認可トランザクションの不在と読取障害を区別する

## 動機

[エラー公開範囲の棚卸し](../done/wi-33421-audit-error-disclosure-boundaries.md)で、認可要求の読取障害が `transaction_unavailable` の説明へ写る経路を見つけた。
`backend/oauth2/internal/handlers_http/authorize_transaction.go` の `transactionRequest` は、Cookieの不在、期限切れ、処理済み状態と `RequestStore.Find` の障害を、すべてerrorとして返す。
`signin_continuation.go` の `SignInContinuation.Find` もそのerrorをそのまま返し、Authenticationのサインイン入口が `transactionErr.Error()` を401の説明へ写す。
`return_to` がある場合は同じ失敗を直接ログインへ進む条件にも使っているため、説明を固定するだけでは分類と診断を直せない。

## 対象範囲

- OAuth2の認可トランザクション取得、再開、同意とAuthenticationのパスワード、TOTP、WebAuthn、MFA登録の入口。
- `PendingAuthorizations.Find` の呼び出し元が、不在と内部障害を型または既知の値で区別できる契約。
- 不在だけが許す直接ログインへのフォールバックと、読取障害での無作用、相関ID付きの診断。

## 対象外

- 認可トランザクションの状態機械の作り直し。
- 一律に説明文を固定し、保存障害も401のまま扱う対応。

## 設計

比較する候補は、所有するポートで不在を値にする方式と、既知の拒否を型付きerrorにする方式である。
着手時に既存の直接ログインと第二要素の呼び出し元を列挙し、正常なフォールバックを維持できる最小の変更を選ぶ。
公開するポートの契約を変える場合は、境界の判断手順に従って比較を記録する。

## 計画

Cookie不在、期限切れ、処理済みと保存先障害を、通常の認可継続と `return_to` の両方で再現する。
業務上の利用不能だけを401へ変換し、読取障害は安全な500と診断へ渡す。

## タスク

- [ ] T001 [Acceptance] 読取ポートへSQLとラップした障害を注入し、最終応答と直接ログインへの分岐を再現する。
- [ ] T002 [App] トランザクションの不在と内部障害を区別し、すべての呼び出し元へ反映する。
- [ ] T003 [Verify] 拒否後のセッション、認可要求、コードとイベントの無変更を検証する。

## 検証

`mise run brief` から影響する規範と契約テストを解決し、各入口の `primary_use_cases` を計画する。
局所検査は `test-go-one`、パッケージ検査は `test-go-package`、最終検査は `verify-changed` を使う。

## リスク

すべての取得失敗を500へ変えると、Cookieを持たない正常な直接ログインが失敗する。
逆にすべてを利用不能へ畳むと、障害時に認証と認可継続の経路が変わり、原因も失われる。
