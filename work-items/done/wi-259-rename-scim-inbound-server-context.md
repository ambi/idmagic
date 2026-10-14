---
status: completed
archived_from: 18a20e15a54838c336457fc95eaa5fcd96d7aa6e
---

# 既存 `Scim` context を inbound-honest な名前 / 構造へ rename・再配置する

[[ADR-141-inbound-identity-sourcing-taxonomy]] 決定 2・4 の目標構造へ SCIM inbound
server を移設した。SCL は `spec/contexts/scim.yaml` → `spec/contexts/sourcing.yaml`
(`context: Sourcing`) へ移し、wi-258 が置いた taxonomy の glossary と統合、context_map の `Scim`
エントリを削除して `Sourcing` に一本化した。Go は `backend/scim/{domain,ports,usecases,handlers_http,
db_memory,db_postgres}` を `backend/sourcing/scim/` 配下へ `git mv` し、`module.go` は context ルート
`backend/sourcing/module.go` (`package sourcing`) に据えた。振る舞い・wire 契約・DB スキーマは不変
(SCIM endpoint パス、`scim:*` scope、`scim_user_refs` / `scim_group_refs` テーブルは変更なし)。

全文は `git show 18a20e15a54838c336457fc95eaa5fcd96d7aa6e:<このファイルのパス>` で読む。
