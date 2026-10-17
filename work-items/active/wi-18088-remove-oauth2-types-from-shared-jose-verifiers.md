---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-18
priority: p2
depends_on: [wi-91505-move-the-jwt-token-signer-into-oauth2]
change_kind: refactor
spec_impact: { kind: none, reason: "共有の JWKS 取得とリプレイ検証から OAuth2 の型への依存を外す。取得元の優先順位、SSRF 拒否、署名検証、リプレイ拒否、HTTP 応答、永続状態とイベントは変えない。" }
---

# 共有の JOSE 検証器が OAuth2 の型に依存しないようにする

## 動機

`tools/check/boundary-debt.json` の `shared-dependency:shared->OAuth2` は、JWTSigner の移動後も残る。
`jwks_resolver.go` が OAuth2Client を、Client Assertion と DPoP の検証器が OAuth2 のリプレイ保存ポートを参照しているためである。

## 対象範囲

- JWKS の取得器は取得元の値を受け取り、OAuth2Client からの変換を利用側へ移す。
- 共有の検証器が必要とする最小のリプレイ保存操作を受け取り、OAuth2 の実装を構造的に適合させる。
- 対応する共有依存を台帳から除く。

## 対象外

- JWKS の優先順位、キャッシュ、SSRF 対策、リプレイ保存の寿命と検証規則の変更。

## 設計

既存の `ResolveJWKSSource` と OAuth2Client 用 `Resolve` の空の取得元に対するエラーの差を確認し、既存の結果を維持する。
D5、D6 に従い、OAuth2Client の解釈は利用側に置き、共有側は JWKS の技術的な取得だけを担う。

## 計画

取得器とリプレイポートの呼び出し元を確認し、既存テストを保って依存を外す。

## タスク

- [ ] T001 [Design] 入力とリプレイポートを比較する。
- [ ] T002 [App] 共有の検証器と呼び出し元を更新する。
- [ ] T003 [Tooling] 解消した共有依存を台帳から除く。
- [ ] T004 [Verify] 既存テストと境界を検証する。

## 検証

- `mise run check-boundaries` と `mise run check-boundary-debt-ratchet`：共有依存を解消する。
- `mise run test-go-package -- ./backend/shared/security/tokens_jose`：取得と検証の結果を保つ。
- `mise run lint-go` と `mise run test-go-changed`：呼び出し元の型と挙動を確認する。

## リスク

空の JWKS と URI のフォールバックを変えると、クライアントの認証結果が変わる。
既存の優先順位とエラーを維持する。
