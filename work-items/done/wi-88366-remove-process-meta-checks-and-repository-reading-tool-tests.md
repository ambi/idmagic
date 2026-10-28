---
status: completed
---

# 効果の薄いリポジトリ検査を削り、tools のテストにリポジトリの状態を検査させない

`mise run spec-diff` は main に対して規範の差分なしを示す。
効果の薄いリポジトリ検査として `command-map`、`slo-references`、文書から作業項目へのリンクを拒否する `work-item-references` を削り、対応する mise タスクと文書の記述も消した。
`documentation-impact` は公開後の告知の宣言（`documentation_impact` の欄）だけを削った。成熟度の昇格に証拠を求める検査は公開前にも働いていたので、`maturity-promotion.ts` に切り出して残した。
実リポジトリの文書や設定を読む tools のテスト 10 本のうち、`check-agent-guidance` と `archived-work-item` は fixture に置き換えた。残りはリポジトリの状態を確かめるテストとして `tools/*/repository/*.repository.ts` へ移し、`test-tools` ではなく `check` の `check-repository-tests` から実行する。
ER 図の網羅のテストは `check-schema-tables` と重複しない（あちらは説明表を照合する）ので、消さずに移した。
`check-repository` の `tool-test-inputs` は、tools のテストが `import.meta.dir` から `tools/` の外を読むことと、引数なしの `createWorkspaceSnapshot()` を拒否する。追加時に 10 本を検出した。
量は、tools の TypeScript が 36,410 行から 35,078 行、mise タスクの定義が 146 から 144、リポジトリ検査が 29 から 27 になった。
`test-tools` は 40 秒から 41 秒でほぼ変わらない。移したテストは合わせて約 2 秒で、遅さの原因ではない。
`spec-impact` の簡素化と、`tools/check` の解析とテストの作りの見直しは対象外とし、別の作業項目で扱う。
