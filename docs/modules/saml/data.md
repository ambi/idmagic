# Saml のデータ

この文書は、Saml の Aggregate を扱う。

## 集約

| Aggregate | 境界の内側にあるもの | 他の Aggregate との関係 |
| --- | --- | --- |
| `SamlServiceProvider` | entityID、許可する ACS の URL、署名の要否（Assertion と Response）、要求の署名の検証の鍵、`ClaimMappingPolicy`、割り当てた IdP プロファイル | `Tenant` を `tenant_id` で参照する。`SamlIdentityProviderProfile` を一つだけ参照する。`Application` から参照される |
| `SamlIdentityProviderProfile` | 名前、モード（`shared` または `dedicated`）、サーバーが生成する entityID | `Tenant` を `tenant_id` で参照する。署名の資格情報は、`SigningKeys` の `XmlFederationSigning` の用途でこのプロファイルをスコープとする |

テナントには、変更できない `default` のプロファイルが必ずあり、複数の SP で共有する。

- **判断**：`shared` と `dedicated` を一つのモデルで表す理由は、[IdP プロファイルを一つの共有できるモデルで表す](decisions.md#idp-プロファイルを一つの共有できるモデルで表す)。
