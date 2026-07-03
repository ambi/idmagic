# idp-ADR-075: Scope signing keys per tenant behind a pluggable KeyProvider

## ステータス
採用。`spec/contexts/signing-keys.yaml` の `models.KeyProvider` / `models.KeyUsage` / `models.SigningKey` (tenant_id) / `models.TenantSigningKey` / `models.SigningKeyRotated` (tenantId) / `interfaces.ListTenantJwks` / `interfaces.RotateTenantSigningKey` / `interfaces.DisableTenantKey` と、`internal/oauth2/ports/key_store.go` ほか Layer 3/4 実装に反映。

## コンテキスト
現状の署名鍵は instance 全体で 1 系統であり、テナント間で共有される。RP が `iss` を厳格検証しない限り、tenant A の active 鍵で署名した token を tenant B の RP が誤受理しうる。`/jwks` は global KeyStore の全鍵を返し、`KeyStore` port (`GetActiveKey` / `GetAllKeys` / `FindByKID` / `Rotate`) は tenant を引数に取らない。秘密鍵マテリアルは in-memory か app DB (`signing_keys` テーブル) に平文で置かれ、実 HSM/KMS を使う経路が無い。

本番マルチテナント IdP としては、(1) テナントごとに独立した署名鍵、(2) 秘密鍵マテリアルをアプリ外へ出さない鍵管理、が要る。あわせて [[idp-wi-36-oauth2-audit-event-tenant-scoping]] から繰り延べた残課題 — `SigningKeyRotated` に帰属テナントが定義できずテナント所属 admin の監査ビューに出ない問題 — を、鍵がテナント帰属を持つ本 ADR で回収する。

鍵管理バックエンドはクラウド KMS (AWS KMS / GCP Cloud KMS) も候補だが、本リポジトリは self-host OSS を default stack とし (PostgreSQL / Valkey)、デモとして特定クラウド SDK とアカウント前提を持ち込みたくない。

## 決定
1. **SigningKey をテナント帰属にする。** `SigningKey` に `tenant_id` を持たせ、`KeyStore` の全操作を tenant で絞る。`kid` は引き続き公開鍵の RFC 7638 thumbprint とし identity に用いるが、列挙・検索・回転・署名鍵選択はすべて tenant 単位で行う。tenant A の JWKS に tenant B の kid は出ない。
2. **KeyProvider を導入する。** 鍵マテリアルの保管場所と署名の実行主体を `KeyProvider` enum で抽象化する: `Local` (in-memory、dev/test)、`Postgres` (app DB に private material、dev/test)、`VaultTransit` (HashiCorp Vault Transit secrets engine。private material は Vault 外に出ず、署名は Vault API 経由で行う)。実 KMS/HSM 相当の鍵管理は `VaultTransit` が満たす。
3. **KeyUsage を導入する。** 当面は `Signing` (OAuth2/OIDC の JWT 署名) のみ。JWE 用暗号鍵や SAML/WS-* の X.509 は本 ADR の範囲外だが、鍵を usage 次元で拡張できる語彙を用意する。
4. **kid にテナントを埋め込まない。** kid は RFC 7638 thumbprint のまま (鍵マテリアルが同じなら移送しても kid は不変)。テナント scoping は格納次元で行う: postgres は `tenant_id` 列、Vault は key name の prefix (`idmagic/{tenant_id}/{kid}` 相当)。
5. **per-tenant JWKS URL は `/realms/{tenant_id}/jwks`。** 当該 tenant の active + verifying (retired-not-expired) 鍵のみ返す。既存 global `/jwks` は default tenant 互換のため維持する (後方互換)。
6. **rotation cadence はテナントごとに評価する。** `SigningKeyMaxAge` (90d) / `SigningKeyMinJwksOverlap` (7d) はテナント単位で評価し、rotation scheduler ([[idp-wi-23-signing-key-rotation-scheduler]]) はテナントごとに回す。
7. **provider 障害時は fail-closed。** KeyProvider (特に Vault) が不達のとき、新規 token 発行を停止する。既発行 token 検証のための JWKS は、取得可能な範囲 (DB/キャッシュのミラー) で返す。
8. **local/postgres は dev/test fallback。** 本番 provider は `VaultTransit`。`Local` / `Postgres` は private material をアプリプロセス / app DB に置く dev/test 用に維持する。
9. **`SigningKeyRotated` に `tenantId` を載せる。** 回転対象鍵の帰属テナントを emit 時に付与し、テナント所属 admin が `/admin/audit_events` で自テナントの鍵ローテーションを確認できるようにする ([[idp-wi-35-audit-event-tenant-scoping]] / wi-36 と同じ emit 時 tenant_id 方針)。

## KeyProvider の署名モデル
`Local` / `Postgres` は private key をプロセス内に持ち、アプリが直接署名する。`VaultTransit` は private key を Vault 内に生成・保持し、署名要求ごとに Vault の `transit/sign/{key}` を呼ぶ。アプリが保持するのは公開鍵 (JWKS 用) のみで、秘密鍵マテリアルは app DB に保存しない (key secrecy)。公開鍵は JWKS 配布と fail-closed 時の検証継続のためにアプリ側へミラーする。

## 却下した代替案
- **Cloud KMS (AWS KMS / GCP Cloud KMS) を直接実装:** cloud SDK 依存と特定クラウドのアカウント / IAM 前提をデモに持ち込む。self-host OSS の default stack と不整合。→ Vault Transit を採用し、KeyProvider 抽象で将来 cloud KMS を足せる余地は残す。
- **kid に tenant prefix を埋め込む:** RFC 7638 thumbprint の安定性を壊し、同一マテリアルの移送で kid が変わる。→ 格納次元でテナント scope。
- **現状維持で iss 厳格検証に委ねる:** RP 実装依存で cross-tenant 誤受理を防げない。→ per-tenant key で構造的に防ぐ。
- **global `/jwks` を廃止し per-tenant のみにする:** 既存 default tenant クライアントを壊す。→ 維持しつつ per-tenant を追加。

## 影響
- **SCL:** `SigningKey` に `tenant_id` / `provider` / `usage`。`KeyProvider` / `KeyUsage` enum。`TenantSigningKey` (テナント鍵ヘルスの value_object)。`ListTenantJwks` / `RotateTenantSigningKey` / `DisableTenantKey` interface。per-tenant 権限 (`TenantKeysRotate` / `TenantKeysDisable` / `SystemKeyHealthRead`)。`SigningKeyRotated` に `tenantId`。
- **Go:** `KeyStore` port を tenant-aware に変更。`jwt_signer` はテナント鍵を選ぶ。discovery に `/realms/{tenant_id}/jwks`。`rotate_signing_key` usecase は帰属テナントを `TenantID` として emit。`internal/shared/spec/events.go` の `SigningKeyRotated` に `TenantID`。Vault Transit adapter を追加。postgres schema に `tenant_id` 列。
- **UI:** admin keys 画面に provider / active kid / rotation 状態。system_admin 向けのテナント別 key health 一覧。
- **運用:** Vault の Transit engine 有効化と tenant ごとの key name、fail-closed 時の挙動、local fallback の注意を README に記す。README の「署名鍵はテナント間で共有される」注意は本 WI 完了記録で除去する。
