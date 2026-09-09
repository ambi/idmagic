# WI-523: 内省の `token_type` が RFC 6749 §5.1 の提示形式になる

作業項目は `wi-523-introspection-token-type-is-not-the-rfc6749-token-type` である。

WI-523 は、`/introspect` の応答の `token_type` を RFC 6749 §5.1 の提示形式に揃える。これまでこの値はトークンの区分名 (`access_token` または `refresh_token`) であり、RFC 7662 §2.2 が参照している語彙のどれでもなかった。同じ 1 本のトークンについて、`/token` は `Bearer` と言い、`/introspect` は `access_token` と言っていたことになる。

有効なアクセストークンの内省は `Bearer` を返す。DPoP で束縛したアクセストークンは RFC 9449 §5 のとおり `DPoP` を返す。mTLS の証明書で束縛したアクセストークンは RFC 8705 §3 のとおり `Bearer` のままである。いずれも発行時に `/token` が返した値と一致する。

リフレッシュトークンの内省は `token_type` を返さなくなった。RFC 6749 §5.1 の種別はアクセストークンを保護リソースへどう提示するかを言うものであり、リフレッシュトークンにはその意味の種別が無い。アクセストークンとリフレッシュトークンの区分を運ぶメンバーは RFC 7662 が定めておらず、この応答にも置いていない。内省を呼ぶ側は `token_type_hint` で区分を指定できる。

無効なトークンの応答は従来どおり `active: false` だけである。提示形式も付かない。

`/token` の応答の `token_type` は変わっていない。

規範上の条件は [RFC7662-INTROSPECT](../../contexts/oauth2/standards.md#oauth-20-token-introspection) が定める。
