---
status: completed
archived_from: 0a6f958e82e03c4352fa2c1402ff5b90b3faccf4
---

# 外から非公開パッケージへの import がないモジュールを Go の internal/ へ移す

`mise run spec-diff -- 96d851960` は規範の変更なしと報告した。
対象の 11 モジュールの実装を `internal/` へ移し、実在する公開契約だけを責務表へ列挙した。
保存、HTTP と worker の構築をルートへ移し、仕様、永続状態、通知とイベントの契約を保った。
非公開 import は 26 件から 25 件へ減り、ほかの境界負債は増えていない。

全文は `git show 0a6f958e82e03c4352fa2c1402ff5b90b3faccf4:<このファイルのパス>` で読む。
