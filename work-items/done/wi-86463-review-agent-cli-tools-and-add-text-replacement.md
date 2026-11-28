---
status: completed
---

# エージェントのツール選択を見直し、構文に依存しない一斉置換を補う

`mise run spec-diff` は規範仕様の差分なしを示した。
文字列の一斉置換に srgn を採用し、AGENTS.md のツール表の書き換えの行を、構造（ast-grep）と文字列（srgn）に分けた。
srgn の行には必須の引数を含む呼び出しの形を置き、`--stdin-detection force-unreadable` がないとパイプの標準入力だけを処理して何も変えずに成功する挙動と、バックスラッシュの解釈を `agent-guidance.md` の「srgn の引数の解釈」に短く書いた。
AGENTS.md と `agent-guidance.md` は利用者の依頼で最小にし、jq、yq、ax、gh の補足、CLI の分担表、採らなかった CLI（`sd`、`fastmod`、`scooter`、`mgrep`、`jaq`、`dasel`）とその理由は書かなかった。
スキルにする案は、置換のたびに本文を読み込み、スキルを読まずに直接呼べば引数の付け忘れを防げないので採らなかった。

`mise.toml` の `[tools]` に rg、fd、ast-grep、yq、gh、ax、srgn を固定し、Homebrew の `setup-cli-tools` を削除して `mise install` に一本化した。
CI がエージェント専用の CLI も取得する増分（展開後 144 MB、キャッシュが外れたときだけ）は許容した。
CI 用の一覧を別に持つと、CI のタスクが新しい CLI を使ったときの足し忘れで CI だけが落ちるためである。

tools の TypeScript を実行するタスクを `setup-tools` に、frontend で bun を実行するタスクを `install-ui` に依存させた。
両タスクは mise の `sources` と `outputs` をやめ、印の新旧をスクリプトで比べる。
`check-agent-guidance` はインラインコードに書いた `mise run <task>` を照合するようになった。
`mise.toml` のタスクは内容ごとの節に分け、節の中を名前の辞書順に並べた。
