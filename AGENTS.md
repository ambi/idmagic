# エージェント指示

エージェントが利用者へ向けて書く文章は日本語にする。作業の報告、提案、確認、要約、差分の説明など、会話へ出す文章をすべて含む。

## 作業ごとの参照先

作業項目では `mise run brief -- <work-item> --target <対象パス>` から、今回必要な節と検証タスクの候補を得る。
対象が複数なら `--target` を繰り返し、境界や作用を変えるときは `--concern boundary` または `--concern effects` を付ける。
候補の参照元と理由を確認し、編集またはレビューを始める前に該当する節を読む。
作業項目がない場合も、この表の条件から読む節を選ぶ。
表の条件コメントと追加取得の使い方は[作業文脈の導出](docs/development/task-context.md)が定める。

| 作業 | 読む文書 |
| --- | --- |
| 作業項目の着手 <!-- context: kinds=* --> | [開発サイクル](docs/development/specification-first-workflow.md#3-開発サイクル)、[証拠の要件](docs/development/specification-first-workflow.md#4-証拠の要件)、[着手時の文脈と設計](docs/formats/work-item-format.md#着手時の文脈と設計)。子節は今回の変更に適用するものだけを読む |
| 機能と振る舞いの変更 <!-- context: kinds=feature,bugfix,operations --> | [仕様の漏れを探す観点](docs/development/specification-first-workflow.md#仕様の漏れを探す観点)、[主要ユースケース](docs/development/testing.md#主要ユースケース) |
| 文書、コメント、UI 文言の執筆または推敲 <!-- context: kinds=docs targets=docs,.agents,AGENTS.md,frontend --> | [文章の言語](docs/development/writing-language.md)。日本語の文章品質と整形には `japanese-tech-writing` スキルを使う |
| コードに含む文章の変更 <!-- context: targets=backend,frontend,tools,.agents,AGENTS.md --> | [文章の言語](docs/development/writing-language.md) |
| 本番コード（`backend/`、`frontend/src/`）の変更 <!-- context: targets=backend,frontend/src --> | [仕様影響の宣言](docs/development/specification-first-workflow.md#仕様影響の宣言)に従って判断と宣言を残す |
| コードの編集またはレビュー <!-- context: kinds=feature,bugfix,refactor,tooling,maintenance,operations targets=backend,frontend,tools --> | [コーディングスタイルの使い方](docs/development/coding-style.md#使い方)。七つの視点のうち、触れる判断の節へ進む |
| テスト水準、実行境界、テストダブルの選択 <!-- context: kinds=feature,bugfix,refactor,tooling,maintenance,operations targets=backend,frontend,tools --> | [テスト方針](docs/development/testing.md#主要ユースケース)。水準の表と、触れる依存の[テストダブル](docs/development/testing.md#テストダブル)を選ぶ |
| モジュール、型、依存、作用、エラーの設計 <!-- context: targets=backend,frontend/src --> | [モジュールの深さと情報隠蔽](docs/design/application/design-guidelines.md#モジュールの深さと情報隠蔽)。触れる型と作用の節へ進む |
| モジュールの追加、分割と統合、公開範囲の拡大、モジュール間の依存の追加、テーブル所有者または組み立て地点の変更 <!-- context: concerns=boundary --> | [境界を選ぶ判断手順](docs/design/application/design-guidelines.md#境界を選ぶ判断手順)、[モジュールの責務](docs/design/architecture/logical.md#モジュールの責務)、[依存規則](docs/design/application/backend.md#モジュール間の依存規則)、[既知リスク](docs/design/architecture/risks.md)。比較の記録を work item の設計に書いてから実装する |
| 横断的な作用と確定の変更 <!-- context: concerns=effects --> | [作用の境界](docs/design/application/design-guidelines.md#作用の境界)、[一貫性の境界と永続化](docs/design/application/design-guidelines.md#一貫性の境界と永続化)、[モジュール間イベント](docs/design/application/backend.md#モジュール間イベント) |
| 公開 HTTP 契約の変更 <!-- context: concerns=contract --> | [API 区分](docs/design/application/api-guidelines.md#api-区分)、[TypeSpec の範囲](docs/formats/specification-format.md#1-typespec-の範囲)。触れる HTTP の判断の節へ進む |
| 認可と信頼境界の変更 <!-- context: concerns=security --> | [主体の種類](docs/design/security/authorization.md#主体の種類)、[テナント境界](docs/design/security/authorization.md#テナント境界)、[脅威モデルの読み方](docs/design/security/threat-model.md#読み方)。触れる信頼境界の節へ進む |
| 品質目標を変える作業 <!-- context: concerns=performance --> | [サービス目標](docs/requirements/quality.md#サービス目標)、[性能設計](docs/design/performance/README.md) |
| 仕様文書の執筆 <!-- context: targets=spec,docs/modules,docs/requirements --> | [要件と例の形式](docs/formats/specification-format.md#5-要件と例) |
| 設計文書の執筆 <!-- context: kinds=docs targets=docs/design --> | [設計領域の形式](docs/formats/design-format.md#1-設計領域) |
| 文書の配置を変える作業 <!-- context: kinds=docs --> | [一次情報源の割り当て](docs/formats/documentation-guide.md#一次情報源の割り当て)から、触れる文書種別の節を選ぶ |
| 開発文書の執筆 <!-- context: kinds=docs targets=docs/development --> | [開発文書の形式](docs/formats/documentation-guide.md#8-開発文書) |
| 開発環境、ビルド、生成、CI、リリース <!-- context: targets=mise.toml,infra,.github --> | [開発文書の索引](docs/development/README.md)から該当する文書を選ぶ |
| リポジトリ内ツールの変更 <!-- context: targets=tools --> | [リポジトリツール](tools/README.md) |

## mise

`mise.toml` は、ツールのバージョン、環境、リポジトリコマンドを集約する唯一の場所である。検証、ビルド、テスト、静的検査、整形、開発サーバー、デモ、コード生成などの基本操作は、下位のツールを直接呼ばず `mise run <task>` で実行する。

実行前に `mise tasks` で該当タスクを探す。一般的な操作にタスクがなければ `mise.toml` へ追加する。

一度だけの調査または移行に使い、反復利用しないコマンドは `mise` タスクにせず直接実行する。

`go test`、`go vet`、`gofmt`、`golangci-lint`、`bun test` のような下位ツールは、1 パッケージだけの確認であっても直接呼ばない。その用途には `mise run test-go-package -- <package>`、`test-go-test`、`format-go`、`lint-go` を使う。

`mise run test-ui-e2e`、`test-ui-e2e-file`、`verify-full` と Chrome／Bun.WebView の起動確認は、localhost の待受けとブラウザーの起動を必要とするため、初回からサンドボックスの外で実行する。
`mise run verify` と Go の HTTP 統合テストも localhost の待受けを使うため、同じ実行方法に従う。
権限昇格が必要な実行環境では、最初の呼び出しから実行ツールの権限昇格を指定する。

テストが規範の被覆を主張する場合は[規範 ID の参照](docs/development/specification-first-workflow.md#規範idをテストから参照する)に従う。
検出能力を測る場合は[ミューテーションテスト](docs/development/specification-first-workflow.md#ミューテーションテスト)で適用条件と手動注入の分担を選び、引数は[Go のミューテーションテスト](docs/development/testing.md#go-のミューテーションテスト)を読む。

## ツール

ファイルの読み書きと値の抽出では、実行環境に公開されている、目的に合う専用ツールを先に使う。
ツールの名前と提供範囲はエージェントの実行環境によって異なるため、特定の名前のツールが必ずあるとは仮定しない。
専用の読み取りツールがなければ、`cat` や `sed -n 'N,Mp'` で必要な範囲を読む。
標準コマンドで足りる読み取りを、Perl やその場限りのスクリプトへ置き換えない。

| 作業 | 使用するもの | 使用しないもの |
| --- | --- | --- |
| ファイルまたは一部の読み取り | 公開されていれば専用の読み取りツール。なければシェル実行ツールで `cat` や `sed -n 'N,Mp'` | 読み取りだけの Perl やその場限りの解析スクリプト |
| ファイルまたは部分木の構造把握 | `ast-grep outline` | 入口を探すための全ファイル読み取り |
| コードの編集 | `apply_patch` または公開されている専用の編集ツール | `sed -i`、ファイルを書き換える `python3` または `bun` のヒアドキュメント |
| JSON の照会、抽出、変更 | `jq` | その場限りの `python3 -c` または `bun -e` |
| YAML の照会、抽出、変更 | `yq` | インデントを前提とする `grep` |
| 内容の検索 | `rg` | `grep -r`、`find … -exec grep` |
| 名前によるファイル検索 | `fd` | `find` |
| 同じ構造または同じ文字列の複数箇所の書き換え | `ast-grep run -p <pattern> -r <rewrite> -U`。一致が 0 件なら終了コード 1 で止まる | 複数行の正規表現、廃止された `sg` 別名 |
| HTML ページの読み取りと抽出 | `ax` | `curl` と解析スクリプト |
| GitHub の Pull Request、Issue、API | `gh` | GitHub API への `curl` |

必要な CLI がなければ `mise run setup-cli-tools` を実行する。
