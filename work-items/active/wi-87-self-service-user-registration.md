---
depends_on: [wi-6-real-email-sender-adapter]
status: pending
authors: ["tn"]
risk: medium
created_at: 2026-07-03
priority: p3
group: identity-management
change_kind: feature
affected_spec:
  - { path: docs/modules/authentication/features/sign-in.md, requirement: REQ-AUTHENTICATION-007 }
  - { path: docs/modules/identity-management/features/user.md, requirement: REQ-IDMANAGEMENT-001 }
---

# テナントが許可したとき、利用者が自分でアカウントを登録できるようにする

## 動機

User を作れるのは、管理者、SCIM、CSV、Directory Connector、外部 IdP の初回のログインだけであり、利用者が自分でアカウントを登録する入口がない。
Keycloak の User registration、Okta と Entra External ID のセルフサービスのサインアップのように、顧客向け（B2C、B2B）の IdP には、テナントが許可した場合に限り利用者が自分で登録し、メールの検証を経て使い始める入口が要る。

## 対象範囲

- メールアドレス、パスワード、表示名による最小の登録を、認証の要らない公開の入口として加える。対象のテナントは解決したテナントに固定する。
- テナントの設定で有効にしたときだけ使える。デフォルトは無効とする。
- 登録した User は、メールを検証するまでログインできない。メールアドレスはテナントの中で一意とし、登録の応答はアカウントの有無で変えない。
- 登録をドメインイベントとして記録し、テナントの User の Hard Quota に従わせる。

## 対象外

- 外部 IdP とソーシャルログインによる初回の登録。既にある。
- CAPTCHA とボットの対策。どの作業項目の対象でもないので、必要になったら起票する。
- 招待による登録。wi-294 が扱う。
- 段階的に属性を集めること。

## 設計

メールの検証のワンタイムトークンは、パスワードレスのログイン（wi-88）と同じ形を使う。
先に完了したほうがその仕組みを置き、後のほうは作り直さない。

## 検証

- 検証前のログインの拒否、重複したメールアドレスの扱い、無効にしたテナントでの拒否を観測する受け入れのテスト。
- `mise run verify-ui`
- `mise run verify`

## リスク

公開の入口は、大量の登録とメールの送信に悪用される。
入口ごとの流量の制限と、テナントのクォータで上限を設ける。
