---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# 起動時の設定を集約・検証し fail-fast させ、単一の設定リファレンスを生成する

REQ-SYSTEM-016 を idmagic だけから全 backend プロセスへ拡張し、環境由来設定を
bootstrap の型付き ConfigLoader に集約した。未知・型不正・範囲外・条件付き必須の
問題は listener、依存接続、seed 適用より前に一括報告し、secret は formatting/JSON/
生成ドキュメントで redaction する。REQ-SYSTEM-017 を追加し、process 固有 default と
条件付き必須を含む `CONFIGURATION.md` を Config metadata から生成して freshness を
リポジトリ検証へ組み込んだ。`just spec-diff` の意味差分は REQ-SYSTEM-017 の追加と
REQ-SYSTEM-016 の変更。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
