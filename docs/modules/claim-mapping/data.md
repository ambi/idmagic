# ClaimMapping のデータ

この文書は、ClaimMapping の Aggregate を扱う。

## 集約

このモジュールは Aggregate を定義しない。
`ClaimMappingPolicy` は値オブジェクトであり、各プロトコルのモジュールの信頼先の Aggregate（`OAuth2Client`、`SamlServiceProvider`、`WsFedRelyingParty`）に埋め込まれる。
このモジュールは、その値の意味と検証だけを定める。

| 値オブジェクト | 内容 |
| --- | --- |
| `ClaimMappingPolicy` | 信頼先ごとの `ClaimMappingRule` の集合と、`NameIdConfiguration` の組 |
| `ClaimMappingRule` | 出力するクレーム型（URI）と、そのソース（ユーザー属性、固定値、`NameID` のいずれか）。必須かどうかを持つ |
| `NameIdConfiguration` | `NameID` の形式と、ソースとする属性 |
| `IssuedClaim` | ポリシーを適用して得た、クレーム型と値の組。プロトコルごとの表現へ変換する前の中間表現 |

- **判断**：ポリシーを独立した Aggregate にしない理由は、[ClaimMappingPolicy を信頼先に埋め込む値オブジェクトにする](decisions.md#claimmappingpolicy-を信頼先に埋め込む値オブジェクトにする)。

`User` の中核の項目（`user_id`、`email`、`name`、`given_name`、`family_name`、`preferred_username`、`email_verified`、ロール）は、テナントの属性定義に現れず、常にソースにできる。
`user_id` は User Aggregate の識別子を指す、プロトコルに依存しない内部の属性キーであり、OIDC の ID Token や UserInfo が発行するクレーム `sub` とは別物である。
両者の対応付けは `OAuth2` の発行側が行う。
