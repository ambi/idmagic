# ClaimMapping Internals

## ResolveEffectiveClaims

`ClaimMappingPolicy` と解決済みの属性から `NameID` と `IssuedClaim[]` を組み立てる。WS-Fed、SAML、OIDC の各 issuer が共有する唯一のクレーム解決経路である。テナントの属性可視性 (`visibility != Private`) と予約済みクレーム型の固定集合は、ポリシーでは緩和できない制約として強制する。

`User` の中核フィールド (`user_id`、`email`、`name`、`given_name`、`family_name`、`preferred_username`、`email_verified`、ロール) は `UserAttributeDef` に現れないため、常に解決対象とする。カスタム属性については、`attribute_defs` にないキー、または `visibility=Private` のキーをソースに持つ規則を下限の検査で拒否する。

`user_id` は User Aggregate の識別子を指す、プロトコルに依存しない内部属性キーであり、OIDC の ID Token や UserInfo が発行するワイヤークレーム `sub` とは別物である。両者の対応付けは OIDC 側の issuer が行う。

## Declarative claim-issuance engine

`ClaimMappingRule` は AD FS 風のクレーム規則言語ではなく、出力するクレーム型 (URI) とそのソース — ユーザー属性、固定値、`NameID` のいずれか — を宣言する。`ClaimMappingPolicy` は RP ごとの規則集合を `NameIdConfiguration` と束ねる。処理系はポリシーと解決済み属性の対応表を受け取り、`IssuedClaim[]` を返す。

入力と出力のどちらもフェイルクローズで扱う。規則に明記されたクレームだけを発行するため、対応付けのない属性がトークンへ漏れることはない。必須規則のソース属性が欠けている場合は、部分的なクレーム集合を返さず、発行そのものを拒否する。WS-Fed、WS-Trust、SAML はいずれもこの処理系を使い、独自にクレームを組み立てない。
