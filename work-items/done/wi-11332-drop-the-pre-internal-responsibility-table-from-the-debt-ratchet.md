---
status: completed
---

# 負債の比較検査から、移行前の責務表を読む互換を外す

負債の比較検査は、基準 revision の責務表も作業ツリーと同じ規則で読むようになった。
公開方式 `legacy`、旧形式の責務表、データベース設計の「所有 Context」の見出しを読む互換と、`legacy` から `internal` へ切り替えたモジュールの利用元の読み替えを外した。
基準 revision がこれらの形を持てば、比較検査は解析の失敗を報告する。
バックエンド設計の公開範囲の節から、移行前の責務表を読む規則を外した。
`spec-diff` に規範の差分はない。

全文は `git log -1 -- work-items/active/wi-11332-drop-the-pre-internal-responsibility-table-from-the-debt-ratchet.md` が示すコミットの親、またはそのコミットで読む。
