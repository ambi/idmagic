---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-11-07
priority: p2
depends_on: [wi-77559-verify-jwt-svid-against-spiffe-rules]
change_kind: feature
affected_spec:
  - { path: docs/modules/workloadidentity/features/attestation-exchange.md, requirement: REQ-WORKLOADIDENTITY-003 }
  - { path: docs/modules/workloadidentity/features/trust-configuration.md, requirement: REQ-WORKLOADIDENTITY-008 }
  - { path: spec/modules/workloadidentity/models.tsp, symbol: IdMagic.Contract.WorkloadTrustBundleRefreshResponse }
---

# SPIFFE の信頼設定では、SPIFFE Bundle Endpoint の形式で鍵を読む

## 動機

信頼設定の `jwks_uri` は、通常の JWKS として取得して全ての鍵を署名の検証に使う。
SPIFFE のトラストバンドル（SPIFFE Bundle Endpoint が返す JWKS）には、`use: "jwt-svid"` の鍵と `use: "x509-svid"` の鍵が混ざり、`spiffe_sequence` と `spiffe_refresh_hint` も載る。

- `x509-svid` の鍵は `kid` を持たないことがあり、JWT-SVID の署名の検証に使ってはならない。現在は区別しない。
- `spiffe_refresh_hint` は、鍵の入れ替えに追従する再取得の間隔を示すが、使っていない。
- Bundle Endpoint の認証方式のうち `https_spiffe`（エンドポイント自身が X.509-SVID で TLS を張る）は、通常の Web PKI の検証では接続できない。

## 対象範囲

- `spiffe` の信頼設定では、`use` が `jwt-svid` の鍵だけを JWT-SVID の検証に使う。
- `spiffe_refresh_hint` を JWKS の再取得の間隔に反映するかを決め、反映する場合は実装する。
- Bundle Endpoint の認証方式 `https_web` に対応することを仕様に書く。`https_spiffe` に対応するかを決める。
- JWKS の再取得の結果（`WorkloadTrustBundleRefreshResponse`）で、`jwt-svid` の鍵の数を返す。

## 対象外

- SPIFFE フェデレーションの、他のトラストドメインのバンドルの自動の取り込み。
- `oidc` の信頼設定の JWKS の扱い。

## 設計

鍵の選別は、取得した JWKS を検証に渡す前の純粋な絞り込みとして `tokens_jose` に置く。
種別は [wi-77559](wi-77559-verify-jwt-svid-against-spiffe-rules.md) が信頼設定に足すものを使う。

`https_spiffe` は、エンドポイントの SPIFFE ID と、最初のバンドル（ブートストラップ）を管理者に登録させる必要があり、信頼設定の項目が増える。
着手時に、利用者の要求があるかで含めるかを決める。

## 計画

1. 選別と再取得の間隔の規則を仕様に書く。
2. 選別を実装し、`x509-svid` の鍵が混ざったバンドルでの検証をテストする。
3. 再取得の結果の表示を管理画面に反映する。

## タスク

- [ ] T001 [Spec] 鍵の選別、再取得の間隔、対応する認証方式を仕様に書く。
- [ ] T002 [Acceptance] `x509-svid` の鍵が混ざったバンドルでの検証の誤りを RED にする。
- [ ] T003 [App] 鍵の選別と再取得の間隔を実装する。
- [ ] T004 [UI] 再取得の結果の鍵の数を、`jwt-svid` の鍵の数として表示する。
- [ ] T005 [Verify] 検証を実行する。

## 検証

- `mise run test-go-package -- ./backend/shared/security/tokens_jose/...` と `./backend/workloadidentity/...`。
- `mise run check-schema`：TypeSpec の変更。

## リスク

- 再取得の間隔を短くすると、取得元への負荷が増える。下限を置く。
