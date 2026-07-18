---
status: accepted
authors: [tn]
created_at: 2026-07-18
---

# ADR-125: client secret を重複期間付き credential として rotation する

## コンテキスト

単一の `client_secret_hash` を置換するだけでは、RP 側の設定差し替え中に認証が停止する。Application を唯一の編集面とする ADR-066 を維持しながら、secret-based OAuth2 client に安全な運用上の切替窓を与える必要がある。

## 決定

- secret は `oauth2_client_secrets` の credential として保存し、平文は rotation 成功応答だけで返す。
- rotation は current credential と overlap 中の previous credential の最大2件を受理する。旧 credential は 1..30 日（既定7日）の expiry を得るか、0日指定で即時 revoke される。
- 既存 `clients.client_secret_hash` は rollout 中に credential が無い client を認証する dual-read/backfill 用として残す。
- Application の OIDC editor だけが rotation endpoint を公開する。private_key_jwt、mTLS、public client は対象外である。
- `ClientSecretRotated` は actor、client、grace だけを監査し、secret と hash をイベントへ含めない。

## 却下した代替案

- client 行の hash を即時置換する: RP の無停止切替を実現できない。
- 任意個数の旧 credential を保持する: 期限・削除・認証時間の管理が増え、通常の単一切替要求を超える。
- 低レベル OAuth client 管理画面に endpoint を置く: Application を唯一の編集面にする既存決定と競合する。

## 影響

- `OAuth2.models.ClientSecretCredential`、`OAuth2.events.ClientSecretRotated`、Application の rotation HTTP 契約を追加する。
- token client authentication は credential を全件照合し、未移行 client の legacy hash も受理する。
- DB schema に `oauth2_client_secrets` を追加する。
