---
context: claim-mapping
updated_at: 2026-08-11
---

# ClaimMapping Specification

## Overview

アイデンティティプリンシパルの属性を外部のリライングパーティー、サービスプロバイダー、クライアントへ公開するための属性公開ポリシーと、プロトコルに依存しないクレーム投影を所有する。属性解決と公開の可否判定を共通化し、OIDC の JSON クレーム、SAML の `AttributeStatement`、WS-Fed のクレーム URI へのワイヤー変換は各プロトコルの Context が所有する。

独立した Context として存在するのは、クレームの発行が XML の署名や搬送とは独立した純粋な変換であり、既存のどの Context にも収まりが悪かったからである。`OAuth2` は OIDC と OAuth に対象を限定しており、WS-* や SAML のリライングパーティーの信頼とアサーションの扱いをそこへ取り込むと、その Context の責務が肥大する。先に切り出すことで、重い XML 署名ライブラリを選定する前に、フェイルクローズと属性最小化の保証を単体テストで固められた。

## Glossary

| Term | Definition | Aliases |
|---|---|---|
| ClaimMappingPolicy | アイデンティティプリンシパルの属性を外部アプリケーション、リライングパーティー、サービスプロバイダー、クライアントへ公開するための属性解決と公開許可の規則。 | ClaimMappingPolicy, attribute release, claim mapping |

## Design

### Internal Interfaces

#### ResolveEffectiveClaims
テナントの属性可視性 (`visibility != Private`) と予約済みクレーム型の固定集合をフェイルクローズの下限として強制し、`ClaimMappingPolicy` と解決済み属性から `NameID` と `IssuedClaim[]` を組み立てる。これは WS-Fed、SAML、OIDC の各 issuer が共有する唯一のクレーム解決経路である。`User` の中核フィールド (`user_id`、`email`、`name`、`given_name`、`family_name`、`preferred_username`、`email_verified`、ロール) は `UserAttributeDef` に現れないため、常に解決対象とする。`user_id` は User Aggregate の識別子を指す、プロトコルに依存しない内部属性キーであり、OIDC ID Token や UserInfo が実際に発行するワイヤークレーム `sub` (RFC 7519 と OIDC Core が定める語彙) とは異なる。カスタム属性について、`attribute_defs` にないキー、または `visibility=Private` のキーをソースに持つ規則は下限の検査で拒否する。

### Declarative claim-issuance engine

`ClaimMappingRule` は AD FS 風のクレーム規則言語ではなく、出力するクレームの型 (URI) とそのソース — ユーザー属性、固定値、`NameID` のいずれか — を宣言する。`ClaimMappingPolicy` はリライングパーティーの規則集合を `NameIdConfiguration` と束ねる。この処理系は、解決済み属性の対応表 (すでにアイデンティティの Aggregate から切り離されている) とポリシーを受け取り、`IssuedClaim[]` を返す。入力側と出力側の両方でフェイルクローズとする。対応付け規則が明示したクレームだけを発行するため、対応付けのない属性がトークンへ漏れることはない。必須規則のソース属性が欠けている場合は、部分的なクレーム集合を返さず、発行そのものを拒否する。WS-Fed、WS-Trust、SAML はそれぞれでクレームを組み立てずに同じ処理系を呼ぶため、フェイルクローズの保証は 3 か所ではなく 1 か所に留まる。

### Design Decisions

- `ClaimMapping` は `OAuth2` および XML フェデレーションプロトコルから分けた独立した Bounded Context である。クレームの発行は XML の署名や搬送とは独立した、プロトコルに依存しない純粋な変換であり、WS-* や SAML のリライングパーティーの信頼を `OAuth2` へ取り込むと、その Context の責務が肥大するからである。
