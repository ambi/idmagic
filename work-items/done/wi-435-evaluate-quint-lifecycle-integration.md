---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# Quint を仕様、設計、開発、テストの検証経路へ組み込めるか実証する

`mise run spec-diff main` は `no normative specification change against main` を返す。規範シナリオ、状態表、遷移表、標準の行、TypeSpec シンボルのいずれも変わっていない。宣言した `spec_impact: none` のとおりである。
この work item が残した変更は work item 2 件だけである。`wi-435` に T001 の訂正、試作結果、採否の判断、再評価条件を記録した。`wi-421` は `depends_on` の解決に伴い、表層言語を「`wi-435` の結果しだい」から直接 TLA+ へ確定し、引き継ぐ 2 つの機構と、確かめるべき前提（`tla2tools.jar` の取得物が `mise.toml` だけで固定できること、Apalache が TLA+ を直接検査しても ITF を出せること）を書き足した。
試作した資材は Out of Scope の定めどおり全て撤去した。撤去したのは `backend/jobs/model/`（Quint 模型 2 本、ITF デコーダー、Go トレースドライバー、生成トレース 7 本）、`tools/model-sync/`、`mise.toml` の `[tools]` 2 行と `[env]` と評価用タスク 8 個、`.gitignore` の 1 行である。撤去後の作業ツリーは work item 2 件の変更だけを含む。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
