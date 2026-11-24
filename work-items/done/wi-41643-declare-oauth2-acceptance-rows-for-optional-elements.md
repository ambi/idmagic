---
status: completed
---

# 連携プロトコルで相手が省略できる要素の受け入れと受け入れの制限を、標準要件と試験に加える

相手が省略できる要素の受け入れと受け入れの制限を、四つのモジュールの標準要件として宣言した。
OAuth2 に `RFC6749-CLIENT-CREDENTIALS-SCOPE`、`RFC7009-TOKEN-TYPE-HINT`、`RFC7523-OPTIONAL-CLAIMS`、`RFC8628-SCOPE-DEFAULT`、`RFC9449-HTU-NORMALIZATION`（partial）、`RFC9449-SERVER-NONCE`（excluded）を加え、`RFC7521-CLIENT-ID-OPTIONAL` に受け入れる六つの入口を明示した。
SAML に `SAML2CORE-AUTHNREQUEST-OMITTED-ATTRIBUTES`、WS-Federation に `WSFED-PASSIVE-OMITTED-PARAMETERS`、外部 OIDC に `OIDC-CORE-ID-TOKEN-RS256`（partial）と `OIDC-CORE-TOKEN-RESPONSE-OPTIONAL` を加えた。
製品コードは変えず、各要件を省いた形を本番の入口へ送る HTTP の受け入れ試験から参照する。
`RFC7523-CLIENT-ASSERTION` と `RFC9449-PROOF` は検査の項目の要件として残し、省略と追加の受け入れを別の要件へ分けた。
試験の作成中に、上流の JWK が `alg` を省くと ID Token を検証できない不備を見つけ、wi-62564 として起票した。
