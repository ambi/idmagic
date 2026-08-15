---
context: claim-mapping
updated_at: 2026-08-15
---

# ClaimMapping Specification

## Overview

プリンシパルの属性を外部の RP、SP、クライアントへ公開するポリシーと、プロトコルに依存しないクレームの組み立てを所有する。属性の解決と公開可否の判定はここに集約し、OIDC の JSON クレーム、SAML の `AttributeStatement`、WS-Fed のクレーム URI への変換は各プロトコルの Context に委ねる。

## Glossary

| Term | Definition | Aliases |
|---|---|---|
| ClaimMappingPolicy | プリンシパルの属性を外部のアプリケーション、RP、SP、クライアントへ公開するための、属性解決と公開許可の規則。 | ClaimMappingPolicy, attribute release, claim mapping |
| IssuedClaim | `ClaimMappingPolicy` を適用して得られる、クレーム型 (URI) と値の組。プロトコルごとのワイヤー表現へ変換する前の中間表現である。 | IssuedClaim |
| NameID | RP や SP に対してプリンシパルを指し示す識別子。`NameIdConfiguration` がソース属性と形式を定める。 | NameID |

## Design

### Internal Interfaces

#### ResolveEffectiveClaims

`ClaimMappingPolicy` と解決済みの属性から `NameID` と `IssuedClaim[]` を組み立てる。WS-Fed、SAML、OIDC の各 issuer が共有する唯一のクレーム解決経路である。テナントの属性可視性 (`visibility != Private`) と予約済みクレーム型の固定集合は、ポリシーでは緩和できない制約として強制する。

`User` の中核フィールド (`user_id`、`email`、`name`、`given_name`、`family_name`、`preferred_username`、`email_verified`、ロール) は `UserAttributeDef` に現れないため、常に解決対象とする。カスタム属性については、`attribute_defs` にないキー、または `visibility=Private` のキーをソースに持つ規則を下限の検査で拒否する。

`user_id` は User Aggregate の識別子を指す、プロトコルに依存しない内部属性キーであり、OIDC の ID Token や UserInfo が発行するワイヤークレーム `sub` とは別物である。両者の対応付けは OIDC 側の issuer が行う。

### Declarative claim-issuance engine

`ClaimMappingRule` は AD FS 風のクレーム規則言語ではなく、出力するクレーム型 (URI) とそのソース — ユーザー属性、固定値、`NameID` のいずれか — を宣言する。`ClaimMappingPolicy` は RP ごとの規則集合を `NameIdConfiguration` と束ねる。処理系はポリシーと解決済み属性の対応表を受け取り、`IssuedClaim[]` を返す。

入力と出力のどちらもフェイルクローズで扱う。規則に明記されたクレームだけを発行するため、対応付けのない属性がトークンへ漏れることはない。必須規則のソース属性が欠けている場合は、部分的なクレーム集合を返さず、発行そのものを拒否する。WS-Fed、WS-Trust、SAML はいずれもこの処理系を使い、独自にクレームを組み立てない。

### Design Decisions

- クレームの発行を `OAuth2` と XML フェデレーションから切り離し、独立した Bounded Context とする。発行は署名や搬送とは独立した純粋な変換であり、WS-* や SAML の RP 信頼を `OAuth2` へ取り込むと、その Context の責務が肥大するからである。
- 規則は宣言的な対応付けに限り、条件分岐や変換関数を持つ規則言語は導入しない。属性最小化とフェイルクローズを、規則を実行せずに静的に検証できる範囲へ保つためである。

## Scenarios

### REQ-CLAIMMAPPING-001: 対応付け規則のないカスタム属性はクレームとして発行されない
- ACTOR System
- GIVEN テナントに `visibility != Private` のカスタム属性が定義されている
- GIVEN RP の `ClaimMappingPolicy` にその属性をソースとする規則がない
- WHEN その RP 向けのクレームを解決する
- THEN 発行されるクレーム集合にその属性は含まれない

### REQ-CLAIMMAPPING-002: 公開できない属性をソースとする規則は発行を拒否する
- ACTOR System
- GIVEN `ClaimMappingPolicy` に、`attribute_defs` にないキーまたは `visibility=Private` のキーをソースとする規則がある
- WHEN その RP 向けのクレームを解決する
- THEN クレームを 1 つも発行せずに拒否する

### REQ-CLAIMMAPPING-003: 必須規則のソース属性が欠けていれば部分的な発行をしない
- ACTOR System
- GIVEN `ClaimMappingPolicy` に必須の規則がある
- GIVEN 解決済み属性にその規則のソース属性がない
- WHEN その RP 向けのクレームを解決する
- THEN 残りの規則だけを適用した部分的なクレーム集合を返さず、発行そのものを拒否する
