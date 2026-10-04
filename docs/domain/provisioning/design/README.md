# Provisioning の設計

この文書は、Provisioning の設計を話題ごとに索引する。
この Context が外へ約束することは、仕様の [Provisioning](../README.md) と各機能仕様が定める。
一つの機能だけの仕組みは、その機能の `design.md` に置く。

| 話題 | 記述した場所 |
| --- | --- |
| アーキテクチャ | [Provisioning のアーキテクチャ](architecture.md) |
| 設計判断 | [Provisioning の重要な設計判断](decisions.md) |
| アプリケーション | 該当なし：管理 API はシステムの[アプリケーション設計](../../../design/application/README.md)に従い、下流の形は[標準仕様](../standards.md)が定める |
| データ | 該当なし：Aggregate とその関係は仕様の[モデル](../README.md#モデル)が、テーブルの定義はスキーマファイルが扱う |
| セキュリティ | 該当なし：下流の URL の検証とテナントの境界は[接続の管理](../connection/README.md)のセキュリティ上の考慮の節が扱う |
| 信頼性 | 該当なし：隔離からの回復と通知は[同期の設計](../synchronization/design.md)が、フル同期の完了は[接続の管理の設計](../connection/design.md)が扱う |
| 性能 | 該当なし：システムの[性能設計](../../../design/performance/README.md)に従い、この Context に割り当てた品質要件はない |
| オブザーバビリティ | 該当なし：反映の結果はドメインイベントで監査に残り、それ以外はシステムの[オブザーバビリティ設計](../../../design/observability/README.md)に従う |
| 検証 | 該当なし：システムの[検証設計](../../../design/verification/README.md)に従う |
| インフラストラクチャ | 該当なし：下流の SaaS への到達性のほかに、実行基盤に固有の要求はない |
| リスク | 該当なし：この Context に固有の既知の欠陥は見つかっていない |
