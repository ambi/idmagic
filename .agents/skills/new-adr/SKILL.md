---
name: new-adr
description: Legacy compatibility skill. Do not create a new ADR; redirect current design to ARCHITECTURE.md and change-specific decisions and alternatives to the work item.
---

# ADR は新規作成しない

`ADR_FORMAT.md` の archive policy に従う。新しい判断を記録する依頼では、現在も有効な設計と短い根拠を `ARCHITECTURE.md`、変更固有の alternatives・risk・履歴を work item に書く。既存 `decisions/` は read-only history として保持する。

ユーザーが明示的に ADR ファイル作成を求めた場合も、現行方針との衝突を説明して配置先を確認するまで新規ファイルを作らない。
