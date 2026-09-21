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
  - { path: docs/domain/provisioning/scenarios.feature.md, requirement: REQ-PROVISIONING-002 }
  - { path: docs/domain/provisioning/scenarios.feature.md, requirement: REQ-PROVISIONING-003 }
  - { path: docs/domain/provisioning/scenarios.feature.md, requirement: REQ-PROVISIONING-004 }
  - { path: docs/domain/provisioning/scenarios.feature.md, requirement: REQ-PROVISIONING-005 }
  - { path: docs/domain/provisioning/scenarios.feature.md, requirement: REQ-PROVISIONING-007 }
  - { path: docs/domain/provisioning/scenarios.feature.md, requirement: REQ-PROVISIONING-008 }
  - { path: docs/domain/provisioning/scenarios.feature.md, requirement: REQ-PROVISIONING-009 }
  - { path: docs/domain/provisioning/scenarios.feature.md, requirement: REQ-PROVISIONING-010 }
  - { path: docs/domain/provisioning/scenarios.feature.md, requirement: REQ-PROVISIONING-011 }
  - { path: docs/domain/provisioning/scenarios.feature.md, requirement: REQ-PROVISIONING-014 }
  - { path: docs/domain/provisioning/scenarios.feature.md, requirement: REQ-PROVISIONING-017 }
  - { path: docs/domain/provisioning/scenarios.feature.md, requirement: REQ-PROVISIONING-018 }
---

# 宣言済みの Provisioning ライフサイクルイベントを状態遷移とともに発行する

## 動機

Provisioning のシナリオは接続登録、配信開始、成功、失敗、隔離、隔離解除、資格情報ローテーションのイベントを観測結果として宣言する。
イベント型は `domain/events.go` に存在するが、本番のユースケース、ハンドラー、アダプターはそれらを発行していない。
型の存在だけでは監査や購読側は状態遷移を観測できない。

## 対象範囲

- 規範が宣言する Provisioning ライフサイクルイベントを、対応する状態変更と同じ作用境界で発行する。
- 発行順序、重複抑止、失敗時の状態との整合を定める。
- 接続管理、配信処理、ジョブ処理の各境界でイベントペイロードを検証する。
- 代表的な成功・失敗・隔離・資格情報ローテーションの受け入れテストを追加する。

## 対象外

- `FullResyncCompleted`。これは resync 追跡を要するため [[wi-22987-track-full-resync-completion]] が扱う。
- 新しいイベント種類や外部イベントブローカー製品の導入。
- シナリオに書かれていないイベントの公開。

## 設計

イベントは状態遷移の副産物ではなく、外部から観測できる結果である。
各ユースケースが任意の通知コールバックを直接持つ設計は、配線の欠落と二重発行を増やす。
既存のドメインイベントとトランザクション境界を調べ、状態保存と同じ成功条件に結び付く単一の発行ポートを置く。

配信の成功・失敗は下流呼び出しの結果に依存するため、job handler が終端状態を保存した後に発行する。
隔離と解除は接続状態を読み戻せるようにしたうえで、それぞれ一度だけ発行する。

## 計画

1. 既存のイベント発行・永続化基盤と Provisioning の全状態遷移を対応付ける。
2. 代表的なイベントの Acceptance RED と Unit RED を確認する。
3. 発行ポートと配線を実装し、各規範イベントへ広げる。
4. 発行を削除・状態保存の前後へ移動する故障を注入して検出能力を確認する。

## タスク

- [ ] T001 [Readiness] 既存イベント基盤と Provisioning の状態遷移を対応付ける。
- [ ] T002 [Acceptance] 登録、配信成功・失敗、隔離、資格情報ローテーションのイベント RED を確認する。
- [ ] T003 [Domain] イベントペイロードと一度だけの発行条件を単体 RED から実装する。
- [ ] T004 [Use Cases] 接続管理、配信、ジョブ処理を発行ポートへ接続する。
- [ ] T005 [Verify] 故障注入、変異、対象パッケージ、`mise run verify` を実行する。

## 検証

- `mise run test-go-package -- ./backend/provisioning/usecases`
- `mise run test-go-package -- ./backend/provisioning/handlers_http`
- `mise run test-go-package -- ./backend/provisioning`
- `mise run test-go-mutation -- backend/provisioning/usecases`
- `mise run verify`

## リスク

状態が変わったのにイベントが無い、またはイベントだけが残ると、監査と再試行処理が実際の状態を誤認する。
状態読み戻しと発行記録を対にして検証し、発行の削除・二重化・順序変更を故障として確認する。
