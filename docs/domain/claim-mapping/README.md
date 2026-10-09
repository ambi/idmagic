# ClaimMapping

## 責務と境界

プリンシパルの属性を外部の RP、SP、クライアントへ公開するポリシーと、プロトコルに依存しないクレームの組み立てを担う。
属性の解決と公開可否の判定はここに集約する。

| 扱わないもの | 担当 |
| --- | --- |
| OIDC の JSON クレームへの変換 | `OAuth2` |
| SAML の `AttributeStatement` への変換 | `SAML` |
| WS-Fed のクレーム URI への変換 | `WS-Federation` |
| ポリシーを埋め込む信頼先の登録と保存 | 信頼先を持つ各プロトコルのモジュール |
| テナントのカスタム属性の定義 | `Tenancy` の属性スキーマ |

## モデル

このモジュールは Aggregate を定義しない。
`ClaimMappingPolicy` は値オブジェクトであり、各プロトコルのモジュールの信頼先の Aggregate（`OAuth2Client`、`SamlServiceProvider`、`WsFedRelyingParty`）に埋め込まれる。
このモジュールは、その値の意味と検証だけを定める。

| 値オブジェクト | 内容 |
| --- | --- |
| `ClaimMappingPolicy` | 信頼先ごとの `ClaimMappingRule` の集合と、`NameIdConfiguration` の組 |
| `ClaimMappingRule` | 出力するクレーム型（URI）と、そのソース（ユーザー属性、固定値、`NameID` のいずれか）。必須かどうかを持つ |
| `NameIdConfiguration` | `NameID` の形式と、ソースとする属性 |
| `IssuedClaim` | ポリシーを適用して得た、クレーム型と値の組。プロトコルごとの表現へ変換する前の中間表現 |

- **判断**：ポリシーを独立した Aggregate にしない理由は、[ClaimMappingPolicy を信頼先に埋め込む値オブジェクトにする](design/decisions.md#claimmappingpolicy-を信頼先に埋め込む値オブジェクトにする)。

`User` の中核の項目（`user_id`、`email`、`name`、`given_name`、`family_name`、`preferred_username`、`email_verified`、ロール）は、テナントの属性定義に現れず、常にソースにできる。
`user_id` は User Aggregate の識別子を指す、プロトコルに依存しない内部の属性キーであり、OIDC の ID Token や UserInfo が発行するクレーム `sub` とは別物である。
両者の対応付けは `OAuth2` の発行側が行う。

## 公開する契約

規則とクレームの形は、TypeSpec の `ClaimMappingRule`、`NameIdConfiguration`、`IssuedClaim`、`ClaimReleaseDeniedError` が定める。
このモジュールは HTTP の操作を持たない。

| 契約 | 相手 | 向き | 内容 |
| --- | --- | --- | --- |
| `IssueClaimsWithFloor` | `OAuth2`（UserInfo と ID Token）、`SAML`、`WS-Federation`（サインインと WS-Trust） | このモジュールが提供する | ポリシー、解決済みの属性、テナントの属性定義から、`NameID` と `IssuedClaim` の集合を返すか、発行を拒否する |
| `ValidateClaimReleaseRules` | アプリケーションの OIDC、SAML、WS-Federation の設定を保存する `Application` | このモジュールが提供する | 保存の前に、規則が公開できる範囲に収まっているかを確かめる |
| `ResolveUserAttributes`、`ResolveTenantAttributeDefs` | クレームを発行する各プロトコルのモジュールと `Application` | このモジュールが提供する | `User` から規則のソースに使う属性の対応表を、テナントから組み込みとカスタムの属性定義を作る |
| `TenantAttributeSchemaRepo` | `Tenancy` の属性スキーマのポートが構造的に満たす | このモジュールが定める | テナントのカスタム属性の定義を読む |

## 機能

| 機能 | 内容 |
| --- | --- |
| [クレームの発行](issuance/README.md) | ポリシーの適用、公開できない属性の拒否、必須の規則の検証 |

| 文書 | 内容 |
| --- | --- |
| [ClaimMapping の用語集](glossary.md) | このモジュールでの語義 |
| [ClaimMapping の設計](design/README.md) | 話題ごとの設計と重要な判断 |
