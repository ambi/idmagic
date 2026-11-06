---
status: pending
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-11-07
priority: p2
depends_on: []
change_kind: refactor
spec_impact: { kind: none, reason: "上流 OIDC の ID Token で受理するアルゴリズム、issuer、audience、有効期限の判定と、MCP gateway と Directory Connector が送る JWT のヘッダー、クレーム、署名方式を変えない。" }
---

# golang-jwt の直接利用を tokens_jose へ統合する

## 動機

[ライブラリ選定の見直し](../done/wi-90339-review-library-adoption.md)で、golang-jwt の直接利用をやめ、自前の `tokens_jose` へ統合すると決めた。
IdP 自身のトークンの発行と、DPoP、client assertion、ID-JAG、SVID、SET の検証は、既に Go 標準の暗号を使う `backend/shared/security/tokens_jose/` などが担っている。
golang-jwt を使うのは本番の 3 ファイルだけである。

- `authentication/internal/federation/protocol_oidc/client.go`：上流 OIDC の ID Token を RS256 で検証する（issuer、audience、有効期限の必須化）。
- `mcpgateway/internal/exchange_http/client.go`：DPoP proof を ES256 で署名する。
- `directoryconnector/internal/api_http/client.go`：client assertion を ES256 で署名する。

JWT の扱いを一つの実装へ寄せれば、アルゴリズムと鍵の対応、クレームの検証の規則を一か所で保てる。
go-jose への全面移行は、JWE や JSON シリアライズなどの不要な入力処理を増やすため採らない。
golang-jwt は go-webauthn の間接依存として残る。

## 対象範囲

- `tokens_jose` に ES256 の署名関数を追加し、2 か所の署名を移す。
- 上流 OIDC の ID Token の検証を、`tokens_jose` の署名検証とクレームの検証へ移す。
- golang-jwt を直接依存から外す。

## 対象外

- `tokens_jose` の既存の検証器の変更。
- go-webauthn が持つ間接依存の除去。

## 設計

ES256 の JWS 署名は、ASN.1 DER ではなく、32 バイトずつの r と s を連結した形で表す（RFC 7518 §3.4）。
既存の `verifyJWTSignature` は ES256 の検証でこの形を扱うので、署名関数は同じ表現を作り、検証器との往復をテストで確かめる。

ID Token の検証は、現在の golang-jwt の設定（`WithValidMethods(RS256)`、`WithIssuer`、`WithAudience`、`WithExpirationRequired`、時刻の注入）と同じ判定を保つ。
`kid` がない場合の鍵の選択と、既存のファジングテスト（`id_token_fuzz_test.go`）を移行後も通す。

## 計画

1. 現在の ID Token の受理と拒否を、既存のテストで網羅しているかを確かめ、欠けていれば先に足す。
2. ES256 の署名関数を追加し、検証器との往復を確かめる。
3. 3 か所を移し、golang-jwt を直接依存から外す。

## タスク

- [ ] T001 [Characterize] ID Token の判定と、送る JWT の形をテストに固定する。
- [ ] T002 [App] ES256 の署名関数を追加する。
- [ ] T003 [App] 3 か所を `tokens_jose` へ移す。
- [ ] T004 [Verify] 直接依存の除去とファジングを確かめる。

## 検証

- `mise run test-go-package -- ./backend/shared/security/tokens_jose/...` と、触れる各パッケージを実行する。
- `mise run test-go-fuzz` で ID Token の解析のファジングを実行する。
- `mise run check-boundaries` で共有パッケージへの依存を確かめる。

## リスク

署名の表現やクレームの検証を誤ると、上流の ID Token の受理や送る JWT の互換性が壊れる。
往復テストと既存の拒否テストで保つ。
