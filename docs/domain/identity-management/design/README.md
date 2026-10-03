# IdManagement の設計

この文書は、IdManagement の設計を話題ごとに索引する。
この Context が外へ約束することは、仕様の [IdManagement](../README.md) と各機能仕様が定める。
一つの機能だけの仕組みは、その機能の `design.md` に置く。

| 話題 | 記述した場所 |
| --- | --- |
| アーキテクチャ | [IdManagement のアーキテクチャ](architecture.md) |
| 設計判断 | [IdManagement の重要な設計判断](decisions.md) |
| アプリケーション | 該当なし：API と UI はシステムの[アプリケーション設計](../../../design/application/README.md)に従い、この Context に固有の方式はない |
| データ | [IdManagement のデータ](data.md) |
| セキュリティ | 該当なし：管理 API の認可は機能 [管理 API の認可](../common/admin-access/README.md) が、システム全体の認可は[認可設計](../../../design/security/authorization.md)が扱う |
| 信頼性 | [IdManagement の信頼性](reliability.md) |
| 性能 | [IdManagement の性能](performance.md) |
| オブザーバビリティ | 該当なし：システムの[オブザーバビリティ設計](../../../design/observability/README.md)に従い、この Context に固有の信号はない |
| 検証 | 該当なし：品質要件の確かめ方は[性能](performance.md)に置き、それ以外はシステムの[検証設計](../../../design/verification/README.md)に従う |
| インフラストラクチャ | 該当なし：`api` と `worker` の実行基盤はシステムの[インフラストラクチャ設計](../../../design/infrastructure/README.md)に従う |
| リスク | [IdManagement のリスク](risks.md) |

横断的概念は次のとおりである。

| 文書 | 内容 |
| --- | --- |
| [CSV の往復変換](csv-transfer.md) | User と Group の CSV のエクスポートとインポートが共有する仕組み |
| [イベントと監査の記録](audit-events.md) | ドメインイベント、監査の記録、下流への通知の経路と、確定との関係 |
