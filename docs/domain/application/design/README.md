# Application の設計

この文書は、Application の設計を話題ごとに索引する。
この Context が外へ約束することは、仕様の [Application](../README.md) と各機能仕様が定める。
一つの機能だけの仕組みは、その機能の `design.md` に置く。

| 話題 | 記述した場所 |
| --- | --- |
| アーキテクチャ | [Application のアーキテクチャ](architecture.md) |
| 設計判断 | [Application の重要な設計判断](decisions.md) |
| アプリケーション | 該当なし：管理 API と UI はシステムの[アプリケーション設計](../../../design/application/README.md)に従い、この Context に固有の方式はない |
| データ | [Application のデータ](data.md) |
| セキュリティ | 該当なし：フェデレーションの関門は[割り当て](../assignment/README.md)と[サインインポリシー](../sign-in-policy/README.md)のセキュリティ上の考慮の節が扱う |
| 信頼性 | 該当なし：Context に固有の障害の経路はない |
| 性能 | 該当なし：システムの[性能設計](../../../design/performance/README.md)に従い、この Context に割り当てた品質要件はない |
| オブザーバビリティ | 該当なし：ポリシーによる拒否はドメインイベントで監査に残り、それ以外はシステムの[オブザーバビリティ設計](../../../design/observability/README.md)に従う |
| 検証 | 該当なし：システムの[検証設計](../../../design/verification/README.md)に従う |
| インフラストラクチャ | 該当なし：実行基盤に固有の要求はない |
| リスク | 該当なし：この Context に固有の既知の欠陥は見つかっていない |
