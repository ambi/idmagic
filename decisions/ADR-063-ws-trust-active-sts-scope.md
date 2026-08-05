---
status: accepted
authors: [tn]
created_at: 2026-07-04
---

# ADR-063: WS-Trust active STS の初期対応範囲を確定する

## コンテキスト

Microsoft 365 などの rich client / legacy active authentication は、ブラウザの WS-Federation
passive profile ではなく SOAP ベースの WS-Trust active endpoint を使う。AD FS 互換の実装では
MEX が endpoint と policy を広告し、`usernamemixed` endpoint が WS-Security UsernameToken を
受けて SAML assertion を RSTR で返す。

WS-Trust は SOAP / WS-Security / WS-Addressing / SAML 署名が重なるため、手広く対応すると
replay・XML wrapping・認証方式混線のリスクが増える。初期対応は Microsoft 365 active sign-in に
必要な Issue binding と UsernameToken に限定する。

## 決定

[[wi-62-ws-trust-active-sts]] の意思決定。[[ADR-059]] の claim 発行、
[[ADR-060]] の SAML assertion 署名、[[ADR-062]] の MEX / federation metadata 公開を前提に、
idmagic が WS-Trust 1.3 active requestor STS として扱う最小範囲を確定する。

対応は WS-Trust 1.3 Issue binding のみとし、`Validate` / `Renew` / `Cancel` は範囲外とする。認証方式も
`usernamemixed` の UsernameToken に限り、Kerberos/IWA の `windowstransport` は別 WI に分ける。
WS-Addressing / WS-Security の必須要素と `AppliesTo` の解決は fail-closed とし、未登録・不整合はすべて
拒否側に倒す。SOAP/WS-Security を広く受理するより、相互運用性より replay / audience 混線防止を優先した
判断であり、Kerberos 同時実装も信頼境界が異なるため見送った。具体的な binding・認証・検証・RSTR の
仕様は [`backend/wsfederation/ARCHITECTURE.md`](../backend/wsfederation/ARCHITECTURE.md) に置く。

## 影響

- `/trust/usernamemixed` が active STS endpoint として動作し、MEX の広告先と一致する。
- WS-Fed passive と WS-Trust active が同じ RP trust、claim mapping、SAML 署名器を共有する。
- `windowstransport` と Hybrid Azure AD Join デバイス登録は提供しないことが明確になる。

## 却下した代替案

- **WS-Trust 全 binding 対応。** Microsoft 365 active sign-in の初期価値に対して攻撃面と検証負荷が大きい。
- **Kerberos `windowstransport` の同時実装。** keytab/SPNEGO/コンピュータアカウント認証は別の信頼境界であり、[[wi-65-kerberos-spnego-inbound-silent-sso]] とも分ける。
- **SOAP/WS-Security の寛容な受理。** 相互運用性よりも replay / audience 混線防止を優先し、必須要素欠落は拒否する。
