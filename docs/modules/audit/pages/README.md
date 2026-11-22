# Audit のページ

この文書は、Audit が主担当となるページの設計を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[Audit のモジュール設計](../README.md) / [UI 全体の索引](../../../design/application/pages/README.md)

## 監査

| ページ | URL | 目的 |
| --- | --- | --- |
| [監査イベント](admin-audit-events.md) | `/admin/audit_events` | テナントの操作と関係者を検索して確認する |
| [システム監査イベント](system-audit-events.md) | `/system/audit-events` | テナントを横断した操作と結果を調べる |
