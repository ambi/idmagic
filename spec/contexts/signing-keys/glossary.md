# SigningKeys Glossary

| Term | Definition | Aliases |
|---|---|---|
| SigningKeys | テナント単位の署名鍵素材について、ライフサイクルと公開を扱う境界。OAuth2 / OIDC は JWK / JWKS、SAML / WS-* は X.509 証明書を使用する。 | KeyMaterial, signing keys |
| Retire | SigningKey を Verifying から Retired に移す。 | retire |
| Archive | SigningKey を Retired から Archived に移す。 | archive |
| Verifying | 署名はしないが、過去発行トークンの検証のため JWKS に残っている状態。 | verifying |
| Retired | JWKS から除去された状態。新規検証には使われない。 | retired |
| Archived | 監査用に長期保管する終端状態。鍵素材は封印される。 | archived |
| KeyProvider | 鍵素材の保管方式と署名の実行主体。Local / Database は秘密鍵をプロセス内に読み込み、アプリケーションが署名する開発・テスト用の方式である。VaultTransit は秘密鍵を Vault 内に保持し、署名を Vault API に委ねる本番用の方式である。Database は特定の製品名を表さない。 | key provider, 鍵プロバイダー |
| VaultTransit | HashiCorp Vault の Transit secrets engine を使う KeyProvider。秘密鍵マテリアルは Vault 外に出ず、署名要求ごとに Vault へ委譲する。 | Vault Transit |
| FailClosed | KeyProvider が不達のとき、新規トークン発行を停止する挙動。既発行トークン検証用の JWKS は取得可能な範囲で返す。強制点は OAuth2.Token の requires が持つ。 | fail-closed, フェイルクローズ |
