# システム境界

この文書は、IdMagic が担う責任と、利用者・外部システム・運用基盤が担う責任の境界を示す。
IdMagic のシステム境界には、ブラウザー UI、公開プロトコルと API、管理 API、永続ジョブとバッチが含まれる。
クラウドのコンピューティング、データベースサービス、DNS、TLS 証明書、上流の IdP とアイデンティティ源、下流の送信先、監視と通知のサービスは境界外にあり、定義した契約を通じて接続する。

次の C4 のシステムコンテキスト図は、人、IdMagic、外部システムと、その間の関係の目的を示す。
矢印は利用または連携の向きであり、物理的な直接接続を意味しない。
ゲートウェイ、プロセス、ネットワークの経路は、[ランタイム](runtime.md)、[デプロイメントアーキテクチャ](deployment.md)、[ネットワーク設計](../infrastructure/network.md)で示す。

```mermaid
---
config:
  c4:
    c4ShapeMargin: 90
---
C4Context
  Person(human, "利用者と管理者", "ブラウザーから使う")
  System_Ext(client, "連携アプリケーション", "OAuth、OIDC、SAML、SCIM のクライアント")
  Person(operator, "運用者")
  System_Ext(upstream, "上流の IdP とアイデンティティ源")
  System(idmagic, "IdMagic", "認証、管理、プロビジョニング")
  System_Ext(platform, "運用基盤", "デプロイ、シークレット、監視")
  System_Ext(downstream, "下流の送信先", "SCIM サーバー、SSF の受信側、SMTP サーバー")
  SystemDb_Ext(postgres, "PostgreSQL", "データベースサービス")
  Rel(human, idmagic, "認証、管理、セルフサービスに使う")
  Rel(client, idmagic, "プロトコルと API で連携する")
  Rel(idmagic, upstream, "認証を委ね、属性を取り込む")
  Rel(idmagic, downstream, "プロビジョニング、イベント、メールを送る")
  Rel(idmagic, postgres, "状態を保存する")
  Rel(operator, platform, "構成し、監視する")
  Rel(platform, idmagic, "デプロイし、シークレットを渡し、観測する")
  UpdateRelStyle(human, idmagic, $offsetX="-205", $offsetY="-10")
  UpdateRelStyle(client, idmagic, $offsetX="10")
  UpdateRelStyle(idmagic, upstream, $offsetX="-74", $offsetY="-8")
  UpdateRelStyle(platform, idmagic, $offsetX="-50", $offsetY="58")
  UpdateRelStyle(idmagic, downstream, $offsetX="-235", $offsetY="10")
  UpdateRelStyle(idmagic, postgres, $offsetX="10")
  UpdateRelStyle(operator, platform, $offsetX="10")
  UpdateLayoutConfig($c4ShapeInRow="3", $c4BoundaryInRow="1")
```

| 外部当事者 | IdMagic の責任 | 外部当事者の責任 |
| --- | --- | --- |
| 利用者とテナント管理者 | 認証、管理、セルフサービスの UI と API | 資格情報と登録情報を適切に扱う |
| 連携アプリケーション | 宣言したプロトコルと標準仕様を実装する | リダイレクト URI、鍵、クライアントの資格情報を管理する |
| 上流の IdP とアイデンティティ源 | レスポンスを検証し、内部のモデルへ変換する | 発行した識別子とアサーションの完全性を保つ |
| 下流の送信先 | 再試行と失敗を、定義した範囲で扱う | 受理の結果と可用性を、契約どおりに返す |
| 基盤の提供者と運用者 | ヘルスチェック、メトリクス、ログ、デプロイできる成果物を提供する | DNS、TLS、シークレット、データベース、コンピューティング、通知経路を構成する |

## ゲートウェイとプラットフォームの実体

利用者と連携アプリケーションの通信を受ける「ゲートウェイ」と、図の「運用基盤」は役割であり、それを担う製品はシステム境界の外で決まる。
[デプロイメントアーキテクチャ](deployment.md)のデプロイプロファイルごとの担い手は、次のとおりである。
構成ファイルがまだ無いものは、末尾に（仮）と書く。

| 役割 | ローカル Docker Compose | 汎用 Kubernetes | Google Cloud |
| --- | --- | --- | --- |
| ゲートウェイ | `frontend` サービスの Caddy が、一つのコンテナで同一オリジンを組み立てる | `idmagic-frontend` の Caddy が同一オリジンを組み立て、TLS の終端はクラスターの Ingress が担う | GKE Ingress で構成する外部アプリケーションロードバランサーが TLS を終端し、その後ろの `idmagic-frontend` が同一オリジンを組み立てる。Cloud CDN が静的アセットをキャッシュする（仮） |
| デプロイの基盤 | Docker Compose | Kubernetes のコントロールプレーンとリリースパイプライン | GKE Autopilot、Artifact Registry、リリースパイプライン（仮） |
| シークレットの基盤 | 構成ファイルに書いた開発用の値 | プラットフォームが先に作る Kubernetes Secret | Secret Manager と Cloud KMS（仮） |
| 監視の基盤 | Prometheus、Loki、Alloy、Grafana のコンテナ | `infra/k8s/monitoring/` の同じ仕組みのマニフェスト、または Prometheus Operator | Managed Service for Prometheus と Cloud Logging（仮）。収集の契約は[オブザーバビリティ設計](../observability/README.md)で定める |

信頼しない入力と制御は[脅威モデル](../security/threat-model.md)で、ネットワークの境界は[ネットワーク設計](../infrastructure/network.md)で定める。
