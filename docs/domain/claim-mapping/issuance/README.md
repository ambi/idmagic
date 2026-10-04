# クレームの発行

## 概要

この文書は、`ClaimMappingPolicy` を適用して、外部の RP、SP、クライアントへ発行するクレームを組み立てる仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | 規則に明記したクレームだけの発行、公開できない属性をソースとする規則の拒否、必須の規則のソースが欠けたときの拒否 |
| 行為者 | System（トークンとアサーションを発行するプロトコルの Context） |
| 扱わないもの | クレームのプロトコルごとの表現と署名は、`OAuth2`、`SAML`、`WS-Federation` が扱う。ポリシーの保存は信頼先を持つ各 Context が扱う |

## モデル

発行は、入力と出力のどちらもフェイルクローズで扱う。
規則に明記したクレームだけを発行し、対応付けのない属性はトークンへ出さない。
発行できない場合は、部分的なクレームの集合を返さず、発行そのものを拒否する。

テナントの属性の可視性（`Private` の属性を公開しない）と、プロトコルが制御するクレーム型の固定の集合は、ポリシーでは緩められない下限である。
固定の集合は `iss`、`sub`、`aud`、`exp`、`iat`、`nbf`、`jti`、`azp`、`nonce`、`at_hash`、`c_hash`、`acr`、`amr`、`sid` である。

- **判断**：規則を宣言的な対応付けに限る理由は、[規則を宣言的な対応付けに限る](../design/decisions.md#規則を宣言的な対応付けに限る)。

## 操作

### プロトコルの Context によるクレームの解決

#### REQ-CLAIMMAPPING-001 対応付け規則のないカスタム属性はクレームとして発行されない

- プロトコルの Context がクレームを解決したとき、ClaimMapping は、ポリシーの規則が生成するクレームだけを返し、どの規則のソースでもない属性をクレームとして返さない。
- `user_attribute` の規則を解決するとき、ClaimMapping は、`source_key` の属性の空白だけでない値をすべて、そのクレームの値として返す。
- `fixed` の規則を解決するとき、ClaimMapping は、`fixed_value` をそのクレームの値として返す。
- `nameid` の規則を解決するとき、ClaimMapping は、NameID の値をそのクレームの値として返す。
- 必須でない規則の値が空の場合、ClaimMapping は、そのクレームを省き、ほかのクレームを返す。
- NameID を解決するとき、ClaimMapping は、`source_attribute` の属性の値のうち、空白だけでない最初の値を返す。
- **例**：EX-CLAIMMAPPING-001-01

#### REQ-CLAIMMAPPING-002 公開できない属性をソースとする規則は発行を拒否する

- 規則の `source_key` または NameID の `source_attribute` が、テナントの属性定義にないキー、または `Private` の属性である場合、ClaimMapping は、クレームを一つも返さずに解決を拒否する。
- ClaimMapping は、User の基本項目（`user_id`、`preferred_username`、`email`、`email_verified`、`name`、`given_name`、`family_name`、`roles`）を、属性定義によらず公開できる属性として扱う。
- 規則の `claim_type` が、前後の空白と大文字小文字を除いて、プロトコルが制御するクレーム型の固定の集合にある場合、ClaimMapping は、クレームを一つも返さずに解決を拒否する。
- **例**：EX-CLAIMMAPPING-002-01

#### REQ-CLAIMMAPPING-003 必須規則のソース属性が欠けていれば部分的な発行をしない

- 必須の規則の値が空の場合、ClaimMapping は、残りの規則のクレームも返さずに解決を拒否する。
- NameID の `source_attribute` の属性に空白だけでない値がない場合、ClaimMapping は、クレームを一つも返さずに解決を拒否する。
- NameID の `format` または `source_attribute` が空か、`claim_type` が空の規則か、`source_key` のない `user_attribute` の規則がある場合、ClaimMapping は、クレームを一つも返さずに解決を拒否する。
- **例**：EX-CLAIMMAPPING-003-01

## セキュリティ上の考慮

下限の検査は、OIDC、SAML、WS-Federation のすべてが通る一つの経路の中で行う。
プロトコルごとに解決を書くと、あるプロトコルでだけ非公開の属性が漏れる形を作れてしまうからである。
テナントの属性定義にないキーは、`Private` の属性と同じく公開できないものとして扱う。
