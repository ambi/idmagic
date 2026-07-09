---
status: completed
authors: ["tn"]
risk: low
created_at: 2026-07-05
---

# SCIM ServiceProviderConfig で authenticationSchemes を申告する

## Motivation
`/scim/v2/ServiceProviderConfig` のレスポンスで `authenticationSchemes` が `null` を
返しており、RFC 7643 §5 に非準拠。同属性は REQUIRED の多値複合属性で、サービス
プロバイダが受け付ける認証方式を宣言する必要がある。一部の外部 IdP コネクタは
この属性を読んで認証方式を自動判定するため、`null` だと連携の自動検出に失敗しうる。

IdMagic の SCIM API は Bearer トークン認証 ([[wi-31-scim2-provisioning]] の ScimToken)
なので、少なくとも `oauthbearertoken` を 1 件申告しなければならない。実装
(`handleGetServiceProviderConfig`) が `AuthenticationSchemes` を一切セットしておらず、
Go の nil スライスが `null` にシリアライズされているのが原因。

## Scope
- **scl**:
  - §3.3 interfaces: `GetScimServiceProviderConfig` の description に、レスポンスが
    対応認証方式 (最低 `oauthbearertoken`) を `authenticationSchemes` で申告する契約を
    明記する。output は opaque な `Map<String,JSON>` のままで構造は変えない。
- **go**:
  - `handleGetServiceProviderConfig` で `AuthenticationSchemes` に Bearer 方式
    (`type: oauthbearertoken`, `name`, `description`, `specUri: RFC 6750`) を 1 件設定する。
- **test**:
  - `scim_test.go` に、`authenticationSchemes` が非 null で `oauthbearertoken` を
    含むことのアサーションを追加する。

## Out of Scope
- bulk / sort / changePassword / etag の対応追加 (いずれも RFC 上 OPTIONAL、別途判断)。
- OAuth 2.0 で発行したアクセストークンの受け入れ (ScimToken glossary に将来余地として記載済み)。
- `authenticationSchemes` に複数方式を列挙すること (現状 Bearer 単一で十分)。

## Verification
- `just yaml-check-work-items`
- `just check-ids`
- `just test-go`
- `just lint-go`
- 手動: `curl .../scim/v2/ServiceProviderConfig` で `authenticationSchemes` が
  `oauthbearertoken` を含む配列で返ることを確認する。

## Risk Notes
レスポンスへの属性追加のみで既存フィールドの意味は変えない。SCIM クライアントは
未知/追加属性を無視するのが原則のため後方互換。認証・ルーティングには触れない。

## Completion
- **Completed At**: 2026-07-05
- **Summary**:
  `/scim/v2/ServiceProviderConfig` の `authenticationSchemes` が Go の nil スライスの
  ため `null` を返し RFC 7643 §5 非準拠だった問題を修正した。`handleGetServiceProviderConfig`
  で Bearer トークン方式 (`type: oauthbearertoken`, specUri: RFC 6750) を 1 件申告する
  ようにした。ドメインモデルの匿名構造体を名前付き型 `AuthenticationScheme` に切り出し、
  ハンドラからの初期化を可読にした。SCL は `GetScimServiceProviderConfig` の description に
  「対応認証方式を authenticationSchemes で申告し、最低 oauthbearertoken を含める」契約を
  明記した (output は opaque な Map<String,JSON> のまま)。
- **Affected Guarantees State**:
  - SCIM 2.0 ServiceProviderConfig が RFC 7643 §5 に準拠すること: satisfied
- **Verification Results**:
  - `just yaml-check-work-items` / `just check-ids`
    - result: ok (All 202 record id(s) OK)
  - `just yaml-check-scl`
    - result: ok (All 12 file(s) OK)
  - `go build ./internal/scim/...`
    - result: ok
  - `just lint-go`
    - result: ok (0 issues)
  - `go test ./internal/scim/...`
    - result: ok (authenticationSchemes が非 null かつ oauthbearertoken を含むことを
      アサートするテストを追加、pass)
  - `just scl-render`
    - result: ok (`spec/idmagic.html` の description のみ更新。openapi/schema は output が
      opaque なため差分なし)
