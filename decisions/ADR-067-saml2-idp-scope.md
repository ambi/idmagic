---
status: accepted
authors: [tn]
created_at: 2026-07-04
---

# ADR-067: SAML 2.0 IdP の対応範囲を確定する

## コンテキスト

OIDC だけでは B2B / enterprise の最低ラインを満たせない取引が多い。Okta / Entra ID /
Keycloak は SAML 2.0 IdP として SP-initiated / IdP-initiated SSO、metadata 公開、
assertion 署名、attribute mapping、Single Logout を提供する。

idmagic には既に WS-Federation passive ([[wi-61-ws-federation-passive-requestor-idp]]) と
WS-Trust active ([[wi-62-ws-trust-active-sts]]) があり、claim 発行エンジン ([[ADR-059]]) と
署名済み SAML assertion アダプタ ([[ADR-060]]) を備える。SAML 2.0 IdP は新しい token 形式や
claim engine を必要とせず、これらを SAML 2.0 Web Browser SSO Profile のワイヤ形式に
包み直す層を足せばよい。XML 署名は誤りやすく署名ラッピング攻撃の温床なので、自前実装しない
方針 ([[ADR-060]]) を引き継ぐ。

## 決定

[[wi-29-saml2-idp]] の意思決定で、idmagic を SAML 2.0 IdP として
振る舞わせる初期スコープを定める。[[ADR-059]] (claim 発行) と
[[ADR-060]] (XML 署名と SAML assertion 署名) を前提に、[[ADR-064]] が分離した
`Saml` bounded context にブラウザ経路と SP 管理を実装する。

初期スコープは SAML 2.0 Web Browser SSO Profile に限り、SAML ECP・encrypted assertion・inbound SAML
federation はフル SAML フレームワーク導入と同様に見送る。既存の claim engine ([[ADR-059]]) と
署名アダプタ ([[ADR-060]]) を SAML 2.0 IdP のために再実装せず、`Saml` bounded context ([[ADR-064]])
にブラウザ経路と SP 管理だけを足す。fail-closed の相互運用ガードは domain に集約し、判定不能・不一致は
拒否側へ倒す。署名は Assertion 署名を既定とし、専用の署名鍵は新設せず WS-Fed / WS-Trust と同じ
federation 署名証明書を流用する ([[ADR-060]])。encrypted assertion を初期必須にしない判断も、
鍵交換・SP 側鍵管理という重い関心事を後回しにする同じ理由による。具体的な binding・署名手順・
`goxmldsig` の enveloped 署名 repositioning hazard・fail-closed ガードの実装箇所は
[`backend/saml/ARCHITECTURE.md`](../backend/saml/ARCHITECTURE.md) に置く。

## 影響

- 署名安全性・audience restriction・open redirect 防止・tenant isolation を、HTTP 経路の前に
  domain / adapter の round-trip 検証で確立できる。
- SP 登録・属性マッピング・NameID format・署名方針は Application Catalog の SAML binding として
  OIDC client / WS-Fed RP と同じ tenant boundary・割当・監査に乗る。
- SAML ECP・encrypted assertion・inbound SAML federation を初期から切り離し、スコープを保てる。

## 却下した代替案

- **フル SAML フレームワーク (crewjam/saml 等) の採用。** IdP/SP の構造と独自の trust / metadata
  モデルを丸ごと持ち込み、既存の claim engine ([[ADR-059]]) と署名アダプタ ([[ADR-060]]) を
  迂回する。署名は goxmldsig、構造は etree、claim は既存エンジンで足り、横断する自由度も保てる。
- **encrypted assertion を初期必須にする。** 鍵交換と SP 側鍵管理の重い関心事を初期に持ち込む。
  署名済み assertion + TLS で初期の機密性は満たせるため、必要時に別 WI とする。
- **SAML 専用の署名鍵・証明書を新設する。** WS-Fed / WS-Trust と同じ federation 署名証明書で
  足り、鍵の用途境界も同じ。証明書ライフサイクルは横断スライスで一括して扱う。
