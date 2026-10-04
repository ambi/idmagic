# Sourcing の設計

この文書は、Sourcing の設計を話題ごとに索引する。
この Context が外へ約束することは、仕様の [Sourcing](../README.md) と各機能仕様が定める。
一つの機能だけの仕組みは、その機能の `design.md` に置く。

| 話題 | 記述した場所 |
| --- | --- |
| アーキテクチャ | [Sourcing のアーキテクチャ](architecture.md) |
| 設計判断 | [Sourcing の重要な設計判断](decisions.md) |
| アプリケーション | 該当なし：SCIM のエンドポイントの形は[標準仕様](../standards.md)と TypeSpec が定める |
| データ | 該当なし：対応の記録は仕様の[モデル](../README.md#モデル)が、テーブルの定義はスキーマファイルが扱う |
| セキュリティ | 該当なし：トークンによるテナントの束縛は[SCIM による取り込み](../scim/README.md)のセキュリティ上の考慮の節が扱う |
| 信頼性 | 該当なし：作成の失敗の後始末は[SCIM による取り込みの設計](../scim/design.md)が扱う |
| 性能 | 該当なし：システムの[性能設計](../../../design/performance/README.md)に従い、この Context に割り当てた品質要件はない |
| オブザーバビリティ | 該当なし：システムの[オブザーバビリティ設計](../../../design/observability/README.md)に従い、この Context に固有の信号はない |
| 検証 | [Sourcing の検証](verification.md) |
| インフラストラクチャ | 該当なし：実行基盤に固有の要求はない |
| リスク | [Sourcing のリスク](risks.md) |
