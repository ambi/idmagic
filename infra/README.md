# インフラストラクチャガイド

この手引きは IdMagic の配備、監視、セキュリティ設定を扱う。

## Kubernetes、監視、負荷スモークテスト

Kubernetes のベース構成は API、UI ゲートウェイ、永続ジョブを処理する `worker` を独立した Deployment に分け、PostgreSQL と共通の OTLP 接続先を使用する。`worker` は実行レーンごとの Deployment（`idmagic-worker-{latency-sensitive,default,bulk}`）に分かれ、それぞれがメトリクス専用の Service を持つ。この Service は `/metrics` だけを公開し、アプリケーション用 HTTP エンドポイントは持たない。プラットフォームが参照先の Secret（`idmagic-<environment>-runtime-secrets`、`idmagic-<environment>-worker-secrets`）を作成した後にだけ、レンダリング済みの環境を適用する。シークレットの値、リリースイメージのダイジェスト、クラウド固有のデータベースエンドポイントは、このリポジトリには決して保存しない。

```bash
mise run check-k8s dev
mise run deploy-k8s dev
```

本番環境は API と UI のレプリカを 3 個、イベントリレーのレプリカを 2 個配置する構成から始める。リリースパイプラインで本番オーバーレイのゼロダイジェストのプレースホルダーを置き換え、検証してから適用する。ロールバックでは直前のリリースのダイジェストオーバーレイを適用する。Kubernetes は以前の ReplicaSet を保持するため、必要なら直ちに `mise run rollback-k8s idmagic-api` を実行できる。

API は `/startupz`、`/livez`、`/readyz` を直接プローブする。その NetworkPolicy が受信を許可するのは UI ゲートウェイと Prometheus のスクレイプ通信だけで、送信先は DNS と PostgreSQL に限る。各 `worker` レーンの NetworkPolicy が受信を許可するのは Prometheus のスクレイプ通信（`/metrics`、ポート 8080）だけで、送信先は DNS と PostgreSQL に限る。`worker` はアプリケーションの通信を処理しないため、レディネスプローブとライブネスプローブを持たない。

`infra/k8s/monitoring` は Docker の例と同じ HTTP RED、認証記録ルール、アラートルールに加え、レーンごとのジョブのゴールデンシグナル（キュー深度、取得レイテンシー、失敗率、再試行率）をまとめる。`TokenLatency`、`TokenErrorRate`、`LoginLatency`、`LoginErrorRate`、可用性の証跡を、リクエスト率、エラー率、レイテンシー、ログイン、トークンの各パネルに対応付ける。`monitoring/operator` ディレクトリは Prometheus Operator がインストール済みの場合にだけ適用する。その `idmagic-worker` ServiceMonitor は 3 個のレーンの Service をすべて対象とする。それ以外の場合は、Prometheus が `idmagic-api` と `idmagic-worker-{latency-sensitive,default,bulk}` の Service にある `/metrics` をスクレイプするよう設定する。

```bash
mise run check-monitoring
mise run deploy-monitoring
mise run deploy-monitoring-operator # Prometheus Operator を使う場合だけ
```

k6 スモークテストは、テナント内に閉じた 1 組の seed フィクスチャーを使い、認可コードと S256 PKCE、リフレッシュトークンのローテーション、クライアントクレデンシャルズを扱う。デフォルトのクライアントには開発用 seed の固定 UUID を使い、テナントをまたぐデータの作成や再利用は行わない。最初に、意図的に seed を投入した開発環境の対象を起動する。デフォルト値を使えない場合は、使い捨てのフィクスチャー用資格情報だけを環境変数で渡す。

```bash
mise run k6-smoke # デフォルト値: http://host.docker.internal:8080/realms/default
# ローカルの `mise run dev-memory` API: mise run k6-smoke http://host.docker.internal:8081 http://localhost:5173
mise run check-k6
```

スモークテストのしきい値は [SLO-TOKEN-LATENCY と SLO-PRIMARY-ERRORS](../docs/capacity.md#service-level-objectives) から導く。数値はここに再掲せず、`load/k6/oauth-smoke.js` が持つ値がその目標に由来することだけを記録する。CI はフィクスチャーを用意した後、隔離したサービス URL に対して同じレシピを実行する。本番テナントに対して実行してはならない。

宣言的な PostgreSQL スキーマだけを再適用する。

```bash
docker compose -f infra/docker/docker-compose.dev.yaml run --rm schema
```

Docker Compose のスタックに対して OAuth / OIDC のデモスクリプトを実行する。

```bash
BASE=http://localhost:8080 ./demo.sh
```

## 設計

## テナントのサブドメインルーティング

Ingress に `*.${TENANT_BASE_DOMAIN}` のワイルドカード DNS とワイルドカード TLS 証明書がある場合にだけ `TENANT_BASE_DOMAIN` を設定する。エンドポイント形式が `subdomain` のテナントには `{realm}.${TENANT_BASE_DOMAIN}` だけで到達でき、パス形式のテナントには `/realms/{realm}` だけで到達できる。アプリケーションはテナントのレスポンスに `Vary: Host` を返すため、CDN やリバースプロキシはキャッシュキーにホストを含めなければならない。証明書の発行と更新はプラットフォームの責任である。

エンドポイント形式を変えると、発行者、Cookie のスコープ、WebAuthn の RP ID が変わる。システムテナントのコンソールで切り替える前に、RP メタデータの変更とパスキーの再登録を調整する。

これらの資材が実装する横断的な実行時設計は、ここではなくリポジトリの設計記録に記載する。高可用性と共有状態、HTTP サーバーの堅牢化、セキュリティレスポンスヘッダーは [`docs/deployment.md`](../docs/deployment.md)、リクエストの相関付けとメトリクスの契約は [`docs/observability.md`](../docs/observability.md) を参照する。このファイルには、スタックを動かすコマンドと設定手順を記載する。
