---
status: completed
---

# golang-jwt の直接利用を tokens_jose へ統合する

`mise run spec-diff` は規範の差分なしを示す。
`tokens_jose` に `SignJWT`（RS256、PS256、ES256）と `VerifyJWT` を足し、`SignPS256` と `SignRS256` は `SignJWT` へ委ねた。
上流 OIDC の ID Token の検証、MCP gateway の DPoP proof、Directory Connector の client assertion を移し、golang-jwt は go-webauthn の間接依存だけになった。
`VerifyJWT` の判定は golang-jwt v5.3.1 の既定に合わせ、`exp: 0` を時刻 0 とすることと JSON の `null` のクレームを拒否することだけが異なる。どちらも ID Token の受理を変えない。
REQ-AUTHENTICATION-001 の時刻、署名方式、issuer、audience の境界をテストに足し、`VerifyJWT` への手の変異（nbf、exp の境界と必須、秒の切り捨て、alg の照合、aud の要素の型）で検出を確かめた。
`alg` の照合の変異は ID Token の経路では署名の検証が先に拒否するため、`tokens_jose` の単体テストで検出する。
確かめた範囲は単体と HTTP の統合テスト、30 秒のファジングであり、実在の上流 IdP との互換性と実 Envoy を通す経路は検証していない。
