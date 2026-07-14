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

### 1. IdentityManagement

- **CSV インポートの入力上限**: `ImportAdminUsers` は CSV を 1 MiB (1,048,576 bytes)、1,000 行、
  1 field あたり 64 KiB を上限として拒否する。この上限自体は `interfaces.ImportAdminUsers.requires`
  に反映済みで、「なぜ 1 MiB か」という運用上の根拠 (メモリ上限を守りつつ通常の管理者運用で
  十分な行数を許容する) を本 ADR に残す。
- **User soft-delete の猶予期間**: `PendingDeletion` から `Deleted` への自動遷移は 30 日
  (2,592,000 秒) を既定とする。Google / Microsoft / Apple 等の業界標準的な猶予期間 (7〜30 日) に
  合わせた値であり、`states.UserLifecycle` の `PendingDeletion → Deleted` transition guard に
  既に反映済み。

### 2. Authentication: パスワードポリシー

- `min_length=12`, `max_length=128`, `history_depth=5` (直近 5 件の履歴と一致するパスワードを拒否)。
  長さと履歴チェックの強制点は `interfaces.ChangePassword.requires` /
  `interfaces.ResetPasswordWithToken.requires` に反映済み。
- `forbid_user_identifier_similarity=true`: パスワードが username / email 等の識別子と類似する
  場合は拒否する。
- `common_password_dictionary=bundled`: バンドル済み一般的パスワード辞書との一致を拒否する。
- `breached_password_check_enabled=false`: 侵害済みパスワードデータベースとの照合は既定で無効。

### 3. Authentication: ログインスロットリング

- per-account: 直近 900 秒窓で失敗 10 回に達するとロックアウト (900 秒)。
- per-IP: 直近 900 秒窓で失敗 30 回に達するとロックアウト (900 秒)。
- `counter_scope=cluster_wide`: カウンタは全レプリカで共有する ([[ADR-105]] の
  `SharedEphemeralStateHA` が定める Valkey 共有ストアに login_throttle を含めているのはこのため)。
- `identifier_hash=sha256`: カウンタキーに使う識別子 (username/IP) は sha256 でハッシュ化する。
- `shared_store_required_for_multi_replica=true`, `degraded_store_behavior=fail_closed`: 共有
  ストア到達不能時はログインを拒否側 (fail_closed) に倒す。強制点自体は
  `interfaces.SubmitBrowserLogin.requires` (`!context.login_throttled`) に反映済み。

### 4. Authentication: TOTP パラメータ (RFC 6238)

- `algorithm=SHA1`, `step_seconds=30`, `digits=6`, `window=1` (前後 1 ステップまで許容),
  `secret_bytes=20` (160 bit)。

### 5. Authentication: WebAuthn パラメータ

- `rp_id_source` / `rp_origins_source=deployment_config` (デプロイ環境の設定から解決),
  `user_verification=preferred`, `attestation=none`, `resident_key=discouraged`,
  `sign_count_regression=reject` (sign counter の逆行はクローン検知として拒否),
  `challenge_bytes=32`, `timeout_seconds=120`。

### 6. Authentication: リカバリコード

- `count=10` (発行数), `code_length=10`, `alphabet="23456789abcdefghijkmnpqrstuvwxyz"`
  (視認性の低い文字を除いた base32 風アルファベット), `hash=sha256` (保存時にハッシュ化),
  `single_use=true`, `regenerate_replaces_all=true` (再生成は既存コード全体を置き換える)。

### 7. Authentication: パスワードリセットトークン

- `ttl=1800s` (30分), `single_use=true`。`models.PasswordResetTokenRecord.fields.expires_at` の
  description と `interfaces.ResetPasswordWithToken.requires` に反映済み。本 ADR には数値のみ記録する。

### 8. Authentication: ログインセッション Cookie

- `http_only=true`, `same_site=Lax`。純粋な transport/cookie 設定であり、対応する SCL model field
  を持たない。

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
