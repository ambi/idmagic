# Audit

複数の Bounded Context が発行するセキュリティ監査イベントを、横断して読むための Read Model にまとめる。監査イベントの検索とエクスポートを行う管理 API、検索属性の定義、個人識別情報の変換方針、保持期間もここで定める。イベントの発行は各 Context の責務とし、`DomainEvent` として蓄積した記録を管理者向けの `AdminAuditEventResponse` として公開する。

| 文書 | 内容 |
|---|---|
| [Audit の用語集](glossary.md) | この Context での語義 |
| [Audit の設計判断](decisions.md) | 設計判断 |
| [Audit の内部設計](internals.md) | 機構の説明 |
| [Audit のシナリオ](scenarios.feature.md) | 受け入れシナリオ |
