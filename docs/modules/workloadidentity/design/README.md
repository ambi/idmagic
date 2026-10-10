# WorkloadIdentity の設計

この文書は、WorkloadIdentity の設計を設計領域ごとに索引する。
このモジュールが外へ約束することは、仕様の [WorkloadIdentity](../README.md) と各機能仕様が定める。
一つの機能だけの仕組みは、その機能の `design.md` に置く。

| 設計領域 | 内容 |
| --- | --- |
| [アーキテクチャ設計](architecture.md) | 文脈と範囲、解決戦略、構成要素、実行時の流れ |
| アプリケーション設計 | 該当なし：管理 API はシステムの[アプリケーション設計](../../../design/application/README.md)に従い、このモジュールに固有の方式はない |
| データ設計 | 該当なし：二つの Aggregate の関係と削除の扱いは仕様の[モデル](../README.md#モデル)が、テーブルの定義はスキーマファイルが扱う |
| インフラストラクチャ設計 | 該当なし：外部の発行者の JWKS への到達性のほかに、実行基盤に固有の要件はない |
| セキュリティ設計 | 該当なし：交換の経路が管理者の権限を通らないことと、テナントの境界は[アテステーションの交換](../attestation-exchange/README.md)のセキュリティ上の考慮の節が扱う |
| 信頼性設計 | 該当なし：JWKS の取得の失敗の扱いは[アテステーションの交換の設計](../attestation-exchange/design.md)が扱う |
| 性能設計 | 該当なし：システムの[性能設計](../../../design/performance/README.md)に従い、このモジュールに割り当てた品質要件はない |
| オブザーバビリティ設計 | 該当なし：拒否の理由は `WorkloadAttestationRejected` で監査に残り、それ以外はシステムの[オブザーバビリティ設計](../../../design/observability/README.md)に従う |
| 検証設計 | 該当なし：システムの[検証設計](../../../design/verification/README.md)に従う |

## 設計の記録

| 記録 | 内容 |
| --- | --- |
| [設計判断](decisions.md) | 代替案を比べた重要な判断 |
