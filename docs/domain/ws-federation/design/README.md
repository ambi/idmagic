# WsFederation の設計

この文書は、WsFederation の設計を話題ごとに索引する。
この Context が外へ約束することは、仕様の [WsFederation](../README.md) と各機能仕様が定める。
一つの機能だけの仕組みは、その機能の `design.md` に置く。

| 話題 | 記述した場所 |
| --- | --- |
| アーキテクチャ | [WsFederation のアーキテクチャ](architecture.md) |
| 設計判断 | [WsFederation の重要な設計判断](decisions.md) |
| アプリケーション | 該当なし：管理 API はシステムの[アプリケーション設計](../../../design/application/README.md)に従い、プロトコルの形は[標準仕様](../standards.md)が定める |
| データ | 該当なし：RP の項目は仕様の[モデル](../README.md#モデル)が、テーブルの定義はスキーマファイルが扱う |
| セキュリティ | [WsFederation のセキュリティ](security.md) |
| 信頼性 | 該当なし：Entra のフェデレーションの障害の切り分けは[RP と Entra フェデレーションの管理の設計](../relying-party/design.md)が扱い、それ以外に固有の仕組みはない |
| 性能 | 該当なし：システムの[性能設計](../../../design/performance/README.md)に従い、この Context に割り当てた品質要件はない |
| オブザーバビリティ | 該当なし：発行と拒否はドメインイベントで監査に残り、それ以外はシステムの[オブザーバビリティ設計](../../../design/observability/README.md)に従う |
| 検証 | [WsFederation の検証](verification.md) |
| インフラストラクチャ | 該当なし：実行基盤に固有の要求はない |
| リスク | 該当なし：この Context に固有の既知の欠陥は見つかっていない |

横断的概念は次のとおりである。

| 文書 | 内容 |
| --- | --- |
| [フェデレーションメタデータ](federation-metadata.md) | `federationmetadata.xml` と MEX が広告する内容と、署名証明書の公開 |
