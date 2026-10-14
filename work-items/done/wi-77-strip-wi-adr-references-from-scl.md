---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# scl.yaml から wi / ADR / commit 参照を除去し純粋な仕様文にする

SCL 正本から wi / ADR / commit 参照を全除去し、純粋な仕様文に整えた。本 WI
起票後に context が物理分割 (wi-31) されたため、scope の「scl.yaml」を SCL spec
全体 (spec/scl.yaml + docs/domain/*.yaml) と解釈して適用した。除去前は 6 ファイル
計 138 箇所の参照があり、除去後は 0 件。型・フィールド・制約・列挙・scenario の
意味は不変。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
