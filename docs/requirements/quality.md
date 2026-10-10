# 品質要件

この文書は、IdMagic の品質要件を定める。品質要件は、目標値と測定境界を一か所に置き、アーキテクチャ、設計、監視、受入試験が同じ ID を参照できるようにする。
現時点で数値を持つのは性能、非 5xx 比率、可用性、キャパシティ受入である。
セキュリティ、アクセシビリティ、互換性などの数値目標のない品質特性の要件は、[品質特性ごとの要件](#品質特性ごとの要件)にまとめる。

## 証拠の区分

| 区分 | 意味 |
| --- | --- |
| 仕様目標 | 満たすべきサービス目標またはキャパシティ受入目標であり、未達は構成または実装の改善を要する |
| 仮定値 | 構成を算出するための未実測の入力値であり、達成済みであることを示さない |
| 実測値 | 日付、ソースのバージョン、実行環境、データ分布、負荷構成、試験時間、結果の保存先を伴う再現可能な実測値 |

現時点で実測値はない。
ステージングでの測定も、まだ行っていない。

## 測定境界

測定境界は、ゲートウェイから API プロセスへ到達し、`http_requests_total` と `http_request_duration_seconds` に記録された HTTP リクエストである。
登録済みのルートパターンとメソッドで集約し、解決済みのパスやテナント ID では集約しない。
現在の計装はゲートウェイより外側の通信を観測しないため、利用者からゲートウェイまでを含む目標としては扱わない。

レイテンシーの母集団には API プロセスが完了した全レスポンスを状態コードにかかわらず含める。
クライアントがレスポンス前に切断したリクエストは母集団から除外するが、成功として数えず `http_request_aborts_total` で別に監視する。
計画停止、デプロイ、依存先障害、流量制限、認証失敗、入力不正は除外しない。

非 5xx 比率は、対象母集団のうち状態コードが 500 未満であるレスポンスの割合とする。
可用性は 5 分の時間区分で評価し、対象リクエストが一件以上完了した区分だけを観測対象とする。
API のメトリクス収集対象が区分を通じて一つ以上利用可能で、対象リクエストに非 5xx レスポンスが一件以上あれば利用可能とする。
要求がない区分は母集団から除外し、メトリクス収集対象が失われた区分は利用不能として数える。

## サービス目標

次の仕様目標は 30 日の移動窓で評価する。
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
| SLO-JWKS-LATENCY | デフォルトテナントと明示テナントの JWKS | p99 ≤ 20 ms | `http_request_duration_seconds` |
| SLO-REGISTER-LATENCY | `POST /register` | p99 ≤ 500 ms | `http_request_duration_seconds` |
| SLO-DEVICE-AUTHORIZATION-LATENCY | `POST /device_authorization` | p99 ≤ 300 ms | `http_request_duration_seconds` |
| SLO-FEDERATION-CALLBACK-LATENCY | OIDC と SAML の外部連携コールバック | p95 ≤ 2 s | `http_request_duration_seconds` |
| SLO-PRIMARY-ERRORS | ログイン、認可、PAR、トークン、失効、UserInfo、動的登録、デバイス認可 | 非 5xx ≥ 99.9% | `http_requests_total` |
| SLO-LOOKUP-ERRORS | introspection、Discovery、Authorization Server Metadata、JWKS | 非 5xx ≥ 99.99% | `http_requests_total` |
| SLO-OAUTH2-AVAILABILITY | OAuth2 と OIDC の対象エンドポイント全体 | ≥ 99.9% | `http_requests_total` とメトリクス収集状態 |
| SLO-TOKEN-AVAILABILITY | `POST /token` | ≥ 99.95% | `http_requests_total` とメトリクス収集状態 |

外部連携コールバックの非 5xx 目標と、セッション一覧のページ別レイテンシー目標は、現在の計装では母集団を定義できないため設けない。

数値目標は、ログイン、OAuth 2.0、OpenID Connect、および上流の OpenID Connect・SAML 連携コールバックを対象とする。
SAML IdP、管理 API、SCIM、監査、ワーカー、バッチの数値目標は、測定境界と運用実績がないため未定義である。
セキュリティとアクセシビリティの要件は、[品質特性ごとの要件](#品質特性ごとの要件)が数値目標ではない形で定める。

## キャパシティ受入れ

[想定ワークロード](../design/performance/capacity.md#想定ワークロード)のデータを投入し、ピークリクエストを同時に 15 分間ウォームアップした後、60 分間処理する。

| ID | エンドポイント | 必要なリクエスト率 | 満たす目標 |
| --- | --- | --- | --- |
| CAP-TOKEN-THROUGHPUT | `POST /token` | 5,000 rps | SLO-TOKEN-LATENCY と SLO-PRIMARY-ERRORS |
| CAP-AUTHORIZE-THROUGHPUT | `GET /authorize` | 1,000 rps | SLO-AUTHORIZE-LATENCY と SLO-PRIMARY-ERRORS |
| CAP-INTROSPECT-THROUGHPUT | `POST /introspect` | 20,000 rps | SLO-INTROSPECT-LATENCY と SLO-LOOKUP-ERRORS |

流量制限で返した 429 は非 5xx 比率には含むが、Required rate の処理済み要求には数えない。

## 復旧目標

ローカルの復元試験は経路の動作を確かめるが、本番の RPO と RTO を確定する証拠ではない。
本番のデータ種別ごとの RPO と RTO は未確定であり、[リカバリ設計](../design/reliability/recovery.md)が不足と確定条件を示す。

## 品質特性ごとの要件

ISO/IEC 25010:2023 の品質特性ごとに、IdMagic に求める要件と、その規範と検証の場所を示す。
外部規範の条項 id は[全体の標準仕様](standards.md)と各モジュールの `standards.md` が定め、この表はその規範への準拠を要件として宣言する。
数値目標のない特性は、数値目標が未定義であることを明示する。

| 品質特性 | 要件 | 規範と検証 |
| --- | --- | --- |
| 機能適合性 | [機能要件](functional.md)の機能群と、割り当て先のモジュールの機能仕様の `REQ-*` を満たす。数値目標は未定義 | 各モジュールの機能仕様、[検証設計](../design/verification/README.md#要件の群ごとの証拠) |
| 性能効率性 | [サービス目標](#サービス目標)のレイテンシーの目標と、[キャパシティ受入れ](#キャパシティ受入れ)の `CAP-*` を満たす | この文書、[性能設計](../design/performance/README.md) |
| 互換性 | 連携の実装者が、OAuth 2.0、OpenID Connect、SAML 2.0、WS-Federation、SCIM 2.0 の標準どおりに接続できる。公開する API の契約は OpenAPI 3.1.1 で記述する。数値目標は未定義 | 各モジュールの `standards.md`、[全体の標準仕様](standards.md) |
| インタラクション能力 | 認証の操作は、[全体の標準仕様](standards.md)が採用した WCAG 2.2 の条項（`WCAG22-*`）を満たし、キーボードだけで完了でき、結果と誤りを支援技術へ伝える。数値目標は未定義 | [全体の標準仕様](standards.md)、[ユーザーインターフェース設計](../design/application/user-interface.md) |
| 信頼性 | [サービス目標](#サービス目標)の非 5xx 比率と可用性の目標、[復旧目標](#復旧目標)を満たす | この文書、[信頼性設計](../design/reliability/README.md) |
| セキュリティ | 各プロトコルのセキュリティ上の義務は各モジュールの `standards.md` の規範を満たす。個人データの扱いは[全体の標準仕様](standards.md#general-data-protection-regulation)の `GDPR-*` を満たす。[脅威モデル](../design/security/threat-model.md)のすべての脅威に、対応する統制か、再検討の条件を伴う受容を記録する。数値目標は未定義 | [セキュリティ設計](../design/security/README.md)、[セキュリティ検証設計](../design/verification/security.md) |
| 保守性 | モジュールの間の依存を[バックエンド設計](../design/application/backend.md#モジュール間の依存規則)の規則に保ち、境界検査で違反を拒否する。数値目標は未定義 | [バックエンド設計](../design/application/backend.md#検査と負債) |
| 柔軟性 | ローカルの Docker Compose と汎用の Kubernetes の二つのデプロイプロファイルで動く。数値目標は未定義 | [アーキテクチャ設計の制約](../design/architecture/README.md#制約)、[デプロイメントアーキテクチャ](../design/architecture/deployment.md#デプロイプロファイル) |
| 安全性 | 人の身体、財産、環境への危害を直接もたらす機能を持たないため、要件を設けない | — |
