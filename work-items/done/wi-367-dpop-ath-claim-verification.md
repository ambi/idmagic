---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# リソースアクセス時の DPoP proof で ath クレームを検証し、proof を access token に束縛する

保護リソースへの DPoP proof が access token に束縛されるようになった。`just spec-diff` の
差分は normative scenario **REQ-OAUTH2-045** の追加と TypeSpec 宣言 **DpopProofClaims** の追加
(加えて standards 行 RFC9449-ATH と Design 節の DPoP 段落)。仕様上の意味の差は、
「DPoP proof は鍵の所有を示す」から「保護リソースに提示する proof は
`ath` = base64url(SHA-256(提示 access token)) を持ち、持たない proof は拒否される」への強化で、
トークンエンドポイントは対象 token が存在しないため従来どおり `ath` を要求しない。
実装では `VerifyDPoP` を経路別の `VerifyDPoPForToken` / `VerifyDPoPForResource` に分割した。
optional 引数ではなく関数分割にしたのは、access token を渡し忘れた呼び出しが
「ath 検証が黙って無効化された保護リソース」にならないようにするため。3 つの呼び出し側は
改名によるコンパイルエラーで全て露出させて振り分けた。比較は `subtle.ConstantTimeCompare`。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
