---
status: accepted
authors: [tn]
created_at: 2026-07-15
---

# ADR-108: SigningKeys の鍵ローテーション・保持期間設定を ARCHITECTURE 層の文書に移す

## コンテキスト

[[ADR-103]] は SCL 3.0 の `objectives` を観測可能な SLI に対する SLO だけに限定し、config/security
policy/retention 設定は ADR または `ARCHITECTURE.md` へ移すことを決定した。wi-210 で
`spec/contexts/signing-keys.yaml` (SCL 2.0) を SCL 3.0 へ移行した際、以下の旧 `invariants` /
`objectives` は `indicator` / `target` / `window` / `budgeting` を持つ観測可能な比率目標ではなく、
単一の設定値・運用方針の集合だった。値そのものは移行によって変更しない。

## 決定

### 1. 署名鍵の最大有効期間 (旧 `SigningKeyMaxAge`)

有効な署名鍵はテナントごとに 90 日を超えない間隔で回転する。回転は運用ジョブが行い、
`RotateTenantSigningKey` (手動即時回転) とは独立した定期処理。強制点となる単一の interface
`requires` / `ensures` は存在しないため、本 ADR を正本とする。

### 2. JWKS 最小重複期間 (旧 `SigningKeyMinJwksOverlap`)

鍵を Verifying から Retired へ移す (`SigningKeyLifecycle` の `Retire` transition) までに、
最低 7 日は新旧両鍵を JWKS に並存させる。

### 3. 署名鍵の監査保持期間 (旧 `SigningKeyArchiveRetention`)

`Archived` (終端状態) に達した鍵材料は、旧鍵で署名された監査トークンの検証のため 7 年間保持する。
削除・完全消去用の独立した interface は現時点で存在しない。

### 4. KeyProviderFailClosed の強制点は OAuth2 context に置く

旧 `invariants.KeyProviderFailClosed` (「KeyProvider が到達不能なテナントでは新規署名をしない」)
が実際に強制される箇所は、トークン発行を行う OAuth2 context の `Token` interface であり、
SigningKeys context 自体は署名・発行 interface を持たない。SigningKeys は `provider_healthy`
という観測可能な信号 (`TenantSigningKey.provider_healthy`、`ListTenantKeyHealth`) だけを所有し、
fail-closed という判断・強制点は wi-210 T004/T007 で `spec/contexts/oauth2.yaml` の `Token.requires`
と対応 scenario へ移す。

## 却下した代替案

- これらの値を `objectives` の新しい kind として残す: [[ADR-103]] の決定を覆さない。
- `SigningKeyMaxAge` を `SigningKey` model の field constraint として表現する: 「作成から90日以内に
  必ず新しい鍵が作られる」という複数レコードにまたがる運用要件であり、単一 entity の field
  constraint では表現できない。
- `KeyProviderFailClosed` を SigningKeys 内の scenario として保持する: SigningKeys にはトークン発行
  interface が無く、強制点のない context に normative behavior を残すと所有 context と実装が乖離する。

## 影響

- `spec/contexts/signing-keys.yaml` の SCL 3.0 版はこれらの `objectives` / `invariants` を持たず、
  本 ADR を鍵ローテーション・保持期間設定の正本として参照する。
- `spec/contexts/oauth2.yaml` の SCL 3.0 版 (wi-210 T004/T007) は `Token` interface の `requires` に
  signing key provider health の条件を追加し、対応する scenario を持つ。
- 値そのものは変更しない。実装・runtime 挙動への影響はない。
