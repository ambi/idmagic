# コントリビューション

## 開発環境とコマンド

ツールの導入、ビルド、生成、テストは [ローカル開発](docs/development/local-development.md) と `mise tasks` を参照してください。基本的なコマンドは下地のツール（`bun`、`go`、`docker` など）を直接呼ばず、`mise run <task>` から実行します。

開発の進め方は [仕様先行の開発ワークフロー](docs/development/specification-first-workflow.md) で定めます。

## Pull Request に求めること

- 一つの意味上の変更につき一つの work item を作る。形式は[作業項目フォーマット](docs/formats/work-item-format.md)に従う。
- 仕様先行、生成と証拠は[開発サイクル](docs/development/specification-first-workflow.md#開発サイクル)と[証拠の要件](docs/development/specification-first-workflow.md#証拠の要件)に従う。
- コードを書くときとレビューするときは、[コーディングスタイル](docs/development/coding-style.md#使い方)の七つの視点で判断する。
- コミットメッセージは Conventional Commits に従い、件名も本文も英語で書く。

文書とコードの言語は [文章の言語](docs/development/writing-language.md) が定めます。同じ内容を 2 つの言語で書かないでください。

## 検証

Pull Request を出す前に、[最終検査の選択](docs/development/specification-first-workflow.md#最終検査の選択)に従って、実際の差分と影響範囲に必要な検査を通してください。

CI の必須検査は [.github/workflows/idmagic-ci.yaml](.github/workflows/idmagic-ci.yaml)、手元の実行順序は[検証の進め方](docs/development/specification-first-workflow.md#検証の進め方)が定めます。

## 脆弱性の報告

脆弱性は Pull Request や公開 issue ではなく、[SECURITY.md](SECURITY.md) の経路で報告してください。
