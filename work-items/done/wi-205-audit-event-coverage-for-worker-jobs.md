---
status: completed
archived_from: 32e41c13f4737800c9992eb48322be3e90b449ef
---

# idmagic-worker から実行される業務操作を監査イベントに漏れなく記録する

`bootstrap.Dependencies.NewEmitFunc` として `EventSink.Emit` + `AuditEventRepo.Append`
の共有ヘルパーを実装し、`cmd/idmagic/server.go` の重複した `emit` クロージャをこれに
置き換えた。`cmd/idmagic-worker/worker.go` は `newAdminUserDeps` ヘルパーを新設し、
`AdminUserDeps.Emit` を同じ共有ヘルパー経由で配線した (`legacyEmit` と同じ
fire-and-forget adaptation)。`spec/contexts/audit.yaml` に
`DomainEventsAreAuditedRegardlessOfProcess` 不変条件を追加した。

全文は `git show 32e41c13f4737800c9992eb48322be3e90b449ef:<このファイルのパス>` で読む。
