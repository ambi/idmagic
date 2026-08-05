---
status: accepted
authors: [tn]
created_at: 2026-07-15
---

# ADR-106: IdentityManagement / Authentication の credential・スロットリング設定を ARCHITECTURE 層の文書に移す

## コンテキスト

[[ADR-103]] は SCL 3.0 の `objectives` を観測可能な SLI に対する SLO だけに限定し、config/security
policy/lifetime 設定は ADR または `ARCHITECTURE.md` へ移すことを決定した。wi-209 で
`spec/contexts/identity-management.yaml` と `spec/contexts/authentication.yaml` (いずれも SCL 2.0)
を SCL 3.0 へ移行した際、以下の `objectives` は `indicator` / `target` / `window` / `budgeting` を
持つ観測可能な比率目標ではなく、単一の設定値・運用方針の集合だった。値そのものは移行によって
変更しない。

## 決定

`objectives` から移した config/security policy/lifetime 値を本 ADR に集約する。値そのものは
SCL 3.0 移行によって変更しない。

### 1. IdentityManagement

- **CSV インポートの入力上限**: `ImportAdminUsers` は CSV を 1 MiB (1,048,576 bytes)、1,000 行、
  1 field あたり 64 KiB を上限として拒否する。この上限自体は `interfaces.ImportAdminUsers.requires`
  に反映済みで、「なぜ 1 MiB か」という運用上の根拠 (メモリ上限を守りつつ通常の管理者運用で
  十分な行数を許容する) を本 ADR に残す。
- **User soft-delete の猶予期間**: `PendingDeletion` から `Deleted` への自動遷移は 30 日
  (2,592,000 秒) を既定とする。Google / Microsoft / Apple 等の業界標準的な猶予期間 (7〜30 日) に
  合わせた値であり、`states.UserLifecycle` の `PendingDeletion → Deleted` transition guard に
  既に反映済み。

### 2. Authentication: パスワードポリシー、ログインスロットリング、TOTP、WebAuthn、
   リカバリコード、パスワードリセットトークン、ログインセッション Cookie

以下の設定値の権威源は本 ADR であり、対応する強制点は各 SCL interface/model の
`requires`/`fields` に反映済み。数値そのものの設計根拠と現在の挙動は
[`backend/authentication/ARCHITECTURE.md`](../backend/authentication/ARCHITECTURE.md) の
Password lifecycle・Login throttling・WebAuthn/passkey MFA and recovery codes 各セクションに
ある。

- パスワードポリシー: `min_length=12`, `max_length=128`, `history_depth=5`,
  `forbid_user_identifier_similarity=true`, `common_password_dictionary=bundled`,
  `breached_password_check_enabled=false`。
- ログインスロットリング: per-account/per-IP とも 900 秒窓で失敗 10 回/30 回に達すると
  900 秒ロックアウト、`counter_scope=cluster_wide`（[[ADR-077]] の共有 Valkey ストアに
  login_throttle を含む）、`identifier_hash=sha256`、`degraded_store_behavior=fail_closed`。
- TOTP (RFC 6238): `algorithm=SHA1`, `step_seconds=30`, `digits=6`, `window=1`,
  `secret_bytes=20`。
- WebAuthn: `rp_id_source`/`rp_origins_source=deployment_config`,
  `user_verification=preferred`, `attestation=none`, `resident_key=discouraged`,
  `sign_count_regression=reject`, `challenge_bytes=32`, `timeout_seconds=120`。
- リカバリコード: `count=10`, `code_length=10`,
  `alphabet="23456789abcdefghijkmnpqrstuvwxyz"`, `hash=sha256`, `single_use=true`,
  `regenerate_replaces_all=true`。
- パスワードリセットトークン: `ttl=1800s`, `single_use=true`。
- ログインセッション Cookie: `http_only=true`, `same_site=Lax`（対応する SCL model field を
  持たない純粋な transport/cookie 設定）。

## 却下した代替案

- 各設定値を対応する model field の `constraints` として無理にモデル化する: TOTP/WebAuthn の
  ceremony パラメータやスロットリングの閾値・バックオフ曲線は実在の field に対応しないものが
  多く、無理に押し込むと表現できない値を暗黙に丸めるか、実体のない model field を作る必要が
  生じる。[[ADR-103]] 自身が「config/tech 選択は ADR/ARCHITECTURE.md」としており、本 ADR の
  範囲で覆さない。
- `objectives` の新しい kind として残す: [[ADR-103]] の決定を覆さない。

## 影響

- `spec/contexts/identity-management.yaml` と `spec/contexts/authentication.yaml` の SCL 3.0 版は
  これらの `objectives` を持たず、本 ADR を credential/スロットリング設定の正本として参照する。
- 値そのものは変更しない。実装・runtime 挙動への影響はない。
