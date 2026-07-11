---
depends_on: [wi-112-prometheus-metrics-and-authentication-golden-signals]
status: pending
authors: ["tn"]
risk: medium
created_at: 2026-06-15
---

# idmagic の Kubernetes・監視・負荷試験資産を整備する

## Motivation
TypeScript 実装の廃止に伴い、旧実装向けの Kubernetes manifests、
Prometheus / Grafana 設定、k6 負荷試験も削除した。これらは実装固有の
endpoint、メトリクス名、コンテナ構成を参照していたため、そのまま
idmagic に移すと実態と乖離する。

一方、本番運用と SCL の objectives を検証するには、Go 実装を基準にした
配備・観測・性能検証の資産が必要である。本 WI で現行の Docker image、
OpenTelemetry 出力、HTTP 契約から再構築する。

## Scope
- **kubernetes**: idmagic/infra/k8s/base に Deployment / Service / ConfigMap / NetworkPolicy / PodDisruptionBudget を追加する。, idmagic API、UI gateway、event relay を責務ごとに分離する。, readiness / liveness probe は Go 実装の `/health` を使う。, signing key rotation の定期実行方法は、既存 use case と実行可能な CLI の有無を確認してから CronJob または別 WI に切り分ける。, dev / prod overlay を用意し、image tag、replica、resource request / limit、外部 secret 参照だけを環境差分にする。
- **monitoring**: OpenTelemetry Collector から取得できる Go 実装の metric 名を確認し、 Prometheus recording rule と alert rule を作る。, SCL objectives の availability / latency / error-rate と alert を 対応付ける。, Grafana dashboard に request rate、error rate、latency、認証失敗、 token 発行、PostgreSQL / Valkey / relay の主要状態を表示する。, ServiceMonitor は Prometheus Operator 利用時だけ適用できる構成にする。
- **load_testing**: k6 で authorization_code + PKCE、token refresh、client_credentials の 最小シナリオを作る。, SCL objectives から閾値を読み取るか、同期確認できる単一設定に集約する。, tenant 境界を越えるデータ再利用をせず、実行ごとに独立した client / user または安全な seed を使う。, ローカル Docker Compose と CI の両方で短時間 smoke を実行できるようにする。
- **documentation**: idmagic/README.md に配備、監視、負荷試験の実行方法を追加する。, 各資産が SCL objectives のどの保証義務を検証するかを記録する。

## Out of Scope
- 特定クラウド専用の Terraform / Pulumi。
- 本番 secret のリポジトリ保存。
- マルチリージョン構成、DR、バックアップ・リストア自動化。
- アプリケーションロジックや HTTP API の変更。

## Plan
- [[ADR-078-kubernetes-health-probes-and-graceful-drain]] で実装済みの `/health/live`、`/health/ready`、`/health/startup` と drain 挙動を Deployment の probe / terminationGracePeriodSeconds に接続する。probe を新設する作業には戻らない。
- `idmagic-api`、frontend gateway、`idmagic-relay` は別 Deployment/Service とし、`PERSISTENCE=postgres_valkey` の API だけを複数 replica 化する。署名鍵 rotator は実行バイナリが未整備なので [[wi-23-signing-key-rotation-scheduler]] に委譲し、本 WI の CronJob から外す。
- `deploy/k8s/base` に Kustomize base、`deploy/k8s/overlays/{dev,prod}` に image tag・replica・resource・external Secret 参照だけを置く。リポジトリに secret 値を置かず、PostgreSQL/Valkey への egress と ingress 経路を NetworkPolicy で明示する。
- 監視資産は [[wi-112-prometheus-metrics-and-authentication-golden-signals]] の metric catalog と `/metrics` を前提にするため、先に dashboard/rule の期待 metric を棚卸しし、不足するアプリ計装を本 WI に混ぜない。
- k6 は既存 HTTP 契約を使う authorization_code+PKCE、refresh、client_credentials の3シナリオを独立させ、SCL objective の latency/error-rate を threshold の正本へ変換する。テストデータ作成と破棄をシナリオ内に閉じる。

## Tasks
- [ ] T001 [Inventory] `just --list` と各 command entry point を確認し、API/UI/relay の port、health path、設定、永続依存、graceful shutdown 時間を配備表にする。
- [ ] T002 [Kubernetes] `deploy/k8s/base` に API/UI/relay の Deployment・Service・ConfigMap、API の PDB、ServiceAccount と共通 label を追加する。
- [ ] T003 [Security] namespace 内 ingress と PostgreSQL/Valkey/DNS への必要 egress だけを許す NetworkPolicy、read-only filesystem 等の pod securityContext、external Secret 参照を追加する。
- [ ] T004 [Overlays] dev/prod overlay に replica、resource request/limit、image digest/tag と環境別 Secret 名を定義し、base に環境差分が混ざらないことを確認する。
- [ ] T005 [Monitoring] wi-112 の metric 名を確認して PrometheusRule、recording rule、Grafana dashboard、任意適用の ServiceMonitor を作る。未実装 metric は依存として明記し、架空の query を置かない。
- [ ] T006 [Load] PKCE、refresh rotation、client_credentials の k6 scenario、tenant-local seed/cleanup と SCL objective 由来の threshold を追加する。
- [ ] T007 [Command Map] manifest build/validation、monitoring lint、k6 smoke に必要な recipe が無ければ `justfile` に追加する。
- [ ] T008 [Docs/Verify] 配備・rollback・dashboard・負荷試験手順を記録し、manifest validation、短時間 k6 smoke、`just verify` を実行する。

## Verification
- Kubernetes manifests を kustomize build し、kubeconform で検証する。
- Prometheus rules を promtool check rules で検証する。
- Grafana dashboard JSON を構文検証する。
- Docker Compose 上で k6 smoke を実行し、閾値を満たす。
- `just test-go-race` を実行し、既存挙動に回帰がない。

## Risk Notes
旧 TypeScript 実装の設定を名前だけ変えて移植すると、存在しない metric や
endpoint を監視する構成になる。最初に Go 実装から観測可能な信号を列挙し、
各 manifest / rule / scenario が現行コードへ辿れる状態で追加する。
