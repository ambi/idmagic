---
status: accepted
authors: [tn]
created_at: 2026-07-04
---

# ADR-005: DPoP をデフォルトのセンダー制約方式とし、mTLS をオプションで提供する

## コンテキスト

OAuth 2.0 の Bearer Token は「持ち主が誰でも使える」設計のため、漏洩時の被害が大きい。
これを緩和する「センダー制約付きトークン (Sender-Constrained Token)」には主に 2 方式がある:

- **DPoP (RFC 9449)**: クライアントが各リクエスト時に JWK で署名した DPoP 証明を提示する。
  クライアント側で鍵管理が必要だが、HTTP リクエストレベルで完結する。
- **mTLS (RFC 8705)**: TLS 層でクライアント証明書を提示する。
  PKI 運用とプロキシ設定が必要。FAPI で長年使われてきた。

FAPI 2.0 はどちらも許可する（§5.3）。

## 決定

本アプリ IdP は DPoP をデフォルト推奨方式とし（WebApp / SPA / ネイティブアプリのいずれにも
実装でき、TLS 終端プロキシの設定変更を要求しない）、mTLS をオプションで提供する（すでに PKI を
運用している B2B / FAPI 系の組織に自然に統合できる）。FAPI プロファイル
(`fapi_profile = "fapi_2_security_profile"`) を宣言したクライアントは少なくとも一方を必須とし、
一般プロファイルではセンダー制約はオプション（クライアントメタデータの `dpop_bound_access_tokens`
で宣言）とする。

現在の設計は [`backend/oauth2/ARCHITECTURE.md`](../backend/oauth2/ARCHITECTURE.md) にある。

## 却下した代替案

- **Bearer Token のみで運用**: 漏洩 1 回で広範囲のリプレイが可能。FAPI 必須要件にも反する
- **DPoP を必須に**: クライアントエコシステムの移行が間に合わない。FAPI でも mTLS は許可
- **証明書ピンニングだけ**: トークンとリクエストのバインディングがないため、リプレイ可能

## 影響

- アクセストークンとリフレッシュトークンの両方を `cnf` でバインドする
  （リフレッシュトークンのバインディングはストアレコードの `sender_constraint` フィールドに保持）
- `/introspect` 応答は `cnf` を含めることで、リソースサーバーが
  DPoP 証明をリクエストレベルで再検証できる
- `requirements.md §7` で DPoP の `iat` クロックスキューを明示しているため、
  ライブラリ差し替え時にも仕様が保たれる
