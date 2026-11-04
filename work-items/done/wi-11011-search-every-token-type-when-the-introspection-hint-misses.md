---
status: completed
---

# イントロスペクションのヒントが外れても、すべての種類のトークンを探す

`/introspect` は `token_type_hint` を探す順序の指定としてだけ使い、ヒントの種類で見つからなければ残りの種類も探すようにした。これまで `access_token` か未知の値のヒントを付けたリフレッシュトークンは、有効でも `active=false` になっていた。

仕様は `docs/modules/oauth2/standards.md` の RFC 7662 の表に受け入れの標準要件 `RFC7662-HINT-MISS`（ヒントの種類で見つからなければほかの種類も探す）と `RFC7662-HINT-UNKNOWN`（未知のヒントをヒントなしと同じに扱う）を、`docs/modules/api-tokens/standards.md` に `RFC7662-API-TOKEN-HINT`（どのヒントでも API アクセストークンに同じ応答を返す）を加えた。

判断：`access_token` のヒントではアクセストークンを先に検証し、無効ならリフレッシュトークンのストアを引く。それ以外のヒントは従来どおりリフレッシュトークンから探す。ヒントを読まない案は、大半の要求にストアの照会を一回足すので採らなかった。リフレッシュトークンのレコードがあれば、無効でもアクセストークンとしては探さない。

確認の限界：受け入れ試験は Memory モジュールで組んだ HTTP スタックの `/introspect` で確かめた。PostgreSQL のストアでは試験していないが、使うのは既存の `FindByHash` だけである。単体試験の 3 箇所が `RefreshStore` を配線せずに `IntrospectDeps` を作っていたため、空のストアを渡すように直した。
