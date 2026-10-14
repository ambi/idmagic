---
status: completed
archived_from: e121c10f76e9fc1819f8f62c92bcaa865718cf30
---

# 設計と仕様の単位の名前を「context」から「module」へそろえ、spec/contexts を spec/modules へ移す

`mise run spec-diff -- main` は「no normative specification change against main」を報告した。TypeSpec の名前空間、タグ、モデル名は変えていない。`spec/generated/openapi/idmagic.openapi.json` は、移動の前後で同じハッシュ（`9332aa4a…`）になった。
`spec/contexts/` を `spec/modules/` へ移し、`spec/main.tsp` の import、検査、描画器、文書とコードのパスの参照を合わせた。ID の文法と配置の表記を `REQ-<MODULE>-NNN`、`EX-<MODULE>-…`、`<module>` に改めた。検査ツールでは、「context」を名前に含む複合語の識別子と `feature-slice-debt.json` のキー（`moduleAliases`）を改めた。単独の `context` は、モジュールの意味で使っているファイルに限って `module` に改めた。Go の `context.Context`、Gherkin の Context 手順、作業項目の `initial_context` を指すものは残した。
コードのコメント 140 か所（72 ファイル）にあった、存在しない `spec/contexts/<module>.yaml` への参照は、`docs/modules/<module>/` へ張り替えた。spec-change の skill にあった、存在しない `tools/check/legacy-spec-layout.json` を指す一文は削除した。
完了済みの記録とリリース文書は、本文を書き換えず、移したファイルへのリンク先だけを張り替えた。完了記録の `affected_spec` は、対応表の接頭辞の対応（`spec/contexts/` → `spec/modules/`）で解決する。`spec-diff` は、履歴の `spec/contexts/` の TypeSpec の宣言を、現在のパスで同定する。これらの移行用の対応は、この作業項目の比較の基準（main）が旧い配置であるあいだだけ必要であり、統合後に別の作業項目で削除する。

全文は `git show e121c10f76e9fc1819f8f62c92bcaa865718cf30:<このファイルのパス>` で読む。
