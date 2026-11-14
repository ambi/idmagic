---
depends_on: [wi-6-real-email-sender-adapter]
status: pending
authors: ["tn"]
risk: medium
created_at: 2026-07-03
priority: p3
group: authentication
change_kind: feature
affected_spec:
  - { path: docs/modules/authentication/features/sign-in.md, requirement: REQ-AUTHENTICATION-007 }
  - { path: docs/modules/authentication/features/password.md, requirement: REQ-AUTHENTICATION-016 }
---

# メールのワンタイムコードとリンクによるパスワードレスのログインを、テナントの設定で有効にする

## 動機

IdMagic 自身が確かめる一次要素はパスワードであり（外部 IdP と Kerberos への委譲を除く）、メールによるパスワードレスの手段はない。
Okta、Entra ID、Keycloak は、メールのマジックリンクやワンタイムコードを一次要素として提供する。
パスワードを覚えず使い回さない体験になり、パスワードのフィッシングと使い回しの危険を避けられる。

## 対象範囲

- ログインの画面でメールアドレスを入力すると、期限付き、単回の使用のコードとリンクを送り、そのどちらかでログインを完了する。
- テナントの設定で有効にしたときだけ使える。デフォルトは無効とする。
- 送信の応答はアカウントの有無で変えず、試行回数と送信の頻度を制限する。
- トークンの AMR と ACR にメールによる認証を反映し、アプリのサインインポリシーが MFA を求める場合は第二要素を続けて求める。

## 対象外

- 第二要素としてのメールと SMS のワンタイムコード。wi-295 が扱う。ワンタイムコードの仕組みは同じ形なので、先に完了したほうが置き、後のほうは作り直さない。
- パスワードを無効にするテナントのポリシー。
- ステップアップでのメールの利用。

## 検証

- 期限切れと再利用のコードとリンクが拒否されることと、無効にしたテナントでは開始できないことを観測する受け入れのテスト。
- `mise run verify-ui`
- `mise run verify`

## リスク

メールの受信箱を奪われると、ログインも奪われる。
MFA を求めるアプリでは第二要素を省略させない。
