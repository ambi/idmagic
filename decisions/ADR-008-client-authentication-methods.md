---
status: accepted
authors: [tn]
created_at: 2026-07-04
---

# ADR-008: クライアント認証方式を 5 種類サポートし、推奨優先順位を明示する

## コンテキスト

OAuth 2.0 はクライアント認証方式を複数定義しており、それぞれセキュリティ強度と運用コストが
異なる。すべてを実装するか、一部に絞るかの判断が必要だった。

業界調査:

- 大半の SaaS IdP は `client_secret_basic` と `client_secret_post` を実装する
- FAPI は `private_key_jwt` と `tls_client_auth` を強く推奨
- public クライアントは `none` のみ（PKCE 必須）

## 決定

以下の 5 方式をすべてサポートする: `private_key_jwt`（一般的な confidential クライアントに推奨）、
`tls_client_auth`（FAPI / B2B、mTLS PKI を持つ組織に推奨）、`none`（public クライアントに必須）、
`client_secret_post` / `client_secret_basic`（レガシー confidential 移行用、許容〜非推奨）。
`client_secret_jwt`（HMAC-SHA256 共有鍵方式）は実装しない — `private_key_jwt` の非対称署名の
ほうが鍵漏洩時の被害が小さく、両方を持つのは冗長なため。クライアント認証が失敗したときは
常に `401 invalid_client` とし、`client_id` が登録されているかどうかを開示しない
（`requirements.md §8`）。

現在の設計は [`backend/oauth2/ARCHITECTURE.md`](../backend/oauth2/ARCHITECTURE.md) にある。

## 却下した代替案

- **すべてを `private_key_jwt` に強制**: レガシー移行が非現実的
- **`client_secret_basic` のみ**: FAPI 要件を満たさない
- **新たな独自方式**: 標準に従わない設計は将来の再生成で破綻する

## 影響

- `client.schema.json` の `token_endpoint_auth_method` enum がこの 5 種類
- Discovery `token_endpoint_auth_methods_supported` がこの 5 種類
- `adapters/http/client-authentication.ts` が認証ロジックを実装する
  （本アプリでは `client_secret_basic` / `client_secret_post` / `none` /
  `private_key_jwt` を実装。`private_key_jwt` の検証規則は ADR-023 を参照。
  `tls_client_auth` はランタイムの mTLS 終端に依存するため枠組みのみ）
