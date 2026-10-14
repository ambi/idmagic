---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# Token endpoint が返さない 403 を契約が宣言している

`Token` から `TokenError403` を取り除き、`OAuthAccessDeniedError` を `TokenError400Body` へ移した。`/token` は RFC 6749 §5.2 のとおり `access_denied` を 400 で返しており、403 を返す経路は無い。`Authorize` の 403 は逆に実在したので消さず、宣言の形を実測に合わせた。`application/json` + `OAuthAccessDeniedError` から `application/problem+json` + `AccessDeniedError` へ変え、隣の `ResumeFederatedAuthorizationError403` と同じ形にした。これは [[wi-382-typespec-contract-wire-fidelity-and-doc-language]] が行った 2 件の入れ替えのうち 1 件を、実測に基づいて戻したことになる。`mise run spec-diff` は `no normative specification change against main` を返す。規範シナリオも標準行も動かしておらず、変わったのは TypeSpec が宣言する応答の集合と本文の形だけである。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
