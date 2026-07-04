---
id: idp-wi-62-ws-trust-active-sts
title: "WS-Trust 1.3 Active Requestor STS (RST/RSTR・MEX) で能動クライアント認証に対応する"
created_at: 2026-06-24
authors: ["tn"]
status: completed
risk: high
---

# Motivation
ブラウザを介さない rich client・レガシー認証・デバイス登録は WS-Federation の
passive プロファイルでは賄えず、SOAP ベースの WS-Trust active requestor を必要と
する。PingFederate の Security Token Service (STS) はこの領域の代表で、RST
(RequestSecurityToken) を受けて token generator が SAML assertion を発行する。
Okta・OneLogin も Microsoft 365 連携でレガシー / active 認証のために WS-Trust の
`usernamemixed` を提供する。AD FS は `/adfs/services/trust/...` 配下に MEX・
`usernamemixed`・`windowstransport` を公開する。

本 WI は idmagic を WS-Trust 1.3 の IP-STS として振る舞えるようにする。WS-Security
/ WS-Addressing を伴う SOAP の RST を受理し、WS-Security UsernameToken (username +
password) で認証して署名済み SAML assertion を RSTR で返す。MEX (Metadata Exchange)
で endpoint と policy を広告し、Entra の federation 設定 (ActiveLogOnUri /
MetadataExchangeUri) の能動経路を成立させる。passive 経路は
[[wi-61-ws-federation-passive-requestor-idp]] が、metadata と claim 発行ルールは
[[wi-63-federation-metadata-and-claims-mapping]] が担う。

本 WI は UsernameToken 認証 (`usernamemixed`) だけで完結し、Kerberos には依存しない。
これは Okta の Office 365 向け WS-Trust が username/password の active 認証に絞っているのと
同じ到達点である。Kerberos/IWA 認証の `windowstransport` (Entra Hybrid Join のデバイス登録
経路) は本シリーズのスコープ外とする (Okta も cloud STS では正面提供しない)。将来必要なら
別 WI とし、その際に再利用できるよう Issue パイプラインは認証方式から独立させておく。

# Scope
- **decision**:
  - 新規 ADR: WS-Trust の対応範囲を確定する。初期対応は WS-Trust 1.3 (`Issue` binding)、 `usernamemixed` (WS-Security UsernameToken による username/password 認証) を必須、 MEX 公開、SAML 1.1 assertion (Entra 互換) を既定・SAML 2.0 を任意とする。WS-Security / WS-Addressing の必須ヘッダ (Timestamp・To・Action・MessageID・ReplyTo) の検証点を定める。 Issue パイプラインは認証方式から独立させ、将来 `windowstransport` (Kerberos/IWA) を 別 WI で足せる拡張点として切る (本シリーズでは windowstransport は範囲外)。
- **scl**:
  - 新規 model: WsTrustRequestSecurityToken / WsTrustResponse / SecurityTokenEndpoint。 assertion は SAML IdP の assertion model を再利用する。
  - 新規 interface: WsTrustIssue / WsTrustMetadataExchange。
  - 新規 event: WsTrustTokenIssued / WsTrustTokenRejected。
  - endpoint ごとの受理する認証方式 (UsernameToken / Windows) を表す model を追加する。
- **go**:
  - SOAP + WS-Security + WS-Addressing の RST を解析し、Timestamp の有効期限・replay・ `AppliesTo` (対象 RP) を検証して RSTR を発行する。XML 署名は SAML IdP と共有 library を使う。
  - UsernameToken の username/password を既存の認証ユースケースで検証する (パスワードハッシャ・ロックアウト・監査を再利用)。
  - `AppliesTo` を登録済み RP に解決し、未登録対象は拒否する (fail-closed)。発行 assertion の AudienceRestriction / Recipient を `AppliesTo` に束縛する。
  - MEX で endpoint・binding・policy (UsernameToken 必須等) を広告する。
- **http**:
  - `/{realm}/trust/mex`・`/{realm}/trust/usernamemixed` を公開する。windowstransport は範囲外。
- **documentation**:
  - README に WS-Trust 対応範囲、active エンドポイント URL、RST/RSTR 例を書く。

# Out of Scope
- WS-Federation passive は [[wi-61-ws-federation-passive-requestor-idp]] で扱う。
- `windowstransport` の Integrated Windows Auth (Kerberos / SPNEGO) と Hybrid Join の デバイス登録。Okta 同様に本シリーズでは未提供とし、必要なら将来別 WI とする。本 WI は 認証方式から独立した Issue パイプラインと、その将来拡張点の提供まで。
- WS-Trust の `Validate` / `Renew` / `Cancel` binding。初期は `Issue` のみ。
- federation metadata 公開と claim 発行ルールは [[wi-63-federation-metadata-and-claims-mapping]] で扱う。

# Verification
- `go test ./...` (in: idmagic)
  - reason: RST 解析・Timestamp/replay・AppliesTo 解決・UsernameToken 認証・RSTR 署名の境界。
- `golangci-lint run ./...` (in: idmagic)
- `go build ./...` (in: idmagic)
- 手動: SOAP クライアントから `usernamemixed` に UsernameToken つき RST を送り、 RSTR の署名・AudienceRestriction・claim が正しいこと、未登録 AppliesTo と 期限切れ Timestamp が拒否されることを確認する。
- 手動: MEX を取得し endpoint / policy が正しく広告されることを確認する。

# Risk Notes
WS-Trust は SOAP / WS-Security / WS-Addressing の手書き処理が XML signature wrapping・
replay・canonicalization のリスクを生む。署名検証は library に委ね、Timestamp・
MessageID・AppliesTo を fail-closed で検証する。windowstransport (Kerberos/IWA) は
本シリーズでは範囲外とし、Issue パイプラインを認証方式から独立させて将来別 WI で
足せる拡張点に留める。

# Completion
- **Completed At**: 2026-06-27
- **Summary**:
  ADR-063 を追加し、WS-Trust 1.3 active STS の初期対応を Issue binding /
  `usernamemixed` に限定して実装した。`/{realm}/trust/usernamemixed` は SOAP 1.2 RST を受理し、
  WS-Addressing `MessageID` / `To` / `Action`、WS-Security UsernameToken、Timestamp、
  `AppliesTo` を fail-closed に検証する。`MessageID` は短期 replay store に記録し、
  `AppliesTo` は登録済み WS-Fed RP に解決する。username/password は既存 UserRepository /
  PasswordHasher / LoginAttemptThrottle を使って検証し、RP の ClaimMappingPolicy と SAML
  署名器を再利用して署名済み assertion を SOAP RSTR で返す。MEX は wi-63 の `trust/mex` が
  `usernamemixed` endpoint を広告する。
- **Verification Results**:
  - `GOCACHE=/tmp/idmagic-cache go test ./internal/wsfederation/...` (in: idmagic)
    - result: ok
  - `GOCACHE=/tmp/idmagic-cache go test ./...` (in: idmagic)
    - result: ok
  - `bun run yaml-check --schema=work-item ../idmagic/work-items/wi-62-ws-trust-active-sts.yaml ../idmagic/work-items/wi-63-federation-metadata-and-claims-mapping.yaml` (in: tools)
    - result: ok
  - `golangci-lint run ./...` (in: idmagic)
    - result: ok: 0 issues (run outside sandbox after package loading failed inside sandbox)
  - `bun --cwd idmagic/ui typecheck`
    - result: ok
  - `bun --cwd idmagic/ui build`
    - result: ok
- **Affected Guarantees State**:
  - XML signature safety: RSTR に含める SAML assertion は samltoken/goxmldsig で署名
  - audience restriction: AppliesTo は登録済み RP に解決し、assertion Audience / Recipient に束縛
  - replay protection: WS-Addressing MessageID を ClientAssertionReplayStore に `wstrust:` prefix で短期記録
  - credential safety: UsernameToken password は既存 PasswordHasher / SentinelPasswordHash / LoginAttemptThrottle を利用
  - audit: WsTrustTokenIssued / WsTrustTokenRejected を発行
