# 採らなかった Cloud Run 案のひな型

このディレクトリは、Google Cloud のデプロイプロファイルを Cloud Run で作る案のひな型を持つ。
**現在の設計はこの案を採らず、GKE Autopilot を使う。**
ここにあるものは、見直す場合に出発点として使えるよう残している。

採らなかった理由と、見直す条件は[プラットフォーム設計](../../../docs/design/infrastructure/platform.md#アーキテクチャの選択)が持つ。
採った構成は次の文書が持つ。

- 実行単位の配置先とプロファイルの一覧：[デプロイメントアーキテクチャ](../../../docs/architecture/deployment.md)
- コンピューティング、データベース、シークレットの注入、スキーマ適用、スケール単位、費用：[プラットフォーム設計](../../../docs/design/infrastructure/platform.md)
- エッジ、セグメンテーション、ファイアウォールルール、Egress、DNS と証明書：[ネットワーク設計](../../../docs/design/infrastructure/network.md)
- 起動時設定の項目、デフォルト値、検証規則：[`CONFIGURATION.md`](../../../CONFIGURATION.md)

## ひな型が現在の設計と食い違う点

ひな型は手を入れずに残しているため、現在の設計と次の点で食い違う。
使う前にこれらを直す。

- `startupProbe` と `livenessProbe` に `/health` を使い、`readinessProbe` を持たない。現在の設計は `/startupz`、`/livez`、`/readyz` を分ける。
- `idmagic-worker` をレーンに分けず、一つの worker pool で全レーンを処理する。
- 専用のサービスアカウントを作らず、デフォルトのサービスアカウントで動く。
- `provision.sh` は、一般提供になった worker pools をまだ `gcloud beta run worker-pools` で呼ぶ。
- Cloud Run の終了猶予は 10 秒で固定されており、汎用 Kubernetes のマニフェストの終了猶予と揃わない。

## 前提

- コンピューティング、データベース、シークレットを単一 VPC、単一リージョンへ置く。
- ひな型の値はプレースホルダーである。実行前に環境へ合わせて置き換える。
- `provision.sh` を実行する前に `gcloud components update` を済ませる。

## デプロイ順序

1. `infra/docker/Dockerfile` からイメージをビルドし、Artifact Registry へ登録する。
2. Cloud SQL のインスタンス、データベース、ユーザーを作る。
3. 接続文字列などのシークレットを Secret Manager へ登録する。
4. `psqldef --apply` でスキーマを適用する。起動時には適用せず、`--enable-drop` は使わない。
5. API を Cloud Run Service、ワーカーを worker pools としてデプロイする。
6. ロードバランサー、Cloud CDN、Cloud Armor、静的アセットのバケット、DNS、TLS を構成する。

ひな型は [`provision.sh`](./provision.sh)（1 から 5）と [`cloudrun-idmagic.yaml`](./cloudrun-idmagic.yaml)（API の Service 定義）にある。
6 のひな型は持たない。

## 置き換える値

| 置き換える対象 | 出現箇所 | 説明 |
|---|---|---|
| `PROJECT`、`REGION` | `provision.sh`、`cloudrun-idmagic.yaml` | プロジェクト ID とリージョン |
| `REPLACE_ME` | `provision.sh` | データベースユーザーのパスワード。Secret Manager へ登録する接続文字列にも同じ値が入る |
| `PROJECT:REGION:idmagic-pg` | `cloudrun-idmagic.yaml` | Cloud SQL の接続名。プライベート IP で接続する場合はこの注釈を外す |
| `https://id.example.com`、`id.example.com` | `cloudrun-idmagic.yaml` | 公開ホスト名。`ISSUER` は Discovery Metadata の `issuer` と一致させ、WebAuthn の RP ID と RP オリジンも同じ名前から決める |
| イメージタグ | `provision.sh`、`cloudrun-idmagic.yaml` | リリースではタグではなくダイジェストを指定する |

インスタンスの種別、ストレージの大きさ、最小と最大のインスタンス数、同時実行数、タイムアウトは、`provision.sh` と `cloudrun-idmagic.yaml` が正本である。
