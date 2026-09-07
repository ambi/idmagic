# 品質要求

品質要求は、目標値と測定境界を一か所に置き、アーキテクチャ、設計、監視、受入試験が同じ ID を参照できるようにする。
現時点で数値を持つのは性能、非 5xx 比率、可用性、容量受入である。
セキュリティとアクセシビリティの義務は[全体規範](../standards.md)と各 Context の規範が持つ。

## 証拠の区分

- **Specification target**：満たすべきサービス目標または容量受入目標であり、未達は構成または実装の改善を要する。
- **Planning assumption**：構成を算出するための未実測の入力値であり、達成済みであることを示さない。
- **Measurement**：日付、ソース版、実行環境、データ分布、負荷構成、試験時間、結果の保存先を伴う再現可能な実測値である。

現時点で Measurement はない。
ステージングでの測定は [wi-282](../../work-items/wi-282-staging-load-testing-and-capacity-validation.md) が扱う。

## 測定境界

測定境界は、ゲートウェイから API プロセスへ到達し、`http_requests_total` と `http_request_duration_seconds` に記録された HTTP リクエストである。
登録済みのルートパターンとメソッドで集約し、解決済みのパスやテナント ID では集約しない。
現在の計装はゲートウェイより外側の通信を観測しないため、利用者からゲートウェイまでを含む目標としては扱わない。

レイテンシーの母集団には API プロセスが完了した全レスポンスを状態コードにかかわらず含める。
クライアントがレスポンス前に切断したリクエストは母集団から除外するが、成功として数えず `http_request_aborts_total` で別に監視する。
計画停止、配備、依存先障害、流量制限、認証失敗、入力不正は除外しない。

非 5xx 比率は、対象母集団のうち状態コードが 500 未満であるレスポンスの割合とする。
可用性は 5 分の時間区分で評価し、対象リクエストが一件以上完了した区分だけを観測対象とする。
API のスクレイプ対象が区分を通じて一つ以上利用可能で、対象リクエストに非 5xx レスポンスが一件以上あれば利用可能とする。
要求がない区分は母集団から除外し、スクレイプ対象が失われた区分は利用不能として数える。

## サービス目標

次の Specification target は 30 日の移動窓で評価する。
他の文書、アラート、負荷試験は ID を参照し、数値を再掲しない。

| ID | 母集団 | 目標 | 指標 |
| --- | --- | --- | --- |
| SLO-LOGIN-LATENCY | `POST /api/auth/login` | p99 ≤ 300 ms | `http_request_duration_seconds` |
| SLO-AUTHORIZE-LATENCY | `GET /authorize` | p99 ≤ 500 ms | `http_request_duration_seconds` |
| SLO-PAR-LATENCY | `POST /par` | p99 ≤ 200 ms | `http_request_duration_seconds` |
| SLO-TOKEN-LATENCY | `POST /token` | p99 ≤ 300 ms | `http_request_duration_seconds` |
| SLO-INTROSPECT-LATENCY | `POST /introspect` | p99 ≤ 50 ms | `http_request_duration_seconds` |
| SLO-REVOKE-LATENCY | `POST /revoke` | p99 ≤ 100 ms | `http_request_duration_seconds` |
| SLO-USERINFO-LATENCY | `GET` と `POST /userinfo` | p99 ≤ 100 ms | `http_request_duration_seconds` |
| SLO-DISCOVERY-LATENCY | OIDC Discovery と OAuth Authorization Server Metadata | p99 ≤ 20 ms | `http_request_duration_seconds` |
| SLO-JWKS-LATENCY | 既定テナントと明示テナントの JWKS | p99 ≤ 20 ms | `http_request_duration_seconds` |
| SLO-REGISTER-LATENCY | `POST /register` | p99 ≤ 500 ms | `http_request_duration_seconds` |
| SLO-DEVICE-AUTHORIZATION-LATENCY | `POST /device_authorization` | p99 ≤ 300 ms | `http_request_duration_seconds` |
| SLO-FEDERATION-CALLBACK-LATENCY | OIDC と SAML の外部連携コールバック | p95 ≤ 2 s | `http_request_duration_seconds` |
| SLO-PRIMARY-ERRORS | ログイン、認可、PAR、トークン、失効、UserInfo、動的登録、デバイス認可 | 非 5xx ≥ 99.9% | `http_requests_total` |
| SLO-LOOKUP-ERRORS | introspection、Discovery、Authorization Server Metadata、JWKS | 非 5xx ≥ 99.99% | `http_requests_total` |
| SLO-OAUTH2-AVAILABILITY | OAuth2 と OIDC の対象エンドポイント全体 | ≥ 99.9% | `http_requests_total` とスクレイプ状態 |
| SLO-TOKEN-AVAILABILITY | `/token` | ≥ 99.95% | `http_requests_total` とスクレイプ状態 |

外部連携コールバックの非 5xx 目標と、セッション一覧のページ別レイテンシー目標は、現在の計装では母集団を定義できないため設けない。

## 容量受入れ

[参照運用プロファイル](../design/performance/capacity.md)のデータを投入し、ピークリクエストを同時に 15 分間ウォームアップした後、60 分間処理する。

| ID | エンドポイント | 必要なリクエスト率 | 満たす目標 |
| --- | --- | --- | --- |
| CAP-TOKEN-THROUGHPUT | `/token` | 5,000 rps | SLO-TOKEN-LATENCY と SLO-PRIMARY-ERRORS |
| CAP-AUTHORIZE-THROUGHPUT | `/authorize` | 1,000 rps | SLO-AUTHORIZE-LATENCY と SLO-PRIMARY-ERRORS |
| CAP-INTROSPECT-THROUGHPUT | `/introspect` | 20,000 rps | SLO-INTROSPECT-LATENCY と SLO-LOOKUP-ERRORS |

流量制限で返した 429 は非 5xx 比率には含むが、Required rate の処理済み要求には数えない。

## 復旧目標

ローカルの復元訓練は経路の動作を確かめるが、本番の RPO と RTO を確定する証拠ではない。
本番のデータ種別ごとの RPO と RTO は未確定であり、[復旧設計](../design/reliability/recovery.md)が不足と確定条件を示す。

## 品質特性の被覆

ISO/IEC 25010:2023 の品質モデルを点検網として使う。
現在の要求が定量化していない機能適合性、相互運用性、使用性、保守性、移植性、安全性については、該当する規範または検証を参照し、測定可能な目標の追加は [wi-419](../../work-items/wi-419-quantification-beyond-performance.md) が扱う。
