---
status: completed
archived_from: 213ad69cecb2c774c9e14d8a8405da698690aa43
---

# 残りの文書の Context をモジュールへ改名し、用語の検査で固定する

`mise run spec-diff` が main に対して示した規範の差分は、規範シナリオ 12 件（REQ-APPLICATION-014、REQ-AUTHORIZATION-010、REQ-CLAIMMAPPING-001、REQ-DATAKEYS-005、REQ-IDMANAGEMENT-012、REQ-IDMANAGEMENT-089、REQ-JOBS-002、REQ-PLATFORM-001〜005）と TypeSpec の宣言 51 件の文面である。
どれも「Context」「公表された言語」を「モジュール」「公開契約」へ差し替えただけで、主体、条件、応答、状態、イベントの意味は変えていない。
文書、仕様の散文、TypeSpec の説明、Go と TypeScript のコメントの旧い語を改名し、各モジュールの判断の文書から廃止した区分の節 21 個を削除した。
`SPECIFICATION_FORMAT.md` の区分を付ける規則は、一覧を責務表に一度だけ書く規則へ書き換えた。
`mise run check-terminology` は、設計の単位としての「Context」と「公開言語」「Published Language」を拒否する。
通すのは `context.Context`、C4 の System Context と、文面を変えなかった標準の 3 行の語である。

全文は `git show 213ad69cecb2c774c9e14d8a8405da698690aa43:<このファイルのパス>` で読む。
