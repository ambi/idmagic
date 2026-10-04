# Authentication の設計

この文書は、Authentication の設計を話題ごとに索引する。
この Context が外へ約束することは、仕様の [Authentication](../README.md) と各機能仕様が定める。
一つの機能だけの仕組みは、その機能の `design.md` に置く。

| 話題 | 記述した場所 |
| --- | --- |
| アーキテクチャ | [Authentication のアーキテクチャ](architecture.md) |
| 設計判断 | [Authentication の重要な設計判断](decisions.md) |
| アプリケーション | 該当なし：API と UI はシステムの[アプリケーション設計](../../../design/application/README.md)に従い、セルフサービス API の境界は[アカウントポータル](../account-portal/README.md)が扱う |
| データ | 該当なし：各 Aggregate の保存は各機能の設計が、認証のイベントの保存は[認証のイベントの記録](authentication-events.md)が扱う |
| セキュリティ | 該当なし：各機能仕様のセキュリティ上の考慮の節と、システムの[認可設計](../../../design/security/authorization.md)が扱う |
| 信頼性 | 該当なし：共有の一時的な状態のストアに到達できないときは失敗させる。各機能の扱いは各機能の設計が扱う |
| 性能 | 該当なし：システムの[性能設計](../../../design/performance/README.md)に従い、この Context に割り当てた品質要件はない |
| オブザーバビリティ | 該当なし：認証のイベントは[認証のイベントの記録](authentication-events.md)が扱い、それ以外はシステムの[オブザーバビリティ設計](../../../design/observability/README.md)に従う |
| 検証 | 該当なし：パースの境界のファジングは各機能の設計の検証の節に置き、それ以外はシステムの[検証設計](../../../design/verification/README.md)に従う |
| インフラストラクチャ | 該当なし：実行基盤に固有の要求はない |
| リスク | 該当なし：この Context に固有の既知の欠陥は見つかっていない |

横断的概念は次のとおりである。

| 文書 | 内容 |
| --- | --- |
| [認証のイベントの記録](authentication-events.md) | 個別の記録と 5 分の集計、相関、種別ごとの保持期間 |
