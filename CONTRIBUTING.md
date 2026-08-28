# コントリビューション

## 開発環境とコマンド

ツールの導入、ビルド、生成、テストは [ローカル開発](docs/development/local-development.md) と `mise tasks` を参照してください。基本的なコマンドは下地のツール（`bun`、`go`、`docker` など）を直接呼ばず、`mise run <task>` から実行します。

開発の進め方は [仕様先行の開発ワークフロー](docs/development/specification-first-workflow.md) が持ちます。

## Pull Request に求めること

- **外部から観測できる振る舞いを変えるなら、仕様を先に更新する。** モデル、API、HTTP 契約、認証方式は TypeSpec が持ち、受け入れシナリオ・用語・準拠する規範・状態遷移・設計判断・機構の説明は、その種類をファイル名が示す `docs/` 配下のファイルが持ちます。書式は [SPECIFICATION_FORMAT.md](SPECIFICATION_FORMAT.md) にあります。
- **1 つの意味上の変更につき 1 つの work item を持つ。** 形式は [WORK_ITEM_FORMAT.md](WORK_ITEM_FORMAT.md) が定め、`affected_spec` に影響する規範 ID または TypeSpec のシンボルを直接書きます。
- **TypeSpec を変更したら生成物を再生成し、互換性を検査する。** `spec/generated/` の生成物は追跡しないため、コミットには含めない。
- **拒否をテストするときは、返したステータスと、拒否が触らなかったものの両方を確かめる。** 前者だけのテストは、「拒否」と応答してから操作を実行してしまう実装に対しても同じように通ります。理由は [仕様先行の開発ワークフロー](docs/development/specification-first-workflow.md#testing-a-refusal) にあります。
- **コミットメッセージは Conventional Commits に従い、件名も本文も英語で書く。**

文書とコードの言語は [AGENTS.md](AGENTS.md) の表が定めます。同じ内容を 2 つの言語で書かないでください。

## 検証

Pull Request を出す前に `mise run verify` を通してください。

**必須の検査の一覧をこの文書に複製しません。** 正本は [.github/workflows/idmagic-ci.yaml](.github/workflows/idmagic-ci.yaml) です。一覧を 2 か所に持つと、検査を足したときに必ず片方が古くなり、古いほうを読んだ人が「通したはずなのに落ちる」に出会います。

手元では、変更したものについて失敗しうる最も安いゲートから順に実行します。全体の検証は最後の 1 回で足ります。順序は [検証のはしご](docs/development/specification-first-workflow.md#5-verification-ladder) が定めます。編集のたびに全体のスイートを回すことが、このリポジトリで時間を失う最もありふれた方法です。

## 脆弱性の報告

脆弱性は Pull Request や公開 issue ではなく、[SECURITY.md](SECURITY.md) の経路で報告してください。
