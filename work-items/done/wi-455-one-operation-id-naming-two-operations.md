---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# 1 つの `operationId` が 2 つ以上の operation を名乗っている状態を解く

`mise run spec-diff` は `OPENAPI31-OPERATION-ID` の追加を報告した。SAML の既定/名前付きプロファイルと GET/POST binding、SAML/OIDC callback、GET/POST EndSession に経路固有の `operationId` を与え、336 operation を一意にした。契約生成器は重複を黙って縮約せず両経路を示して拒否する。provisioning と sharedsignals の配送一覧 handler 名も分離し、status drift の曖昧さを解消した。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
