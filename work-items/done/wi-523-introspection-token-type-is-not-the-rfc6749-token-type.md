---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# 内省の `token_type` が RFC 6749 §5.1 の種別ではなく、トークンの区分を名乗っている

`/introspect` の `token_type` が、トークンの区分名から RFC 6749 §5.1 の提示形式になった。
制約なしのアクセストークンは `Bearer`、DPoP 束縛は RFC 9449 §5 の `DPoP`、mTLS 束縛は
RFC 8705 §3 のとおり `Bearer` を名乗る。リフレッシュトークンと無効なトークンは何も名乗らない。
`mise run spec-diff` は規範差分なしを返す。追加も削除もした宣言が無く、変わったのは
`IntrospectionResponse.token_type` の型 (`string` から `"Bearer" | "DPoP"`) と doc だからである。
規範の側では、本項目は既存の `RFC7662-INTROSPECT` を実装で満たしにいった記録である。
実装は 3 つに分かれる。提示形式の規則を `backend/oauth2/token/domain` の
`PresentationTokenType` へ 1 つだけ置き、`/token` に写されていた同じ 4 行 6 か所をそこへ寄せ、
内省のユースケースが `active` のときだけ同じ関数で導く。`ports.IntrospectionResult` からは
`TokenType` を落とした。提示形式は `SenderConstraint` から導けるので、JWT を検証する層が
提示形式の語彙を知っている必要が無い。
**規則を 1 つにしたことが効いた。** 「`/token` と `/introspect` が一致する」は、7 か所を
読み比べる約束ではなく、同じ関数を通るという構造になった。E2E も約束ではなく、発行時に
返った値と内省の値を突き合わせる形で読んでいる。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
