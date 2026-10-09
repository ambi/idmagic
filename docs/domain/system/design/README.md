# System の設計

この文書は、System の設計を話題ごとに索引する。
このモジュールが外へ約束することは、仕様の [System](../README.md) と各機能仕様が定める。
一つの機能だけの仕組みは、その機能の `design.md` に置く。

| 話題 | 記述した場所 |
| --- | --- |
| アーキテクチャ | [System のアーキテクチャ](architecture.md) |
| 設計判断 | [System の重要な設計判断](decisions.md) |
| アプリケーション | 該当なし：API と UI の方式はシステムの[アプリケーション設計](../../../design/application/README.md)が扱い、このモジュールは経路の分け方を[API の境界](../api-boundary/README.md)で定める |
| データ | 該当なし：このモジュールは永続化を持たない |
| セキュリティ | 該当なし：認可トランザクションとポータルのトークンは[ホステッド UI とポータルの設計](../hosted-ui/design.md)が、システム全体の認可は[認可設計](../../../design/security/authorization.md)が扱う |
| 信頼性 | 該当なし：プローブは[運用](../operations/README.md)が、システム全体の可用性は[信頼性設計](../../../design/reliability/README.md)が扱う |
| 性能 | 該当なし：過負荷のときの受け付けの制御は[アドミッションコントロールの設計](../admission-control/design.md)が、キャパシティの計画は[キャパシティ設計](../../../design/performance/capacity.md)が扱う |
| オブザーバビリティ | 該当なし：システムの[オブザーバビリティ設計](../../../design/observability/README.md)が扱う |
| 検証 | 該当なし：システムの[検証設計](../../../design/verification/README.md)に従う |
| インフラストラクチャ | 該当なし：システムの[インフラストラクチャ設計](../../../design/infrastructure/README.md)と[配置の設計](../../../design/architecture/deployment.md)が扱う |
| リスク | 該当なし：システム全体の[リスク](../../../design/architecture/risks.md)が扱う |
