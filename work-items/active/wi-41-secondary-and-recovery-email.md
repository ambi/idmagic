---
depends_on: []
status: pending
authors: ["tn"]
risk: medium
created_at: 2026-06-21
priority: p3
group: identity-management
change_kind: feature
affected_spec:
  - { path: docs/modules/identity-management/features/account.md, requirement: REQ-IDMANAGEMENT-017 }
  - { path: docs/modules/authentication/features/password.md, requirement: REQ-AUTHENTICATION-016 }
---

# 本人が復旧用のメールアドレスと追加のメールアドレスを登録し、検証できるようにする

## 動機

アカウントポータルは、プライマリのメールアドレスの変更（再検証を含む）までを扱う（[REQ-IDMANAGEMENT-017](../../docs/modules/identity-management/features/account.md)）。
Google、Okta、Keycloak のアカウントの画面は、プライマリのほかに追加のメールアドレスと、アカウントの復旧の宛先を持つ。
プライマリのメールアドレスを失った利用者は、パスワードの再設定のメールを受け取れない。

## 対象範囲

- 復旧用のメールアドレスを一つと、追加のメールアドレスを本人が登録、検証、削除できるようにする。追加のメールアドレスをプライマリに切り替えられるようにする。
- 検証は、メールアドレスの変更と同じワンタイムトークンの方針（ハッシュだけの保存、単回の使用、期限）に従う。
- パスワードの再設定で、検証済みの復旧用のメールアドレスを宛先に使えるようにする（[REQ-AUTHENTICATION-016](../../docs/modules/authentication/features/password.md)）。

## 対象外

- 電話番号の登録と検証。SMS の送信の手段を持つ wi-295 が扱う。
- 連絡先を使ったステップアップと通知。
- 管理者による連絡先の代理の編集。

## 設計

着手時に、連絡先を属性として持つか、専用のテーブルを持つかを決め、IdManagement の判断に書く。
属性を薄く保つ方針で複数の値の連絡先を一度落とした経緯があるので、複数のメールアドレスを持つことの影響（SCIM の `emails`、CSV、ログインの識別子との関係）を先に確かめる。

## 検証

- 検証していないメールアドレスを復旧の宛先に使えないことを観測する受け入れのテスト。
- `mise run verify-ui`
- `mise run verify`

## リスク

復旧用のメールアドレスは、奪われるとアカウントの乗っ取りの経路になる。
登録と変更をステップアップの対象にし、プライマリの宛先へ変更を通知する。
