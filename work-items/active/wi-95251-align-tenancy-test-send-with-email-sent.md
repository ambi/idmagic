---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-17
priority: p3
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/tenancy/notification-template/README.md, requirement: REQ-TENANCY-018 }
---

# 通知の試し送りが EmailSent を発行するかどうかを、仕様と実装で一致させる

## 動機

EX-TENANCY-018-01 は、操作者が `SendTestNotification` を呼ぶと「宛先は操作者本人に固定され、`EmailSent` が発行される」と述べる。
実装の `SendTestNotification`（`backend/tenancy/internal/usecases/manage_notification_templates.go`）はイベントを発行しない。依存の `NotificationTemplateDeps` は `Emit` を持たない。
例を引くテスト `TestSendTestNotificationGoesToTheActorOnly` は、送信先と配送の結果だけを確かめ、イベントを確かめない。
要件の REQ-TENANCY-018 の本文は `EmailSent` に触れておらず、例だけが要件にない作用を述べている。

ほかのメールの送信（パスワードのリセット、メールアドレスの変更）は、送信を試みたとき、成否によらず `EmailSent` を発行し、SIEM が送信の失敗をこのイベントで観測する。

## 対象範囲

- 試し送りが `EmailSent` を発行するかどうかを決め、REQ-TENANCY-018 の本文、EX-TENANCY-018-01、実装、テストを一致させる。
- 発行する場合は、Tenancy の `domain` に `EmailSent` の構造体を置き、`purpose` の値を決め、要件に発行する条件を書く。

## 対象外

- 試し送りの宛先、言語、応答の形の変更。
- 通知の送信境界（`backend/shared/notification` の `Notifier`）がイベントを発行する形への変更。

## 設計

着手時に次の問いを決める。

| 問い | 候補 | 判断の材料 |
| --- | --- | --- |
| 試し送りは `EmailSent` を発行するか | 発行する（仕様に合わせて実装を直す）、発行しない（例から作用を消す） | 試し送りも外部へのメールの送信であり、送信の失敗を SIEM が観測できるか。試し送りは応答で `delivered` を返すので、操作者自身は結果を知れる |
| 発行する場合の `purpose` | `notification_test` など | TypeSpec の `EmailSent.purpose` の説明が挙げる値（`password_reset`、`email_change`、`email_verification`）との整合 |

発行する場合、イベントは Tenancy が所有する。
具象のイベントはモジュールの `domain` に置き、ワイヤ表現は TypeSpec の `EmailSent` に従う。

## タスク

- [ ] T001 [Spec] 発行するかどうかを決め、REQ-TENANCY-018 と EX-TENANCY-018-01 を一致させる。
- [ ] T002 [Use Cases] 決めた結果に合わせて `SendTestNotification` とテストを直す。
- [ ] T003 [Verify] 変更を検証する。

## 検証

- `mise run test-go-package -- ./backend/tenancy/...`
- `mise run check-spec`、`mise run check-unspecified-vocabulary`、`mise run check-event-contract`

## リスク

- 発行する場合、監査ログに試し送りの記録が増える。宛先は操作者本人のアドレスのハッシュだけを記録する。
