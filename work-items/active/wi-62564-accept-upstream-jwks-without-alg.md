---
status: pending
authors: [tn]
risk: low
created_at: 2026-11-25
priority: p1
group: spec-conformance
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/authentication/standards.md, requirement: OIDC-CORE-ID-TOKEN-KID-OPTIONAL }
  - { path: docs/modules/authentication/standards.md, requirement: OIDC-CORE-ID-TOKEN-RS256 }
---

# 上流の JWK Set で `alg` を省いた RSA 鍵を、ID Token の検証に使う

## 動機

外部 OIDC の callback は、上流の JWK Set のうち `alg` が `RS256` の RSA 鍵だけを署名の検証の候補にする。
`backend/authentication/internal/federation/protocol_oidc/client.go` の `jwksDocument.rsaKeys` が、`key.Alg != "RS256"` の鍵を読み飛ばすためである。
JWK の `alg` は RFC 7517 §4.4 で任意であり、省く OP もある。
そうした上流では、ID Token の `kid` が鍵と一致しても `JWKS has no supported signing key` で検証が失敗し、利用者がサインインできない。

`use` は省略と `sig` を受け入れており（`OIDC-CORE-ID-TOKEN-KID-OPTIONAL`）、`alg` だけが省略を受け入れない。
[[wi-41643-declare-oauth2-acceptance-rows-for-optional-elements]] の試験の作成中に見つけた。

## 対象範囲

- `rsaKeys` の候補の条件を、`alg` が省略または `RS256` の RSA 鍵に改める。
- `OIDC-CORE-ID-TOKEN-KID-OPTIONAL` の「署名に使える RSA 鍵」の条件に、`alg` の省略を明記する。
- `alg` を省いた JWK だけを公開する上流で、callback が federated セッションを発行する受け入れ試験を足す。

## 対象外

- RS256 以外の署名アルゴリズムへの対応。`OIDC-CORE-ID-TOKEN-RS256` が制限として宣言している。
- `alg` が `RS256` 以外の値を持つ RSA 鍵（`RS384` など）の扱い。現状どおり候補から外す。

## 設計

候補の条件を `key.KTY == "RSA" && (key.Alg == "" || key.Alg == "RS256") && (key.Use == "" || key.Use == "sig")` にする。
ID Token のヘッダーの `alg` は `OIDC-CORE-ID-TOKEN-RS256` に従い引き続き RS256 だけを受け入れるため、`alg` を省いた鍵を RS256 以外の検証に使う経路は生じない。

採らない案：`alg` を省いた鍵を、ヘッダーの `alg` に従って任意のアルゴリズムに使う。
アルゴリズムの取り違えの攻撃の面を広げ、RS256 だけという宣言とも合わない。

## 計画

1. `spec-change` で `OIDC-CORE-ID-TOKEN-KID-OPTIONAL` の鍵の条件を更新する。
2. `oidc_upstream_acceptance_test.go` の組み立てで `alg` を省いた JWK を返し、受け入れ RED を確かめる。
3. `rsaKeys` を直して GREEN にする。

## タスク

- [ ] T001 [Spec] `OIDC-CORE-ID-TOKEN-KID-OPTIONAL` に `alg` の省略を明記する。
- [ ] T002 [Acceptance] `alg` を省いた JWK の上流で callback がセッションを発行する試験の RED を確認する。
- [ ] T003 [App] `rsaKeys` の候補の条件を改めて GREEN にする。
- [ ] T004 [Verify] 変更を検証する。

## 検証

- `mise run check-spec`：標準要件の変更の形式を確かめるため。
- `mise run test-go-package -- ./backend/authentication/internal/federation/...`：受け入れ試験と既存の `kid` の試験を通すため。

## リスク

`alg` を省いた暗号化用の鍵を候補に入れる懸念は、`use` の条件（省略または `sig`）が変わらないため増えない。
