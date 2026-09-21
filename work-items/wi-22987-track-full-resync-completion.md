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
  - { path: docs/domain/provisioning/scenarios.feature.md, requirement: REQ-PROVISIONING-013 }
---

# Full Resync の配信完了を追跡し、全対象が収束したときに完了イベントを発行する

## 動機

`EX-PROVISIONING-013-01` は、scope 内の全 subject に配信を作成し、すべてが収束したとき `FullResyncCompleted` を発行すると宣言する。
現行の `StartFullResync` は配信を作るだけで、非同期配信群を同じ resync として追跡せず、実装コメントでもイベント未実装を明記している。

## 対象範囲

- Full Resync ごとの対象配信、成功、失敗、完了を追跡する状態を実装する。
- すべての対象配信が終端状態になった時点で `FullResyncCompleted` を発行する。
- `REQ-PROVISIONING-013` を参照する単体テストと、実配線での最終イベントを観測する受け入れテストを追加する。

## 対象外

- Full Resync の対象選択規則の変更。
- 外部 SCIM の同期アルゴリズムや差分計算の追加。
- 他の Provisioning イベントの発行。

## 設計

非同期配信の一括処理には、個々の `ProvisioningDelivery` だけでは表せない親の追跡単位が必要である。
親状態は開始時の対象集合と終端結果を持ち、ジョブハンドラーが配信を終端化した後に更新する。
完了判定とイベント発行は同じ永続化境界で一度だけ成立させる。

## 計画

1. 配信・ジョブ・イベントの既存永続化境界を調査する。
2. 全対象の登録と、最後の成功または失敗が完了を確定する RED を作る。
3. 親状態、配信への関連付け、終端時の集計とイベント発行を実装する。
4. 結合経路と変異を検証する。

## タスク

- [ ] T001 [Readiness] resync 親状態の所有者とトランザクション境界を決める。
- [ ] T002 [Acceptance] `EX-PROVISIONING-013-01` の E2E RED を確認する。
- [ ] T003 [Domain] 全対象の終端判定と一度だけの完了発行を単体 RED から実装する。
- [ ] T004 [Use Cases] 開始・配信処理・イベント発行を接続する。
- [ ] T005 [Verify] 変異、対象パッケージ、`mise run verify` を実行する。

## 検証

- `mise run test-go-package -- ./backend/provisioning/usecases`
- `mise run test-go-package -- ./backend/provisioning`
- `mise run test-go-mutation -- backend/provisioning/usecases`
- `mise run verify`

## リスク

早すぎる完了イベントは未配信の subject を成功と誤認させ、重複発行は監査と運用の判断を壊す。
複数配信・失敗・再試行を含む状態遷移を、イベントの回数まで含めて検証する。
