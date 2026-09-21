---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-21
priority: p2
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/domain/provisioning/scenarios.feature.md, requirement: REQ-PROVISIONING-006 }
---

# 猶予期間を持つ User 削除の下流配信を延期し、再割り当て時に取り消す

## 動機

`EX-PROVISIONING-006-01` と `EX-PROVISIONING-006-02` は、削除後 7 日間は delete を配信せず、猶予期間中の再割り当てで予約済みの配信を取り消すことを宣言する。
現行の `CaptureLifecycleEvent` は `TriggerUserDeleted` をただちに delete の `ProvisioningDelivery` へ変換するため、この規範に一致しない。

## 対象範囲

- `REQ-PROVISIONING-006` の猶予期間中の非送出、経過後の delete 配信、再割り当てによる取消を実装する。
- 配信予定と取消を永続化し、worker が期限到来を処理できる境界を設ける。
- 実時間を直接読むのではなく、期限判定へ明示した時刻を渡す。
- 規範 ID を参照する単体テストと、下流への DELETE を観測する受け入れテストを追加する。

## 対象外

- `on_delete=deactivate` と `on_delete=none` の既存意味の変更。
- User 以外の猶予期間。
- 既存の下流 SCIM クライアントのプロトコル変更。

## 設計

猶予期間は即時の `ProvisioningDelivery` を遅延実行する問題ではなく、再割り当てで取り消せる予約状態を持つ問題である。
予約を通常の pending 配信へ偽装すると、worker が期限前に送出した場合と取消対象を区別できない。
期限、接続、User、元イベントを型付きの予約として保存し、期限到来時にだけ配信を生成する。

再割り当ては同じ User と Application の有効な予約だけを取り消す。
異なる接続や別バージョンのイベントを取り消さない。

## 計画

1. `REQ-PROVISIONING-006` の既存捕捉経路とジョブ永続化を読んで予約の所有境界を決める。
2. 受け入れ RED と単体 RED を、期限前、期限後、再割り当ての順に確認する。
3. 予約の保存、期限処理、取消を実装し、下流への効果を検証する。
4. 変更した Go パッケージの変異結果を確認して完了する。

## タスク

- [ ] T001 [Readiness] 予約状態と期限処理の所有境界を決める。
- [ ] T002 [Acceptance] `EX-PROVISIONING-006-01` と `-02` の E2E RED を確認する。
- [ ] T003 [Domain] 猶予予約と取消の単体 RED を GREEN にする。
- [ ] T004 [Use Cases] 削除捕捉、再割り当て、期限到来を接続する。
- [ ] T005 [Verify] 変異、対象パッケージ、`mise run verify` を実行する。

## 検証

- `mise run test-go-package -- ./backend/provisioning/usecases`
- `mise run test-go-package -- ./backend/provisioning`
- `mise run test-go-mutation -- backend/provisioning/usecases`
- `mise run verify`

## リスク

期限前の送出と取消漏れは、外部システムでの誤削除を招く。
期限の直前・直後と再割り当ての競合を、予約の状態遷移と明示時刻を使うテストで固定する。
