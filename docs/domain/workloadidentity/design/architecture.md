# WorkloadIdentity のアーキテクチャ

この文書は、WorkloadIdentity の文脈、解決戦略、構成要素、実行時の流れを扱う。
システム全体の構成は[論理アーキテクチャ](../../../design/architecture/logical.md)と[ランタイムアーキテクチャ](../../../design/architecture/runtime.md)に従う。

## 文脈と範囲

この Context がほかの Context と結ぶ契約は、仕様の[公開する契約](../README.md#公開する契約)が定める。
この Context から外への依存の向きは次のとおりである。

| 相手 | 向き | 実現方式 |
| --- | --- | --- |
| `IdManagement` | 関連付けの作成と交換が、`Agent` とその束縛を読む | 相手の `agent` の `ports.AgentRepository` を使う |
| `OAuth2` | Token Exchange が、この Context の検証を呼ぶ | 相手が定める `WorkloadTokenVerifier` のポートを、この Context のアダプターが実装する |
| 共有の部品 | JWT の署名の検証、JWKS の解決と最後に取得できた鍵の保持 | `backend/shared/security/tokens_jose` |

## 解決戦略

| 方針 | 理由 |
| --- | --- |
| 外部のアテステーションを検証する RP に徹する | SPIRE などの基盤を同梱せずに、既存の実行環境の資格情報を使える。詳細は[判断](decisions.md#外部のアテステーションを検証する-rp-に徹し長期のシークレットを配らない) |
| 交換の結果を、登録した関連付けだけで決める | トークンの内容で対応先が変わらない |
| 複数の一致を拒否する | 関連付けの追加が既存のワークロードの対応先を変えない。詳細は[判断](decisions.md#複数の関連付けに一致した主体は優先順位で選ばず拒否する) |

## 構成要素

コードは機能スライスを持たず、一つの層の構成である。

| 機能仕様 | 主なユースケースとハンドラー |
| --- | --- |
| [信頼設定と関連付けの管理](../trust-configuration/README.md) | `usecases/admin_trust_bundles.go`、`usecases/admin_bindings.go`、`handlers_http/routes.go` |
| [アテステーションの交換](../attestation-exchange/README.md) | `usecases/verify_workload_attestation.go`、`usecases/adapter.go`、`verification_jose/verifier.go` |

| 層 | 責務 |
| --- | --- |
| `domain` | `WorkloadTrustBundle`、`AgentWorkloadBinding` と値の検証、主体のパターンの照合、`WorkloadIdentityGrant`、ドメインイベント |
| `ports` | 二つの Aggregate の Repository と、署名を検証する `WorkloadSVIDVerifier` |
| `usecases` | 管理の操作と、アテステーションの検証の手順 |
| `verification_jose` | `WorkloadSVIDVerifier` の実装。共有の部品で署名を検証し、JWKS の取得の失敗を扱う |
| `handlers_http` | テナント単位の管理 API |
| `db_postgres`、`db_memory` | `ports` の PostgreSQL の実装と、テストとローカルの構成で使うメモリの実装 |
| `testing_contract` | 二つの実装が同じ契約を満たすことを確かめる共通のテスト |

## 実行時の流れ

| 流れ | 契機 | 実行する場所 | 詳細 |
| --- | --- | --- | --- |
| 管理 API | 解決済みのテナントの `/api/admin/v1/workload-identity/...` への要求 | `api` が、ハンドラーからユースケースを同期的に呼ぶ | [信頼設定と関連付けの管理](../trust-configuration/README.md) |
| アテステーションの交換 | `OAuth2` のトークンエンドポイントへの Token Exchange の要求 | `api` の `OAuth2` のユースケースが、この Context の検証を同期的に呼ぶ | [アテステーションの交換の設計](../attestation-exchange/design.md) |
