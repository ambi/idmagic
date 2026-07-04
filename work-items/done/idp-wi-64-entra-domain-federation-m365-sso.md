---
id: idp-wi-64-entra-domain-federation-m365-sso
title: "Microsoft Entra domain federation で Microsoft 365 サインインと無音 SSO に対応する"
created_at: 2026-06-24
authors: ["tn"]
status: completed
risk: high
---

# Motivation
本シリーズの capstone。idmagic を Microsoft Entra ID (旧 Azure AD) の検証済み
ドメインに対する federated IdP として設定し、Microsoft 365 のブラウザ / リッチクライアント
サインインと、ドメイン参加 PC からの無音 SSO を成立させる。Okta・PingFederate・OneLogin は
いずれも AD FS の代替として Entra/M365 と federation し、passive (ブラウザ) は
WS-Federation、active (rich client) は WS-Trust で token を発行する。Entra 側は
`IssuerUri` / `PassiveLogOnUri` / `ActiveLogOnUri` / `MetadataExchangeUri` / 署名証明書を
登録し (`Update-MgDomainFederationConfiguration` 相当)、発行 token に `UPN` と `ImmutableID`
(オンプレ objectGUID 由来の sourceAnchor) が含まれることを必須とする。

本 WI は wi-61 (passive)・wi-62 (active/MEX)・wi-63 (metadata/claims)・
[[wi-65-kerberos-spnego-inbound-silent-sso]] (passive WIA / 無音 SSO)
を組み合わせ、Entra が要求する厳密な claim 形状・issuer・エンドポイント仕様を満たして
実テナントに対する end-to-end 連携を検証する。ここでの「依存」はゴールが全部品を集約する
という意味であり、wi-61〜63・wi-65 が互いに依存するわけではない (各 WI は単体で完結する)。

到達点は Okta 同等とする。すなわち Microsoft 365 のサインイン (passive / active) と無音 SSO
までを成立させ、**Hybrid Azure AD Join のデバイス登録は本 WI のスコープ外の既知制約**とする。
デバイス登録は WS-Trust `windowstransport` でコンピュータアカウントを Kerberos 認証する経路を
要するが、Okta も cloud STS では正面提供しておらず (managed/PHS や AD FS 併存で回避するのが
通例)、本 WI も同じ立場を取る。これを ADR と運用ドキュメントで明示する。

# Scope
- **decision**:
  - 新規 ADR: Entra federation の準拠プロファイルを確定する。required claims (`UPN` = `http://schemas.xmlsoap.org/claims/UPN`、`ImmutableID` / sourceAnchor = `nameidentifier` を persistent format で、objectGUID の base64 表現)、`IssuerUri` の規約、passive/active エンドポイントの URL 形、署名証明書要件 (鍵長・SHA-256 署名)、SAML 1.1 assertion を既定とすることを定める。1 テナントで複数の検証済みドメインを federation する場合の issuerUri / ドメイン → プロファイル mapping 規則も定める (Okta の Office 365 app 相当)。Hybrid Azure AD Join のデバイス登録は windowstransport + コンピュータアカウント Kerberos を要するためスコープ外とし、その理由 (Okta も同様の制約) と 回避策 (managed/PHS への切替や AD FS 併存) をドキュメント方針として明記する。
  - ADR-065 で Entra domain federation profile、sourceAnchor 変換、Hybrid Join 非対応の判断を記録する。
- **scl**:
  - 新規 model: EntraFederationProfile (issuerUri / sourceAnchor source / claim preset)。 既存の WsFedRelyingParty / claim mapping を Entra プリセットで束ねる。
  - 新規 interface: ConfigureEntraFederation (sourceAnchor 属性の指定と検証)。
  - 新規 event: EntraFederationConfigured。
- **go**:
  - Entra プリセットを適用すると wi-63 の claim mapping に UPN / ImmutableID / nameidentifier(persistent) が自動構成され、未充足なら設定を拒否する (fail-closed)。
  - sourceAnchor (objectGUID 等) を base64 化して ImmutableID として発行する変換を実装する。
  - 1 テナントの複数検証済みドメインを federation できるよう、ドメイン → issuerUri / Entra プロファイルの mapping を持ち、サインイン時に正しいプロファイルへ解決する。
  - passive / active / MEX / metadata エンドポイントが Entra の登録値と一致する URL で 公開されることを保証する (wi-61 / wi-62 / wi-63 を配線)。ドメイン PC 無音 SSO は wi-65 を配線する。
  - Hybrid Join のデバイス登録は未提供であることを設定時に診断・明示し、回避策を案内する。
- **ui**:
  - admin に Entra federation セットアップ画面 (issuerUri・sourceAnchor 属性選択・PowerShell 設定値の表示) を追加する。
- **documentation**:
  - README / 運用ドキュメントに Entra federation 設定手順 (`Update-MgDomainFederationConfiguration` に渡す値)、required claims、無音 SSO の前提、および Hybrid Join デバイス登録が 範囲外であること (Okta 同様の既知制約) と回避策を書く。

# Out of Scope
- WS-Fed passive / WS-Trust active / metadata の基盤実装 (wi-61 / wi-62 / wi-63)。
- passive WIA / Kerberos 無音 SSO 本体は [[wi-65-kerberos-spnego-inbound-silent-sso]]。本 WI は Entra プリセット配線・診断・実テナント検証まで。
- Hybrid Azure AD Join のデバイス登録 (WS-Trust windowstransport + コンピュータアカウント Kerberos)。Okta 同様の既知制約として未提供とし、必要なら将来別 WI とする。
- Entra Connect (オンプレ同期) の同梱。sourceAnchor の供給はオンプレ側責務とする。
- Microsoft 365 以外の Entra 統合アプリ個別検証。
- Microsoft Entra ID / Microsoft 365 の実テナントに対する end-to-end 接続検証 (domain federation 切替を伴う破壊的検証)。WS-* シリーズ共通の実テナント検証として [[wi-79-entra-real-tenant-end-to-end-verification]] に集約する。本 WI はローカル検証 (unit / integration / curl) とプリセット配線・診断までで完了とする。

# Verification
- `GOCACHE=/tmp/idmagic-cache go test ./...` (in: idmagic)
  - reason: Entra プリセットの claim 充足判定・sourceAnchor 変換・issuer 一致・未充足拒否の境界。
- `golangci-lint run ./...` (in: idmagic)
- `go build ./...` (in: idmagic)
- `bun --cwd idmagic/ui typecheck`
- `bun --cwd idmagic/ui build`
- 手動: Hybrid Join 設定を試みると未提供である旨が API 応答 / UI で診断・案内されることを確認する。
- 実テナント (passive / active サインイン・無音 SSO・claim 形状) の end-to-end 検証は [[wi-79-entra-real-tenant-end-to-end-verification]] に集約する。

# Risk Notes
Entra は claim 形状・issuer・署名要件に厳格で、誤ると無言のサインイン失敗や
sourceAnchor 不一致によるアカウント重複を招く。required claims は fail-closed で
強制し、sourceAnchor は不変属性に束縛する。Hybrid Azure AD Join のデバイス登録は
windowstransport + コンピュータアカウント Kerberos を要するため本 WI では提供せず
(Okta 同様の既知制約)、managed/PHS への切替や AD FS 併存を回避策として案内する。
無理に擬似実装しない。実テナント検証は破壊的変更 (ドメイン federation 切替) を伴うため
検証用テナントで行う ([[wi-79-entra-real-tenant-end-to-end-verification]] に集約)。

# Completion
- **Completed At**: 2026-06-28
- **Summary**:
  Entra domain federation profile (ADR-065) を WS-Federation context の RP preset として
  実装した。管理 UI 「Entra ドメインフェデレーション」(/admin/federation/entra) と
  `POST /api/admin/wsfed/entra-federation` で、検証済み domain・sourceAnchor 属性・IssuerUri を
  受け取り、UPN / ImmutableID / persistent NameID を必須 claim とする WsFedRelyingParty を
  upsert する。応答は Entra へ登録する PassiveLogOnUri / ActiveLogOnUri / MetadataExchangeUri と
  署名証明書の入手先、Hybrid Join 非対応の既知制約を返す。sourceAnchor は objectGUID の GUID 文字列を
  Microsoft byte order で base64 化 (既に base64 ならそのまま) して ImmutableID に正規化し、設定時は
  既存 user の欠落・重複・変換不能を fail-closed で拒否、発行時 (passive wi-61 / active wi-62) も
  ApplyEntraProfile → IssueClaims で required claim 未充足を fail-closed で拒否する。複数の検証済み
  ドメインは domain ごとに issuerUri (既定 urn:idmagic:entra:<domain>) の RP として共存し、サインイン時は
  wtrealm で正しい profile へ解決する。README に Update-MgDomainFederationConfiguration への値マッピング・
  required claims・無音 SSO 前提 (wi-65)・Hybrid Join 範囲外と回避策を追記した。
- **Verification Results**:
  - `go build ./...` (in: idmagic)
    - result: ok
  - `GOCACHE=/tmp/idmagic-cache go test ./...` (in: idmagic)
    - result: ok
  - `golangci-lint run ./...` (in: idmagic)
    - result: 0 issues
  - `bun --cwd idmagic/ui typecheck`
    - result: ok
  - `bun --cwd idmagic/ui build`
    - result: ok
- **Affected Guarantees State**:
  - claim 準拠: preset は UPN / ImmutableID / nameidentifier を Required とし、未充足は設定時・発行時とも fail-closed
  - issuer 一貫性: 発行 token の issuer / wtrealm / audience を IssuerUri に揃える
  - sourceAnchor 安定性: ImmutableID は不変属性 (objectGUID 等) を Microsoft byte order base64 で正規化
  - 制約の明示: Hybrid Join 非対応を API 応答 known_limitations と UI Alert で診断・案内
  - tenant isolation: Entra profile と RP は tenant ごとに分離し、複数ドメインは issuerUri で解決
  - audit: EntraFederationConfigured を発行
