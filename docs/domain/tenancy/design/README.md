# Tenancy の設計

この文書は、Tenancy の設計を話題ごとに索引する。
この Context が外へ約束することは、仕様の [Tenancy](../README.md) と各機能仕様が定める。
一つの機能だけの仕組みは、その機能の `design.md` に置く。

| 話題 | 記述した場所 |
| --- | --- |
| アーキテクチャ | [Tenancy のアーキテクチャ](architecture.md) |
| 設計判断 | [Tenancy の重要な設計判断](decisions.md) |
| アプリケーション | 該当なし：API と UI はシステムの[アプリケーション設計](../../../design/application/README.md)に従い、この Context に固有の方式はない |
| データ | [Tenancy のデータ](data.md) |
| セキュリティ | 該当なし：テナント分離の規則は[認可設計](../../../design/security/authorization.md)が、各機能の考慮は機能仕様のセキュリティ上の考慮の節が扱う |
| 信頼性 | 該当なし：Context に共通する仕組みはなく、ブランド設定とテナント設定の退避の扱いは各機能の設計に置く |
| 性能 | 該当なし：システムの[性能設計](../../../design/performance/README.md)に従い、この Context に割り当てた品質要件はない |
| オブザーバビリティ | 該当なし：システムの[オブザーバビリティ設計](../../../design/observability/README.md)に従い、この Context に固有の信号はない |
| 検証 | 該当なし：システムの[検証設計](../../../design/verification/README.md)に従う |
| インフラストラクチャ | 該当なし：`tenant_base_domain` の要否はテナントの解決の設計が扱い、それ以外はシステムの[インフラストラクチャ設計](../../../design/infrastructure/README.md)に従う |
| リスク | [Tenancy のリスク](risks.md) |
