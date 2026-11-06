---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-11-07
priority: p2
depends_on: []
change_kind: refactor
spec_impact: { kind: none, reason: "WebAuthn の登録と認証のセレモニー、返す options、受理と拒否の判定、保存するセッションの形を変えない。外部ライブラリの型が現れる位置だけを変える。" }
---

# go-webauthn の型を authentication モジュールの内部へ閉じ込める

## 動機

[ライブラリ選定の見直し](../done/wi-90339-review-library-adoption.md)で、WebAuthn は go-webauthn を維持すると決めた。
検証器の自前化は、実機と認証器ごとの癖（Windows Hello の RSA 鍵、EdDSA、署名カウンター 0、BE と BS の組み合わせ）への対応を確かめる手段がなく、採らない。
go-passkeys は [v0.4.1 の VerifyAssertion](https://github.com/go-passkeys/go-passkeys/blob/v0.4.1/webauthn/webauthn.go) が UP を検査しないため採らない。

一方、外部の型が公開ポートと複数の層に漏れている。
`webauthn.SessionData` は `authentication/webauthn/ports` と DB とメモリの実装に、`webauthn.WebAuthn` はユースケース、HTTP 層、`module.go`、`wiring.go`、`cmd/internal/bootstrap/deps.go`、`shared/http/server_http/routes.go` に現れる。
ライブラリの更新や置き換えの影響を、authentication モジュールの内部のアダプターに限る。

## 対象範囲

- 公開ポートの `SessionData` を、authentication モジュールが所有する型へ置き換える。
- `webauthn.WebAuthn` の構築を authentication モジュールの内部へ移し、組み立て地点と HTTP 層から外部の型を外す。
- DB に保存するセッションの JSON の形を保つ。

## 対象外

- 受け付ける attestation の形式の変更。これは [wi-48351](wi-48351-limit-webauthn-attestation-formats.md) が扱う。
- go-webauthn 以外のライブラリへの置き換え。

## 設計

境界の判断手順を着手時に適用し、ポートの型、組み立て地点、モジュール間の依存の変化を記録する。
薄いラッパーを足すだけでは第三者コードは減らないため、目的は入力処理の削減ではなく、型の広がりを一つのアダプターに限ることだと明記する。

## 計画

1. 保存するセッションの形と、セレモニーの応答を特性化テストに固定する。
2. 所有する型とアダプターを作り、ポート、DB、ユースケース、HTTP 層、組み立てを移す。

## タスク

- [ ] T001 [Design] 境界の判断の記録を設計に書く。
- [ ] T002 [Characterize] セッションの保存形とセレモニーの応答を固定する。
- [ ] T003 [Adapters] 外部の型をアダプターへ閉じ込める。
- [ ] T004 [Verify] 境界と依存を確かめる。

## 検証

- `mise run test-go-package -- ./backend/authentication/...` を実行する。
- `mise run check-boundaries` を通す。

## リスク

セッションの直列化の形が変わると、移行中のセレモニーが失敗する。
保存形を特性化テストで固定する。
