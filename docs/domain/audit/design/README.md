# Audit の設計

この文書は、Audit の設計を話題ごとに索引する。
この Context が外へ約束することは、仕様の [Audit](../README.md) と各機能仕様が定める。
一つの機能だけの仕組みは、その機能の `design.md` に置く。

| 話題 | 記述した場所 |
| --- | --- |
| アーキテクチャ | [Audit のアーキテクチャ](architecture.md) |
| 設計判断 | [Audit の重要な設計判断](decisions.md) |
| アプリケーション | 該当なし：管理 API はシステムの[アプリケーション設計](../../../design/application/README.md)に従い、この Context に固有の方式はない |
| データ | 該当なし：検索属性の副表は[監査イベントの検索の設計](../event-search/design.md)が、保持期間と削除は[データのライフサイクル](../../../design/data/lifecycle.md)が扱う |
| セキュリティ | 該当なし：経路による範囲の固定とカーソルの扱いは[監査イベントの検索](../event-search/README.md)のセキュリティ上の考慮の節が扱う |
| 信頼性 | 該当なし：記録の追記の失敗の扱いは配信点の側にあり、[Audit のリスク](risks.md)に載せる |
| 性能 | 該当なし：システムの[性能設計](../../../design/performance/README.md)に従い、この Context に割り当てた品質要件はない |
| オブザーバビリティ | 該当なし：システムの[オブザーバビリティ設計](../../../design/observability/README.md)に従い、この Context に固有の信号はない |
| 検証 | [Audit の検証](verification.md) |
| インフラストラクチャ | 該当なし：実行基盤に固有の要求はない |
| リスク | [Audit のリスク](risks.md) |
