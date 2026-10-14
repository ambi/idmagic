---
status: completed
archived_from: 610341bde014725da511dbb909e29fdcfbeaa085
---

# 外部が値を決める識別子に資源の上限を与える

外部が値を決める識別子の「上限なし」を、宣言された上限に置き換えた。意味上の差分は 3 点である。
- **暗黙の上限を宣言された上限にした**。これらの列は主キーや一意索引の成分であり、PostgreSQL の btree v4 が索引行 1 件を 2704 バイトに制限する。つまり上限は既に存在し、宣言されておらず、超過は `SQLSTATE 54000` として 500 で返っていた。`wi-128` が長さ違反に与えた 422 の扱いから漏れていた。上限を宣言し、超過が btree に到達する前に止まるようにした。回帰テストは `TestAdminServiceProvider_RejectsMultibyteEntityIDOverTheByteCeiling`（着手時は 201 を返し、PostgreSQL へ渡していた）。
- **上限を 2 段にした**。契約の上限はコードポイント、索引の鍵の成分にはバイトの資源上限を重ねる。コードポイントだけでは足りない。標準が 1024 文字と定める entity ID でも、すべて 3 バイト文字なら 3072 バイトになり、宣言した契約を満たす値のまま btree で落ちる。資源をバイトで測るのは、btree が制限しているものがバイトだからである。`docs/SPECIFICATION.md` の「単位はコードポイント」という原則の例外に、「標準自身がオクテットで定めている値」に加えて「資源そのものがバイトで測られる場合」を足した。
- **違反の伝え方を接点ごとに分けた**。管理 API は `wi-128` どおり 422 / `field_length_exceeded`。SAML・WS-Federation・OAuth 2.0・WebAuthn の接点は、そのプロトコルが定めるエラー形で返す。AuthnRequest を送ってきた相手に Problem Details を返しても読めない。プロトコル接点では型付きの `*spec.LengthError` を返さないことを `TestParseAuthnRequestLengthErrorIsNotAFieldLengthViolation` が守る。

全文は `git show 610341bde014725da511dbb909e29fdcfbeaa085:<このファイルのパス>` で読む。
