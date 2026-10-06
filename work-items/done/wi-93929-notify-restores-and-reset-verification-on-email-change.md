---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-04
priority: p1
depends_on: [wi-17076-limit-specifications-to-external-contracts-written-once]
change_kind: bugfix
evidence_policy: risk-based-v4
documentation_impact:
  level: release_note
  reason: 復元した User が下流で戻ることと、管理者がメールアドレスを変えると確認済みが外れることを、運用者へ知らせる必要がある。
  references:
    - { kind: release_note, path: docs/releases/changes/wi-93929-notify-restores-and-reset-verification-on-email-change.md }
initial_context:
  specification:
    - docs/domain/identity-management/user/lifecycle.md#REQ-IDMANAGEMENT-049
    - docs/domain/identity-management/user/README.md#REQ-IDMANAGEMENT-045
    - docs/domain/provisioning/synchronization/README.md#REQ-PROVISIONING-006
  typespec: []
  source:
    - backend/idmanagement/user/usecases/admin_users.go
    - backend/provisioning/usecases/capture.go
    - backend/provisioning/usecases/execute_task.go
    - backend/provisioning/domain/reconcile.go
  tests:
    - backend/idmanagement/user/usecases/user_rules_test.go
    - backend/provisioning/usecases/capture_test.go
    - backend/provisioning/e2e_lifecycle_events_test.go
  stop_before_reading: [frontend, spec/contexts]
primary_use_cases:
  - id: restore-notifies-re-enable
    requirement: REQ-IDMANAGEMENT-049
    observable_result: 管理者が PendingDeletion の User を復元すると、下流のプロビジョニングへその User の再有効化が一度だけ通知される。
    boundary: acceptance
    test: { path: backend/idmanagement/user/usecases/user_rules_test.go, name: TestRestoreUserNotifiesProvisioningOfTheReEnabledUser, task: test-go-race }
    fault_model: 復元が通知しない、または削除や属性の変更として通知する。
  - id: re-enable-cancels-scheduled-deprovisions
    requirement: REQ-PROVISIONING-006
    observable_result: 猶予期間つきの削除を予約した User が再び有効になると、すべての接続の削除の予約が取り消される。
    boundary: unit
    test: { path: backend/provisioning/usecases/capture_test.go, name: TestCaptureLifecycleEvent_ReEnablingCancelsEveryConnectionsScheduledDeprovision, task: test-go-race }
    fault_model: 再有効化が予約を取り消さない、または一つの接続の予約だけを取り消す。
  - id: restore-within-grace-keeps-the-downstream-user
    requirement: REQ-PROVISIONING-006
    observable_result: 猶予期間 7 日の接続で削除を予約した User を猶予期間内に復元すると、期限の経過後も下流へ DELETE が送られない。
    boundary: e2e
    reason: 復元の通知が IdManagement の通知の境界、Provisioning の通知の変換、捕捉の取消を通って届くことは、モジュールを組んだ経路でしか確かめられない。
    test: { path: backend/provisioning/e2e_lifecycle_events_test.go, name: TestE2E_RestoreWithinTheGracePeriodCancelsTheDELETE, task: test-go-race }
    fault_model: 復元の通知が取消に届かず、期限の経過後に下流へ DELETE が送られる。
  - id: admin-email-change-resets-verification
    requirement: REQ-IDMANAGEMENT-045
    observable_result: 管理者が email_verified を指定せずに確認済みのメールアドレスを変えると email_verified が false になり、changed_fields に email と email_verified が載る。同じ要求で email_verified=true を指定すると true のまま保存する。
    boundary: acceptance
    test: { path: backend/idmanagement/user/usecases/user_rules_test.go, name: TestUpdateUserResetsEmailVerifiedWhenTheAddressChanges, task: test-go-race }
    fault_model: メールアドレスの変更が email_verified を戻さない、指定した値を無視して false にする、または changed_fields に email_verified を載せない。
affected_spec:
  - { path: docs/domain/identity-management/user/lifecycle.md, requirement: REQ-IDMANAGEMENT-049 }
  - { path: docs/domain/identity-management/user/README.md, requirement: REQ-IDMANAGEMENT-045 }
  - { path: docs/domain/provisioning/synchronization/README.md, requirement: REQ-PROVISIONING-006 }
---

# User の復元を下流へ通知し、管理者がメールアドレスを変えたら確認済みの状態を戻す

## 動機

User の状態や連絡先が変わっても、それに伴う作用の一部が起きていない。

| 対象 | 今の挙動 | 起きる問題 |
| --- | --- | --- |
| 復元 | 削除の予約は下流のプロビジョニングへ User の削除として通知するが、復元は通知しない | 下流の宛先では、復元した User が削除されたまま残る |
| 管理者によるメールアドレスの変更 | 変更前に確認済みなら、`email_verified` が `true` のまま残る | 所有を確かめていない新しいアドレスを、確認済みとして扱う |

## 対象範囲

- 管理者が User を復元したとき、下流のプロビジョニングへ通知する。
- 管理者が User のメールアドレスを別の値へ変えたとき、`email_verified` を `false` にする。
- REQ-IDMANAGEMENT-049 と REQ-IDMANAGEMENT-045 の要件文と、それを表す例を改める。

## 対象外

- 本人によるメールアドレスの変更。
  確認のリンクを経由して確認済みにするので、変えない。
- 通知の失敗の扱いの統一。
  Group と User で異なる点は、別に扱う。
- CSV の取り込みと SCIM の受信によるメールアドレスの変更。
  どちらも `email_verified` を戻さないが、取り込みは `email_verified` の列を持ち、SCIM の受信は上流を正とするので、扱いを別に決める。

## 設計

### 要件の差分

| 要件 | 変更 | 新しい要件文 |
| --- | --- | --- |
| REQ-IDMANAGEMENT-049 | 「下流のプロビジョニングへ通知しない」を置き換える | 管理者が User を復元したとき、IdManagement は、User の再有効化として下流のプロビジョニングへ通知する。 |
| REQ-IDMANAGEMENT-045 | 「メールアドレスだけを変えたとき `email_verified` を変えない」を置き換える | 管理者が `email_verified` を指定せずにメールアドレスを別の値へ変えたとき、IdManagement は、`email_verified` を `false` にし、値が変わったなら `changed_fields` に `email_verified` を載せる。管理者が同じ要求で `email_verified` を指定したとき、IdManagement は、指定した値を保存する。 |
| REQ-PROVISIONING-006 | 一文を足す | 猶予期間の間に User が再び有効になったとき、Provisioning は、すべての接続でその User の削除の予約を取り消す。 |

例は EX-IDMANAGEMENT-049-03 と EX-IDMANAGEMENT-045-03 の期待結果を改め、取消の境界は EX-PROVISIONING-006-03 として足す。

### 通知の種類

復元は、既存の `user_enabled` と同じ通知（`operation=update`）として送る。
下流の扱いは `DeprovisionPolicy.on_delete` ごとに次のとおり収束し、通知の種類を増やさずに済む。

| `on_delete` | 削除の予約のときに起きたこと | 復元の通知で起きること |
| --- | --- | --- |
| `deactivate` | 下流で `active=false` | リンクがあるので PATCH で `active=true` に戻す |
| `delete`（猶予期間なし） | 下流から DELETE し、リンクを消した | リンクがないので、実行が作成へ切り替える |
| `delete`（猶予期間あり） | 削除の予約だけを保存した | 予約を取り消し、更新を送る |
| `none` | 何も送らない | 更新を送る（下流の状態は変わらない） |

予約の取消は `user_enabled` を受けたときに行う。
無効化からの再有効化には予約がないので、取消は何もしない。
接続の状態によらず取り消す点は、割り当ての追加による取消と同じである。

### 仕様にない振る舞いの分類

| 観点 | 見つけた振る舞い | 分類 |
| --- | --- | --- |
| 外向きの作用 | 復元しても猶予期間つきの削除の予約が残り、期限に下流を削除する | (a) REQ-PROVISIONING-006 に取消を足す（利用者に確認済み） |
| 項目の間の不変条件 | メールアドレスと `email_verified=true` を同じ要求で指定した更新 | (a) 指定した値を優先する（利用者に確認済み） |
| 同じ種類の操作 | CSV の取り込みと SCIM の受信もメールアドレスを変えるが、`email_verified` を戻さない | 対象外へ移す |

## 計画

1. 要件の差分を書く。
2. 受け入れ境界で RED を確認し、実装する。

## タスク

- [x] T001 [Spec] REQ-IDMANAGEMENT-049、REQ-IDMANAGEMENT-045、REQ-PROVISIONING-006 の差分を書く。検査：`mise run check-spec`。
- [x] T002 [Acceptance] 復元の通知（EX-IDMANAGEMENT-049-03）と、確認済みの状態のリセット（EX-IDMANAGEMENT-045-03）の RED を確認し、GREEN にする。検査：`mise run test-go-test -- ./backend/idmanagement/user/usecases <test>`。
- [x] T003 [App] 再有効化による予約の取消（EX-PROVISIONING-006-03）の RED を確認し、GREEN にする。検査：`mise run test-go-test -- ./backend/provisioning/usecases <test>` と `mise run test-go-test -- ./backend/provisioning <test>`。
- [x] T004 [Verify] 変異試験、`mise run lint-go`、`mise run verify` で検証する。

## 検証

- `mise run check-work-items`
- `mise run verify`

## リスク

- 下流の宛先が、削除した User の再作成を受け付けない場合がある。
  通知の種類を決める前に、Provisioning の宛先ごとの扱いを確かめる。

## 完了

- **Completed At**: 2026-10-07
- **Summary**:
  `mise run spec-diff` は main に対する規範仕様の差分として REQ-IDMANAGEMENT-045、REQ-IDMANAGEMENT-049、REQ-PROVISIONING-006 の変更を報告した。
  管理者が User を復元すると、IdManagement は再有効化（`user_enabled`）として下流のプロビジョニングへ通知する。下流で無効化した User は有効に戻り、削除した User は既存の実行経路が作り直す。
  Provisioning は User の再有効化を受けると、接続の状態によらず、すべての接続でその User の猶予期間つきの削除の予約を取り消す。
  管理者が `email_verified` を指定せずにメールアドレスを変えると `email_verified` を `false` にし、値が変われば `changed_fields` に `email_verified` を載せる。同じ要求で指定した `email_verified` はその値を保存する。管理画面はすでにこの規則どおりに値を送っており、変更していない。
  例 EX-IDMANAGEMENT-049-03 と EX-IDMANAGEMENT-045-03 の期待結果を改め、EX-PROVISIONING-006-03 を加えた。リリースノートを追加した。
- **Primary Use Case Evidence**:
  - id: restore-notifies-re-enable
    red: 復元が通知しなかったため、TestRestoreUserNotifiesProvisioningOfTheReEnabledUser が「notifications=[], want [{userID:alice trigger:user_enabled}]」で失敗した。
    fault_injection: RestoreUser から notifyProvisioning の呼び出しを外すと、TestRestoreUserNotifiesProvisioningOfTheReEnabledUser が同じ表明で失敗した。
  - id: re-enable-cancels-scheduled-deprovisions
    red: 再有効化が予約を取り消さなかったため、TestCaptureLifecycleEvent_ReEnablingCancelsEveryConnectionsScheduledDeprovision が、二つの予約が残ったことを報告して失敗した。
    fault_injection: 取消の条件に最初の接続だけを加えると、TestCaptureLifecycleEvent_ReEnablingCancelsEveryConnectionsScheduledDeprovision が、予約が一つ残ったことを報告して失敗した。
  - id: restore-within-grace-keeps-the-downstream-user
    red: 復元しても予約が残ったため、TestE2E_RestoreWithinTheGracePeriodCancelsTheDELETE が「downstream received DELETE /Users/remote-user-1 after the restore cancelled it」で失敗した。
    fault_injection: RestoreUser から notifyProvisioning の呼び出しを外すと、TestE2E_RestoreWithinTheGracePeriodCancelsTheDELETE が同じ表明で失敗した。
  - id: admin-email-change-resets-verification
    red: メールアドレスの変更が確認済みを戻さなかったため、TestUpdateUserResetsEmailVerifiedWhenTheAddressChanges が「verified=true, want the new address unverified」で失敗した。
    fault_injection: 変異器（test-go-mutation）が `in.EmailVerified != nil` と `updated.EmailVerified != user.EmailVerified` の否定を生成し、どちらも TestUpdateUserResetsEmailVerifiedWhenTheAddressChanges が検出した。
- **Change-Resistance Results**:
  `mise run test-go-mutation -- backend/idmanagement/user/usecases` と `mise run test-go-mutation -- backend/provisioning/usecases` は、変更した行に生存変異を残さなかった。生存変異は admin_users.go、admin.go、execute_task.go、reconcile.go の既存の行にあり、この作業の範囲外として残す。
  変異器が表せない故障として、復元の通知の配線の除去と、取消を一つの接続に限る置き換えを手で注入し、上の各テストが検出することを確かめた。
- **Verification Results**:
  - `mise run check-spec` - 成功
  - `mise run check-work-items` - 成功
  - `mise run lint-go` - 成功
  - `mise run verify` - 成功
  - `mise run test-ui-e2e` - 対象外。管理画面は変えておらず、変更はブラウザーへ届く送信内容を変えない。
