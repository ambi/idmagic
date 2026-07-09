---
status: pending
authors: ["tn"]
risk: high
created_at: 2026-07-03
---

# 管理者による代理ログイン (impersonation) を監査付きで導入する

## Motivation
サポート業務では「ユーザとして画面を見て操作する」代理ログインが、再現困難な
不具合や設定問題の調査に有効。代表的な IdP も提供する:

- Keycloak: user impersonation。
- Okta / Entra: 管理者による代理アクセス。

一方 impersonation は権限昇格・なりすまし・監査欠落のリスクが大きい。本 WI は、
専用権限・対象と時間の境界・厳格な監査 (actor chain、ADR-049 と整合)・
impersonation 中の可視バナー・機微操作の禁止を備えた形で導入する。

## Scope
- **decision**:
  - 新規 ADR: impersonation の許可条件 (専用権限 / 対象制限 / 時間制限)、監査 (開始・終了・操作を impersonator を含む actor chain で記録、 ADR-049 と整合)、可視バナーによる不可視ななりすまし防止、機微操作 (パスワード / MFA 変更 / 削除) の禁止範囲、テナント設定での有効化を記録する。
- **scl**:
  - §3.3 interfaces: StartImpersonation / EndImpersonation を追加する。
  - §3.7 permissions: impersonation は専用権限に固定し fail-closed とする。
  - §3.4 states/events: ImpersonationStarted / ImpersonationEnded を追加し、 session に impersonator を保持する。
  - §3.5 invariants: 全操作が impersonator を含めて監査され、cross-tenant impersonation を禁止することを明示する。
- **go**:
  - impersonation usecase を追加し、session に actor + impersonator を持たせ、 機微操作をブロックし、全操作を監査する。
- **http**:
  - admin からの開始 / 終了と、impersonation バナー用 context を追加する。
- **ui**:
  - AdminUsers 詳細に「代理ログイン」、全画面に impersonation 中バナーを 追加する。
- **documentation**:
  - README に impersonation の権限・制約・監査を追記する。

## Out of Scope
- 恒久的な delegated access ([[wi-94-delegated-administration]] とは別)。
- cross-tenant impersonation。
- impersonation 中のパスワード / MFA 変更・アカウント削除。

## Verification
- `just test-go`
- `just lint-go`
- `just build-go`
- `just typecheck-ui`
- `just lint-ui`
- `just build-ui`
- 手動: 専用権限を持つ admin が代理ログイン → バナーが出て対象ユーザ画面を閲覧 できる。機微操作が禁止され、開始 / 終了 / 操作が監査に impersonator 付きで 残ることを確認する。
- 手動: 権限が無い admin / cross-tenant では impersonation が拒否されることを確認する。

## Risk Notes
impersonation は本質的に権限昇格・なりすましであり、監査欠落や機微操作の許可は
重大インシデントに直結する。専用権限・fail-closed・全操作監査・機微操作禁止・
cross-tenant 禁止・可視バナーを必須ガードとしてテストで担保する。
