# IdGovernance の設計

この文書は、IdGovernance の設計を話題ごとに索引する。
この Context が外へ約束することは、仕様の [IdGovernance](../README.md) と各機能仕様が定める。
一つの機能だけの仕組みは、その機能の `design.md` に置く。

| 話題 | 記述した場所 |
| --- | --- |
| アーキテクチャ | [IdGovernance のアーキテクチャ](architecture.md) |
| 設計判断 | [IdGovernance の重要な設計判断](decisions.md) |
| アプリケーション | 該当なし：管理 API と UI はシステムの[アプリケーション設計](../../../design/application/README.md)に従い、この Context に固有の方式はない |
| データ | 該当なし：Aggregate とその関係は仕様の[モデル](../README.md#モデル)が、テーブルの定義はスキーマファイルが扱う |
| セキュリティ | 該当なし：テナントの境界は各機能仕様のセキュリティ上の考慮の節が扱う |
| 信頼性 | 該当なし：Job の投入の失敗と再試行の扱いは[ワークフローの実行の設計](../workflow-run/design.md)が扱う |
| 性能 | 該当なし：システムの[性能設計](../../../design/performance/README.md)に従い、この Context に割り当てた品質要件はない |
| オブザーバビリティ | 該当なし：実行とステップの結果はドメインイベントで監査に残り、それ以外はシステムの[オブザーバビリティ設計](../../../design/observability/README.md)に従う |
| 検証 | 該当なし：システムの[検証設計](../../../design/verification/README.md)に従う |
| インフラストラクチャ | 該当なし：実行基盤に固有の要求はない |
| リスク | [IdGovernance のリスク](risks.md) |
