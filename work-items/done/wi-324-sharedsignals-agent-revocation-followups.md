---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# SharedSignals エージェント失効の実装中に保留した項目 (Hard Quota・新規 token 発行停止・認証経路統一) を解消する

wi-58 が意図的に先送りした 4 項目のうち、Out of Scope の `ManagementApiClient` を除く 4 つを解消した。
normative scenario は 4 つ増えた (`just spec-diff`: REQ-OAUTH2-046 / REQ-OAUTH2-047 /
REQ-SHAREDSIGNALS-009 / REQ-SHAREDSIGNALS-010)。失われた、あるいは意味が変わった scenario は無い。
- **所有者オフボードが新規発行に効くようになった** (REQ-OAUTH2-046)。これまで所有者を無効化・削除しても
`Agent.Status` は変わらず、配下 Agent は `client_credentials` で新しい token を取り続けられた。
所有者の有効性を発行のたびに解決する形にし、client_credentials と CIBA の両経路を通した。
Agent の `status` は書き換えないので、所有者が復帰すれば発行も自動的に再開する。
oauth2 の Design (Agent principals) にもこの規則を durable な設計として書いた。
- **失効判定を迂回する access token 検証経路が無くなった** (REQ-OAUTH2-047)。denylist と
revocation epoch の判定を `AccessTokenIsRevoked` として切り出し、`/introspect` と
admin/account portal の Bearer 認証が同じ 1 つの規則を通るようにした。
- **`ssf_streams` が Hard Quota resource になった** (REQ-SHAREDSIGNALS-009)。specification が
宣言済みだった `QuotaExceededError` に実装が追いつき、宣言と実装の乖離が解消した。
送信側と受信側で 1 つの上限 (既定 20) を共有し、削除で利用量が戻る。
- **外部 transmitter の SET を受理できるようになった** (REQ-SHAREDSIGNALS-010)。RFC 9493 の
`iss_sub` と `opaque` を解釈する (標準要件 RFC9493-SUBID-FORMAT / RFC9493-SUBID-ISS-SUB を新設)。
`email` は User 側の失効を持たない現状では反映先が無いため対象外とし、理由を `## Design` に残した。
- 契約の差分は `TenantQuota` / `TenantUsage` / `TenantQuotaUpdateRequest` への `ssf_streams` 追加のみで、
既存フィールドの削除・型変更・required 化は無い。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
