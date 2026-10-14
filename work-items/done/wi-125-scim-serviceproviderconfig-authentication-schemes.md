---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# SCIM ServiceProviderConfig で authenticationSchemes を申告する

`/scim/v2/ServiceProviderConfig` の `authenticationSchemes` が Go の nil スライスの
ため `null` を返し RFC 7643 §5 非準拠だった問題を修正した。`handleGetServiceProviderConfig`
で Bearer トークン方式 (`type: oauthbearertoken`, specUri: RFC 6750) を 1 件申告する
ようにした。ドメインモデルの匿名構造体を名前付き型 `AuthenticationScheme` に切り出し、
ハンドラからの初期化を可読にした。SCL は `GetScimServiceProviderConfig` の description に
「対応認証方式を authenticationSchemes で申告し、最低 oauthbearertoken を含める」契約を
明記した (output は opaque な Map<String,JSON> のまま)。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
