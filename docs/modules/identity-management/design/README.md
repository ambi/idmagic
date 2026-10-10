# IdManagement の設計

この文書は、IdManagement の設計を設計領域ごとに索引する。
このモジュールが外へ約束することは、仕様の [IdManagement](../README.md) と各機能仕様が定める。
一つの機能だけの仕組みは、その機能の `design.md` に置く。

| 設計領域 | 内容 |
| --- | --- |
| [アーキテクチャ設計](architecture.md) | 文脈と範囲、解決戦略、構成要素、実行時の流れ |
| アプリケーション設計 | 該当なし：API と UI はシステムの[アプリケーション設計](../../../design/application/README.md)に従い、このモジュールに固有の方式はない |
| [データ設計](data.md) | Aggregate と永続化、保持と削除、排他と一意性、テナントの属性スキーマ |
| インフラストラクチャ設計 | 該当なし：`api` と `worker` の実行基盤はシステムの[インフラストラクチャ設計](../../../design/infrastructure/README.md)に従う |
| セキュリティ設計 | 該当なし：管理 API の認可は機能 [管理 API の認可](../admin-access/README.md) が、システム全体の認可は[認可設計](../../../design/security/authorization.md)が扱う |
| [信頼性設計](reliability.md) | 障害の症状、原因、直し方 |
| [性能設計](performance.md) | 実効ポリシーを超えるエクスポート |
| オブザーバビリティ設計 | 該当なし：システムの[オブザーバビリティ設計](../../../design/observability/README.md)に従い、このモジュールに固有の信号はない |
| 検証設計 | 該当なし：品質要件の確かめ方は[性能](performance.md)に置き、それ以外はシステムの[検証設計](../../../design/verification/README.md)に従う |

## 設計の記録

| 記録 | 内容 |
| --- | --- |
| [設計判断](decisions.md) | 代替案を比べた重要な判断 |
| [リスク](risks.md) | 既知の欠陥、技術的負債、受容したリスク |

横断的概念は次のとおりである。

| 文書 | 内容 |
| --- | --- |
| [CSV の往復変換](csv-transfer.md) | User と Group の CSV のエクスポートとインポートが共有する仕組み |
| [イベントと監査の記録](audit-events.md) | ドメインイベント、監査の記録、下流への通知の経路と、確定との関係 |
