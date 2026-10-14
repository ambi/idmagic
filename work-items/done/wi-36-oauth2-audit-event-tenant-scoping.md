---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# oauth2 / token / consent / client 系の監査イベントにも emit 時点で tenant_id を載せる

[[wi-35-audit-event-tenant-scoping]] の残課題だった OAuth2 component の
18 イベントに tenantId を追加し、emit 時点で所属テナントを載せた。これで
クライアント登録・同意付与/失効・認可コード・トークン発行/失効/再利用検出・
PAR・デバイスフローの各イベントが、テナント所属 admin の
`/admin/audit_events` に出るようになった。wi-35 と同じく newAuditEventRecord
と監査フィルタは一切変更していない (既存の payload["tenantId"] 抽出が効く
ようになっただけ)。
SigningKeyRotated は意図的に対象外。現状の署名鍵はインスタンス全体で
1 系統のため帰属テナントが定義できず、tenant_id 空のまま残る。per-tenant
鍵を入れる [[wi-32-kms-hsm-and-per-tenant-signing-keys]] で対応する。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
