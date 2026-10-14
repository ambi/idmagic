---
status: completed
---

# 作業項目の完了を、完了記録を書かずに要約へ縮めて移す手順にする

`mise run spec-diff` に規範の差分はない。
完了は `## 要約` を書いて `mise run move-work-item -- <id>`（中止は `--cancel`）を実行するだけになった。
コマンドは全文に完了時の検査（`affected_spec` の解決、`primary_use_cases` のテストの実在、宣言した告知文書）を適用し、状態、題名、要約だけに縮めて `work-items/done/` へ移す。
縮めた記録は `archived_from` を持たず、仕様影響の検査は全文を `work-items/active/` のパスの履歴から読む。
完了節の証拠、`evidence_policy`、`initial_context`、公開前の `documentation_impact`、`brief` の下書き一覧、`archive-work-items` をなくした。
RED と故障注入は手順として残し、記録しない。
差分は 37 ファイルで 343 行の追加と 1,665 行の削除で、該当する開発文書 6 本は 173,386 バイトから 166,438 バイトになった。
wi-82135 は新しい手順で中止した。

全文は `git log -1 -- work-items/active/wi-38309-complete-work-items-by-collapsing-instead-of-recording.md` が示すコミットの親、またはそのコミットで読む。
