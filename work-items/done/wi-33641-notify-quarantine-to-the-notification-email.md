---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-25
risk_notes: |
  通知が届かないと、隔離で止まった下流の反映に管理者が気付かない。逆に宛先を誤ると、接続名や隔離の理由を第三者へ送る。
priority: p2
depends_on: []
change_kind: feature
evidence_policy: risk-based-v3
documentation_impact:
  level: release_note
  reason: 接続を隔離すると notification_email へメールが届くようになり、管理画面の通知テンプレート一覧に新しいキーが加わる。運用者は宛先の設定と文面の上書きを見直す必要がある。
  references:
    - { kind: release_note, path: docs/releases/changes/wi-33641-notify-quarantine-to-the-notification-email.md }
initial_context:
  specification:
    - docs/domain/provisioning/scenarios.feature.md#REQ-PROVISIONING-011
    - docs/domain/provisioning/internals.md
    - docs/domain/tenancy/internals.md
  typespec:
    - IdMagic.Contract.NotificationTemplateKey
    - IdMagic.Contract.ConnectionQuarantined
  source:
    - backend/shared/notification/ports/template.go
    - backend/shared/notification/template/render.go
    - backend/shared/notification/template/notifier.go
    - backend/shared/notification/template/defaults/ja.yaml
    - backend/provisioning/usecases/job_handler.go
    - backend/provisioning/usecases/reconcile.go
    - backend/provisioning/module.go
    - backend/cmd/idmagic-worker/worker.go
    - frontend/src/features/admin-settings/NotificationTemplatesTab.tsx
    - infra/schema/postgres.sql
  tests:
    - backend/provisioning/usecases/job_handler_test.go
    - backend/provisioning/e2e_reconcile_test.go
  stop_before_reading: [backend/provisioning/client_scim, backend/provisioning/db_postgres, backend/tenancy]
affected_spec:
  - { path: docs/domain/provisioning/scenarios.feature.md, requirement: REQ-PROVISIONING-011 }
  - { path: spec/contexts/tenancy/models.tsp, symbol: IdMagic.Contract.NotificationTemplateKey }
primary_use_cases:
  - id: quarantine-notifies-notification-email
    requirement: REQ-PROVISIONING-011
    observable_result: 誤削除ガードで接続を隔離すると、notification_email へ隔離を知らせるメールが一通届き、管理者が隔離を解除すると health が ok に戻る。
    unit_test: { path: backend/provisioning/usecases/job_handler_test.go, name: TestProvisioningTaskHandler_QuarantineNotifiesTheNotificationEmailOnce, task: test-go-race }
    e2e_test: { path: backend/provisioning/e2e_reconcile_test.go, name: TestE2E_ReconcileOverTheGuardNotifiesAndIsResumed, task: test-go-race }
    unit_fault_model: 隔離の共通処理が通知を組み立てても、宛先を notification_email ではなく別の値にするか、隔離のたびでなく失敗のたびに送る。
    e2e_fault_model: Module.ReconcileDeps が worker から受け取った Notifier をインクリメンタル同期の依存へ渡さず、誤削除ガードの隔離でメールが送られない。
---

# 接続の隔離を `notification_email` へメールで通知する

## 動機

`EX-PROVISIONING-011-01` は、誤削除ガードで接続を隔離したら `notification_email` へ通知すると宣言している。
TypeSpec の `ProvisioningConnection.notification_email` も「失敗と隔離を通知する宛先」と説明している。

実装はどの経路でも通知しない。
連続失敗による隔離（`recordConsecutiveFailure`）も、[[wi-61629-quarantine-connections-on-the-accidental-deletion-guard]] が加える誤削除ガードによる隔離も、`ConnectionQuarantined` を発行するだけで、メールは送らない。
隔離はプロビジョニングを止める状態なので、管理者が気付かなければ、下流の反映はいつまでも止まったままになる。

wi-61629 はこの通知を対象外にした。
メールは共有の `Notifier` を通り、テンプレートのキーは Tenancy の TypeSpec `NotificationTemplateKey` と双子定義である。
キーを足すと、Tenancy の公開契約、OpenAPI のベースライン、管理画面の通知テンプレート一覧（ja と en）まで波及し、ガードとは独立に受け入れられる変更になるためである。

## 対象範囲

- 隔離を知らせる通知テンプレートのキーを `NotificationTemplateKey` へ足し、ja と en の組込み既定を用意する。
- 連続失敗と誤削除ガードの双方で、接続を隔離したときに `notification_email` へ送る。宛先が未設定なら送らない。
- `EX-PROVISIONING-011-01` を名指すテストを書き、`tools/check/example-coverage-debt.json` から外す。

## 対象外

- 隔離以外の失敗（`dead_letter` など）の通知。
- 通知の再送や配送の保証。
- 本文へ Application の表示名を載せること。Provisioning は `ApplicationRepository` へ依存しておらず、隔離という例外的な状況の通知のためだけに依存を足さない。本文は `application_id` を載せ、管理画面の接続一覧で health から隔離された接続を探せる。
- OpenAPI の互換性ベースラインの更新。`docs/development/release.md` に従い、リリースの準備で固定する。

## 設計

### テンプレート

`NotificationTemplateKey` へ `ProvisioningConnectionQuarantined: "provisioning_connection_quarantined"` を足す。
Go の双子定義 `TemplateKeyProvisioningConnectionQuarantined`、差し込み変数の許可集合、プレビュー用のサンプル値、ja と en の組込み既定、`notification_templates.template_key` の CHECK、管理画面の型とラベル（ja と en）を揃える。

| 差し込み変数 | 値 |
| --- | --- |
| `application_id` | `ProvisioningConnection.ApplicationID` |
| `quarantine_reason` | 保存した `ProvisioningConnection.QuarantineReason`（長さの上限で切り詰めた後の値） |
| `quarantined_at` | `QuarantinedAt` を `2006-01-02 15:04 UTC` で書いた値 |

宛先は User ではないので `RecipientLocale` は空とし、テナントの既定言語から解決する。
全キー共通の `user_display_name` は空文字列を渡す。
渡さないと、それを参照するテナントの上書きが描画に失敗し、メールが送られなくなるためである。

### 送る場所

連続失敗と誤削除ガードの両経路が通る `quarantineConnection` で、`ConnectionQuarantined` の発行の後に送る。

```go
// quarantineNotification は隔離した接続の通知を組み立てる。宛先が未設定か空白だけなら false を返す。
func quarantineNotification(conn *domain.ProvisioningConnection) (notificationports.Notification, bool)

func quarantineConnection(ctx context.Context, repo ports.ProvisioningConnectionRepository, emitFn func(spec.DomainEvent),
    notifier notificationports.Notifier, conn *domain.ProvisioningConnection, reason string, now time.Time) error
```

`JobHandlerDeps` と `ReconcileDeps` に `Notifier notificationports.Notifier` を足す。nil なら送らない。
`Module.JobHandlerDeps` と `Module.ReconcileDeps` は `emit` と同じく `notifier` を引数で受け取り、worker は `deps.Notification.Notifier` を渡す。
`Notifier` は fail-open なので、送信の失敗は隔離の保存とイベントの発行を取り消さない。

採用しない案として、`ConnectionQuarantined` を購読して送る方法がある。
イベントの購読から通知を送る仕組みは Provisioning になく、この通知一つのために購読の配線を足すより、既に隔離を保存して発行する一か所で送るほうが小さい。

## タスク

- [x] T001 [Spec] `NotificationTemplateKey` へキーを足し、Tenancy と Provisioning の `internals.md` へ差し込み変数と送る条件を書く。`mise run check-spec`、`mise run check-api-compat`。
- [x] T002 [Acceptance] `TestE2E_ReconcileOverTheGuardNotifiesAndIsResumed`（`EX-PROVISIONING-011-01`）の RED を確認する。`mise run test-go-test -- ./backend/provisioning TestE2E_ReconcileOverTheGuardNotifiesAndIsResumed`。
- [x] T003 [Catalog] テンプレートのキーと既定文面を足し、`mise run test-go-package -- ./backend/shared/notification/template` を GREEN にする。
- [x] T004 [App] `TestProvisioningTaskHandler_QuarantineNotifiesTheNotificationEmailOnce` と宛先未設定のテストの RED を確認し、`quarantineConnection` で送る。`mise run test-go-test -- ./backend/provisioning/usecases <test>`。
- [x] T005 [Adapters] `Module` と worker を配線し、受け入れテストを GREEN にする。管理画面の型とラベルを足す。
- [x] T006 [Verify] `mise run test-go-mutation -- backend/provisioning/usecases`、`mise run verify`、`mise run test-ui-e2e`。

## 検証

- `mise run check-contract-drift`
- `mise run check-api-compat`
- `mise run verify`

## 完了

- **Completed At**: 2026-09-26
- **Summary**:
  `mise run spec-diff` は main に対する規範の差分なしと報告した。`REQ-PROVISIONING-011` の文面は変えず、`EX-PROVISIONING-011-01` が既に宣言していた `notification_email` への通知を実装した。
  公開契約では、TypeSpec の `NotificationTemplateKey` に `provisioning_connection_quarantined` を追加した（`mise run check-api-compat` は非互換なしと判定した）。
  連続失敗と誤削除ガードのどちらで接続を隔離しても、隔離を保存して `ConnectionQuarantined` を発行した後、`notification_email` へアプリケーション ID、隔離の理由、隔離の時刻を載せたメールを一通送る。宛先が未設定か空白だけなら送らない。
  ja と en の組込み既定、差し込み変数の許可集合、`notification_templates` の CHECK、管理画面の型とラベルを揃え、`EX-PROVISIONING-011-01` を被覆の負債台帳から外した。
- **Primary Use Case Evidence**:
  - id: quarantine-notifies-notification-email
    unit_red: 隔離の共通処理がメールを送らないため、TestProvisioningTaskHandler_QuarantineNotifiesTheNotificationEmailOnce が `notifications = [], want exactly [{... To:provisioning-alerts@example.test Key:provisioning_connection_quarantined ...}]` で失敗した。
    e2e_red: Module.ReconcileDeps で組み立てたインクリメンタル同期が誤削除ガードで隔離してもメールが送信境界へ届かないため、TestE2E_ReconcileOverTheGuardNotifiesAndIsResumed が `mails sent = 0, want exactly one quarantine notice` で失敗した。
    unit_fault_injection: 通知の宛先を notification_email から接続のアプリケーション ID に替えると、TestProvisioningTaskHandler_QuarantineNotifiesTheNotificationEmailOnce が `To:app-1` を報告して失敗した。
    e2e_fault_injection: Module.ReconcileDeps が受け取った notifier を ReconcileDeps へ渡さないようにすると、TestE2E_ReconcileOverTheGuardNotifiesAndIsResumed が `mails sent = 0` で失敗した。
- **Change-Resistance Results**:
  `mise run test-go-mutation -- backend/provisioning/usecases` は 192 件中 178 件を検出した。`job_handler.go` の追加部分（`quarantineNotification` の宛先の判定と `quarantineConnection` の送信）に生き残った変異はない。
  `reconcile.go:72` の 3 件はこのパッケージのテストで被覆されず、親パッケージの TestE2E_ReconcileOverTheGuardNotifiesAndIsResumed と TestE2E_ReconcileOverTheGuardQuarantinesWithoutDeprovisioning が担う。`reconcile.go:163` の生存は割り当ての読み込み（`assignedUserIDs`）の既存の分岐で、この変更とは関係ない。
  宛先未設定のテスト TestProvisioningTaskHandler_QuarantineSendsNothingWithoutANotificationEmail は、実装前から通る防御のテストであり RED は観測していない。宛先の判定の否定変異は変異ツールが検出した。
  `Module.JobHandlerDeps` の notifier の配線は E2E で表明していない。連続失敗の経路の E2E（`e2e_lifecycle_events_test.go`）は notifier に nil を渡す。
- **Verification Results**:
  - `mise run check-spec` - 成功
  - `mise run check-api-compat` - 成功
  - `mise run lint-go` - 成功
  - `mise run test-go-changed` - 成功
  - `mise run verify` - 成功
  - `mise run test-ui-e2e` - 初回は `admin can reset a user authenticator from the browser` が既知の WebKit の不具合（oven-sh/bun#43412、evaluate の応答なし）で失敗し、再実行で 39 件すべて成功
