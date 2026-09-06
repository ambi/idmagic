---
depends_on: []
status: pending
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-09-07
priority: p1
change_kind: bugfix
affected_spec:
  - { path: docs/contexts/api-tokens/standards.md, requirement: RFC9449-API-TOKEN-DPOP }
  - { path: spec/contexts/oauth2/models.tsp, symbol: IdMagic.Contract.DpopProofClaims }
---

# 保護リソースが DPoP 証明の `htu` をパスとして照合するので、適合クライアントが到達できない

## Motivation

[[wi-500-back-api-tokens-standards-rows-with-tests]] が `RFC9449-API-TOKEN-DPOP` にテストを対応付けているときに見つけた。

`backend/shared/http/support_http/auth.go:150` は、保護リソースの DPoP 証明を検証するとき `RequestHTU(c, "")` を期待値として渡す。`RequestHTU` は `strings.TrimRight(base, "/") + c.Request().URL.Path` なので、base が空文字なら返るのはパスだけ、たとえば `/realms/default/api/admin/v1/users` である。`verifyDPoP` は証明の `htu` をこれと完全一致で照合する。

同じ関数を、トークンエンドポイント（`backend/oauth2/handlers_http/token_handler.go:84`）と `/userinfo`（`userinfo_handler.go:77`）は `RequestHTU(c, d.Issuer)` で呼ぶ。こちらは絶対 URL になる。**同じ製品の中で、DPoP 証明の `htu` に 2 つの別々の形を要求している。**

RFC 9449 §4.2 は `htu` を「the HTTP target URI, without query and fragment parts」と定めており、`spec/contexts/oauth2/models.tsp:607` も `htu: url` と宣言している。したがって適合クライアントは絶対 URL を送る。それは保護リソースで必ず `htu mismatch` になる。

**送信者制約付きの API アクセストークンは、適合クライアントからは 1 度も使えない。** `/token` で受け取ったトークンを持って保護リソースへ行くと、同じクライアントが同じ規則で作った証明が今度は拒否される。この経路を通せるのは、保護リソースにだけパスを送るように作り分けた非適合クライアントだけである。

`docs/contexts/oauth2/internals.md:37` は「Proof の検証はパラメーター化せず、エンドポイントの種類で分ける」と書いているが、分けると書いてあるのは `ath` の要否であって `htu` の形ではない。`htu` がエンドポイントごとに違う形になることは、どの正典文書にも書かれていない。

## Scope

- 保護リソースの `htu` 期待値を、他のエンドポイントと同じ絶対 URL に揃える。
- 絶対 URL を持つ証明が保護リソースで受理されること、およびパスだけの証明が受理されないことを、保護されたエンドポイントから観測する。
- テナントの正規ロケーションが 2 形（`/realms/{realm}` と subdomain）あることを踏まえて期待値を組み立てる。`RequestHTU(c, d.Issuer)` を単純に写すと、path style で prefix が二重になる（`TenantURL` の doc コメントが同じ罠を挙げている）。
- `backend/shared/http/server_http/api_token_standards_test.go` の `TestDPoPBoundApiTokenVerifiesEveryProofElement` が、いまパスを無傷の値として使っている注記を落とす。

## Out of Scope

- `htm`、`iat`、`jti`、`ath`、サムプリントの検証。いずれも期待どおり働いている。
- mTLS による送信者制約。

## Risk Notes

- **期待値を絶対 URL にすると、prefix の二重化で全経路が落ちる。** テナントの 2 形それぞれについて、クライアントが送ったままの URL を復元できていることを観測する。
- **リバースプロキシ越しのスキームとホスト。** 期待値をリクエストの `Host` から組むか、テナントの正規 issuer から組むかで、プロキシ配下の振る舞いが変わる。どちらを採るかを Design で決める。

## Verification

- 保護リソースへ絶対 URL の `htu` を持つ DPoP 証明を提示した送信者制約付き API アクセストークンが、
  保護されたエンドポイントへ到達する。
- 別のリソースの絶対 URL を持つ証明は到達しない。
- テナントの正規ロケーションが path style と subdomain style のどちらでも、同じことが成り立つ。
- `mise run verify`
