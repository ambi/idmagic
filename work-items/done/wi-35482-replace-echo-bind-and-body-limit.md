---
status: cancelled
---

# Echo の Bind と BodyLimit を標準の処理へ置き換える

中止した。
Echo を維持すると決めた以上、Echo に任せられる入力処理は任せる方針とし、`c.Bind` と `middleware.BodyLimit` はそのまま使う。
`BodyLimit` は REQ-SYSTEM-027 を満たし、`Bind` の JSON 以外の本文も読む挙動は、置き換えの負担に見合う差ではない。
同じ方針で、クライアント IP の解決は wi-16357 で Echo の `IPExtractor` と `RealIP` に寄せる。
セキュリティヘッダーは、自動送信のフォームの CSP を応答ごとに差し替える処理が残り、Echo の `Secure` へ移せるのは固定の部分だけで、HSTS の付与が `X-Forwarded-Proto` に依存するようになるため、今の実装を維持する。
