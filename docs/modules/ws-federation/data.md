# WsFederation のデータ

この文書は、WsFederation の Aggregate を扱う。

## 集約

| Aggregate | 境界の内側にあるもの | 他の Aggregate との関係 |
| --- | --- | --- |
| `WsFedRelyingParty` | `wtrealm`、許可する `wreply` の集合、audience、トークンの型（SAML 1.1 または SAML 2.0）、`ClaimMappingPolicy`、Entra のフェデレーションの定型設定 | `Tenant` を `tenant_id` で参照する。`Application` から参照される |
| `EntraFederationObservationRecord` | 一つのドメインの最新の観測値、比べた期待値の指紋と証明書の拇印、差分、取得時刻と確認時刻 | `Tenant` を `tenant_id` で、Entra の RP のドメインの組をドメイン名で参照する。RP の保存と別に書く |

RP のトークンの型は、指定がなければ SAML 1.1 とする。
Entra と AD FS の WS-Fed のデフォルトに合わせるためである。
