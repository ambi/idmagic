---
status: completed
archived_from: 4d622cdaf97c6bebaa7a1e0ce14c485365a4461b
---

# SCL 3.0 の規範要素を検証対象として安定参照可能にする

SCL 3.0 の規範要素を `context`、`kind`、所有要素名で指す閉じた構造化参照と、standard
requirement 用の `standard` + `requirement` 参照を言語仕様へ追加した。workspace context map と
context spec を検査・index 化する共通 resolver、未知 context/kind/element/requirement・context
不一致・重複 requirement ID・位置ベース field を拒否する fixture/test を実装した。HTML renderer
は同じ参照 tuple から canonical 表示と衝突しない anchor を生成し、authorization 3種の同名要素と
standard 間の同一 requirement ID を区別する。SCL YAML shape、`spec_version: "3.0"`、既存19件の
SCL YAML は変更していない。

全文は `git show 4d622cdaf97c6bebaa7a1e0ce14c485365a4461b:<このファイルのパス>` で読む。
