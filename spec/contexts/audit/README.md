# Audit

複数の Bounded Context が発行するセキュリティ監査イベントを横断して読むための Read Model を所有する。監査イベントの検索とエクスポートを行う管理 API、検索属性の定義、個人識別情報の変換方針、保持期間もここで定める。イベントの発行は各 Context の責務とし、`DomainEvent` として蓄積した記録を管理者向けの `AdminAuditEventResponse` として公開する。

| File | Content |
|---|---|
| [glossary.md](glossary.md) | この Context での語義 |
| [decisions.md](decisions.md) | 設計判断 |
| [internals.md](internals.md) | 機構の説明 |
| [scenarios.md](scenarios.md) | 受け入れシナリオ |
