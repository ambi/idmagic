---
status: completed
archived_from: b701beaa702c675cffa1dc6a57712736a7a17d5f
---

# MCP resource indicator を device_code グラントへ拡張し RFC8707 を完全準拠にする

RFC 8707 resource indicator の audience 限定を device_code グラント
(`ExchangeDeviceCode`, RFC 8628 §3.4 の token polling) へ拡張し、
`RFC8707-MCP-RESOURCE-BINDING` requirement を `adoption: partial` から
`adoption: required` へ格上げした。これにより Authorize / PushAuthorizationRequest /
Token(authorization_code redemption・refresh rotation・client_credentials・
device_code・token-exchange) の全経路で resource パラメータ指定時の audience 厳格
限定が一様に適用され、RFC 8707 は完全準拠になった。実装は client_credentials
(wi-262) と同型のパターンを踏襲し、`/device_authorization` (初期リクエスト) 自体は
無変更のまま、token polling リクエストでのみ resource を受理する設計とした
(RFC 8707 §2 が token request での指定を正規の使い方として認めているため)。
refresh token rotation は wi-262 で実装済みの resource 保持機構をそのまま再利用し、
device_code 発行の refresh token もローテーション後に audience 限定を保持する。
全層 test-first (RED 確認 → GREEN) で実装した。

全文は `git show b701beaa702c675cffa1dc6a57712736a7a17d5f:<このファイルのパス>` で読む。
