---
status: completed
---

# JWK Set の鍵が一つの上流の ID Token を、`kid` なしでも検証する

上流の RS256 の ID Token が `kid` を省略したとき、署名用 RSA 鍵がちょうど一つなら検証する標準要件 `OIDC-CORE-ID-TOKEN-KID-OPTIONAL` を追加し、`REQ-AUTHENTICATION-001` と例を更新した。
署名候補から暗号化用の鍵を除外し、JWK 自体の `kid` の省略を許し、同じ識別子の複数鍵も個数を保って扱う。
複数候補、未知の識別子、不正な署名とクレームは拒否する。
単体試験と実物の OIDCDriver を通る callback の受け入れ試験で、既存 User への `federated` セッションの発行を検証する。
RS256 以外の署名アルゴリズムへの対応と受け入れの制限の整理は対象外で、後者は wi-82993 が扱う。
