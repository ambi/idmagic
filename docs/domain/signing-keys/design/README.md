# SigningKeys の設計

この文書は、SigningKeys の設計を話題ごとに索引する。
この Context が外へ約束することは、仕様の [SigningKeys](../README.md) と各機能仕様が定める。
一つの機能だけの仕組みは、その機能の `design.md` に置く。

| 話題 | 記述した場所 |
| --- | --- |
| アーキテクチャ | [SigningKeys のアーキテクチャ](architecture.md) |
| 設計判断 | [SigningKeys の重要な設計判断](decisions.md) |
| アプリケーション | 該当なし：管理 API はシステムの[アプリケーション設計](../../../design/application/README.md)に従い、この Context に固有の方式はない |
| データ | [SigningKeys のデータ](data.md) |
| セキュリティ | 該当なし：鍵素材の置き場所は仕様の[モデル](../README.md#モデル)が、鍵の種類ごとの扱いはシステムの[秘密情報の設計](../../../design/security/secrets.md)が扱う |
| 信頼性 | 該当なし：提供元の障害の扱いは[鍵の提供元と健全性](../provider/README.md)のモデルの節が扱う |
| 性能 | 該当なし：システムの[性能設計](../../../design/performance/README.md)に従い、この Context に割り当てた品質要件はない |
| オブザーバビリティ | 該当なし：提供元の健全性は制御面の健全性の一覧で観測し、それ以外はシステムの[オブザーバビリティ設計](../../../design/observability/README.md)に従う |
| 検証 | 該当なし：システムの[検証設計](../../../design/verification/README.md)に従う |
| インフラストラクチャ | 該当なし：Vault の配置はシステムの[秘密情報の設計](../../../design/security/secrets.md)が扱う |
| リスク | [SigningKeys のリスク](risks.md) |
