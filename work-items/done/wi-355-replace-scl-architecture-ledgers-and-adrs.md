---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# 専用SCL・Architecture台帳・新規ADRを標準契約とMarkdown仕様へ置き換える

専用 SCL YAML を context-local TypeSpec と requirements Markdown に置換し、実行時 YAML loader と
独自 renderer/generator を撤去した。状態遷移は requirements の言語非依存表、API 契約は TypeSpec、
認可実行規則は Go を正本とした。architecture.yaml と全量 ledger/traceability 検査を削除し、path/import
から導く禁止境界検査へ置換した。既存 ADR は履歴として保持し、新規判断は current-state docs と work item
に記録する方針へ変更した。Cedar 評価は wi-354、RA という名称自体の再評価は wi-356 に分離した。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
