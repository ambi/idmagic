---
status: accepted
authors: [tn]
created_at: 2026-07-04
---

# ADR-023: private_key_jwt クライアント認証の検証規則

## コンテキスト

ADR-008 で `private_key_jwt` を「推奨クライアント認証方式」と定め、
Discovery (`token_endpoint_auth_methods_supported`) と grant matrix
(`token_endpoint_auth_methods`) に宣言済みだった。しかし実装は「枠組みのみ」で、
Discovery が広告する方式を実際には検証できない（仕様が約束しているのに実装が応えない）
状態だった。FAPI クライアントはこの方式でしか認証できないため、これは適合性の欠落である。

`private_key_jwt` (RFC 7521 / RFC 7523) は、クライアントが自分の秘密鍵で署名した
JWT (`client_assertion`) をトークンエンドポイント等に提示する方式。共有秘密
(`client_secret_*`) より鍵漏洩時の被害が小さく、IdP 側に検証用の秘密を置かない。

## 決定

（ADR-008 を実装に落とす）

`client_assertion` の検証規則を固定する。alg は PS256 / ES256 のみ（`alg: none` や HMAC は
アルゴリズム混乱攻撃防止のため拒否）、`iss === sub === client_id` (RFC 7523 §3)、`aud` は
このサーバーの issuer 識別子・エンドポイント URL のいずれかに一致、署名鍵はクライアント登録の
インライン `jwks` を優先し無ければ `jwks_uri` から解決、`exp` は必須かつ寿命を有界化して
リプレイ窓を確定させる、`jti` は単回使用（DPoP の jti とは別名前空間・別 TTL の独立ポート —
責務が異なるため）、`client_assertion` と Basic/secret の同時提示は `invalid_request`
(RFC 6749 §2.3) とする。

現在の設計は [`backend/oauth2/ARCHITECTURE.md`](../backend/oauth2/ARCHITECTURE.md) にある。

## 影響

- `ClientAssertionReplayStore` ポート + memory / valkey アダプタを追加。
- `authenticateClient` に `ClientAuthOptions { issuer, clientAssertionReplayStore }` を渡す。
  /token・/par・/introspect・/revoke の各ルートが issuer と replay store を保持する。
- `register-client.ts` は `private_key_jwt` クライアントに `jwks` または `jwks_uri` を要求し、
  secret は secret ベース方式のときだけ発行する。
- `verifyClientAssertion` は HTTP Context 非依存の純粋関数として切り出し、単体テスト可能
  （`adapters/http/client-authentication.test.ts`）。

## jwks_uri の SSRF 対策（実装メモ）

`jwks_uri` は登録時に許可ホスト/スキーム (https のみ・内部 IP 拒否) を検証する前提。
本アプリはインライン `jwks` を主経路とし、`jwks_uri` は production の枠組みとして残す。

## 却下した代替案

- **client_secret_jwt (HMAC)**: ADR-008 で却下済み（対称鍵は漏洩時の被害が大きい）。
- **jti リプレイを DpopReplayStore と共用**: TTL・名前空間・監査意味論が異なるため分離した。
- **aud をトークンエンドポイント URL のみ許可**: issuer 識別子を aud とする実装が多く、
  相互運用性のため両方を受理する。
