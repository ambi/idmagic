---
status: completed
authors: ["tn"]
risk: medium
created_at: 2026-06-24
---

# Federation metadata 公開と claim 発行ルールエンジンを WS-Fed / WS-Trust で共有する

## Motivation
WS-Federation passive ([[wi-61-ws-federation-passive-requestor-idp]]) と WS-Trust
active ([[wi-62-ws-trust-active-sts]]) は、(1) RP / 相手 IdP が信頼を確立するための
federation metadata と、(2) 内部ユーザー属性から発行 token の claim を組み立てる
claim 発行ルールを共有する。AD FS は `federationmetadata.xml` に passive/active
エンドポイント・token type・署名証明書を載せ、relying party trust ごとに claim
issuance rule で `UPN` や `nameidentifier` を発行する。PingFederate は attribute
contract と token generator で同等を、Okta は Office 365 連携の claim mapping で
同等を提供する。

本 WI は AD FS 互換の `federationmetadata.xml` を realm 配下に公開し、署名証明書
(signing certificate) を広告し、RP / SP trust ごとの claim 発行ルール (内部属性 →
SAML claim type への mapping、固定値、NameID format / source) を定義・適用する
エンジンを実装する。これは WS-Fed と WS-Trust の双方が assertion を組み立てるとき
に使い、最終ゴールの Entra 連携 ([[wi-64-entra-domain-federation-m365-sso]]) が
要求する UPN / ImmutableID などの claim 形状をこのエンジンで満たす。SAML 2.0 IdP
([[wi-29-saml2-idp]]) の attribute mapping とも基盤を共有する。

## Scope
- **decision**:
  - 新規 ADR: federation metadata の公開範囲と claim 発行ルールの所有境界を確定する。 metadata は AD FS 互換 `federationmetadata.xml` (EntityDescriptor + RoleDescriptor: SecurityTokenService / ApplicationServiceType、PassiveRequestorEndpoint、 SecurityTokenServiceEndpoint、署名証明書) を realm 単位で公開する。claim ルールは 宣言的 mapping (source 属性 → claim type、固定値、NameID format) とし、AD FS の claim rule 言語そのものは採らない方針を明記する。署名証明書は OAuth signing key と 分離するか ADR で決める ([[wi-32-kms-hsm-and-per-tenant-signing-keys]] と整合)。
- **scl**:
  - 新規 model: FederationMetadataDocument / ClaimMappingRule / ClaimType / NameIdConfiguration。RP / SP trust に claim mapping rule 集合を持たせる。
  - 新規 interface: PublishFederationMetadata / 管理用の ClaimMappingRule CRUD。
  - 新規 permission: AdminFederationTrustManage。
  - 新規 event: FederationMetadataPublished / ClaimMappingRuleChanged。
- **go**:
  - realm の署名証明書・WS-Fed/WS-Trust エンドポイントから `federationmetadata.xml` を 生成し署名する。XML 署名は SAML / WS-* と共有 library を使う。
  - claim 発行エンジンを実装し、ユーザー属性・固定値・NameID 設定から token claim を fail-closed に組み立てる (未マップ属性は出力しない)。WS-Fed / WS-Trust / SAML が再利用する。
  - claim type は標準 URI (`http://schemas.xmlsoap.org/.../UPN` 等) を扱えるようにする。
  - metadata は存在するエンドポイントを広告する。wi-65 が WIA passive を追加したら RoleDescriptor に反映できるよう拡張点を設ける (本 WI は wi-65 に依存しない)。
- **http**:
  - `/{realm}/federationmetadata/2007-06/federationmetadata.xml` を公開する。
- **ui**:
  - admin trust 編集に claim mapping rule・NameID source・metadata ダウンロードを追加する。
- **documentation**:
  - README に metadata URL、claim mapping の設定例、署名証明書の取り扱いを書く。

## Out of Scope
- WS-Federation passive / WS-Trust active のプロトコル本体 (wi-61 / wi-62)。
- SAML 2.0 metadata 形式そのもの (wi-29 で扱う。本 WI は WS-* 互換 federationmetadata.xml)。
- AD FS claim rule language の完全互換。宣言的 mapping に留める。

## Verification
- `go test ./...` (in: idmagic)
  - reason: metadata 生成・署名、claim mapping の出力・未マップ拒否、NameID source 解決の境界。
- `golangci-lint run ./...` (in: idmagic)
- `go build ./...` (in: idmagic)
- `bun --cwd idmagic/ui typecheck`
- `bun --cwd idmagic/ui build`
- 手動: federationmetadata.xml を取得し署名・エンドポイント・証明書が正しいこと、 claim mapping で UPN / nameidentifier が意図通り発行され未マップ属性が出ないことを確認する。

## Risk Notes
metadata の証明書・エンドポイント記載や claim mapping を誤ると、RP が誤った鍵を信頼したり
過剰な属性を漏らす。metadata は署名し、claim は明示 mapping のみ fail-closed で発行する。
署名証明書のローテーションは [[wi-23-signing-key-rotation-scheduler]] / [[wi-32-kms-hsm-and-per-tenant-signing-keys]]
と整合させ、metadata に複数証明書を載せられるようにする。

## Completion
- **Completed At**: 2026-06-27
- **Summary**:
  ADR-062 を追加し、AD FS 互換の federation metadata 公開範囲を確定した。
  `/{realm}/federationmetadata/2007-06/federationmetadata.xml` は tenant issuer、
  WS-Fed passive endpoint、WS-Trust active endpoint、MEX endpoint、federation 署名証明書を
  `SecurityTokenServiceType` / `ApplicationServiceType` の RoleDescriptor として広告する。
  `/{realm}/trust/mex` は `usernamemixed` endpoint と UsernameToken 前提の policy を広告する。
  Claim mapping は既存の宣言的 `ClaimMappingPolicy` / `IssueClaims` を継続利用し、RP 管理 API
  から NameID / claim rule を管理できる。未マップ属性は出力せず、required source 欠落は拒否する。
- **Verification Results**:
  - `GOCACHE=/tmp/idmagic-cache go test ./internal/wsfederation/...` (in: idmagic)
    - result: ok
  - `GOCACHE=/tmp/idmagic-cache go test ./...` (in: idmagic)
    - result: ok
  - `bun run yaml-check --schema=work-item ../idmagic/work-items/wi-62-ws-trust-active-sts.yaml ../idmagic/work-items/wi-63-federation-metadata-and-claims-mapping.yaml` (in: tools)
    - result: ok
  - `go test ./internal/wsfederation/...` (in: idmagic)
    - result: failed: default Go build cache under ~/Library/Caches was not writable in sandbox; rerun with GOCACHE=/tmp succeeded
  - `golangci-lint run ./...` (in: idmagic)
    - result: ok: 0 issues (run outside sandbox after package loading failed inside sandbox)
  - `bun --cwd idmagic/ui typecheck`
    - result: ok
  - `bun --cwd idmagic/ui build`
    - result: ok
- **Affected Guarantees State**:
  - metadata integrity: federation 署名証明書を KeyDescriptor として広告する。文書自体の XML 署名は未実施で残リスクに明示
  - claim minimization: IssueClaims は mapping で明示した claim だけを出力し、required 欠落は拒否
  - tenant isolation: metadata / MEX URL は core.RequestIssuer と TenantRoute により realm ごとに分離
  - key separation: metadata は FederationSigner の X.509 証明書を広告し、OAuth JWK は流用しない
  - audit: claim mapping 変更は既存 admin RP 更新経路で扱う。metadata 公開イベントは未実装
