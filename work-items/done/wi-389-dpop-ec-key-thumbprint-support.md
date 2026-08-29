---
depends_on: []
status: completed
authors: [tn]
risk: low
created_at: 2026-08-22
priority: p2
change_kind: bugfix
evidence_policy: risk-based-v2
initial_context:
  specification: [docs/contexts/oauth2/standards.md]
  typespec: []
  source:
    - backend/shared/security/tokens_jose/dpop_verifier.go
  tests:
    - backend/shared/security/tokens_jose/dpop_verifier_test.go
    - backend/oauth2/handlers_http/userinfo_handler_test.go
  stop_before_reading: [frontend, spec, backend/signingkeys]
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
- [x] T001 [Domain] `jwkThumbprint` (dpop_verifier.go) を kty 分岐に書き換え、EC (`crv`/`kty`/`x`/`y`)
  を正しい正規メンバー集合として扱う。未対応 kty は明示エラーで拒否する。
  Unit RED: `TestJWKThumbprintFollowsRFC7638CanonicalForm`。標準行: `RFC9449-TOKEN-BINDING`。
- [x] T002 [Test] wi-129 で固定した `TestVerifyDPoPES256ProofFailsAtThumbprint` を
  `TestVerifyDPoPAcceptsES256Proof` へ書き換えた。RFC 7638 の既知ベクタと照合する直接テスト
  `TestJWKThumbprintFollowsRFC7638CanonicalForm` を追加した。
- [x] T003 [Acceptance] `/userinfo` で ES256 proof が受理されることの受け入れ検査
  `TestUserInfoDPoPAcceptsES256Proof` を置いた。シナリオ: `REQ-OAUTH2-045`。
- [x] T004 [Verify] `mise run verify` を通した。

## Verification
- `mise run verify-go`
- `mise run test-go -- ./backend/shared/security/tokens_jose/...` で DPoP 関連テストが全て pass
  し、ES256 proof が `cnf.jkt` 付きで受理されることを確認する。

## Risk Notes
サムプリント値の計算方式を変えるため、既に発行済みの `cnf.jkt` (RSA 鍵由来) が本修正の前後で
変わらないことを確認する — RSA 分岐の必須メンバー集合 `{e, kty, n}` は変更しないため、既存の
RSA-DPoP クライアントの jkt 値に影響は無いはずだが、修正後にテストで明示的に固定する。

## Completion

- **Completed At**: 2026-08-30
- **Summary**:
  `jwkThumbprint` の正規メンバー集合を `kty` で分岐させ、EC は RFC 7638 §3.2 が定める `{crv, kty, x, y}` で計算するようにした。宣言どおり受理される ES256 DPoP proof が、署名検証を通ったあと最終段の jkt 算出で必ず落ちていた食い違いが解けている。未対応の `kty` は空のサムプリントで通さず明示エラーで拒否する (fail-closed)。RSA の集合 `{e, kty, n}` は変えていないので、発行済みの RSA 由来 `cnf.jkt` は動かない。`mise run spec-diff` は `no normative specification change against main` を返す。標準行 `RFC9449-TOKEN-BINDING` は元から MUST を宣言しており、実装がそれに追いついた変更である。
- **Acceptance RED Evidence**:
  - **Test**: `TestUserInfoDPoPAcceptsES256Proof` (`backend/oauth2/handlers_http/userinfo_handler_test.go`)
  - **Requirement**: N/A: 該当する規範は `docs/contexts/oauth2/standards.md` の標準行 `RFC9449-TOKEN-BINDING` (MUST) であり、`REQ-` シナリオではない。近接する `REQ-OAUTH2-045` は `ath` による結び付けを述べる別の要件なので、ここには挙げない。
  - **Observed Failure**: `valid ES256 proof status=400 body={"error":"invalid_token","error_description":"DPoP key binding mismatch"}`
  - **Detection Reason**: 保護リソース `/userinfo` という、呼び出し側が結合の成否を観測できる最も狭い境界で、正しい ES256 proof の受理と、別の EC 鍵で署名した proof の拒否を対で確認する。受理だけを見るテストは jkt を照合しない実装にも通るため、結合が実際に効いていることを拒否側で固定した。期待値の jkt は正規メンバー集合から検証対象とは独立に組み立てており、実装を呼び戻していない。
- **Unit RED Evidence**:
  - **Test**: `TestJWKThumbprintFollowsRFC7638CanonicalForm` (`backend/shared/security/tokens_jose/dpop_verifier_test.go`)
  - **Requirement**: N/A: 上と同じ理由で、対応する `REQ-` シナリオを持たない。固定しているのは RFC 7638 §3.2 の正規メンバー集合と、それを要求する `RFC9449-TOKEN-BINDING` である。
  - **Observed Failure**: EC の部分試験が `jwk missing required member: e`、未対応 kty の部分試験が `expected unsupported-kty rejection, got jwk missing required member: e`。RSA の部分試験は当初から通った。
  - **Detection Reason**: サムプリントの値そのものを固定する。RSA は RFC 7638 §3.1 が本文に書き下している `NzbLsXh8uDCcd-6MNwXF4W_7noWXFZAfHkxZsRGC9Xs` と照合するので、EC 分岐の追加が既存の RSA 由来 jkt を動かせば落ちる。EC は RFC 7515 Appendix A.3.1 の鍵に対して RFC 7638 §3 の手順をこの実装とは独立に適用した値を固定するので、メンバー集合や順序を取り違えた実装は通らない。`y` を落とした jwk と `OKP` の jwk が、集合を緩めた実装と fail-open した実装をそれぞれ落とす。
- **Change-Resistance Results**:
  リスクは `low` のため必須ではないが、代表的な誤実装を 2 つ入れて実測した。EC 分岐の必須メンバーから `y` を除くと、`TestJWKThumbprintFollowsRFC7638CanonicalForm` の EC ベクタ照合が値の相違で落ち、`TestUserInfoDPoPAcceptsES256Proof` も jkt 不一致で落ちた。`default` 分岐を `required = []string{"kty"}` に緩めると、未対応 kty の部分試験が落ちた。
- **Verification Results**:
  - `mise run verify` - passed (exit 0)
  - `mise run test-go-package -- ./backend/shared/security/tokens_jose/...` - ok
  - `mise run test-go-package -- ./backend/oauth2/handlers_http/...` - ok
  - `mise run spec-diff` - `no normative specification change against main`
