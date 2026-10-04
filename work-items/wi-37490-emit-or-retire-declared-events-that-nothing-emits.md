---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-04
priority: p1
depends_on: [wi-35451-rewrite-remaining-context-specifications-in-the-requirement-format]
change_kind: bugfix
affected_spec:
  - { path: docs/domain/authentication/security-notification/README.md, requirement: REQ-AUTHENTICATION-031 }
  - { path: docs/domain/provisioning/connection/README.md, requirement: REQ-PROVISIONING-002 }
  - { path: docs/domain/oauth2/authorization/README.md, requirement: REQ-OAUTH2-053 }
---

# 宣言しているのに発行されないドメインイベントを、発行するか宣言から外す

## 動機

wi-35451 の語彙の検査で、宣言しているのにどの経路も発行しないイベントが見つかった。
特に `SessionImpersonationStarted` が発行されないので、REQ-AUTHENTICATION-031 が求める `impersonation` の必須のセキュリティ通知が、なりすまされた本人に届かない。

| Context | 発行されないイベント |
| --- | --- |
| Authentication | `AuthenticationStepCompleted`、`AuthenticationStepFailed`、`MfaChallengeIssued`、`MfaChallengeSucceeded`、`MfaChallengeFailed`、`SessionStarted`、`SessionRefreshed`、`SessionImpersonationStarted`、`SessionImpersonationEnded` |
| Provisioning | `ProvisioningConnectionUpdated`、`ProvisioningConnectionDisabled`、`ProvisioningConnectionDeleted` |
| OAuth2 | `AuthorizationDetailsRejected` |

## 対象範囲

- イベントごとに、発行する（要件に書く）か、宣言と保持期間の表から外すかを決めて実装する。
- なりすましのセッションの開始で `SessionImpersonationStarted` を発行し、`impersonation` の通知が届くようにする。
- 語彙の検査の許容リストから、扱ったイベントを外す。

## 対象外

- 新しいイベントの追加。

## 設計

監査の利用者が依存し得るイベント（セッション、なりすまし、接続の変更、認可詳細の拒否）は発行する側へ倒す。
発行する経路を持たない中間の段のイベント（`AuthenticationStep*`、`MfaChallenge*`）は、同じ事実を記録する既存のイベントがあるかを確かめてから、外すかを決める。
決めた結果は、この記録の設計に表で残す。

## タスク

- [ ] T001 [Spec] イベントごとの扱いを決め、要件に書く。
- [ ] T002 [Acceptance] なりすましの通知が届かないことを RED で確認する。
- [ ] T003 [App] 発行を実装し、外すイベントの宣言を消す。
- [ ] T004 [Verify] `mise run check-unspecified-vocabulary` と `mise run verify` を通す。

## 検証

- `mise run verify`

## リスク

- 宣言から外すイベントを、監査の検索の画面やエクスポートが名前で参照している場合がある。外す前に参照を検索する。
