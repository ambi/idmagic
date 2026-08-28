# 継続的インテグレーション

## 正本

CI で実行するジョブと検査の正本は [`.github/workflows/idmagic-ci.yaml`](../../.github/workflows/idmagic-ci.yaml) である。この文書は一覧を複製せず、失敗を切り分ける原則だけを持つ。ジョブを追加または削除するときはワークフローを変更し、対応する `mise` タスクが無ければ先に `mise.toml` へ追加する。

## CI で検証しないもの

**infra の宣言的アセットは CI で検証しない。** Docker Compose の構成、Kubernetes のオーバーレイ、Prometheus の規則、k6 のモジュールには、それぞれ `mise run check-compose` / `check-k8s` / `check-monitoring` / `check-k6` があるが、どの集約タスクにも CI にも入れていない。いずれも kustomize、kubeconform、promtool、k6 のイメージを取得してから実行するため、1 回あたりの所要が他の全検査の合計を超える。得られるのは変更頻度の低いアセットの構文検査であり、**最も変わらない入力に、最も重い検査を毎 Pull Request で払う形になる**ので採らない。

これらのタスクは `mise tasks` から個別に呼べる状態を保つ。`infra/` を変更したときは、対応するタスクを手元で実行してから Pull Request を出す。PostgreSQL のスキーマ収束 (`check-schema`) だけは例外で、データの意味に直結するため CI が持つ。

## 手元での再現

失敗したステップが呼ぶ `mise run <task>` を同じ固定ツール版で実行する。複数の検査をまとめたジョブが落ちた場合は、ログにある最初の失敗を再現し、その検査が緑になってからジョブ全体へ戻る。検査を恒常的に無効化せず、不安定な検査は所有者と期限を持つ work item で修復または削除する。

Pull Request の要件は [CONTRIBUTING.md](../../CONTRIBUTING.md)、変更中に検査を広げる順序は [検証のはしご](specification-first-workflow.md#5-verification-ladder) が持つ。
