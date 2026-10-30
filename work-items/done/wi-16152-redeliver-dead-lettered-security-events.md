---
status: completed
---

# 配信不能になった SET の配送をやり直せるようにする

管理者がストリームの `dead_letter` の配送をすべて `pending` に戻す `RedeliverDeadLetteredDeliveries` を追加した（REQ-SHAREDSIGNALS-013）。
`SecurityEventDeliveryLifecycle` に `dead_letter` → `pending` の遷移を加え、`dead_letter` は終端でなくなった。REQ-SHAREDSIGNALS-006 と REQ-SHAREDSIGNALS-011 もこれに合わせて改めた。
追加した TypeSpec は `SecurityEventDeliveryRedeliveryRequested`、`SsfStreamDisabledError`、`SsfStreamNotFoundError` である。

起票時の案は Provisioning にそろえて新しいレコードを作ることだったが、Provisioning は実際には同じタスクを戻していたので、同じ行を戻す方式を選んだ。
SET を保つので `jti` と `event_timestamp` は変わらず、RFC 8935 の 2 節が認める再送になり、受信側は重複を見分けられる。
無効化したストリームは 409 `ssf_stream_disabled` で拒否し、対象がなければ 200 と件数 0 を返す。

テストは実装と同時に書いたので、RED は完成後に経路、試行の回数の初期化、SQL の状態条件を一時的に壊して観測した。

残るリスクは次のとおりである。
保存した SET は再署名しないので、署名鍵が JWKS から外れた後（ローテーションの 7 日後）のやり直しは受信側で検証できず、再び `dead_letter` になる（`risks.md`）。
無効化したストリームの、すでに `pending` または `failed` の配送を `worker` が送る既存の振る舞いは変えていない。
SSF の最終版は末尾の 14,556 字を確認していない。
