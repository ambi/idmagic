---
status: completed
archived_from: b3a5efb24199796755b7b1cd1d3235a759f6c4b6
---

# OIDC authorization request の重複パラメータと prompt を仕様どおりに fail-closed で処理する

OIDC prompt を一意な空白区切り token 集合として解析し、重複・未知 token・`none` との併用を
`invalid_request` とした。`/authorize` と `/par` は security-sensitive parameter の重複を raw
input のまま拒否し、PAR の `request_uri` と他の authorization parameter の混在を拒否する。
`prompt=none` のログイン／同意不能時は UI を開始せず、登録済み redirect URI へ state と RFC 9207
`iss` を保った error を返す。

全文は `git show b3a5efb24199796755b7b1cd1d3235a759f6c4b6:<このファイルのパス>` で読む。
