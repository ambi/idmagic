---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-11-08
priority: p3
depends_on: []
change_kind: refactor
spec_impact: { kind: none, reason: "X-Forwarded-For から信頼するホップ数だけさかのぼってクライアント IP を決める規則と、その結果を使うログインの抑制、レート制限、アクセス条件の判定を変えない。" }
---

# クライアント IP の抽出を一つの実装にまとめる

## 動機

信頼するホップ数だけ X-Forwarded-For をさかのぼる処理は、`support_http.ExtractClientIP` と `application/usecases.AccessGate.ClientIP` の二つが同じ内容を持つ。
`signin` と `oauth2` の HTTP 層にも、前者へ委ねるだけの非公開の別名 `extractClientIP` がある。
同じ判定の二つの実装は、片方だけを直す変更で、ログインの抑制とアプリケーションのアクセス条件が別々のクライアント IP を見る状態を作り得る。
wi-34240 を中止した際に、この整理だけを独立して行うことにした。

## 対象範囲

- `AccessGate.ClientIP` の重複した実装を除き、判定を `support_http.ExtractClientIP` の一か所にする。
- `signin` と `oauth2` の `extractClientIP` を除き、呼び出し元から共有の関数を直接呼ぶ。

## 対象外

- 信頼するホップ数の規則、空文字を返す条件、`TRUSTED_FORWARDED_HOPS` の意味の変更。
- `Forwarded` ヘッダー（RFC 7239）への対応。

## 設計

`application/usecases` は HTTP のアダプターに依存しないので、`AccessGate.ClientIP` から `support_http` を呼ぶ形は採らない。
クライアント IP の解決は HTTP の要求を読む処理であり、アクセス条件の判定ではない。
そこで、`AccessGate` はクライアント IP を引数で受け取るだけにし、SAML と WS-Federation の HTTP 層が `support_http.ExtractClientIP` で解決して渡す。
信頼するホップ数は両モジュールの HTTP 層が既に持つ `TrustedForwardedHops` を使う。
持たない場合は、`GateTrustedForwardedHops` の配線を HTTP 層の依存へ移す。
着手時に `wsfederation/internal/handlers_http/service.go` の `ClientIP` の供給元と、SAML の依存の組み立てを読み、どちらの形かを確定する。

採らない案：判定を `shared/` の HTTP 以外のパッケージへ移し、両方から呼ぶ。
共有パッケージを一つ増やすが、呼び出し元はどちらも HTTP の要求を持つので、アダプターに置く今の所有を変える理由にならない。

## 計画

1. `AccessGate.ClientIP` の呼び出し元と、ホップ数の配線を確かめる。
2. HTTP 層で解決した値を渡す形へ移し、`AccessGate.ClientIP` と `GateTrustedForwardedHops` の不要になった配線を除く。
3. `extractClientIP` の別名を除く。

## タスク

- [ ] T001 [Refactor] SAML と WS-Federation のクライアント IP を `support_http.ExtractClientIP` で解決する。
- [ ] T002 [Refactor] `signin` と `oauth2` の `extractClientIP` を除く。
- [ ] T003 [Verify] 既存テストと境界の検査を通す。

## 検証

- 判定の規則は `support_http` の `misc_helpers_test.go` が固定している。`AccessGate.ClientIP` と `extractClientIP` のテストは、同じ入力と期待値のまま共有の関数のテストへ寄せる。
- 変更したパッケージとその逆依存は `mise run verify-changed` で確かめる。
- 境界は `mise run check-boundaries` で確かめる。

## リスク

ホップ数の配線を取り違えると、CIDR のアクセス条件が別の IP で判定される。
`TRUSTED_FORWARDED_HOPS` が 0 のときに空を返して fail-closed になることを、HTTP 層の既存の E2E で確かめる。
