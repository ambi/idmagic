---
status: completed
archived_from: 5ce8ffc3bfb1dc24ab0dd8b489e483eccd45698f
---

# 監査イベントを OAuth2 から独立した audit bounded context へ切り出す

監査イベントの読み出し API (`ListAdminAuditEvents` / `GetAdminAuditEvent` /
`ExportAdminAuditEvents`)、検索属性 registry、PII 変換、`AdminAuditEventsRead`
permission、保持期間 objective を OAuth2 context から独立した `Audit` bounded context
(`spec/contexts/audit.yaml`) へ移設した。Go 側は `backend/audit/{ports,usecases,
adapters/http,adapters/persistence/{memory,postgres}}` を新設し、`backend/oauth2`・
`backend/authentication`・`backend/bootstrap`・中央 HTTP router の import 参照を
audit port へ切り替えた。`TenantSaltStore` port も、監査検索の PII 相関ハッシュに
本質的な役割を持つため audit context へ移した (OAuth2 の emit 経路は
`auditports.TenantSaltStore` を参照する側になる)。`ARCHITECTURE.md` の Context Map
に `Audit` 行を追加し、`sqlc.yaml` / postgres の queries・sqlcgen も audit 側へ分離した。
HTTP route (`/api/admin/audit_events` 系)・JSON レスポンス形状・検索意味論・認可判定は
変更していない。UI 側には SCL bounded context を示す明示的な参照が無く、既存 URL も
変えていないため追加の改修は不要だった。副次的に、`just yaml-check` に抜けていた
`--architecture` チェックを `justfile` の `yaml-check` composite recipe へ組み込んだ。

全文は `git show 5ce8ffc3bfb1dc24ab0dd8b489e483eccd45698f:<このファイルのパス>` で読む。
