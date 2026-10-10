# ClaimMapping のアーキテクチャ

この文書は、ClaimMapping の文脈、解決戦略、構成要素、実行時の流れを扱う。
システム全体の構成は[論理アーキテクチャ](../../../design/architecture/logical.md)と[ランタイムアーキテクチャ](../../../design/architecture/runtime.md)に従う。

## 文脈と範囲

このモジュールがほかのモジュールと結ぶ契約は、仕様の[公開する契約](../README.md#公開する契約)が定める。
このモジュールから外への依存の向きは次のとおりである。

| 相手 | 向き | 実現方式 |
| --- | --- | --- |
| `IdManagement` | 属性の解決が、`User` と属性定義の型を使う | 相手の `user` の `domain` を参照する |
| `Tenancy` | テナントのカスタム属性の定義を読む | このモジュールが定めるインターフェースを、相手の属性スキーマのポートが構造的に満たす。相手への import の依存はない |

## 解決戦略

| 方針 | 理由 |
| --- | --- |
| クレームの解決を、すべてのプロトコルが共有する一つの経路にする | あるプロトコルでだけ非公開の属性が漏れる形を作らせない |
| 規則を宣言的な対応付けに限る | 公開できる範囲を、規則を実行せずに検査できる。詳細は[判断](decisions.md#規則を宣言的な対応付けに限る) |
| 発行を、作用を持たない関数にする | ポリシー、属性、属性定義だけから結果が決まり、呼び出し元のトランザクションや署名と独立に検証できる |

## 構成要素

コードは機能スライスを持たず、`domain` と `ports` の二つのパッケージである。
クレームの発行は入力だけで結果が決まる計算なので、`domain` に置く。
下限を通らない発行（`issueClaims`）は `domain` の外へ公開せず、ほかのモジュールは `IssueClaimsWithFloor` だけを通る。

| パッケージ | 主な要素 | 対応する機能仕様 |
| --- | --- | --- |
| `domain` | `ClaimMappingPolicy`、`ClaimMappingRule`、`NameIdConfiguration`、`IssuedClaim`、`IssueClaimsWithFloor`（下限の検査と発行）、`ValidateClaimReleaseRules`（保存時の検査）、`ResolveUserAttributes`、`MergeTenantAttributeDefs` | [クレームの発行](../issuance/README.md) |
| `ports` | `TenantAttributeSchemaRepo`（テナントの属性スキーマの読み取り）、`ResolveTenantAttributeDefs`（スキーマを読んで組み込みの定義と合成する） | [クレームの発行](../issuance/README.md) |

## 実行時の流れ

| 流れ | 契機 | 実行する場所 | 詳細 |
| --- | --- | --- | --- |
| クレームの発行 | ID Token、UserInfo、SAML アサーション、WS-Fed のトークンの発行 | 発行するモジュールのユースケースが、属性と属性定義を解決してから `IssueClaimsWithFloor` を呼ぶ | [クレームの発行の設計](../issuance/design.md) |
| 保存時の規則の検査 | アプリケーションの OIDC、SAML、WS-Federation の設定の更新 | `Application` の管理 API が、保存の前に `ValidateClaimReleaseRules` を呼ぶ | [クレームの発行の設計](../issuance/design.md) |
