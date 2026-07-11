---
depends_on: []
status: pending
authors: ["tn"]
risk: high
created_at: 2026-07-09
---

# 管理者が認証器をリセットし再登録を強制する緊急復旧導線を追加する

## Motivation

現状 idmagic では TOTP / WebAuthn を喪失したユーザーの復旧経路が、本人管理の backup
recovery code **のみ**である（[ADR-087](file:///Users/tn/src/idmagic/decisions/ADR-087-webauthn-phishing-resistant-mfa.md)）。
recovery code も失うと復旧経路が実質ゼロになり、単一障害点になっている。Explore 調査でも
管理者による認証器リセットや緊急ロック解除の導線は存在しないことを確認した。

Okta（Reset Authenticators）、Microsoft Entra ID（Authentication Administrator による
リセット + Require re-register）、Keycloak（OTP credential 削除 + required action 再登録）は
いずれも「管理者による認証器リセット + 次回ログイン時の再登録強制」を企業向けの緊急
backstop として備える。idmagic はこの層を欠いている。[ADR-088](file:///Users/tn/src/idmagic/decisions/ADR-088-layered-account-recovery.md)
の第 2 層としてこの導線を仕様化する。

## Scope

- `spec/contexts/authentication.yaml`:
  - 管理者操作 interface（対象ユーザーの認証器リセット。TOTP factor / WebAuthn credential /
    recovery code の削除と、次回ログイン時の MFA 再登録強制フラグ設定）。
  - 認証器リセットに伴うドメイン状態（`User.mfa_enrolled` の再計算、再登録要求状態）。
  - 監査イベント（authenticator reset requested / completed、re-enrollment required）。
- `spec/contexts/application.yaml` および管理 UI:
  - ユーザー詳細画面での「認証器をリセット」操作、リセット対象 factor の選択、確認 UX。
  - 権限モデル（管理者 / 委任管理者スコープ）を既存の admin 操作に揃える。
- Authentication use cases: リセット実行、`mfa_enrolled` 再計算、再登録強制状態のセット、
  再登録強制と [wi-127](file:///Users/tn/src/idmagic/work-items/wi-127-mfa-enrollment-onboarding-and-enforcement.md) の enrollment-required flow の接続。
- OAuth2 browser login handlers: リセット済みユーザーは次回ログインで MFA 再登録 flow に入る。
- Persistence adapters: 再登録強制状態 / required action の保存（必要に応じて）。

## Out of Scope

- 本人確認（ID プルーフィング / ライブネス）ベースのセルフサービス復旧（SSAR 相当）。ADR-088 で
  将来検討に回した。
- 管理者発行の時限アクセスパス（TAP 相当）。必要なら別 work item。ここでは「リセット + 再登録強制」に絞る。
- メールルートによる自動復旧（[wi-41](file:///Users/tn/src/idmagic/work-items/wi-41-secondary-and-recovery-email.md) の範疇）。
- 手段冗長化の推進（ADR-088 第 1 層。別 work item）。

## Plan

- 方針:
  - リセットは既存の管理者操作・権限モデル・監査枠組みに揃え、新しい認可軸を増やさない。
  - リセットで対象 factor を削除し、`mfa_enrolled` を残存要素に応じて再計算する。全 factor を
    失った場合は「次回ログインで MFA 再登録を要求」状態にし、wi-127 の enrollment-required flow へ接続する。
  - リセット単体では新しい factor を作らない（管理者が任意の factor を勝手に登録できると別の
    なりすまし面になる）。あくまで削除 + 再登録要求に留める。
  - 全操作を監査イベント化し、誰が誰の何をリセットしたかを追跡可能にする。
- 参考にする外部パターン: Okta Reset Authenticators、Entra ID Require re-register MFA、
  Keycloak OTP 削除 + required action。
- 却下する代替案:
  - 管理者が新 factor を直接登録して渡す: 管理者経由のなりすまし面を作るため不可。削除 + 再登録要求に限定。
  - リセット時に recovery code を自動再発行して管理者に見せる: 平文コードが管理者を経由し漏洩面が広がる。行わない。
- 未決定事項: 委任管理者（[wi-94](file:///Users/tn/src/idmagic/work-items/wi-94-delegated-administration.md)）とのスコープ境界、リセット時のユーザー通知メール
  （[wi-90](file:///Users/tn/src/idmagic/work-items/wi-90-account-security-notification-emails.md)）連携の要否。

## Tasks

- [ ] T001 [SCL] 認証器リセット interface、再登録強制状態、監査イベント、管理 UX を `authentication.yaml` / `application.yaml` に仕様化する。
- [ ] T002 [Domain] リセット後の `mfa_enrolled` 再計算と再登録要求状態の判定を追加する。
- [ ] T003 [UseCase] 管理者リセット use case（対象 factor 削除 + 再登録要求セット）を追加する。
- [ ] T004 [UseCase] リセット済みユーザーのログインを wi-127 の enrollment-required flow に接続する。
- [ ] T005 [Admin/UI] ユーザー詳細画面にリセット操作・対象選択・確認 UX を追加する。
- [ ] T006 [Audit] リセット要求 / 完了 / 再登録要求を監査イベントに出す。
- [ ] T007 [Verify] E2E で、全 factor リセット後の再登録強制ログイン、部分リセット後の残存要素動作、権限外操作の拒否を固定する。

## Verification

- `just yaml-check`
- `just test-go`
- `just verify-ui`
- `just test-ui-e2e`
- 手動確認:
  - 管理者が対象ユーザーの全認証器をリセットすると、そのユーザーは次回ログインで MFA 再登録を求められる。
  - 一部 factor のみリセットした場合、残存要素で引き続きログインできる。
  - 権限を持たない操作者はリセットできない。
  - リセット・再登録要求が監査イベントに記録される。

## Risk Notes

リスクは高い。認証器リセットは認証境界を管理者権限で越える操作であり、乱用や設計ミスは
なりすまし・恒久ロックアウトに直結する。緩和策として、リセットは削除 + 再登録要求に限定して
管理者による factor 直接登録を禁じ、既存の admin 権限・監査枠組みに揃え、全操作を監査イベント
必須とする。再登録強制は wi-127 の fail-closed な enrollment-required flow を再利用する。
