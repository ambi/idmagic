---
status: pending
authors: ["tn"]
risk: high
created_at: 2026-07-10
---

# 管理者ロールの JIT 特権昇格を導入する

## Motivation
現状の admin / system_admin 権限は User.roles または scoped admin assignment によって
常時有効になる。Microsoft Entra Privileged Identity Management のような Just-In-Time
昇格がないと、管理者が普段から強い権限を持ち続け、侵害時の影響が大きい。

本 WI は、対象ユーザーが理由、期限、step-up、必要なら承認を満たしたときだけ
管理者ロールを一時的に有効化する privileged role activation を導入する。

## Scope
- **scl**:
  - `IdentityManagement` に PrivilegedRoleEligibility / PrivilegedRoleActivation / ActivationStatus を追加する。
  - admin role の effective evaluation に「eligible だが未 activate」と「active activation」を区別して反映する。
  - activation request / approve / deny / expire / revoke events を追加する。
  - `Authentication` の step-up / MFA を activation 前提条件として参照する。
- **go**:
  - eligibility / activation repository、期限切れ評価、effective roles への activation 差し込みを実装する。
  - activation 時に step-up recency と MFA を検証する。
- **http**:
  - 自分の eligible role 一覧、activation 申請、承認/拒否、取消 API を追加する。
- **ui**:
  - admin shell に「特権を有効化」導線、承認者向け queue、現在有効な activation 表示を追加する。
- **documentation**:
  - README に JIT 昇格の利用手順、最大期間、監査イベントを追記する。

## Out of Scope
- 委任管理のスコープモデル自体。これは `wi-94-delegated-administration` が扱う。
- 外部 PAM / ticketing system 連携。
- break-glass アカウント運用。
- cross-tenant privileged access。

## Plan
- 常時 role assignment と JIT eligibility を別 model として持ち、effective roles には active activation だけを足す。
- activation は短い最大期限を持ち、期限切れは read-time evaluation と将来 job sweep の両方に対応できるようにする。
- high risk 操作なので activation には MFA step-up と理由入力を必須にする。
- 初期は自己 activation + 監査から始め、承認必須 role は同一 model の policy で表現する。

## Tasks
- [ ] T001 [SCL] Eligibility / Activation model、state、events、effective role invariant を追加する。
- [ ] T002 [Decision] activation 条件、期限上限、承認要否、break-glass 境界を ADR に記録する。
- [ ] T003 [App] activation usecase と effective roles 評価を実装する。
- [ ] T004 [HTTP] activation / approval API を追加する。
- [ ] T005 [UI] activation と承認 queue の UI を追加する。
- [ ] T006 [Verify] SCL、Go、UI、手動シナリオを検証する。

## Verification
- `just yaml-check`
- `just check-ids`
- `just test-go`
- `just verify-ui`
- 手動: eligible なユーザーが step-up 後に admin role を短時間有効化し、期限後に権限が消えることを確認する。
- 手動: eligible でないユーザー、期限切れ activation、承認拒否済み activation では管理 API が拒否されることを確認する。

## Risk Notes
特権昇格の誤実装は権限昇格脆弱性になる。eligibility と active activation を厳密に分け、期限・step-up・承認状態を全 admin 認可で fail-closed に評価する。activation / revocation は監査イベントに必ず残す。
