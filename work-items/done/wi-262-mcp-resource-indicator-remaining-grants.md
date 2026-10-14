---
status: completed
archived_from: e51192cb3ab459b5cb58d2c1b8e65eed91d2096b
---

# MCP resource indicator を client_credentials / refresh rotation へ拡張する

RFC 8707 resource indicator の audience 限定を client_credentials グラントと
authorization_code の refresh token rotation へ拡張した。SCL の
`RFC8707-MCP-RESOURCE-BINDING` requirement を更新し、対象外は device_code のみに
縮小した（wi-56 時点では client_credentials・refresh rotation・device_code の
3 経路が対象外だった）。`token_handler.go` の client_credentials 分岐に
`usecases.ResolveResourceIndicator` を配線し、未登録・Disabled な resource は
fail-closed で拒否、登録済み Active な resource は access token の aud をそこに
限定する。`domain.RefreshTokenRecord` に `resource` フィールドを追加し、
`GenerateInitialRefreshToken`/`RotateRefreshToken` で保持・引き継ぎ、
`refresh_tokens.go` のローテーション時に `Audiences` として `SignAccessToken` へ
渡すことで、authorization_code + offline_access + resource で取得した refresh
token をローテーションしても audience 限定が失われないようにした。Postgres
`refresh_tokens` テーブルに `resource TEXT` (nullable) 列を追加し sqlc を再生成した。
全層 test-first (RED 確認 → GREEN) で実装した。

全文は `git show e51192cb3ab459b5cb58d2c1b8e65eed91d2096b:<このファイルのパス>` で読む。
