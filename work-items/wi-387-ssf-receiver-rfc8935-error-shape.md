---
status: pending
authors: [tn]
risk: low
created_at: 2026-08-19
change_kind: bugfix
priority: p3
depends_on: []
affected_spec:
  - { path: spec/contexts/sharedsignals/models.tsp, symbol: IdMagic.Contract.SecurityEventRejectedError }
---

# SharedSignals 受信エンドポイントのエラー本体を RFC 8935 §2.3 の形にする

## Motivation

`POST /ssf/streams/{stream_id}/events` の拒否応答は `writeSecurityEventReceiverError` が `{"error": code, "message": description}` を返す。RFC 8935 §2.3 が定めるのは `{"err": ..., "description": ...}` である。この接点は「標準が形を定めるから Problem Details を適用しない」という理由で例外扱いにしているのに、その標準の形になっていない。

`wi-335` の Risk Notes がこれを記録し、`wi-382` は「契約が実装に合わせる」という原則の外にあるとして Out of Scope に置いたうえで、実装がいま返している `{error, message}` のほうを契約に書いた。契約を先に正しい形にすると、どちらが正なのか読めなくなるためである。

## Scope

- `writeSecurityEventReceiverError` の出力を `{"err": ..., "description": ...}` に変える。
- `SecurityEventRejectedError` / `SecurityEventTokenTooLargeError` / `SecurityEventStreamNotFoundError` の 3 モデルを新しい形に合わせ、`@doc` から「RFC に追従できていない」という注記を外す。
- 受信側の適合を確認する試験を足す。

## Out of Scope

- 送信側 (transmitter) の SET 形式。RFC 8417 に従っており変更しない。
- 管理 API 側のエラー本体。Problem Details のままとする。

## Verification

- `mise run verify`
- 手動確認: 不正な SET を送ると `{"err": ..., "description": ...}` が返る。
- 手動確認: 生成された OpenAPI の 400 / 404 / 413 が同じ形を記述している。

## Risk Notes

`{error, message}` を読んでいる外部 transmitter があれば壊れる。この接点は外部から呼ばれるため、変更前に配備先で実際に受信している相手を確認する。相手が居ない配備では単純な置換で済む。
