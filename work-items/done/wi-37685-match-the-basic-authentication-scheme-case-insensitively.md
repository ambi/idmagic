---
status: completed
---

# クライアント認証の `Basic` の方式名を、大文字と小文字を区別せずに照合する

クライアント認証は、`Authorization` ヘッダーの方式名 `Basic` を大文字と小文字を区別せずに照合する。
`basic` や `BASIC` で送った `client_secret_basic` の要求は、`/token`、PAR、イントロスペクション、失効で認証を通る。
複数の認証方式の混在の判定も同じ照合を使うので、小文字の `basic` と `client_secret` を併せた要求は `invalid_request` で拒否する。
規則は `standards.md` の RFC 9110 の節に RFC9110-AUTH-SCHEME-CASE-INSENSITIVE として宣言した。REQ-OAUTH2-007 の要件文は変えていない。
組み込みの失効クライアントの判定（`handleRevoke`）は、ヘッダーの有無だけを見るため変えていない。
