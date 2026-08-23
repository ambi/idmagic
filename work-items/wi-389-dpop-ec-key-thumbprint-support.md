---
depends_on: []
status: pending
authors: [tn]
risk: low
created_at: 2026-08-22
priority: p2
change_kind: bugfix
affected_spec:
  - { path: docs/contexts/oauth2/standards.md, requirement: RFC9449-TOKEN-BINDING }
---

# DPoP proof の JWK サムプリント計算を EC (ES256) 鍵にも対応させる

## Motivation
`backend/shared/security/tokens_jose/dpop_verifier.go` は DPoP proof の `alg` として
`PS256` と `ES256` の両方を宣言どおり受理する (`alg must be PS256 or ES256`)。ES256 proof は
`publicKeyFromJWK` で EC 鍵として正しくパースされ、`verifyJWTSignature` の署名検証も通る。

しかし最終段の `jwkThumbprint` (RFC 7638 の JWK Thumbprint、`cnf.jkt` に載る値) が RSA 専用の
必須メンバー集合 `{e, kty, n}` を決め打ちしており、EC の JWK にはこれらのフィールドが存在しない
(`kty`, `crv`, `x`, `y` を持つ) ため、`jwk missing required member: e` で必ず失敗する。結果として、
署名が正しい ES256 DPoP proof も `verifyDPoP` の最終段で必ず拒否される。

`RFC9449-TOKEN-BINDING` (DPoP 利用時はアクセストークンに `jkt` を含め、提示された proof の鍵と
照合する) は MUST 要件であり、宣言どおり ES256 を受理する以上この結合は ES256 鍵でも成立しなければ
ならない。現状は宣言と実装が食い違っている。

[[wi-129-backend-test-coverage-improvement]] でカバレッジ追加中に発見した (テストは現状の拒否
挙動をそのまま固定しており、本 WI で修正後にテストごと更新する)。

## Scope
- `jwkThumbprint` を `kty` で分岐させ、EC の場合は RFC 7638 §3.2 が定める正規メンバー集合
  `{crv, kty, x, y}` でサムプリントを計算する。
- 未対応の `kty` (RSA/EC 以外) は明示的なエラーで拒否する (fail-closed を保つ)。
- `[[wi-129-backend-test-coverage-improvement]]` で現状の拒否挙動を固定した
  `TestVerifyDPoPES256ProofFailsAtThumbprint` を、ES256 proof が受理されることを検証するテストに
  書き換える。

## Out of Scope
- DPoP 以外 (client_assertion 等) の JWK サムプリント計算。`client_assertion_verifier.go` は
  `jwkThumbprint` を使っていない (署名検証のみ) ため影響しない。
- P-256 以外の EC curve (P-384 等) のサポート。`publicKeyFromJWK` は現状 P-256 のみを受理しており、
  本 WI もそれに合わせる。
- DPoP proof の `alg` 選択ポリシーやクライアント向けドキュメントの変更。

## Design
`jwkThumbprint` を次のように `kty` 分岐にする。`encoding/json.Marshal` は `map[string]any` の
キーを Unicode コードポイント順にソートして出力するため、各分岐の必須メンバー集合さえ正しければ
RFC 7638 §3.1 の lexicographic ordering 要件は自動的に満たされる。

```go
func jwkThumbprint(jwk map[string]any) (string, error) {
    kty, _ := jwk["kty"].(string)
    var required []string
    switch kty {
    case "RSA":
        required = []string{"e", "kty", "n"}
    case "EC":
        required = []string{"crv", "kty", "x", "y"}
    default:
        return "", fmt.Errorf("unsupported kty for thumbprint: %q", kty)
    }
    ...
}
```

検討した代替案:
- **jwk の形から RSA/EC を推測する (kty を見ない)**: `n`/`e` の有無で判定できなくもないが、
  `kty` は JWK 自体が申告している値であり、それを無視して形状から推測するのは自己申告と実体の
  food-for-thought な不一致を握りつぶす方向になる。`publicKeyFromJWK` も既に `kty` で分岐して
  いるので、同じ分岐軸に揃える。
- **EC のときだけ x/y の欠落を許容する (mixed required set)**: RFC 7638 の正規形はアルゴリズム
  ごとに固定のメンバー集合であり、緩めると thumbprint の値が RFC 準拠でなくなる (他実装と一致
  しなくなる)。採用しない。

## Plan
1. `jwkThumbprint` を上記の分岐に書き換える (dpop_verifier.go)。
2. `TestVerifyDPoPES256ProofFailsAtThumbprint` (wi-129 で追加) を
   `TestVerifyDPoPAcceptsES256Proof` に戻し、ES256 proof が `JKT` 付きで受理されることを検証する。
   `jwkThumbprint` の RFC 7638 準拠を確認する直接テスト (EC 鍵で既知のサムプリント値と照合する
   fixture、または RSA/EC 双方の canonical member set を確認するテスト) を追加する。
3. `publicKeyFromJWK` や `verifyJWTSignature` 側の変更は不要 (既に EC 対応済み)。

## Tasks
- [ ] T001 [Domain] `jwkThumbprint` (dpop_verifier.go) を kty 分岐に書き換え、EC (`crv`/`kty`/`x`/`y`)
  を正しい正規メンバー集合として扱う。未対応 kty は明示エラーで拒否する。
- [ ] T002 [Test] wi-129 で固定した `TestVerifyDPoPES256ProofFailsAtThumbprint` を
  ES256 proof の受理を検証するテストへ書き換える。RSA/EC 双方で thumbprint が RFC 7638 の
  既知ベクタ (または相互に一貫した値) と一致することを確認する直接テストを追加する。
- [ ] T003 [Verify] `mise run verify-go` を通す。

## Verification
- `mise run verify-go`
- `mise run test-go -- ./backend/shared/security/tokens_jose/...` で DPoP 関連テストが全て pass
  し、ES256 proof が `cnf.jkt` 付きで受理されることを確認する。

## Risk Notes
サムプリント値の計算方式を変えるため、既に発行済みの `cnf.jkt` (RSA 鍵由来) が本修正の前後で
変わらないことを確認する — RSA 分岐の必須メンバー集合 `{e, kty, n}` は変更しないため、既存の
RSA-DPoP クライアントの jkt 値に影響は無いはずだが、修正後にテストで明示的に固定する。
