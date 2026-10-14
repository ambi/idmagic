---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# 不完全な `id_token_hint` によるログアウト対象の誤解決を拒否する

`mise run spec-diff` は `changed scenarios: REQ-OAUTH2-024` を返す。規範の差分はこの Rule 一つで、
拒否の具体例が 2 件 (EX-OAUTH2-024-05、EX-OAUTH2-024-06) 増えた。標準行 OIDC-LOGOUT-ID-TOKEN-HINT の
文言は変えていない。行が既に「署名、発行者、audience、subject、`sid` を検証して対象を解決する」と
述べており、足りなかったのは宣言ではなく、その条件を破る入力を名指しする具体例だった。
製品の側では、`/end_session` が `id_token_hint` から対象を決める条件が 3 つ増えた。
`sub` または `aud` を欠くヒントは `VerifyIDTokenHint` が claim を運ぶ前に落とす。
`sid` を欠くヒントと、`sid` の LoginSession の主体と `sub` が食い違うヒントは `ResolveEndSession` が
`invalid_request` で拒否する。`sid` を欠くヒントがブラウザー Cookie のセッションへ降格する経路は無くなった。
`sid` の必須化は共通の verifier ではなくログアウトの use case に置いた。同じ verifier を使う CIBA は
`sid` を持たない ID Token をヒントに受け取るためである。
検証の材料としては、Authentication の LoginSession の主体を読む `ports.LoginSessionOwnerLookup` が増え、
HTTP 層の `loginSessionOwner` が `SessionManager` からそれを満たす。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
