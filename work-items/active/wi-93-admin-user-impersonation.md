---
depends_on: [wi-43-account-portal-step-up-auth, wi-44-authentication-event-store-and-search]
status: pending
authors: ["tn"]
risk: high
created_at: 2026-07-03
priority: p3
group: authentication
change_kind: feature
affected_spec:
  - { path: docs/modules/identity-management/features/admin-access.md, requirement: REQ-IDMANAGEMENT-014 }
  - { path: docs/modules/authentication/features/session.md, requirement: REQ-AUTHENTICATION-021 }
  - { path: docs/modules/authentication/features/security-notification.md, requirement: REQ-AUTHENTICATION-031 }
---

# 管理者による代理ログインを、権限、対象、期限、監査を限って導入する

## 動機

サポート業務では、利用者として画面を見て操作する代理ログインが、再現の難しい不具合や設定の問題の調査に役立つ。
Keycloak の user impersonation、Okta と Entra の管理者による代理のアクセスがこれに当たる。
一方、代理ログインは権限の昇格、なりすまし、監査の欠落の危険が大きい。

TypeSpec は `SessionImpersonationStarted` と `SessionImpersonationEnded` を宣言し、本人への `impersonation` の通知も[セキュリティ通知](../../docs/modules/authentication/features/security-notification.md)に定めているが、どちらも発行する経路がない。
2026-10-25 の棚卸しで、重複する [wi-59470](../done/wi-59470-administrator-impersonation-sessions.md) の範囲をこの項目へ統合した。

語彙の検査の例外を減らすことだけを理由に着手せず、サポート業務での需要を確かめてから着手する。
新しい機能領域なので、優先度は p3 とする。

## 対象範囲

- 代理ログインの開始と終了。専用の権限を持つ管理者だけが、同じテナントの User を対象に、期限付きで開始できる。テナントの設定で有効にしたときだけ使える。
- 代理ログイン中のセッションに代理の管理者を保持し、すべての操作の監査に代理の管理者を行為者の連鎖として記録する。
- 開始と終了で `SessionImpersonationStarted` と `SessionImpersonationEnded` を発行し、本人へ通知する。
- 代理ログイン中であることを画面に常に表示する。
- パスワードと MFA の変更、アカウントの削除など、機微な操作を代理ログイン中は拒否する。

## 対象外

- 恒久的な委任管理。wi-94 が扱う。
- テナントをまたぐ代理ログイン。制御面の運用者が利用者として入ることも含めない。

## 設計

着手時に、行為者の連鎖を監査イベントとトークンにどう表すかを、[OAuth2 の委譲と行為者の連鎖の判断](../../docs/modules/oauth2/decisions.md)と揃えて決める。
代理ログイン中に発行したトークンを外部のアプリへ渡すかどうかも、このときに決める。

## 検証

- 機微な操作の拒否、期限での終了、監査とイベントと通知の発行を観測する受け入れのテスト。
- `mise run verify`

## リスク

代理ログイン中の操作が本人の操作として記録されると、監査が意味を失う。
すべての監査イベントに代理の管理者が載ることを、操作の一覧と照合して確かめる。
