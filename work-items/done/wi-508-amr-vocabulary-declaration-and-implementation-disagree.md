---
status: completed
archived_from: 854f521daf0113319b826044107f04189d1f24c1
---

# `amr` の語彙について、standards.md と scenarios.feature.md が食い違っている

`mise run spec-diff` は `REQ-AUTHENTICATION-036` の追加と `RFC8176-AMR-VOCABULARY` の変更を返す。
復旧コードで成立した第二要素が MFA の要求を満たすようになった。`amr` に `rc` が加わると `acr` が
`urn:idmagic:acr:mfa` へ上がり、以後の認可要求は第二要素を再び求めない。`LoginSession.amr` の語彙は
`federated` を加えた 8 語となり、語彙の外の値を持つセッションは保存されなくなった。語彙の宣言は
`backend/authentication/domain` の 1 か所にあり、`LoginSession.Validate()` がそれを使う。この検証は
それまで production のどこからも呼ばれていなかったので、`SessionManager` が `amr` を書く 2 か所へ
配線するところまでが 1 組の変更である。

全文は `git show 854f521daf0113319b826044107f04189d1f24c1:<このファイルのパス>` で読む。
