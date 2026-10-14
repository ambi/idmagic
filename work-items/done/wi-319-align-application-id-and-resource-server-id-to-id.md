---
status: completed
archived_from: 408af72425cd94dbbcb6a5ea97d8cd8a6f926598
---

# `applications.application_id` と `mcp_resource_servers.resource_server_id` を `id` に統一する

Motivation で指摘した誤った一括除外を是正し、`mcp_resource_servers.resource_server_id`
(小規模・単一 context)と `applications.application_id`(6 bounded context に波及)の
両方を `id` に統一した。SCL → Postgres schema → Go(domain/usecases/db_postgres/
db_memory/handlers_http)→ frontend の順で一貫させた。
`Application` を参照する他 entity の FK フィールド(`ApplicationAssignment.application_id`、
`AppSignInPolicy`/`ProvisioningConnection` の FK 型 identity、`OAuth2Client` /
`SamlServiceProvider` / `WsFedRelyingParty` / `WorkflowActionDef` の `application_id`)は
wi-318 の `federated_identities.provider_id` と同じ理屈で列名を維持し、参照先の型だけを
`id` に追従させた。
作業中に Design 段階では想定していなかった追加スコープが判明した: `provisioning.yaml` の
admin interface 群(`RegisterProvisioningConnection` 等 11 本)も `application_id` を
`Application` 自身の identity として path param に持っており、`ProvisioningConnection` の
FK 型 identity(維持)とは別に `id` へ統一する対象だった。同様に `GetApplicationIcon` は
wi-318 で既に内側の `object_key` を `id` 化していたため、外側の `application_id` を
`id` にすると同一パスに `{id}/{id}` の重複が生じる。これは唯一 `id` 化を見送った箇所で、
`application_id` のまま維持した。
test-first からの逸脱 (self-attest): 本 WI も wi-318 と同様、新規振る舞いを追加しない
列名/フィールド名のリネームであり、新しい RED テストは書いていない。リネーム後にコンパイル
エラー・JSON キー不一致となった既存テストを追従させ、`just verify` の green を確認した。

全文は `git show 408af72425cd94dbbcb6a5ea97d8cd8a6f926598:<このファイルのパス>` で読む。
