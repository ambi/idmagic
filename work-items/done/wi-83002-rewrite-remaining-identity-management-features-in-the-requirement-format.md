---
status: completed
archived_from: 82daf5688f3c7d3c39648d60aa5d8f11903fe3a7
---

# IdManagement の残りの機能仕様を、EARS 形式の要件だけで書く方式へ書き直す

`mise run spec-diff -- main` の結果、規範の差分は REQ-IDMANAGEMENT-002 から 088 のうち対象の 10 機能の要件と REQ-IDMANAGEMENT-047 の本文の書き直しと、AgentLifecycle、DataExportLifecycle、DynamicMembershipEvaluationLifecycle の状態遷移の変更（状態遷移表（マトリクス形式）の追加と、自己遷移、再試行、Agent の `Deleted` の追加）だけであり、要件の追加と削除はない。
10 機能の `README.md` は 1,376 行から 1,095 行になり、各機能の全文書は 3,516 行から 3,235 行になった。三つの状態遷移表を加えた後の値である。
操作ごとの要約表、担保手段の欄、要判断の欄 11 件を消した。要約表にだけあった事実は要件へ移し、要判断の未決定の点は wi-97149 に残っている。
`tools/check/unspecified-vocabulary-debt.json` の 17 件のうち、要件に書いた 11 件を消した。残る 6 件は (c) として wi-45298 に起票した。
状態遷移表と許容リストから見つけた未記載の振る舞いを設計の表のとおりに分類し、(c) を wi-45298、wi-17379、wi-74002 として起票した。

全文は `git show 82daf5688f3c7d3c39648d60aa5d8f11903fe3a7:<このファイルのパス>` で読む。
