# 継続的インテグレーション

## 一次情報

push と Pull Request で実行するジョブと検査の一次情報は [`.github/workflows/idmagic-ci.yaml`](../../.github/workflows/idmagic-ci.yaml) である。
main に対する Go race 検査の定期実行の一次情報は [`.github/workflows/idmagic-race.yaml`](../../.github/workflows/idmagic-race.yaml) である。

## CI で検証するもの

| 分類 | 検証内容 |
| --- | --- |
| 仕様と文書 | TypeSpec、規範シナリオ、文書リンク、用語、work item |
| バックエンド | 静的検査、単体・統合テスト、ビルド、データベーススキーマ |
| フロントエンド | 型検査、単体テスト、ビルド、ブラウザー E2E |
| リポジトリ | 生成物の差分、依存関係、構成上の規則 |

ジョブを追加または削除するときはワークフローを変更する。

仕様とリポジトリ検査はすべての変更で実行する。
製品検査の選択は `mise run plan-ci -- <base> <head>` が一か所で判断し、workflow の各ジョブはその出力を参照する。
PR は base/head、push は before/after の commit を比較する。
削除と移動は新旧のパスを含めて扱う。
空差分、比較基準の不在、差分の取得失敗では全群を選ぶ。

製品検査の対象外にできる文書は `work-items/` と `docs/development/` の Markdown に限定する。
これらを読むリポジトリ検査は維持する。
frontend の変更は UI とブラウザー、backend の局所変更は Go とブラウザーを選ぶ。
共有 Go コードと組み立て入口、Go の依存、スキーマ入力、公開契約、規範文書、検証基盤、未知の入力は全群を選ぶ。
複数種類の変更では各選択の和集合を取る。
文書の拡張子だけを根拠に、規範と実行可能な仕様例を除外しない。

既存名の `Verify specification, backend, and UI` が計画、仕様、Go、UI、ブラウザー、スキーマの結果を集約する。
計画と常時実行の検査、および選択したすべての群が成功した場合だけ成功する。
選択した群の失敗、取消、スキップ、結果の欠落は成功として扱わない。
対象外は計画の理由と `skipped` の結果を表示し、成功したテスト数へ加えない。
計画の欠落と不正な形式、対象外のジョブの失敗も拒否する。
workflow 全体の paths フィルターは置かず、計画は毎回起動し、最終チェックは先行ジョブの失敗時も起動する。
GitHub Actions の [依存ジョブと `always()` の仕様](https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax#jobsjob_idneeds)に従う。
ジョブごとに必要な依存だけを準備する。
仕様ジョブは `check-ci-spec` で生成物とリポジトリ規則を検査する。
公開 API の互換性検査もこの中で実行し、直後に同じ検査を繰り返さない。
設定、ルート優先度、ゲートウェイの Go 参照検査は、通常テストの直後に同じジョブで実行する。
Go テストが作ったビルド結果を利用し、仕様ジョブでバックエンドを別にコンパイルする費用を避ける。
手元の `check` は引き続き両方の検査を含む。

push と Pull Request の Go 全体テストは `mise run test-go` による通常構成とし、race 検出器を使わない。
data race の自動検査は、main の全体を `mise run test-go-race` で検査する定期実行だけが担う。
実行は毎日一回（UTC 18:17）と手動の起動（`workflow_dispatch`）で、実行履歴や経過日数による切り替えは行わない。
構成は Go 全体テストと同じく DB を必須にし、検証の証拠を失敗時も artifact として残す。
通常テストの成功は data race を検査した証拠にはしない。

定期実行が失敗すると、GitHub は cron を最後に変更した利用者へ通知する。
data race は実行のたびに現れるとは限らないので、次の順で原因を探す。

1. 失敗した実行の artifact とログから、`DATA RACE` を報告したパッケージとテストを特定する。
2. 失敗した実行と、直前に成功した実行の、それぞれのコミットを Actions の実行一覧で確かめる。原因はその範囲にある。
3. 手元で、該当パッケージを race 検出器付きで実行して再現する（`mise run test-go-package -- <package>`）。一度で現れなければ、繰り返し実行する。
4. 再現したら範囲を二分して原因のコミットを特定し、通常の作業項目として直す。

既存バージョン API との互換性は[公開状態](release.md#公開状態と日常開発)に応じて検査する。
初回公開前は互換性比較を省略し、公開後は破壊的変更を拒否する。
現在の TypeSpec と実装の整合、セキュリティとデータの検査は公開状態によらず実行する。

Go の検証では DB を必須にし、起動失敗やスキップを成功にしない。
変更操作のうち静的解析が不完全なものには、同じ実行で成功した契約テストを要求する。
変更基準には Pull Request の base SHA または push 前の SHA を渡す。
判定と代替証拠の形式は[テスト方針](testing.md#検証結果の区分)が定める。
実行結果の要約を job summary に表示し、失敗した場合も `.verification/` の構造化結果と生の Go イベントを artifact に保存する。
CI の通常テストの報告には `.verification/test-go.json` を明示し、別の race 実行の結果を参照しない。
テスト実行まで到達しなかった場合は結果の欠落として表示され、以前の実行を成功の証拠に使わない。

## CI で検証しないもの

**infra の宣言的アセットは CI で検証しない。** Docker Compose の構成、Kubernetes のオーバーレイ、Prometheus の規則、k6 のモジュールには、それぞれ `mise run check-compose` / `check-k8s` / `check-monitoring` / `check-k6` があるが、どの集約タスクにも CI にも入れていない。いずれも kustomize、kubeconform、promtool、k6 のイメージを取得してから実行するため、1 回あたりの所要が他の全検査の合計を超える。得られるのは変更頻度の低いアセットの構文検査であり、**最も変わらない入力に、最も重い検査を毎 Pull Request で払う形になる**ので採らない。

これらのタスクは `mise tasks` から個別に呼べる状態を保つ。`infra/` を変更したときは、対応するタスクを手元で実行してから Pull Request を出す。PostgreSQL のスキーマ収束 (`check-schema`) だけは例外で、データの意味に直結するため CI で実行する。

## 手元での再現

検査の選択は `mise run plan-ci -- <base> <head>` の JSON で確認できる。
CI は同じ計画と群ごとの結果を job summary に表示する。
最終チェックの再現には `CI_NEEDS` に計画ジョブの出力と各ジョブの結果を含む `needs` JSON を渡して `mise run check-ci-results` を実行する。

失敗したステップが呼ぶ `mise run <task>` を同じ固定したツールのバージョンで実行する。集約ゲート (`check`、`verify-spec`、`verify`) は最初の失敗で打ち切らず、落ちたゲートを全部並べて終わるので、ログが挙げた分をまとめて手元で再現できる。検査を恒常的に無効化せず、不安定な検査は所有者と期限を持つ work item で修復または削除する。

ブラウザー E2E (`test-ui-e2e`) は手元の `verify` には入っていない。Go のビルド、API サーバー、開発サーバー、初期データの投入という起動をこのテストだけが要求し、静的検査と単体テストのたびに毎回払う費用ではないからである。手元では `mise run verify-full` で、CI では独立した job で実行する。

Pull Request の要件は [コントリビューション](../../CONTRIBUTING.md)で、変更中に検査を広げる順序は [検証の進め方](specification-first-workflow.md#検証の進め方)で定める。
