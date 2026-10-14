---
status: completed
archived_from: 57db91ef1f545849b9ccfb1b9e0041c29a3dc3f0
---

# wsfederation コンテキストへバックエンド・コンテキストローカリティを横展開する

WS-Federation 固有の relying party・token type・federation event を
`backend/wsfederation/domain/` へ移設し、ClaimMapping 型だけを `backend/shared/spec` に残置した。
memory / PostgreSQL repository を context 配下へ同居させ、PostgreSQL は 4 本の静的 sqlc
クエリへ置換した。`WsFederation.Module` が RP repository と route 登録を所有し、中央 Deps と
bootstrap は Module だけを配線する形にした。ARCHITECTURE.md と ADR-090 の実測も同期した。

全文は `git show 57db91ef1f545849b9ccfb1b9e0041c29a3dc3f0:<このファイルのパス>` で読む。
