---
status: completed
archived_from: 9f147844390880d989d14a3d018b5a82665f3f67
---

# SCL 規範要素・Architecture・実装・検証証跡を直接結ぶ追跡グラフを構築する

`verification/manifest.yaml` と分離した execution evidence を正本に、SCL 規範要素、
Architecture module、宣言済み check / `just` recipe、revision 付き実行結果を直接結ぶ
四層 workspace traceability graph を実装した。selector ごとの coverage、未知 target、
孤立・重複 binding、欠落 / stale evidence、期限切れ baseline を機械可読 report で
分類し、report-only と strict gate を `just` に接続した。Work Item に direct
`affected_spec`、`change_kind`、`initial_context`、理由付き `spec_impact: none` を追加し、
旧 Go `AssuranceManifest` は workspace manifest へ移行した。

全文は `git show 9f147844390880d989d14a3d018b5a82665f3f67:<このファイルのパス>` で読む。
