---
status: completed
archived_from: bb8b8175200d963c2b68946eff1c6aa4d4a57935
---

# 旧形式の仕様文書の経路を検査と生成器から取り除く

`mise run spec-diff -- main` の結果は「no normative specification change against main」であり、規範の差分はない。
`tools/check/legacy-spec-layout.json` を削除し、すべての Context が `design/README.md` を持つことを Context ごとに確かめる検査に置き換えた。
文書の配置の判定、要件の配置の検査、規則の検査、セキュリティ統制の検査、`render-docs`、`spec-route` から、Context と機能の階層の旧形式のファイルを受け付ける経路を外した。到達しなくなった節の順序の検査（`verifySectionOrder`）も削除した。
`spec-diff` の旧形式の読み取りと、システムの `docs/domain/scenarios.feature.md` の経路は残した。
ツールのテストの準備データを印を持つ Context へ移し、印のない Context と旧形式のファイル種別を拒否するテストを加えた。
`SPECIFICATION_FORMAT.md` の第 10 節を「旧形式の扱い」に改め、根拠の文書と `WORK_ITEM_FORMAT.md` の言及を現状に合わせた。
生成したサイトはページの集合が main と同じで、差は未決事項の説明文の 1 文と、編集した規約の文書だけである。

全文は `git show bb8b8175200d963c2b68946eff1c6aa4d4a57935:<このファイルのパス>` で読む。
