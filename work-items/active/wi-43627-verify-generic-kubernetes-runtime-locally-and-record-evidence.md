---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-13
priority: p2
depends_on: [wi-47518-align-shared-document-placement-and-design-records]
change_kind: tooling
spec_impact:
  kind: none
  reason: "汎用 Kubernetes の既存構成をローカルの専用クラスターで起動して検証し、その証拠を保存する。製品の HTTP 応答、認証方式、永続状態の形式、ドメインイベント、外向きの通知と、本番構成および品質目標は維持する。"
---

# 汎用 Kubernetes の構成をローカルで起動し、条件付きの稼働証拠を残す

## 動機

[デプロイメントアーキテクチャ](../../docs/design/architecture/deployment.md#デプロイプロファイル)と[システム全体のリスク](../../docs/design/architecture/risks.md)は、汎用 Kubernetes の構成は静的検査を通るが稼働証拠がないと記録している。
wi-592 は構成の生成、wi-55532 は本番準備の不足の判定を扱い、実際の起動を検証する手順そのものは扱わない。

利用者は、今回の完了対象を汎用 Kubernetes のローカル検証とし、GKE の実環境検証は既存の本番準備項目へ明記することを選んだ。
クラウドのアカウントや外部環境の指定なしに、専用のローカルクラスターで再現可能な稼働証拠を収集する。
静的検査、ローカルの起動、本番の検証を別の事実として扱う。

## 対象範囲

- kind の専用ローカルクラスターで、汎用 Kubernetes の base と dev overlay を使って API、frontend、三つの worker レーンと batch の起動を検証する。
- PostgreSQL、スキーマの適用、テスト専用シークレット、初期データ、イメージのビルドと取り込みを、再現できる検証前提として用意する。
- 正式な Service と frontend の中継を通したプローブ、静的画面、Discovery、認証とトークンの smoke、worker の指標、batch の完了を観測する。
- 起動失敗、期限超過、部分成功、後片付け失敗を区別した実行結果と証拠を保存する。
- 開発手順、デプロイの検証状況、リスクの現在の扱いを、実際に得た証拠の範囲へ合わせる。

## 対象外

- GKE、Cloud SQL、本番のクラスターと実際のクラウド権限の使用。wi-55532 の後続工程へ記載する。
- dev 以外の overlay の稼働検証、GKE の構成作成、構成の生成器。wi-592 が扱う。
- AZ とノードの故障耐性、HA、SLO と CAP の達成、RPO/RTO、負荷試験。既存の wi-165、wi-282 などが扱う。
- NetworkPolicy の強制を伴うテナントやネットワークの分離の実証。kind の標準 CNI で起動した結果を、その証拠にはしない。
- 本番構成の未実装機能を検証を通すために一括で追加すること。
- 既存クラスター、既存 namespace、既存の kubeconfig の current-context の変更や削除。

## 設計

### ローカル実行の境界

新しい入口は `mise.toml` に追加する `verify-kubernetes-runtime` タスクとする。
kind のバージョンと必要な CLI は mise に固定し、Docker はリポジトリの既存の導入方針に従う。
実行器は `tools/kubernetes-runtime-verification/` に置き、CLI の実行、期限、証拠、後片付けを担当する。
単体テストではプロセス実行と時計をダブルにし、実際の稼働テストでは Docker と Kubernetes と PostgreSQL を実物として使う。

各実行には無作為な実行 ID を付け、専用の cluster 名、namespace、kubeconfig、証拠ディレクトリを使う。
kubectl は明示した kubeconfig と context を必須とし、利用者の current-context を参照しない。
実行器が所有を記録した cluster と作業用リソースだけを後片付けし、同名の既存 cluster がある場合は使わず別名にする。
成功と失敗のどちらでも証拠を確定してから後片付けする。

### 検証前提と対象構成

検証専用の overlay を `infra/k8s/overlays/verification/` に置き、dev overlay を参照する。
既存のアプリケーション構成を写した別構成は作らない。
差分はローカルのイメージ、レプリカ数、PostgreSQL の参照、シークレットと初期データの設定、外部の検証前提だけに限定する。
変更した値と検証対象の値を証拠へ記録し、本番の構成と同一だとは扱わない。

PostgreSQL は検証 namespace 内の専用インスタンスとし、既存の Docker Compose と同じ固定バージョンのイメージを使う。
物理スキーマの一次情報を適用してからアプリケーションを起動する。
初期データは既存の seed プロファイルと起動時の投入経路を再利用する。
現在の製品イメージに seed の独立実行ファイルがないことを、本番の Job の検証済み扱いで隠さない。
追加の初期データが必要なら検証 fixture として定義し、既存の起動契約と同じ初期化を通す。

製品と frontend のイメージは既存の Dockerfile からビルドし、kind へ取り込む。
検証用の参照には内容で区別できるタグとイメージ ID を使い、公開レジストリへ push しない。
API と worker の ConfigMap および Secret の参照、レーンの選択、Service と frontend の中継は既存構成を使う。
OTel はローカルの検証用 Collector を既存の Compose の構成に従って接続し、exporter の送信失敗を成功として隠さない。
保存と検索、本番の Collector の構成はこの証拠の対象にしない。

### 観測する結果

| 対象 | 合格条件 |
| --- | --- |
| スキーマと PostgreSQL | スキーマ適用が成功し、API の受付可否が PostgreSQL への到達を伴って成功する |
| API | Service 経由で `/livez`、`/readyz`、`/startupz` が既存契約に従って応答し、Discovery が有効な発行者と経路を示す |
| frontend | 静的画面を返し、同じ入口から API と Discovery へ中継できる |
| 認証とトークン | 既存の OAuth smoke を frontend の入口から実行し、既存の成功条件をすべて満たす |
| worker | 三つのレーンの Pod が起動し、Service の `/metrics` が応答する。追加されていない `/livez` を必須にしない |
| batch | 既存の retention-sweep CronJob から検証用 Job を作り、正常終了を Kubernetes の Job 状態で確認する |
| 証拠と後片付け | 必須の観測が全て成功し、証拠が現在の入力に結び付き、所有した cluster と一時資産の後片付けが完了する |

この表の一部だけの成功では、実行全体を成功にしない。
起動前の依存不足は実行不可として報告し、Pod の起動待ちと Job は期限付きで待つ。
失敗時には Secrets の値とシークレットを含む環境を出力せず、対象名、段階、状態、必要なログだけを採取する。

### 証拠の形式

結果は追跡しない `.verification/kubernetes-runtime/<run-id>/` に保存する。
manifest は実行 ID、Git revision、差分の有無と検証対象入力の digest、プロファイル、overlay、ツールと Kubernetes のバージョン、イメージ ID、PostgreSQL と初期データの条件、開始と終了時刻、段階ごとの結果、観測結果、後片付けの結果、適用除外を持つ。
Secret 値、トークン、資格情報、Cookie は保存しない。
入力が変わった証拠を読み取る場合は stale と示し、現在の検証済み結果へ昇格させない。
実行器は成功、失敗、実行不可を明示し、すべての必須観測と後片付けが成功した場合だけ終了コード 0 を返す。

kind ではなく利用者の既存クラスターへ適用する案は、権限と環境の選択が毎回必要になるため採らない。
静的レンダリングだけを証拠にする案は、実際の起動と接続を検出できないため採らない。
本番の構成とは別の簡易アプリケーション manifest を作る案は、対象構成の誤配線を検出できないため採らない。

## 計画

1. 実行器と証拠の型、専用 cluster の所有、期限と後片付けのインターフェースを実装する。
2. 既存の dev 構成を再利用する検証 overlay、PostgreSQL、スキーマ、初期データ、イメージの前提を整える。
3. 全必須観測を実装し、実際のローカルクラスターで通し実行する。
4. 失敗と stale を検出する証拠とテストを揃え、実際の成功結果に限定して文書を更新する。

## タスク

- [ ] T001 [Tooling] mise のツールと実行タスク、実行器、結果と証拠の型を追加する。
- [ ] T002 [Unit] 期限超過、必須観測の欠落、部分成功、実行不可、stale、後片付け失敗と他の context への誤操作を検出する RED を作る。
- [ ] T003 [Infra] 既存の dev 構成を参照する検証 overlay と PostgreSQL、スキーマ、初期データ、Collector の前提を整える。
- [ ] T004 [Runtime] 専用 cluster とイメージを準備し、API、frontend、認証、worker と batch の必須観測を実装する。
- [ ] T005 [Evidence] 実際の通し実行で成功と後片付けを確認し、機密値を含まない証拠を保存する。
- [ ] T006 [Docs] 開発手順、デプロイの検証状況とリスクをローカルの条件付き証拠へ合わせ、GKE と本番の未検証を維持する。
- [ ] T007 [Verify] 局所テスト、構成検査、変更耐性と最終検査を完了する。

## 検証

- `mise run test-tools-file -- <実行器のテストファイル>` で判定、期限、入力の同一性、証拠、機密値の除外、後片付けの故障を検証する。
- 既存の `mise run check-k8s` に verification overlay を検査する経路を追加し、dev と verification を検査する。
- 追加した `verify-kubernetes-runtime` タスクで実物の Docker、Kubernetes、PostgreSQL と製品イメージを使った通し実行を行う。
- frontend の中継先を誤る、batch の完了を待たない、worker の必須観測を落とす故障を、通し実行または実行器の局所テストが検出することを確認する。
- 起動とクラスタ操作はサンドボックス外で実行し、実行前にネットワーク、Docker、ツールの必要な依存を準備する。
- `mise run typecheck-tools`、`mise run lint-tools`、`mise run check-command-map`、`mise run check-links` を実行し、実際の差分に応じて `mise run verify-spec` を選ぶ。
- 完了記録には通し実行の実際の結果、条件、証拠パス、後片付け、除外と未検証を記す。
- 実行器と架空の成功 fixture だけでは本項目を完了しない。

## リスク

ローカルの稼働を本番の品質達成として読める記述にすると、未検証のリスクが隠れる。
証拠はプロファイル、入力、環境と観測の範囲へ限定する。
既存の context とリソースを扱わず、作成した専用 cluster の所有と後片付けを明示的に記録する。
観測用ログに Secret 値とトークンを残さず、取得する情報を制限する。
