---
id: idp-wi-92-configurable-password-policy
title: "設定可能なパスワードポリシー (長さ / 履歴 / 有効期限) を導入する"
created_at: 2026-07-03
authors: ["tn"]
status: pending
risk: medium
---

# Motivation
現状のパスワード検証は breach チェック ([[wi-24-hibp-breached-password-checker]])
と最低限に留まり、テナントがポリシーを調整できない。代表的な IdP は password
policy を設定可能にする:

- Keycloak: password policies (length / history / expiry / regex 等)。
- Okta / Entra: password policy (length / complexity / age / history)。

一方 NIST SP 800-63B は「複雑性の強制」「定期変更の強制」を非推奨とし、最小長 +
breach チェックを中心に据える。本 WI は**モダンなガイドラインを既定**としつつ、
テナントが必要に応じて履歴・有効期限・追加要件を opt-in できる設定可能ポリシーを
導入する。

# Scope
- **decision**:
  - 新規 ADR: サポートするルール (最小長 / breach / 履歴 / 有効期限 / 任意の 追加要件) と既定値を決める。既定は NIST 準拠 (最小長 + breach 中心、複雑性 / 定期変更は既定 off)。履歴用 past-hash の保管方針、有効期限は RequiredAction.UpdatePassword と連携する点を記録する。
- **scl**:
  - §3.2 models: PasswordPolicy を追加する。
  - §3.3 interfaces: ChangePassword / ResetPasswordWithToken / RegisterUser ([[wi-87-self-service-user-registration]]) の検証にポリシーを反映する。
  - §3.4 states/events: PasswordPolicyUpdated を追加する。
  - §3.5 invariants: ポリシーは全パスワード設定経路で一貫適用され、履歴は hash 比較で平文を保持しないことを明示する。
  - tenancy: AdminSettings に password policy 設定を追加する。
- **go**:
  - policy 評価器と password history store (hash) を追加し、既存 HIBP チェックと 合成する。有効期限超過で UpdatePassword required action を付与する。
- **http**:
  - admin の policy 設定エンドポイントと、検証失敗時の要件メッセージを追加する。
- **ui**:
  - AdminSettingsPage にポリシー編集、ChangePasswordPage / RegisterPage に 要件表示を追加する。
- **documentation**:
  - README にポリシー項目と既定値 (NIST 準拠) を追記する。

# Out of Scope
- パスワードなし強制 / passwordless-only ポリシー。
- 外部辞書サービスとの連携 (breach は既存 HIBP に留める)。
- 高度なリアルタイム strength meter。

# Verification
- `just test-go`
- `just lint-go`
- `just build-go`
- `just typecheck-ui`
- `just lint-ui`
- `just build-ui`
- 手動: ポリシーを設定 → 要件を満たさないパスワードが change / reset / register で拒否される。履歴に一致するパスワードへの再設定が拒否されることを確認する。

# Risk Notes
ポリシーを厳しくしすぎると UX を損ない、緩いと無意味。既定を NIST 準拠に置き、
複雑性 / 定期変更は opt-in にする。履歴 hash の取り扱い (平文比較禁止) と、全設定
経路での一貫適用漏れが主なリスクで、経路ごとにテストを置く。
