---
status: completed
archived_from: 13f9fb9d32c5865b5f4ec696c8e094ae0bb7f6e1
---

# brief が読ませる開発文書の量を測り、変更に適用しない節を候補から外す

`mise run spec-diff` に規範の差分はない。
`brief` は、各候補について最初の子見出しまでの本体の量と、子節の名前と量を表示し、本体の合計を示す。
`spec_impact: none` の変更では、`--concern contract` がない限り TypeSpec のファイルを候補にしない。
計画では 3 節と 4 節を子節へ分ける予定だったが、`brief` が子節を量とともに並べるので、文書の構造は変えなかった。
AGENTS.md の着手の行は、証拠に限らず子節を当てはまるものだけ読むと改め、作業文脈の導出へ表示と TypeSpec の条件を記した。
形式とワークフローの文書から、ほかの節と重なる 2 文を削った。`docs/development/` と `docs/formats/` は 313,546 バイトから 313,502 バイトになった。
wi-66580 の `brief` は、変更前は全体で約 48 KB の節と 150 KB の TypeSpec を示していたが、変更後は本体の合計が 23 KB で、TypeSpec を含まない。

全文は `git show 13f9fb9d32c5865b5f4ec696c8e094ae0bb7f6e1:<このファイルのパス>` で読む。
