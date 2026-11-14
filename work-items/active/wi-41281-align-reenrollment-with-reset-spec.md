---
status: pending
authors: [tn]
risk: medium
created_at: 2026-11-14
change_kind: bugfix
priority: p3
depends_on: []
affected_spec:
  - { path: docs/modules/authentication/features/mfa.md, requirement: REQ-AUTHENTICATION-022 }
---

# 認証器の全リセット後の再登録の強制について規範と実装を揃える

## 動機

REQ-AUTHENTICATION-022 は「再登録が必要な User が次に正しいパスワードでログインしたとき、同じログインセッションを `pending_purpose=Enrollment` にする」と、サインインポリシーによらない形で書いている。
実装は、実効のサインインポリシーが MFA を求めるときだけ再登録を強制し、求めないときはパスワードだけでログインを完了させる（2026-11-14 に [[wi-93893-verify-control-plane-mfa-recovery]] の受け入れテストで観測）。

## 対象範囲

- どちらへ揃えるかを決める。候補は、ポリシーによらず再登録を強制するよう実装を直す、ポリシーが MFA を求めるときだけと規範を直す、の二つである。
- 決めた側を規範とテストで固定する。

## 対象外

- 復旧の手段そのもの。

## タスク

- [ ] T001 [Decision] 揃える側を決める。
- [ ] T002 [Spec] 規範を更新する。
- [ ] T003 [Acceptance] MFA を求めないポリシーでの全リセット後のログインを受け入れテストで固定する。

## 検証

- 最終検査は `mise run verify-changed -- <着手時の基準リビジョン>` とする。

## リスク

実装を直す場合、MFA を求めないテナントの利用者が、リセット後のログインで再登録を求められるようになる。
