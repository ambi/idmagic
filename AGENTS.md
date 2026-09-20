# エージェント指示

エージェントが利用者へ向けて書く文章は日本語にする。作業の報告、提案、確認、要約、差分の説明など、会話へ出す文章をすべて含む。

## 作業ごとの参照先

作業に該当する文書を、編集またはレビューを始める前に読む。

| 作業 | 読む文書 |
| --- | --- |
| 機能、振る舞い、設計の変更 | [仕様先行の開発ワークフロー](docs/development/specification-first-workflow.md)。仕様文書の形式は [SPECIFICATION_FORMAT.md](SPECIFICATION_FORMAT.md)、work item の形式は [WORK_ITEM_FORMAT.md](WORK_ITEM_FORMAT.md)、文書体系は [DOCUMENTATION_GUIDE.md](DOCUMENTATION_GUIDE.md) |
| コードの編集またはレビュー | [コーディングスタイル](docs/development/coding-style.md)。七つの視点を実装とレビューの判断に用いる |
| テスト水準、実行境界、テストダブルの選択 | [テスト方針](docs/development/testing.md) |
| モジュール、型、依存、作用、エラーの設計 | [設計ガイドライン](docs/design/application/design-guidelines.md) |
| 開発環境、ビルド、生成、CI、リリース | [開発文書の索引](docs/development/README.md)から該当する文書を選ぶ |

## mise

`mise.toml` は、ツールのバージョン、環境、リポジトリコマンドを集約する唯一の場所である。検証、ビルド、テスト、静的検査、整形、開発サーバー、デモ、コード生成などの基本操作は、下位のツールを直接呼ばず `mise run <task>` で実行する。

実行前に `mise tasks` で該当タスクを探す。一般的な操作にタスクがなければ `mise.toml` へ追加する。

一度だけの調査または移行に使い、反復利用しないコマンドは `mise` タスクにせず直接実行する。

`go test`、`go vet`、`gofmt`、`golangci-lint`、`bun test` のような下位ツールは、1 パッケージだけの確認であっても直接呼ばない。`mise run test-go-package -- <package>`、`test-go-test`、`format-go`、`lint-go` がその用途を持つ。

テストが宣言済みの `REQ-*`、`EX-*`、標準 id を被覆したと主張するときは、テスト関数の直上へ `//spec:covers <id>[, <id>]: <このテストが何を固定しているか>` を置く。この形だけが被覆と数えられるので、散文で id に言及するのは安全である。規約は[仕様先行の開発ワークフロー](docs/development/specification-first-workflow.md)の「Citing a normative id from a test」が定める。

テストの検出能力を測るときは `mise run test-go-mutation -- <package-directory>` を使う。手で変異を書くのは、変異器が表現できない「配線を外す」「デフォルトの分岐を差し替える」種類だけとする。読み方は[仕様先行の開発ワークフロー](docs/development/specification-first-workflow.md)の Mutation testing が定める。

## ツール

ファイルの読み書きと値の抽出では、目的に合う専用ツールを先に使う。

| 作業 | 使用するもの | 使用しないもの |
| --- | --- | --- |
| ファイルまたは一部の読み取り | Read ツールの `offset` と `limit` | `cat`、`head`、`tail`、`sed -n 'N,Mp'` |
| ファイルまたは部分木の構造把握 | `ast-grep outline` | 入口を探すための全ファイル読み取り |
| コードの編集 | Edit ツール、同じリテラルの一括置換には `sd` | `sed -i`、ファイルを書き換える `python3` または `bun` のヒアドキュメント |
| JSON の照会と抽出 | `jq` | その場限りの `python3 -c` または `bun -e` |
| YAML の照会と抽出 | `yq` | インデントを前提とする `grep` |
| 内容の検索 | `rg` | `grep -r`、`find … -exec grep` |
| 名前によるファイル検索 | `fd` | `find` |
| 同じ構造で識別子だけ異なる複数箇所 | `ast-grep` | 複数行の正規表現、廃止された `sg` 別名 |
| リテラルの一括置換 | `sd` | `sed -i` |
| HTML ページの読み取りと抽出 | `ax` | `curl` と解析スクリプト |
| GitHub の Pull Request、Issue、API | `gh` | GitHub API への `curl` |

必要な CLI がなければ `mise run setup-cli-tools` を実行する。
