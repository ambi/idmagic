---
status: completed
archived_from: 102ad14ea373dad7590a86c2f2dedf290f9c9c1c
---

# アカウント自己管理 API を API アクセストークンの scope で制御できるようにする

管理画面発行 token と OAuth grant 発行 token の wire format を RFC 9068 JWT に統一した。管理画面発行では OAuth client の事前登録を不要とし、realm 組み込み public client `idmagic-api-token`、`sub=user_id`、credential 識別子 `jti` を使用する。account scope を resource server の route 単位で enforce し、管理発行 token の lifecycle record を RFC 7662 introspection と RFC 7009 revocation に統合した。Authorization Code + PKCE / Device Authorization の user-bound grant は account scope を発行でき、client credentials と user subject のない token exchange は拒否する。RFC 6750 error、realm API audience、任意 DPoP、RFC 9728 metadata も共通境界へ結線した。

全文は `git show 102ad14ea373dad7590a86c2f2dedf290f9c9c1c:<このファイルのパス>` で読む。
