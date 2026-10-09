# DataKeys の設計

この文書は、DataKeys の設計を話題ごとに索引する。
このモジュールが外へ約束することは、仕様の [DataKeys](../README.md) と各機能仕様が定める。
一つの機能だけの仕組みは、その機能の `design.md` に置く。

| 話題 | 記述した場所 |
| --- | --- |
| アーキテクチャ | [DataKeys のアーキテクチャ](architecture.md) |
| 設計判断 | [DataKeys の重要な設計判断](decisions.md) |
| アプリケーション | 該当なし：HTTP の操作は健全性の一覧の一つだけで、システムの[アプリケーション設計](../../../design/application/README.md)に従う |
| データ | 該当なし：鍵のバージョンの保存と破棄の扱いは[DEK のライフサイクル](../lifecycle/README.md)のモデルの節が、テーブルの定義はスキーマファイルが扱う |
| セキュリティ | [DataKeys のセキュリティ](security.md) |
| 信頼性 | 該当なし：再暗号化の再開と取りこぼしの回収は[DEK のライフサイクルの設計](../lifecycle/design.md)が扱う |
| 性能 | 該当なし：システムの[性能設計](../../../design/performance/README.md)に従い、このモジュールに割り当てた品質要件はない |
| オブザーバビリティ | 該当なし：システムの[オブザーバビリティ設計](../../../design/observability/README.md)に従い、このモジュールに固有の信号はない |
| 検証 | 該当なし：システムの[検証設計](../../../design/verification/README.md)に従う |
| インフラストラクチャ | 該当なし：マスターキーのプロバイダーの配置はシステムの[秘密情報の設計](../../../design/security/secrets.md)が扱う |
| リスク | [DataKeys のリスク](risks.md) |
