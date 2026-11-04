---
status: completed
---

# `client_id` を省いた `private_key_jwt` のクライアント認証を受け入れる

`private_key_jwt` のクライアント認証で、`client_id` のフォーム値を省いた要求を受け入れるようにした。
RFC 7521 §4.2 は `client_id` を OPTIONAL とし、クライアントをアサーションの subject で識別する。公式の MCP TypeScript SDK v2 の `PrivateKeyJwtProvider` は `client_id` を送らないため、これまで `invalid_client` で拒否していた。

仕様は REQ-OAUTH2-028 に、省いた要求をアサーションの `sub` の登録で同じ規則のまま確かめる要件と、`iss` か `sub` と異なる `client_id` を拒否する要件を加えた。`standards.md` に RFC 7521 の節と `RFC7521-CLIENT-ID-OPTIONAL` を加えた。これまで RFC 7523 の検証の規則だけを取り込み、土台の RFC 7521 を宣言していなかった。

判断：未検証の `sub` は登録を探す先にだけ使う（`UnverifiedClientAssertionSubject`、既存の `UnverifiedIDJAGIssuer` と同じ形）。認証方式、署名の鍵、`iss == sub == client_id`、`aud`、有効期間、`jti` の検証は変えていない。

確認の限界：受け入れ試験は製品の HTTP サーバーの `/token` で確かめた。実クライアントでは、この修正でビルドした IdMagic に対して wi-36722 の再現試験を回避策なしで実行し、13 の確認がすべて通った。PAR、イントロスペクション、失効は同じ認証処理を通るが、`client_id` を省いた要求の試験は `/token` だけである。
