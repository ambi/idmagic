# 継続的インテグレーション

## 一次情報

CI で実行するジョブと検査の一次情報は [`.github/workflows/idmagic-ci.yaml`](../../.github/workflows/idmagic-ci.yaml) である。

## CI で検証するもの

| 分類 | 検証内容 |
| --- | --- |
| 仕様と文書 | TypeSpec、規範シナリオ、文書リンク、用語、work item |
| バックエンド | 静的検査、単体・統合テスト、ビルド、データベーススキーマ |
| フロントエンド | 型検査、単体テスト、ビルド、ブラウザー E2E |
| リポジトリ | 生成物の差分、依存関係、構成上の規則 |

ジョブを追加または削除するときはワークフローを変更する。

## CI で検証しないもの

**infra の宣言的アセットは CI で検証しない。** Docker Compose の構成、Kubernetes のオーバーレイ、Prometheus の規則、k6 のモジュールには、それぞれ `mise run check-compose` / `check-k8s` / `check-monitoring` / `check-k6` があるが、どの集約タスクにも CI にも入れていない。いずれも kustomize、kubeconform、promtool、k6 のイメージを取得してから実行するため、1 回あたりの所要が他の全検査の合計を超える。得られるのは変更頻度の低いアセットの構文検査であり、**最も変わらない入力に、最も重い検査を毎 Pull Request で払う形になる**ので採らない。

これらのタスクは `mise tasks` から個別に呼べる状態を保つ。`infra/` を変更したときは、対応するタスクを手元で実行してから Pull Request を出す。PostgreSQL のスキーマ収束 (`check-schema`) だけは例外で、データの意味に直結するため CI で実行する。

## 手元での再現

失敗したステップが呼ぶ `mise run <task>` を同じ固定したツールのバージョンで実行する。集約ゲート (`check`、`verify-spec`、`verify`) は最初の失敗で打ち切らず、落ちたゲートを全部並べて終わるので、ログが挙げた分をまとめて手元で再現できる。検査を恒常的に無効化せず、不安定な検査は所有者と期限を持つ work item で修復または削除する。

ブラウザー E2E (`test-ui-e2e`) は手元の `verify` には入っていない。Go のビルド、API サーバー、開発サーバー、初期データの投入という起動をこのテストだけが要求し、静的検査と単体テストのたびに毎回払う費用ではないからである。手元では `mise run verify-full` で、CI では独立した job で実行する。

Pull Request の要件は [コントリビューション](../../CONTRIBUTING.md)で、変更中に検査を広げる順序は [検証の段階](specification-first-workflow.md#5-検証の段階)で定める。
