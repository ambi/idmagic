# SigningKeys

テナント単位の署名鍵素材について、ライフサイクル、ローテーション、公開の重複期間、監査をプロトコル横断で担う。OAuth2 / OIDC は JWK / JWKS と JWT 署名器、SAML / WS-* は X.509 証明書と XML 署名アダプターを使用するが、鍵の用途、ローテーション、テナント間の分離に関する規則はここに集約する。

鍵プロバイダーの選択と保管もここに含む。鍵素材は署名ポートと公開鍵ポートを通じてのみ提供し、各プロトコル形式への直列化は OAuth2、SAML、WS-Federation のアダプターに委ねる。

| File | Content |
|---|---|
| [glossary.md](glossary.md) | この Context での語義 |
| [states.md](states.md) | 状態と遷移 |
| [decisions.md](decisions.md) | 設計判断 |
| [internals.md](internals.md) | 機構の説明 |
| [scenarios.md](scenarios.md) | 受け入れシナリオ |
