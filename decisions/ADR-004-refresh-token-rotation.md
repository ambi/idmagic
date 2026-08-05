---
status: accepted
authors: [tn]
created_at: 2026-07-04
---

# ADR-004: リフレッシュトークンを毎回ローテーションし、再利用検出時にファミリー一括失効する

## コンテキスト

リフレッシュトークンは（access_token と比較して）長寿命のため、漏洩した場合の被害が大きい。
RFC 9700 §4.14 は、特に public クライアント（SPA / ネイティブ）について、
ローテーションと再利用検出を強く推奨している:

> The authorization server MUST ... rotate the refresh token on every use.
> ... If a refresh token is presented that was already used, the authorization
> server MUST revoke all refresh tokens that were issued based on the
> originally issued refresh token.

## 決定

すべてのクライアント（public / confidential 問わず）に対してリフレッシュトークンの
ローテーションを必須とする。使用のたびにローテーションし、`family_id` で祖先・子孫の
チェーンを追跡する。すでにローテーション済み（`rotated=true`）のトークンが再提示されたら、
同じ `family_id` のトークンをすべて失効させる（ファミリー一括失効）。並行使用（タブを 2 枚
開いている SPA など）もリプレイ攻撃と区別せず、常に片方成功・片方失効とし、設計を単純に保つ。
`absolute_expires_at` は発行時に設定し（30 日固定）、ローテーションしても延長しない。

現在の設計は [`backend/oauth2/ARCHITECTURE.md`](../backend/oauth2/ARCHITECTURE.md) にある。

## 却下した代替案

- **リフレッシュトークンを長寿命のままローテーションしない**: 漏洩時の被害が指数的に大きい
- **public クライアントだけローテーション**: confidential クライアントも侵入は起こりうる。
  「全クライアント一律」のほうが運用と監査が単純
- **5 秒 grace period**: 実装が複雑化。Valkey のような外部状態ストアが必要。
  本アプリは "single success, family-revoke on reuse" を採用

## 影響

- `RefreshTokenRecord` は `family_id` / `parent_id` / `rotated` / `revoked` フィールドを持つ
- リフレッシュトークン用ストアは「`family_id` での一括失効」を効率的にできる必要がある
- 監査ログは `RefreshTokenRotated` と `RefreshTokenReuseDetected` を区別する
- `requirements.md §5` と `scenarios.feature` にローテーション動作を明記する
