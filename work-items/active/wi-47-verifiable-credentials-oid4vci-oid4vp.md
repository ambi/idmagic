---
depends_on: []
status: pending
authors: ["tn"]
risk: high
created_at: 2026-06-21
priority: p3
group: federation-protocols
change_kind: feature
affected_spec:
  - { path: docs/modules/oauth2/features/authorization.md, requirement: REQ-OAUTH2-005 }
  - { path: docs/modules/oauth2/features/protocol-endpoints.md, requirement: REQ-OAUTH2-014 }
---

# 検証可能なクレデンシャルを OID4VCI でウォレットへ発行する

## 動機

eIDAS 2.0 と EUDI Wallet、ISO mDL、Microsoft Entra Verified ID のように、ウォレットに入れる検証可能なクレデンシャル（Verifiable Credentials）が標準化と実運用の段階に入っている。
IdP は、従来の ID Token とアクセストークンの発行に加えて、利用者のウォレットへクレデンシャルを発行する役割（OpenID for Verifiable Credential Issuance、OID4VCI）と、ウォレットからの提示を検証する役割（OpenID for Verifiable Presentations、OID4VP）を担い始めている。
IdMagic は ID Token とアクセストークンの発行だけを持ち、選択的な開示と保有者の束縛を伴うクレデンシャルに対応しない。

起票時は発行と検証を一つの項目にしていたが、発行者と検証者は別の役割で、必要な鍵、状態、相互運用の確認が違う。
2026-11-14 の整理で、この項目は発行に絞り、検証は発行が動いてから別に起票する。
新しい機能領域なので、優先度は p3 とする。

## 対象範囲

- SD-JWT VC の形式で、認可コードのフローと事前認可コードのフローによるクレデンシャルの発行を加える。クレデンシャルのオファーは QR とディープリンクで渡す。
- 発行者のメタデータ（`/.well-known/openid-credential-issuer`）、クレデンシャルのエンドポイント、保有者の束縛（`cnf`）、選択的な開示を持つ。
- Token Status List による失効の公開と、管理者による失効の操作を持つ。
- 管理コンソールでクレデンシャルの構成を定義し、発行の履歴を見て失効させる。利用者はアカウントポータルからウォレットへ取得する。
- 署名には SigningKeys の用途別の署名鍵を使う。

## 対象外

- OID4VP による提示の検証。発行が動いてから別に起票する。
- mdoc（ISO 18013-5）。
- 特定のウォレットとの相互運用の認定の取得。
- 端末に束縛した鍵と Secure Enclave との連携。
- 発行者の信頼の連鎖（trust list）の構築。

## 設計

着手時に、発行を OAuth2 に置くか新しいモジュールに置くかを、[境界を選ぶ判断手順](../../docs/design/application/design-guidelines.md#境界を選ぶ判断手順)で決める。
OID4VCI はトークンエンドポイントに事前認可コードのグラントを足すので、OAuth2 との関係が判断の中心になる。
クレデンシャルに載せる属性は、ClaimMapping の開示の規則から選ぶ。

## 検証

- 事前認可コードと認可コードの両方で、ウォレット役のテストクライアントがクレデンシャルを取得し、失効の後に状態のリストが失効を示すことを観測する受け入れのテスト。
- `mise run verify`

## リスク

発行の仕様は改訂が続いており、固定したドラフトのバージョンと最終版の差が相互運用を壊す。
固定するバージョンを `standards.md` に宣言する。
