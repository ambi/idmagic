---
status: accepted
authors: [tn]
created_at: 2026-07-11
---

# ADR-095: transaction-bound event を共通 command envelope に集約する

## コンテキスト

ADR-094 は業務状態と event log を同一 transaction で確定することを決めたが、最初の移行では
context ごとの HTTP handler が transaction runner、recorder、correlation ID、bridging emit を
個別に組み立てる形になった。この形では共通の失敗処理や legacy bridge の変更が各 context に波及し、
mutation を追加するたびに同じ配線を再実装する危険がある。

## 決定

`backend/shared/eventlog.CommandRunner` を、業務 command と transaction-bound emitter の唯一の
実行境界にする。adapter は `CommandRunner.Run` へ業務操作だけを渡し、受け取る
`eventlog.Command` の `Context` と `Emit` を use case dependency に注入する。共通 envelope が
transaction、correlation ID、event log append、legacy audit/outbox bridge、エラー伝播を所有する。

## 却下した代替案

- 各 handler が `TxRunner.Run` と `NewBridgingEmit` を直接呼ぶ: 配線と失敗処理の変更点が全 context に散る。
- HTTP request 全体を middleware transaction にする: ADR-094 の短い command transaction 方針に反し、外部 I/O を含む危険がある。
- event emit 失敗を fire-and-forget のままにする: `EventLogAtomicWithBusinessState` を満たせない。

## 影響

- SCL の既存 invariant `spec/contexts/system.yaml` の `EventLogAtomicWithBusinessState` を実装で一貫して担保する。
- 今後の mutation は `CommandRunner` を利用し、transaction 配線を再実装しない。
- `backend/shared/eventlog` は ADR-094 の technical shared capability として command envelope も所有する。新しい bounded context は追加しない。
