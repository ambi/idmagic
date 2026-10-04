# フェデレーションメタデータ

この文書は、WsFederation が公開する `federationmetadata.xml` と MEX の仕組みを扱う。
公開のエンドポイントは、パッシブサインインと能動的な STS の両方の利用者が使う。

## アーキテクチャ

各テナントの `/{realm}/federationmetadata/2007-06/federationmetadata.xml` で AD FS 互換の `federationmetadata.xml` を公開し、テナントの発行者（デフォルトテナントでは `/realms/default`）を entityID として広告する。
WS-Fed の RP と Microsoft Entra のドメインフェデレーションは、別の導入手順を使わずに、発行者、エンドポイント、署名証明書を検出できる。

| 要素 | 広告する内容 |
| --- | --- |
| `RoleDescriptor` | `SecurityTokenServiceType` と `ApplicationServiceType` の二つ |
| エンドポイント | `PassiveRequestorEndpoint`、`SecurityTokenServiceEndpoint`、`MetadataEndpoint` |
| `KeyDescriptor` | 役割ごとに、現在有効な署名証明書と、有効期限内の検証用の証明書 |

署名証明書は、OAuth と OIDC の JWK の形を再利用せず、WS-* が本来使う X.509 の形で公開する。
鍵の用途、ローテーション、重なりは `SigningKeys` の責務のままであり、メタデータは WS-* の利用者がすでに期待するものを広告すれば足りるからである。
有効期限内の検証用の証明書も広告するので、RP は計画されたローテーションの前後でも検証を続けられる。

`/{realm}/trust/mex` は、能動的な STS の探索として、`usernamemixed` のエンドポイントと、UsernameToken を必須とするその方針を公開する。
RST と RSTR のやり取り自体は、メタデータの一部ではない。

## セキュリティ

メタデータと MEX はテナントの公開の Discovery であり、認証を要さない。
公開するのは、発行者、エンドポイント、署名証明書に限る。
