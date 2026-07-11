---
depends_on: []
status: pending
authors: ["tn"]
risk: low
created_at: 2026-07-04
---

# Prometheus メトリクスエンドポイントと認証ゴールデンシグナルを公開し SLO 計測を可能にする

## Motivation
現状 idmagic は OpenTelemetry トレーシング (wi-107) の導入は計画しているが、
Prometheus/OpenMetrics 形式で収集できるアプリケーションメトリクスを一切公開して
いない。ADR-018 のリテンション表はメトリクス時系列を Prometheus で 30 日保持し
SLO 計測ウィンドウとする前提を置いているが、その計測点が実装されていない。

本番 IdP の運用では、トレース (個別リクエストの追跡) とは別に、集約された
ゴールデンシグナル — login 成功/失敗率、token 発行レイテンシとエラー率、
login throttle ヒット、依存先 (DB/Valkey) の健全性 — を時系列で継続監視し、
アラートと SLO の根拠にする必要がある。Prometheus scrape 可能な /metrics
エンドポイントと主要カウンタ/ヒストグラムを提供する。

## Scope
- **scl**: spec/contexts/system.yaml objectives に MetricsExposition を追加し、/metrics の公開、RED (Rate/Errors/Duration) を満たす HTTP メトリクスと認証ゴールデンシグナル (login/token/throttle) のカタログを宣言する。
- **go**: OpenTelemetry metrics または prometheus client を用いて /metrics (OpenMetrics) を公開する。ADR-017 (OTel を観測インターフェースとする) と整合させる。, HTTP RED メトリクス (リクエスト数・エラー・レイテンシ histogram) をミドルウェアで収集する。既存 http_request_aborts_total の abort 分類 label と整合させる。, 認証ドメインのゴールデンシグナル (login 成否 counter、token 発行 counter/latency、throttle ヒット counter) を計装する。
- **monitoring**: docker compose / deploy に Prometheus scrape 設定例と最小ダッシュボードを追加する。

## Out of Scope
- 分散トレーシング本体 (既存 wi-107)。
- アラートルール / PagerDuty 等の通知連携。
- 負荷試験資産 (既存 wi-11 の範囲)。

## Verification
- just verify
- 手動: /metrics を curl し、login/token/throttle と HTTP RED のメトリクスが OpenMetrics 形式で出力されることを確認する。

## Risk Notes
メトリクスの label カーディナリティ (tenant / client_id 等) が過大にならないよう、
高カーディナリティ次元は慎重に選ぶ。ADR-017/018 との整合 (OTel を第一の観測 IF と
するか prometheus client を併用するか) を実装前に確認する。
