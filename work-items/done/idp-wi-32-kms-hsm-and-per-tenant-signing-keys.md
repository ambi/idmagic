---
id: idp-wi-32-kms-hsm-and-per-tenant-signing-keys
title: "HSM / KMS 実鍵管理と per-tenant signing keys を導入する"
created_at: 2026-06-20
authors: ["tn"]
status: completed
risk: high
---

# Motivation
README は現状、署名鍵がテナント間で共有されるため、RP が `iss` を厳格検証しないと
tenant 間で token を誤受理しうる前提を明記している。本番マルチテナント IdP としては、
per-tenant signing key と実 KMS/HSM による秘密鍵管理が必要である。

あわせて、[[wi-36-oauth2-audit-event-tenant-scoping]] から繰り延べた残課題を
本 WI で回収する: `SigningKeyRotated` イベントは現状の「インスタンス全体で
署名鍵 1 系統」モデルでは帰属テナントが定義できず、emit 時 tenant_id 付与
([[wi-35-audit-event-tenant-scoping]] / wi-36 の方針) の対象外として
tenant_id 空のまま記録されている。そのためテナント所属 admin の監査ビュー
(`/admin/audit_events`) に鍵ローテーションが出てこない。per-tenant 鍵を
入れる本 WI で `SigningKeyRotated` に帰属テナントが定まるので、同イベントにも
tenant_id を載せ、wi-35/wi-36 で揃えた「監査イベントはテナント帰属を持つ」
原則を全イベントで満たす。

# Scope
- **decision**:
  - 新規 ADR: per-tenant key の lifecycle、kid 命名、JWKS URL、rotation cadence、 KMS/HSM adapter 方針、local/postgres fallback の位置付けを定義する。
- **scl**:
  - TenantSigningKey / KeyProvider / KeyUsage / KeyLifecycle objective を追加する。
  - RotateTenantSigningKey / ListTenantJwks / DisableTenantKey を追加する。
  - `SigningKeyRotated` event の payload に `tenantId: { type: String }` を 追加する (wi-36 から繰り延べた残課題)。per-tenant 鍵の帰属テナントを表す。 spec/scl.yaml の event 定義と internal/spec/events.go の struct を twin で 更新し、spec HTML を再生成する。
- **go**:
  - KeyStore port を tenant-aware に変更する。
  - `/realms/{tenant_id}/jwks` は当該 tenant の active + retired-not-expired 鍵だけ返す。
  - token issuer は issuer tenant の signing key を選ぶ。
  - KMS adapter を 1 つ選定して実装する。初期は AWS KMS または GCP Cloud KMS のどちらか 1 つ。
  - local/postgres KeyStore は dev/test 用 fallback として維持する。
  - key rotation scheduler (`wi-23`) を tenant-aware に拡張する。
  - `internal/oauth2/usecases/rotate_signing_key.go` の `SigningKeyRotated` emit に、回転対象鍵の帰属テナント (tenant-aware KeyStore から取得) を `TenantID` として載せる。これによりテナント所属 admin の監査ビューに鍵ローテーションが出るようになる (wi-35/wi-36 と同じ emit 時 tenant_id 方針)。
- **ui**:
  - admin keys page に tenant key provider / active kid / rotation 状態を表示する。
  - system_admin は tenant ごとの key health を一覧できるようにする。
- **documentation**:
  - README の「署名鍵はテナント間で共有される」注意を完了記録で除去する。
  - KMS の IAM 権限、key policy、ローカル fallback の注意を書く。

# Out of Scope
- BYOK / customer-managed per-customer key UI。
- encrypted ID Token / JWE。
- SAML assertion key の統合。SAML は別 WI で扱う。

# Verification
- `go test ./...` (in: idmagic)
- `golangci-lint run ./...` (in: idmagic)
- `go build ./...` (in: idmagic)
- `bun --cwd idmagic/ui typecheck`
- `bun --cwd idmagic/ui lint`
- `bun --cwd idmagic/ui build`
- 手動: tenant A/B で token を発行し、それぞれの `/jwks` に別 kid が出ることを確認する。
- 手動: tenant A の JWKS だけでは tenant B token の kid を解決できないことを確認する。

# Risk Notes
KeyStore port の変更は token issuer、discovery、rotation、admin key UI に波及する。
まず local/postgres を tenant-aware にして全テストを通し、その後 KMS adapter を追加する。

# Completion
- **Completed At**: 2026-07-03
- **Summary**:
  ADR-075 で signing key を pluggable な `KeyProvider` の背後で per-tenant に
  scope することを採用した。SCL には KeyProvider / KeyUsage / TenantSigningKey /
  TenantKeyHealth と RotateTenantSigningKey / DisableTenantKey / ListTenantJwks /
  ListTenantKeyHealth interface、TenantJwksIsolation / KeyProviderFailClosed
  invariant、SystemKeyHealthRead permission を追加し、`SigningKeyRotated` payload
  に `tenantId` を twin (scl.yaml + events.go) で足して spec HTML を再生成した。

  Go では KeyStore port を tenant-aware にし (ctx の tenancy.TenantID で解決)、
  `/realms/{tenant_id}/jwks` は当該 tenant の鍵だけを返す。InMemory / postgres
  KeyStore を tenant scope 化し、rotate_signing_key.go は帰属テナント付きで
  SigningKeyRotated を emit する。KMS adapter は当初案の AWS/GCP クラウド KMS を
  採らず、ユーザ判断により self-host OSS スタックと整合する HashiCorp Vault
  Transit を選定した (ADR-075 で cloud KMS を rejected として記録)。VaultKeyStore
  は秘密鍵を Vault 外に出さず署名を transit/sign へ委譲し、公開鍵だけをミラーする。
  jwt_signer は crypto.Signer 抽象に寄せ、local RSA と Vault 署名を両対応にした。
  KEY_PROVIDER=vault で選択し、未設定時は local/postgres を dev/test fallback とする。

  UI は admin keys ページに provider 列 / detail と緊急 disable を足し、rotate を
  自テナント admin/system_admin に開放、system_admin 向けに /admin/keys/health で
  テナント別 provider health を一覧する画面を追加した。README には per-tenant 鍵の
  isolation、Vault Transit の運用 (Transit engine 有効化 / key policy / fail-closed /
  local fallback) と KEY_PROVIDER / VAULT_* 環境変数を追記した。
- **Verification Results**:
  - `go test ./...` (in: idmagic)
    - result: 全パッケージ pass。tenant JWKS isolation / Vault fake / rotate tenantId のテストを含む。
  - `golangci-lint run ./...` (in: idmagic)
    - result: 0 issues。
  - `bun run build` (in: idmagic/ui)
    - result: typecheck / lint / build pass。route tree に /admin/keys/health を生成。
  - `just yaml-check-scl` (in: .)
    - result: SCL 全ファイル OK。spec HTML / JSON Schema / OpenAPI を再生成。
- **Affected Guarantees State**:
  - guarantee: tenant isolation: tenant A の JWKS に tenant B の kid が出ない。
  - state: passed
  - guarantee: key secrecy: KMS/HSM 利用時、秘密鍵 material を app DB に保存しない。
  - state: passed
  - guarantee: rotation continuity: grace 期間中の旧鍵は tenant JWKS に残る。
  - state: passed
  - guarantee: fail-closed: key provider failure 時は新規 token 発行を止める。
  - state: passed
  - guarantee: audit: key create / rotate / retire / provider error を監査に残す。
  - state: passed
  - guarantee: audit tenant-scoping: `SigningKeyRotated` が tenant_id を持ち、テナント 所属 admin が自テナントの鍵ローテーションを `/admin/audit_events` で確認できる。
  - state: passed
