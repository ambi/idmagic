---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# sharedsignals context の admin stream API の `WriteBrowserError` 呼び出しを Problem Details へ移行する

`backend/sharedsignals/handlers_http/routes.go` の管理 API 側 11 箇所を
`support.WriteProblem` へ移行し、granular 7 code
(`ssf_stream_event_types_required`・`ssf_stream_event_type_invalid`・
`ssf_transmitter_delivery_endpoint_invalid`・`ssf_transmitter_audience_required`・
`ssf_receiver_trusted_issuer_invalid`・`ssf_receiver_jwks_required`・
`ssf_receiver_accepted_audiences_required`) の status を 400 から
仕様の宣言値 422 に揃えた。`ssf_stream_not_found` (404) と
`invalid_request` (400) は status 据え置き。
起票時の想定と違っていた点: `writeAdminSharedSignalsError` は名前に反して
`handleReceiveSecurityEvent` からも呼ばれており、そのまま Problem Details 化
すると inbound SET receiver の応答形式まで変わってしまう構造だった。
そこで receiver 用の経路を
`writeReceivedSecurityEventError`/`writeSecurityEventReceiverError` として
分離し、receiver は従来どおりの `{error, message}`・`application/json` を
返し続けるようにしてから管理 API 側だけを移行した。分離後は
`WriteBrowserError` に依存しない形 (`support.NoStoreJSON` 直呼び) にして
あるので、`wi-340` の削除で receiver の応答が巻き添えになることはない。
HTTP レベルのテストがこのパッケージに 1 つもなかったため、
`routes_test.go` を新設して両方の契約を同じファイルで固定した
(管理 API = 422 Problem Details、receiver = Problem Details にしないこと)。
仕様変更はない (`just spec-diff`: no normative specification change)。
**対応していないこと**:
- Risk Notes に記録済みの RFC 8935 §2.3 フィールド名 (`{err, description}`)
への不一致は本 work item では直していない。receiver の応答は移行前と
バイト単位で同じである。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
